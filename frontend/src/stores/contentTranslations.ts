import { defineStore } from 'pinia'
import { ref } from 'vue'
import type {
  ContentTranslationLang,
  lookupContentTranslations as LookupFn
} from '@/api/contentTranslations'

// 内容自动翻译（fork 本地功能）：管理员手写文案的译文缓存，只服务于显示层。
//
// 组件在渲染时调 translate()（经 composables/useContentTranslation.ts 的 tx()）：命中缓存返回译文，
// 否则返回原文并把文本排进查询队列。队列 50 ms 防抖后按 ≤200 条分批查询公开接口；接口说
// pending 时隔 10 秒把仍没有译文的文本再查一次，最多 3 次。查过但没有译文的文本记下来，
// 不会每次渲染都重查。译文到达时 version +1，读过它的渲染 / computed 随之刷新。
//
// 绝不要把 translate() 的结果写回数据对象，也不要拿它去做搜索、过滤、路由或 revision 计算。
//
// 本模块刻意不静态 import `@/i18n` 与 `@/api/*`：GroupBadge 这类基础组件都会经 tx() 引到这里，
// 而很多组件测试只 mock 了 vue-i18n 的 useI18n；静态引入 api client（它 import `@/i18n`，
// 后者在模块求值时调 createI18n）会让这些测试在 import 阶段就失败。语言由调用方传入
// （组件里取 $i18n.locale，见 composable），查询接口在第一次真正查询时按需加载。

export const CONTENT_TRANSLATION_DEBOUNCE_MS = 50
export const CONTENT_TRANSLATION_PENDING_RETRY_MS = 10_000
export const CONTENT_TRANSLATION_PENDING_MAX_RETRIES = 3
/** 单次查询最多 200 条（后端同一上限） */
export const CONTENT_TRANSLATION_LOOKUP_MAX_TEXTS = 200
/** 单条超过 20 000 字符后端直接忽略 */
export const CONTENT_TRANSLATION_MAX_TEXT_LENGTH = 20000
/** 查询失败的文本冷却这么久后，允许下一次渲染重新查询 */
const FAILURE_COOLDOWN_MS = 60_000
/** 请求体上限 256 KiB；按每字符 3 字节粗估，给 JSON 转义留余量 */
const MAX_CHUNK_BYTES = 192 * 1024

const HAN_RE = /\p{Script=Han}/u
const KANA_RE = /[\p{Script=Hiragana}\p{Script=Katakana}]/u
const CJK_OR_KANA_RE = /[\p{Script=Han}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Hangul}]/u

/** zh* → zh，ja* → ja，en* → en；其他值返回 null */
export function normalizeContentLang(value: unknown): ContentTranslationLang | null {
  if (typeof value !== 'string') return null
  const lower = value.trim().toLowerCase()
  if (lower.startsWith('zh')) return 'zh'
  if (lower.startsWith('ja')) return 'ja'
  if (lower.startsWith('en')) return 'en'
  return null
}

/** 没有组件上下文时的语言兜底：i18n 初始化 / 切换语言时会同步写 <html lang> */
export function documentContentLang(): ContentTranslationLang {
  try {
    return normalizeContentLang(document.documentElement.getAttribute('lang')) ?? 'en'
  } catch {
    return 'en'
  }
}

/**
 * 不值得查询的文本：空白、超长、或一眼看上去已经是当前语言（便宜的文字系统判断）。
 * zh：含汉字且不含假名；en：不含任何中日韩文字；ja：含假名。
 */
export function shouldSkipContentTranslation(text: string, lang: ContentTranslationLang): boolean {
  if (!text || !text.trim()) return true
  if (text.length > CONTENT_TRANSLATION_MAX_TEXT_LENGTH) return true
  if (lang === 'zh') return HAN_RE.test(text) && !KANA_RE.test(text)
  if (lang === 'en') return !CJK_OR_KANA_RE.test(text)
  if (lang === 'ja') return KANA_RE.test(text)
  return false
}

function chunkTexts(texts: string[]): string[][] {
  const chunks: string[][] = []
  let current: string[] = []
  let bytes = 0
  for (const text of texts) {
    const size = text.length * 3 + 8
    if (
      current.length > 0 &&
      (current.length >= CONTENT_TRANSLATION_LOOKUP_MAX_TEXTS || bytes + size > MAX_CHUNK_BYTES)
    ) {
      chunks.push(current)
      current = []
      bytes = 0
    }
    current.push(text)
    bytes += size
  }
  if (current.length > 0) chunks.push(current)
  return chunks
}

function bucket<T>(map: Map<ContentTranslationLang, T>, lang: ContentTranslationLang, create: () => T): T {
  let value = map.get(lang)
  if (value === undefined) {
    value = create()
    map.set(lang, value)
  }
  return value
}

let lookupLoader: Promise<typeof LookupFn> | null = null

function loadLookup(): Promise<typeof LookupFn> {
  if (!lookupLoader) {
    lookupLoader = import('@/api/contentTranslations').then((mod) => mod.lookupContentTranslations)
    lookupLoader.catch(() => {
      lookupLoader = null
    })
  }
  return lookupLoader
}

// 模块级：同一页面会话里查询失败只打一次日志，避免接口不可用时刷屏
let failureLogged = false

export const useContentTranslationsStore = defineStore('contentTranslations', () => {
  /** 缓存里每多出 / 变化一条译文就 +1；translate() 读它，译文到达后依赖它的视图会刷新 */
  const version = ref(0)

  // 以下都不是响应式状态：translate() 在渲染中调用，登记查询不能反过来触发重新渲染
  const cache = new Map<ContentTranslationLang, Map<string, string>>()
  /** 已排队 / 查询中 / 查过但没有译文（含等待 pending 重查、失败冷却中）的文本 */
  const requested = new Map<ContentTranslationLang, Set<string>>()
  const queue = new Map<ContentTranslationLang, Set<string>>()
  let flushTimer: ReturnType<typeof setTimeout> | null = null

  function cached(text: string, lang: ContentTranslationLang): string | undefined {
    return cache.get(lang)?.get(text)
  }

  /**
   * 返回 text 在 lang 下的译文；没有译文时返回原文，并在需要时登记一次查询。
   * 只用于显示。lang 缺省时取 <html lang>。
   */
  function translate(text: string | null | undefined, lang: ContentTranslationLang = documentContentLang()): string {
    if (text === null || text === undefined) return ''
    if (shouldSkipContentTranslation(text, lang)) return text
    // 读 version 建立响应式依赖：译文到达后调用方会重新求值
    void version.value
    const hit = cached(text, lang)
    if (hit !== undefined) return hit
    request(text, lang)
    return text
  }

  function request(text: string, lang: ContentTranslationLang) {
    const seen = bucket(requested, lang, () => new Set<string>())
    if (seen.has(text)) return
    seen.add(text)
    bucket(queue, lang, () => new Set<string>()).add(text)
    if (flushTimer !== null) clearTimeout(flushTimer)
    flushTimer = setTimeout(flush, CONTENT_TRANSLATION_DEBOUNCE_MS)
  }

  function flush() {
    flushTimer = null
    const batches = [...queue.entries()]
    queue.clear()
    for (const [lang, texts] of batches) {
      for (const chunk of chunkTexts([...texts])) {
        void lookup(lang, chunk, 0)
      }
    }
  }

  async function lookup(lang: ContentTranslationLang, texts: string[], attempt: number): Promise<void> {
    let response: Awaited<ReturnType<typeof LookupFn>> | undefined
    try {
      const lookupFn = await loadLookup()
      response = await lookupFn(lang, texts)
    } catch (err) {
      if (!failureLogged) {
        failureLogged = true
        console.warn('[content-translations] lookup failed; showing original text', err)
      }
      setTimeout(() => {
        const seen = requested.get(lang)
        for (const text of texts) seen?.delete(text)
      }, FAILURE_COOLDOWN_MS)
      return
    }

    const translations =
      response && typeof response.translations === 'object' && response.translations !== null
        ? response.translations
        : {}
    const langCache = bucket(cache, lang, () => new Map<string, string>())
    const missing: string[] = []
    let changed = false
    for (const text of texts) {
      const value = Object.prototype.hasOwnProperty.call(translations, text) ? translations[text] : undefined
      if (typeof value === 'string' && value.trim() !== '') {
        if (langCache.get(text) !== value) {
          langCache.set(text, value)
          changed = true
        }
      } else {
        missing.push(text)
      }
    }
    if (changed) version.value++

    if (response?.pending === true && missing.length > 0 && attempt < CONTENT_TRANSLATION_PENDING_MAX_RETRIES) {
      setTimeout(() => {
        void lookup(lang, missing, attempt + 1)
      }, CONTENT_TRANSLATION_PENDING_RETRY_MS)
    }
  }

  return {
    version,
    translate,
    cached
  }
})
