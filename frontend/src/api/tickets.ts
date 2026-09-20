/**
 * 工单（fork 本地功能）：用户侧接口，只能操作自己的工单。
 * 客服侧接口见 api/admin/tickets.ts。
 */

import { apiClient } from './client'
import type { FetchOptions, PaginatedResponse } from '@/types'

export type TicketStatus = 'open' | 'replied' | 'closed'
export type TicketCategory = 'account' | 'billing' | 'api' | 'other'
/** 用户视图里客服折叠为 staff；客服视图保留真实角色 admin / operator */
export type TicketMessageAuthorRole = 'user' | 'staff' | 'admin' | 'operator'

export interface SupportTicket {
  id: number
  title: string
  category: TicketCategory
  status: TicketStatus
  /** 客服回复后为 true，打开详情即清零 */
  user_unread: boolean
  message_count: number
  last_message_at: string
  closed_at?: string | null
  created_at: string
  updated_at: string
}

export interface SupportTicketMessage {
  id: number
  ticket_id: number
  author_role: TicketMessageAuthorRole
  /** 只有客服视图带作者邮箱 */
  author_email?: string
  body: string
  created_at: string
}

export interface TicketDetail<T extends SupportTicket = SupportTicket> {
  ticket: T
  messages: SupportTicketMessage[]
}

export interface TicketReplyResponse<T extends SupportTicket = SupportTicket> {
  ticket: T
  message: SupportTicketMessage
}

export interface TicketCountResponse {
  count: number
}

export interface CreateTicketPayload {
  title: string
  category: TicketCategory
  body: string
}

export interface TicketListFilters {
  status?: TicketStatus | ''
  category?: TicketCategory | ''
}

export async function list(
  page: number,
  pageSize: number,
  filters: TicketListFilters = {},
  options: FetchOptions = {}
): Promise<PaginatedResponse<SupportTicket>> {
  const params: Record<string, unknown> = { page, page_size: pageSize }
  if (filters.status) params.status = filters.status
  if (filters.category) params.category = filters.category
  const { data } = await apiClient.get<PaginatedResponse<SupportTicket>>('/tickets', { params, signal: options.signal })
  return data
}

export async function create(payload: CreateTicketPayload): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>('/tickets', payload)
  return data
}

export async function get(id: number): Promise<TicketDetail> {
  const { data } = await apiClient.get<TicketDetail>(`/tickets/${id}`)
  return data
}

export async function reply(id: number, body: string): Promise<TicketReplyResponse> {
  const { data } = await apiClient.post<TicketReplyResponse>(`/tickets/${id}/messages`, { body })
  return data
}

export async function close(id: number): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>(`/tickets/${id}/close`)
  return data
}

export async function reopen(id: number): Promise<SupportTicket> {
  const { data } = await apiClient.post<SupportTicket>(`/tickets/${id}/reopen`)
  return data
}

export async function unreadCount(): Promise<TicketCountResponse> {
  const { data } = await apiClient.get<TicketCountResponse>('/tickets/unread-count')
  return data
}

export const ticketsAPI = {
  list,
  create,
  get,
  reply,
  close,
  reopen,
  unreadCount
}

export default ticketsAPI
