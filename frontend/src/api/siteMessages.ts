/**
 * 站内信（fork 本地功能）：用户侧接口，只读写自己的收件箱。
 * 管理端发送 / 查看某用户历史见 api/admin/users.ts 的 sendSiteMessage / listSiteMessages。
 */

import { apiClient } from './client'
import type { FetchOptions, PaginatedResponse } from '@/types'

export type SiteMessageCategory = 'security' | 'admin' | 'system'
/** 用户视图只区分系统发出 / 工作人员发出，不暴露具体是谁 */
export type SiteMessageFrom = 'system' | 'staff'

export interface SiteMessage {
  id: number
  category: SiteMessageCategory
  title: string
  /** Markdown */
  content: string
  from: SiteMessageFrom
  read_at: string | null
  created_at: string
}

export interface SiteMessageListFilters {
  unreadOnly?: boolean
  category?: SiteMessageCategory | ''
}

export interface SiteMessageCountResponse {
  count: number
}

export interface SiteMessageMarkAllResponse {
  updated: number
}

export async function list(
  page: number,
  pageSize: number,
  filters: SiteMessageListFilters = {},
  options: FetchOptions = {}
): Promise<PaginatedResponse<SiteMessage>> {
  const params: Record<string, unknown> = { page, page_size: pageSize }
  if (filters.unreadOnly) params.unread_only = 1
  if (filters.category) params.category = filters.category
  const { data } = await apiClient.get<PaginatedResponse<SiteMessage>>('/site-messages', {
    params,
    signal: options.signal
  })
  return data
}

export async function unreadCount(): Promise<SiteMessageCountResponse> {
  const { data } = await apiClient.get<SiteMessageCountResponse>('/site-messages/unread-count')
  return data
}

export async function markRead(id: number): Promise<void> {
  await apiClient.post(`/site-messages/${id}/read`)
}

export async function markAllRead(): Promise<SiteMessageMarkAllResponse> {
  const { data } = await apiClient.post<SiteMessageMarkAllResponse>('/site-messages/read-all')
  return data
}

export const siteMessagesAPI = {
  list,
  unreadCount,
  markRead,
  markAllRead
}

export default siteMessagesAPI
