/**
 * 工单（fork 本地功能）：客服侧接口。管理员与运维管理员同权（运维的写操作在后端
 * operatorWriteScope 里直接放行，不走审批），看全部工单并回复 / 关闭 / 重开。
 */

import { apiClient } from '../client'
import type { FetchOptions, PaginatedResponse } from '@/types'
import type {
  SupportTicket,
  TicketAttachmentUploadResult,
  TicketCategory,
  TicketCountResponse,
  TicketDetail,
  TicketReplyResponse,
  TicketStatus
} from '@/api/tickets'

export interface AdminSupportTicket extends SupportTicket {
  user: { id: number; email: string }
  closed_by_role?: string
}

export type AdminTicketDetail = TicketDetail<AdminSupportTicket>
export type AdminTicketReplyResponse = TicketReplyResponse<AdminSupportTicket>

export interface AdminTicketListFilters {
  status?: TicketStatus | ''
  category?: TicketCategory | ''
  /** 匹配标题或用户邮箱 */
  search?: string
}

export async function list(
  page: number,
  pageSize: number,
  filters: AdminTicketListFilters = {},
  options: FetchOptions = {}
): Promise<PaginatedResponse<AdminSupportTicket>> {
  const params: Record<string, unknown> = { page, page_size: pageSize }
  if (filters.status) params.status = filters.status
  if (filters.category) params.category = filters.category
  if (filters.search?.trim()) params.search = filters.search.trim()
  const { data } = await apiClient.get<PaginatedResponse<AdminSupportTicket>>('/admin/tickets', {
    params,
    signal: options.signal
  })
  return data
}

export async function get(id: number): Promise<AdminTicketDetail> {
  const { data } = await apiClient.get<AdminTicketDetail>(`/admin/tickets/${id}`)
  return data
}

export async function reply(id: number, body: string): Promise<AdminTicketReplyResponse> {
  const { data } = await apiClient.post<AdminTicketReplyResponse>(`/admin/tickets/${id}/messages`, { body })
  return data
}

export async function close(id: number): Promise<AdminSupportTicket> {
  const { data } = await apiClient.post<AdminSupportTicket>(`/admin/tickets/${id}/close`)
  return data
}

export async function reopen(id: number): Promise<AdminSupportTicket> {
  const { data } = await apiClient.post<AdminSupportTicket>(`/admin/tickets/${id}/reopen`)
  return data
}

export async function openCount(): Promise<TicketCountResponse> {
  const { data } = await apiClient.get<TicketCountResponse>('/admin/tickets/open-count')
  return data
}

/** 客服侧上传工单图片附件（multipart 字段 file），与用户侧附件互不可见 */
export async function uploadAttachment(file: File): Promise<TicketAttachmentUploadResult> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await apiClient.post<TicketAttachmentUploadResult>('/admin/tickets/attachments', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
  return data
}

export const ticketsAPI = {
  list,
  get,
  reply,
  close,
  reopen,
  openCount,
  uploadAttachment
}

export default ticketsAPI
