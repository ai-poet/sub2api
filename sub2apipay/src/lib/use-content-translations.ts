'use client';

/**
 * 内容自动翻译的浏览器端：按原文查译文，查到就显示译文，查不到就显示原文。
 * 只用于渲染——接口数据、下单参数一律保持原文（见 docs/CONTENT_TRANSLATION.md）。
 */
import { useCallback, useEffect, useMemo, useSyncExternalStore } from 'react';
import type { Locale } from '@/lib/locale';
import { buildAppApiPath } from '@/lib/public-path';

export type Translate = (text: string) => string;

/** 支付服务查询接口单次最多 200 条、请求体最多 256 KiB */
const MAX_BATCH_TEXTS = 200;
const MAX_BATCH_BYTES = 240 * 1024;
/** 后端会忽略超过这个长度的单条文本 */
const MAX_TEXT_LENGTH = 20_000;
/** `pending` 时隔 10 秒补查缺的文本，最多 3 次 */
const RETRY_DELAY_MS = 10_000;
const MAX_RETRIES = 3;

type TranslationMap = ReadonlyMap<string, string>;

interface LocaleStore {
  /** 原文（去首尾空白）→ 译文；每次更新都换一个新 Map，供 useSyncExternalStore 比较 */
  translations: TranslationMap;
  /** 请求过的文本（有结论、正在请求或等待补查），本页生命周期内不再重复请求 */
  requested: Set<string>;
  listeners: Set<() => void>;
}

const EMPTY_TRANSLATIONS: TranslationMap = new Map();
const stores = new Map<Locale, LocaleStore>();

function getStore(locale: Locale): LocaleStore {
  let store = stores.get(locale);
  if (!store) {
    store = { translations: EMPTY_TRANSLATIONS, requested: new Set(), listeners: new Set() };
    stores.set(locale, store);
  }
  return store;
}

const HAN = /\p{Script=Han}/u;
const KANA = /[\p{Script=Hiragana}\p{Script=Katakana}]/u;
const HANGUL = /\p{Script=Hangul}/u;
const LETTER = /\p{L}/u;

/** 文本看起来已经是目标语言：zh = 有汉字且没有假名；en = 不含中日韩文字 */
export function looksLikeLocale(text: string, locale: Locale): boolean {
  const hasHan = HAN.test(text);
  const hasKana = KANA.test(text);
  if (locale === 'zh') return hasHan && !hasKana;
  return !hasHan && !hasKana && !HANGUL.test(text);
}

/** 需要查译文的文本：去首尾空白、去空、去重，跳过已是目标语言或不含文字的 */
export function pickTextsToTranslate(texts: readonly unknown[], locale: Locale): string[] {
  const seen = new Set<string>();
  const picked: string[] = [];
  for (const value of texts) {
    if (typeof value !== 'string') continue;
    const text = value.trim();
    if (!text || text.length > MAX_TEXT_LENGTH || seen.has(text)) continue;
    if (!LETTER.test(text) || looksLikeLocale(text, locale)) continue;
    seen.add(text);
    picked.push(text);
  }
  return picked;
}

function splitIntoBatches(texts: string[]): string[][] {
  const encoder = new TextEncoder();
  const batches: string[][] = [];
  let current: string[] = [];
  let bytes = 0;
  for (const text of texts) {
    const size = encoder.encode(JSON.stringify(text)).length + 1;
    if (current.length > 0 && (current.length >= MAX_BATCH_TEXTS || bytes + size > MAX_BATCH_BYTES)) {
      batches.push(current);
      current = [];
      bytes = 0;
    }
    current.push(text);
    bytes += size;
  }
  if (current.length > 0) batches.push(current);
  return batches;
}

async function fetchBatch(locale: Locale, texts: string[], attempt: number): Promise<void> {
  const store = getStore(locale);
  let received: Record<string, unknown> = {};
  let pending = false;
  try {
    const response = await fetch(buildAppApiPath('/api/content-translations'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ lang: locale, texts }),
    });
    if (response.ok) {
      const data = (await response.json()) as { translations?: unknown; pending?: unknown } | null;
      if (data?.translations && typeof data.translations === 'object' && !Array.isArray(data.translations)) {
        received = data.translations as Record<string, unknown>;
      }
      pending = data?.pending === true;
    }
  } catch {
    // 查不到就显示原文
  }

  const asked = new Set(texts);
  let next: Map<string, string> | null = null;
  for (const [source, translated] of Object.entries(received)) {
    if (!asked.has(source) || typeof translated !== 'string' || translated.trim() === '') continue;
    if (store.translations.get(source) === translated) continue;
    next ??= new Map(store.translations);
    next.set(source, translated);
  }
  if (next) {
    store.translations = next;
    store.listeners.forEach((listener) => listener());
  }

  const missing = texts.filter((text) => !store.translations.has(text));
  if (pending && missing.length > 0 && attempt < MAX_RETRIES) {
    setTimeout(() => void fetchBatch(locale, missing, attempt + 1), RETRY_DELAY_MS);
  }
}

function requestTranslations(locale: Locale, texts: string[]): void {
  const store = getStore(locale);
  const fresh = texts.filter((text) => !store.requested.has(text));
  if (fresh.length === 0) return;
  for (const text of fresh) store.requested.add(text);
  for (const batch of splitIntoBatches(fresh)) void fetchBatch(locale, batch, 0);
}

const getServerSnapshot = (): TranslationMap => EMPTY_TRANSLATIONS;

/**
 * 给一组要显示的文案查译文，返回 `tx(text)`：有译文返回译文，否则原样返回。
 * 文案列表变化时一次批量查询；结果按语言缓存在模块里，组件重新挂载不会重复请求。
 */
export function useContentTranslations(locale: Locale, texts: readonly unknown[]): Translate {
  const subscribe = useCallback(
    (listener: () => void) => {
      const store = getStore(locale);
      store.listeners.add(listener);
      return () => {
        store.listeners.delete(listener);
      };
    },
    [locale],
  );
  const getSnapshot = useCallback(() => getStore(locale).translations, [locale]);
  const translations = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

  // 内容不变时 key 不变，数组引用变化不会触发重复请求
  const key = useMemo(() => JSON.stringify(pickTextsToTranslate(texts, locale)), [texts, locale]);

  useEffect(() => {
    const wanted = JSON.parse(key) as string[];
    if (wanted.length > 0) requestTranslations(locale, wanted);
  }, [key, locale]);

  return useCallback<Translate>(
    (text) => {
      if (typeof text !== 'string') return text;
      return translations.get(text.trim()) ?? text;
    },
    [translations],
  );
}
