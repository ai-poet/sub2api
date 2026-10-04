/**
 * 内容自动翻译（fork 本地功能）：公开的只读查询接口。
 *
 * 只查缓存，永远不会触发模型调用；查不到的文本不出现在 translations 里，显示原文即可。
 * 契约见 docs/CONTENT_TRANSLATION.md。显示层封装见 stores/contentTranslations.ts 与
 * composables/useContentTranslation.ts —— 组件不要直接调这里（批量上限等常量也在 store 里）。
 */

import { apiClient } from './client'

export type ContentTranslationLang = 'zh' | 'en' | 'ja'

export interface ContentTranslationLookupResponse {
  lang: ContentTranslationLang
  /** 键是请求里原样发来的文本 */
  translations: Record<string, string>
  /** 有文本正在排队翻译，可隔 10 秒左右再查一次（最多 3 次） */
  pending: boolean
}

export async function lookupContentTranslations(
  lang: ContentTranslationLang,
  texts: string[]
): Promise<ContentTranslationLookupResponse> {
  const { data } = await apiClient.post<ContentTranslationLookupResponse>(
    '/content-translations/lookup',
    { lang, texts }
  )
  return data
}

export const contentTranslationsAPI = {
  lookup: lookupContentTranslations
}

export default contentTranslationsAPI
