/**
 * 内容自动翻译（fork 本地功能）：管理接口。仅 admin 可用（operator 不在白名单里）。
 * 契约见 docs/CONTENT_TRANSLATION.md。
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { ContentTranslationLang } from '../contentTranslations'

export type { ContentTranslationLang } from '../contentTranslations'

export interface ContentTranslationConfig {
  enabled: boolean
  /** 管理员自己的一把 API Key 的 id；null 表示未选 */
  api_key_id: number | null
  model: string
  /** 留空走本机回环；只请求 {base_url}/v1/chat/completions */
  base_url: string
  languages: ContentTranslationLang[]
  /** GET 额外返回的只读字段：所选 Key 的名称，找不到时为空 */
  api_key_name?: string
}

export type ContentTranslationConfigInput = Omit<ContentTranslationConfig, 'api_key_name'>

export interface ContentTranslationTestResult {
  translated: string
  latency_ms: number
}

export interface ContentTranslationStatus {
  enabled: boolean
  running: boolean
  sources: number
  translated: number
  pending: number
  last_run_at: string | null
  last_error: string
  last_error_at: string | null
}

export interface ContentTranslationItem {
  id: number
  source_hash: string
  target_lang: ContentTranslationLang
  source_text: string
  translated_text: string
  model: string
  manual: boolean
  in_use: boolean
  created_at: string
  updated_at: string
  last_seen_at: string | null
}

export interface ContentTranslationListQuery {
  lang?: ContentTranslationLang | ''
  q?: string
  page?: number
  page_size?: number
}

export async function getConfig(): Promise<ContentTranslationConfig> {
  const { data } = await apiClient.get<ContentTranslationConfig>('/admin/content-translations/config')
  return data
}

export async function updateConfig(config: ContentTranslationConfigInput): Promise<ContentTranslationConfig> {
  const { data } = await apiClient.put<ContentTranslationConfig>('/admin/content-translations/config', config)
  return data
}

/** 用请求体里的配置翻译一段示例文本（不保存） */
export async function testConfig(config: ContentTranslationConfigInput): Promise<ContentTranslationTestResult> {
  const { data } = await apiClient.post<ContentTranslationTestResult>(
    '/admin/content-translations/config/test',
    config,
    // 一次真实的模型调用，可能比默认 30 秒更久
    { timeout: 120000 }
  )
  return data
}

export async function getStatus(): Promise<ContentTranslationStatus> {
  const { data } = await apiClient.get<ContentTranslationStatus>('/admin/content-translations/status')
  return data
}

/** 立即扫描一次（异步），返回状态 */
export async function sync(): Promise<ContentTranslationStatus> {
  const { data } = await apiClient.post<ContentTranslationStatus>('/admin/content-translations/sync')
  return data
}

export async function list(
  query: ContentTranslationListQuery = {}
): Promise<PaginatedResponse<ContentTranslationItem>> {
  const params: Record<string, unknown> = {}
  if (query.lang) params.lang = query.lang
  if (query.q && query.q.trim()) params.q = query.q.trim()
  if (query.page) params.page = query.page
  if (query.page_size) params.page_size = query.page_size
  const { data } = await apiClient.get<PaginatedResponse<ContentTranslationItem>>(
    '/admin/content-translations',
    { params }
  )
  return data
}

/** 改写译文，改后标为人工译文 */
export async function updateItem(id: number, translatedText: string): Promise<ContentTranslationItem> {
  const { data } = await apiClient.put<ContentTranslationItem>(`/admin/content-translations/${id}`, {
    translated_text: translatedText
  })
  return data
}

/** 删除一条译文，下一轮重新翻译 */
export async function deleteItem(id: number): Promise<void> {
  await apiClient.delete(`/admin/content-translations/${id}`)
}

/** 清空全部机器译文（人工译文保留） */
export async function clearMachine(): Promise<{ deleted: number }> {
  const { data } = await apiClient.delete<{ deleted: number }>('/admin/content-translations', {
    params: { scope: 'machine' }
  })
  return data
}

export const contentTranslationsAPI = {
  getConfig,
  updateConfig,
  testConfig,
  getStatus,
  sync,
  list,
  updateItem,
  deleteItem,
  clearMachine
}

export default contentTranslationsAPI
