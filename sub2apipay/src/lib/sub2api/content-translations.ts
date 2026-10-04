/**
 * 内容自动翻译（fork 本地功能，接口约定见仓库根目录 docs/CONTENT_TRANSLATION.md）。
 *
 * 支付服务把管理员手写的文案（在售套餐、启用中的活动与渠道、PAY_HELP_TEXT）登记给后端，
 * 后端负责翻译并永久缓存；浏览器显示时再按原文查译文。
 *
 * 只在显示层替换：这里不改任何接口返回——桌面客户端按原始套餐名 / 描述分流，替换会让分流出错。
 */
import { prisma } from '@/lib/db';
import { getEnv } from '@/lib/config';
import { getInternalPayHeaders } from '@/lib/internal-auth';
import { buildInternalUrl } from './client';

/** 后端 `PUT /api/internal/pay/content-translations/sources` 的条数上限 */
export const MAX_PAY_TRANSLATION_SOURCES = 1000;
/** 后端公开查询接口的条数上限 */
export const MAX_LOOKUP_TEXTS = 200;
/** 后端会忽略超过这个长度的单条文本 */
export const MAX_TRANSLATION_TEXT_LENGTH = 20_000;

const REQUEST_TIMEOUT_MS = 10_000;
const SYNC_DEBOUNCE_MS = 2_000;
const STALE_SYNC_INTERVAL_MS = 10 * 60 * 1000;

const SOURCES_PATH = '/api/internal/pay/content-translations/sources';
const LOOKUP_PATH = '/api/v1/content-translations/lookup';

export interface ContentTranslationLookup {
  translations: Record<string, string>;
  pending: boolean;
}

/**
 * 套餐 / 渠道的 features 存的是 JSON 字符串数组（管理端按行拆分后 JSON.stringify）。
 * 兼容 `{ text }` 形式的条目；解析失败或不是数组时当作没有特性。
 */
export function parseFeatureTexts(raw: string | null | undefined): string[] {
  if (!raw) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }
  if (!Array.isArray(parsed)) return [];
  const texts: string[] = [];
  for (const item of parsed) {
    if (typeof item === 'string') {
      texts.push(item);
    } else if (item && typeof item === 'object' && typeof (item as { text?: unknown }).text === 'string') {
      texts.push((item as { text: string }).text);
    }
  }
  return texts;
}

/** 收集支付服务需要登记翻译的文案：去首尾空白、去空、去重，最多 1000 条 */
export async function collectPayTranslationSources(now: Date = new Date()): Promise<string[]> {
  const [plans, promotions, channels] = await Promise.all([
    prisma.subscriptionPlan.findMany({
      where: { forSale: true },
      orderBy: { sortOrder: 'asc' },
      select: { name: true, description: true, features: true, productName: true },
    }),
    // 启用中且尚未结束的活动（含还没开始的，开始前就译好）
    prisma.rechargePromotion.findMany({
      where: { enabled: true, OR: [{ endsAt: null }, { endsAt: { gt: now } }] },
      orderBy: { sortOrder: 'asc' },
      select: { name: true, description: true },
    }),
    prisma.channel.findMany({
      where: { enabled: true },
      orderBy: { sortOrder: 'asc' },
      select: { name: true, description: true, features: true },
    }),
  ]);

  const seen = new Set<string>();
  const texts: string[] = [];
  const add = (value: string | null | undefined) => {
    if (typeof value !== 'string') return;
    const text = value.trim();
    if (!text || text.length > MAX_TRANSLATION_TEXT_LENGTH || seen.has(text)) return;
    seen.add(text);
    texts.push(text);
  };

  // 帮助文案每个页面都显示，排在最前面，避免被条数上限截掉
  add(getEnv().PAY_HELP_TEXT);
  for (const plan of plans) {
    add(plan.name);
    add(plan.description);
    for (const feature of parseFeatureTexts(plan.features)) add(feature);
    add(plan.productName);
  }
  for (const promotion of promotions) {
    add(promotion.name);
    add(promotion.description);
  }
  for (const channel of channels) {
    add(channel.name);
    add(channel.description);
    for (const feature of parseFeatureTexts(channel.features)) add(feature);
  }

  return texts.slice(0, MAX_PAY_TRANSLATION_SOURCES);
}

/** 把当前文案全量登记给后端（后端按全量替换处理）。失败只记日志，从不抛给调用方。 */
export async function syncPayTranslationSources(): Promise<boolean> {
  try {
    const texts = await collectPayTranslationSources();
    const response = await fetch(buildInternalUrl(SOURCES_PATH), {
      method: 'PUT',
      headers: getInternalPayHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ texts }),
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    });
    if (!response.ok) {
      console.warn(`[content-translations] source sync failed: HTTP ${response.status}`);
      return false;
    }
    return true;
  } catch (error) {
    console.warn(
      '[content-translations] source sync failed:',
      error instanceof Error ? error.message : String(error),
    );
    return false;
  }
}

// 进程级状态挂在 globalThis 上：开发模式热重载、不同路由的打包副本共用同一份
interface SyncState {
  timer: ReturnType<typeof setTimeout> | null;
  lastSyncAt: number;
  chain: Promise<unknown>;
}

const globalForSync = globalThis as unknown as { __payContentTranslationSync?: SyncState };

function syncState(): SyncState {
  if (!globalForSync.__payContentTranslationSync) {
    globalForSync.__payContentTranslationSync = { timer: null, lastSyncAt: 0, chain: Promise.resolve() };
  }
  return globalForSync.__payContentTranslationSync;
}

/** 串行执行：后发起的同步一定在前一次之后发出，最后一次 PUT 反映最新数据 */
function runSync(): Promise<boolean> {
  const state = syncState();
  state.lastSyncAt = Date.now();
  const next = state.chain.then(() => syncPayTranslationSources());
  state.chain = next.catch(() => undefined);
  return next;
}

/** 管理端写操作之后调用：2 秒防抖，一串连续修改只同步一次。不等待、不抛错。 */
export function schedulePayTranslationSync(): void {
  const state = syncState();
  if (state.timer) clearTimeout(state.timer);
  const timer = setTimeout(() => {
    state.timer = null;
    void runSync();
  }, SYNC_DEBOUNCE_MS);
  // 不让一个待发的同步拖住进程退出
  (timer as unknown as { unref?: () => void }).unref?.();
  state.timer = timer;
}

/** 用户页面接口里调用：本进程每 10 分钟最多同步一次（启动后第一次调用一定会同步）。不等待、不抛错。 */
export function syncPayTranslationSourcesIfStale(): void {
  try {
    const state = syncState();
    if (state.lastSyncAt > 0 && Date.now() - state.lastSyncAt < STALE_SYNC_INTERVAL_MS) return;
    void runSync();
  } catch (error) {
    console.warn(
      '[content-translations] stale sync skipped:',
      error instanceof Error ? error.message : String(error),
    );
  }
}

/** 仅供测试：清掉防抖定时器与节流时间戳 */
export function resetPayTranslationSyncStateForTests(): void {
  const state = syncState();
  if (state.timer) clearTimeout(state.timer);
  state.timer = null;
  state.lastSyncAt = 0;
  state.chain = Promise.resolve();
}

/**
 * 查后端的译文缓存（公开只读接口，不会触发模型调用）。失败时返回空结果，调用方显示原文即可。
 * 附带内部令牌：所有用户的查询都从支付服务这一个 IP 发出，后端可据此区分。
 */
export async function lookupContentTranslations(lang: string, texts: string[]): Promise<ContentTranslationLookup> {
  const empty: ContentTranslationLookup = { translations: {}, pending: false };
  if (texts.length === 0) return empty;

  try {
    const response = await fetch(buildInternalUrl(LOOKUP_PATH), {
      method: 'POST',
      headers: getInternalPayHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ lang, texts }),
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    });
    if (!response.ok) {
      console.warn(`[content-translations] lookup failed: HTTP ${response.status}`);
      return empty;
    }

    const payload = (await response.json()) as { data?: { translations?: unknown; pending?: unknown } } | null;
    const data = payload?.data;
    const raw = data?.translations;
    const requested = new Set(texts);
    const entries =
      raw && typeof raw === 'object' && !Array.isArray(raw)
        ? Object.entries(raw as Record<string, unknown>).filter(
            (entry): entry is [string, string] =>
              requested.has(entry[0]) && typeof entry[1] === 'string' && entry[1].trim() !== '',
          )
        : [];

    return {
      // Object.fromEntries 建的是自有属性，"__proto__" 这类键也不会改原型
      translations: Object.fromEntries(entries),
      pending: data?.pending === true,
    };
  } catch (error) {
    console.warn('[content-translations] lookup failed:', error instanceof Error ? error.message : String(error));
    return empty;
  }
}
