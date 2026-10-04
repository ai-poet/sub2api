import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const state = vi.hoisted(() => ({
  lookup: vi.fn()
}))

// store 在第一次真正查询时才动态 import 查询接口
vi.mock('@/api/contentTranslations', () => ({
  lookupContentTranslations: state.lookup
}))

import {
  CONTENT_TRANSLATION_DEBOUNCE_MS,
  CONTENT_TRANSLATION_PENDING_RETRY_MS,
  normalizeContentLang,
  shouldSkipContentTranslation,
  useContentTranslationsStore
} from '@/stores/contentTranslations'

/** 不传 lang 时 store 读 <html lang> */
function setLang(lang: string) {
  document.documentElement.setAttribute('lang', lang)
}

/** 推进假时钟，并等动态 import 的查询模块就绪、查询结果落地 */
async function advance(ms: number) {
  await vi.advanceTimersByTimeAsync(ms)
  await vi.dynamicImportSettled()
  await vi.advanceTimersByTimeAsync(0)
}

type LookupResult = { lang: string; translations: Record<string, string>; pending: boolean }

function respond(translations: Record<string, string> = {}, pending = false) {
  return (lang: string, texts: string[]): Promise<LookupResult> =>
    Promise.resolve({
      lang,
      translations: Object.fromEntries(texts.filter((t) => t in translations).map((t) => [t, translations[t]])),
      pending
    })
}

describe('content translations store (fork)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    setActivePinia(createPinia())
    setLang('en')
    state.lookup.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('batches misses after the debounce and returns translations once they arrive', async () => {
    state.lookup.mockImplementation(respond({ 分组描述: 'Group description', 公告标题: 'Notice' }))
    const store = useContentTranslationsStore()

    expect(store.translate('分组描述')).toBe('分组描述')
    expect(store.translate('公告标题')).toBe('公告标题')
    // 同一文本重复调用不会重复登记
    store.translate('分组描述')

    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS - 1)
    expect(state.lookup).not.toHaveBeenCalled()

    await advance(1)
    expect(state.lookup).toHaveBeenCalledTimes(1)
    expect(state.lookup).toHaveBeenCalledWith('en', ['分组描述', '公告标题'])

    expect(store.version).toBe(1)
    expect(store.translate('分组描述')).toBe('Group description')
    expect(store.translate('公告标题')).toBe('Notice')
  })

  it('restarts the debounce while texts keep arriving', async () => {
    state.lookup.mockImplementation(respond())
    const store = useContentTranslationsStore()

    store.translate('一')
    await advance(30)
    store.translate('二')
    await advance(30)
    expect(state.lookup).not.toHaveBeenCalled()

    await advance(20)
    expect(state.lookup).toHaveBeenCalledTimes(1)
    expect(state.lookup).toHaveBeenCalledWith('en', ['一', '二'])
  })

  it('splits a large queue into chunks of at most 200 texts', async () => {
    state.lookup.mockImplementation(respond())
    const store = useContentTranslationsStore()

    for (let i = 0; i < 450; i++) store.translate(`文本${i}`)
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)

    expect(state.lookup).toHaveBeenCalledTimes(3)
    expect(state.lookup.mock.calls.map((call) => (call[1] as string[]).length)).toEqual([200, 200, 50])
  })

  it('does not request texts that are blank, too long or already in the current locale', async () => {
    state.lookup.mockImplementation(respond())
    const store = useContentTranslationsStore()

    expect(store.translate('')).toBe('')
    expect(store.translate('   ')).toBe('   ')
    expect(store.translate(null)).toBe('')
    expect(store.translate(undefined)).toBe('')
    expect(store.translate('Plain English text')).toBe('Plain English text')
    expect(store.translate('汉'.repeat(20001))).toBe('汉'.repeat(20001))
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(state.lookup).not.toHaveBeenCalled()

    setLang('zh')
    store.translate('中文描述')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(state.lookup).not.toHaveBeenCalled()

    // 中文界面下的英文与日文文案都需要查
    store.translate('English group')
    store.translate('グループ説明')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(state.lookup).toHaveBeenCalledWith('zh', ['English group', 'グループ説明'])
  })

  it('classifies scripts per locale', () => {
    expect(shouldSkipContentTranslation('中文', 'zh')).toBe(true)
    expect(shouldSkipContentTranslation('日本語のテキスト', 'zh')).toBe(false)
    expect(shouldSkipContentTranslation('Claude Max', 'zh')).toBe(false)
    expect(shouldSkipContentTranslation('Claude Max', 'en')).toBe(true)
    expect(shouldSkipContentTranslation('Claude 国模', 'en')).toBe(false)
    expect(shouldSkipContentTranslation('グループ', 'en')).toBe(false)
    expect(shouldSkipContentTranslation('グループ', 'ja')).toBe(true)
    expect(shouldSkipContentTranslation('中文', 'ja')).toBe(false)
  })

  it('normalizes locale codes', () => {
    expect(normalizeContentLang('zh-CN')).toBe('zh')
    expect(normalizeContentLang('en-US')).toBe('en')
    expect(normalizeContentLang('ja-JP')).toBe('ja')
    expect(normalizeContentLang('fr')).toBeNull()
    expect(normalizeContentLang(undefined)).toBeNull()
  })

  it('remembers texts without a translation instead of re-requesting them on every render', async () => {
    state.lookup.mockImplementation(respond())
    const store = useContentTranslationsStore()

    store.translate('没有译文')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(state.lookup).toHaveBeenCalledTimes(1)

    expect(store.translate('没有译文')).toBe('没有译文')
    await advance(CONTENT_TRANSLATION_PENDING_RETRY_MS * 4)
    expect(state.lookup).toHaveBeenCalledTimes(1)
  })

  it('re-looks up still-missing texts after 10 s while pending, at most 3 times', async () => {
    state.lookup.mockImplementation(respond({ 已有: 'Ready' }, true))
    const store = useContentTranslationsStore()

    store.translate('已有')
    store.translate('排队中')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(state.lookup).toHaveBeenNthCalledWith(1, 'en', ['已有', '排队中'])

    await advance(CONTENT_TRANSLATION_PENDING_RETRY_MS - 1)
    expect(state.lookup).toHaveBeenCalledTimes(1)
    await advance(1)
    expect(state.lookup).toHaveBeenNthCalledWith(2, 'en', ['排队中'])

    await advance(CONTENT_TRANSLATION_PENDING_RETRY_MS * 10)
    expect(state.lookup).toHaveBeenCalledTimes(4)
    expect(store.translate('排队中')).toBe('排队中')
  })

  it('stops the pending loop once the translation arrives', async () => {
    state.lookup
      .mockImplementationOnce(respond({}, true))
      .mockImplementationOnce(respond({ 排队中: 'Queued' }, false))
    const store = useContentTranslationsStore()

    store.translate('排队中')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    await advance(CONTENT_TRANSLATION_PENDING_RETRY_MS)
    expect(state.lookup).toHaveBeenCalledTimes(2)
    expect(store.translate('排队中')).toBe('Queued')

    await advance(CONTENT_TRANSLATION_PENDING_RETRY_MS * 5)
    expect(state.lookup).toHaveBeenCalledTimes(2)
  })

  it('swallows lookup failures, returns the original text and logs at most once', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    state.lookup.mockRejectedValue(new Error('network down'))
    const store = useContentTranslationsStore()

    store.translate('失败一')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    store.translate('失败二')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)

    expect(state.lookup).toHaveBeenCalledTimes(2)
    expect(store.translate('失败一')).toBe('失败一')
    expect(warn.mock.calls.length).toBeLessThanOrEqual(1)
  })

  it('keeps a separate cache per locale', async () => {
    state.lookup.mockImplementation((lang: string, texts: string[]) =>
      Promise.resolve({
        lang,
        translations: Object.fromEntries(texts.map((t) => [t, `${lang}:${t}`])),
        pending: false
      })
    )
    const store = useContentTranslationsStore()

    store.translate('分组')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(store.translate('分组')).toBe('en:分组')

    setLang('zh')
    expect(store.translate('Group')).toBe('Group')
    await advance(CONTENT_TRANSLATION_DEBOUNCE_MS)
    expect(state.lookup).toHaveBeenLastCalledWith('zh', ['Group'])
    expect(store.translate('Group')).toBe('zh:Group')
    expect(store.translate('分组', 'en')).toBe('en:分组')
  })
})
