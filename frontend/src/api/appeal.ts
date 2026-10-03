/**
 * 封禁申诉会话（fork 本地功能）。
 *
 * 用独立的 axios 实例，而不是 apiClient：
 *  - 只带 X-Appeal-Token，永远不带 Authorization（申诉令牌不是登录凭证，也不能混进登录态）；
 *  - 不走 apiClient 的全局 401 处理（那里会清掉登录信息并跳转登录页）。
 * 错误一律归一成 { status, code, reason, message, metadata }，由调用方决定怎么处理。
 */

import axios, { type AxiosError, type AxiosInstance } from 'axios'
import { getLocale } from '@/i18n'
import { getAPIBaseURL } from './url'
import type { PaginatedResponse } from '@/types'
import type { SiteMessage } from './siteMessages'
import type { SupportTicket, TicketDetail, TicketReplyResponse, TicketAttachmentUploadResult } from './tickets'
import { readAppealSession } from '@/utils/appeal'

export const APPEAL_TOKEN_HEADER = 'X-Appeal-Token'

export interface AppealSessionInfo {
  email: string
  status: string
  expires_at: string
  has_active_appeal: boolean
}

export interface AppealApiError {
  status: number
  code?: number | string
  reason?: string
  message: string
  metadata?: Record<string, unknown>
}

interface Envelope<T> {
  code: number
  message?: string
  reason?: string
  metadata?: Record<string, unknown>
  data: T
}

function normalizeError(error: AxiosError<Envelope<unknown>>): AppealApiError {
  const data = error.response?.data as Partial<Envelope<unknown>> | undefined
  return {
    status: error.response?.status ?? 0,
    code: data?.code,
    reason: data?.reason,
    message: data?.message || error.message || 'Network error',
    metadata: data?.metadata
  }
}

export const appealClient: AxiosInstance = axios.create({
  baseURL: getAPIBaseURL(),
  timeout: 30000
})

appealClient.interceptors.request.use((config) => {
  const session = readAppealSession()
  if (session?.token) {
    config.headers.set(APPEAL_TOKEN_HEADER, session.token)
  }
  config.headers.set('Accept-Language', getLocale())
  // 显式去掉 Authorization，即使将来有人给 axios 配了默认头
  config.headers.delete('Authorization')
  return config
})

appealClient.interceptors.response.use(
  (response) => {
    const body = response.data as Envelope<unknown> | Blob
    if (body instanceof Blob) return response
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) {
        response.data = body.data
        return response
      }
      return Promise.reject({
        status: response.status,
        code: body.code,
        reason: body.reason,
        message: body.message || 'Unknown error',
        metadata: body.metadata
      } satisfies AppealApiError)
    }
    return response
  },
  (error: AxiosError<Envelope<unknown>>) => Promise.reject(normalizeError(error))
)

export async function getSession(): Promise<AppealSessionInfo> {
  const { data } = await appealClient.get<AppealSessionInfo>('/appeal/session')
  return data
}

export async function logout(): Promise<void> {
  await appealClient.post('/appeal/logout')
}

export async function listSiteMessages(page = 1, pageSize = 20): Promise<PaginatedResponse<SiteMessage>> {
  const { data } = await appealClient.get<PaginatedResponse<SiteMessage>>('/appeal/site-messages', {
    params: { page, page_size: pageSize }
  })
  return data
}

export async function markSiteMessageRead(id: number): Promise<void> {
  await appealClient.post(`/appeal/site-messages/${id}/read`)
}

export async function listTickets(page = 1, pageSize = 20): Promise<PaginatedResponse<SupportTicket>> {
  const { data } = await appealClient.get<PaginatedResponse<SupportTicket>>('/appeal/tickets', {
    params: { page, page_size: pageSize, category: 'appeal' }
  })
  return data
}

export async function getTicket(id: number): Promise<TicketDetail> {
  const { data } = await appealClient.get<TicketDetail>(`/appeal/tickets/${id}`)
  return data
}

export async function createTicket(title: string, body: string): Promise<SupportTicket> {
  const { data } = await appealClient.post<SupportTicket>('/appeal/tickets', { title, body })
  return data
}

export async function reply(id: number, body: string): Promise<TicketReplyResponse> {
  const { data } = await appealClient.post<TicketReplyResponse>(`/appeal/tickets/${id}/messages`, { body })
  return data
}

export async function uploadAttachment(file: File): Promise<TicketAttachmentUploadResult> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await appealClient.post<TicketAttachmentUploadResult>('/appeal/tickets/attachments', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
  return data
}

export async function fetchAttachment(key: string): Promise<Blob> {
  const response = await appealClient.get('/appeal/tickets/attachments/content', {
    params: { key },
    responseType: 'blob'
  })
  return response.data
}

export const appealAPI = {
  getSession,
  logout,
  listSiteMessages,
  markSiteMessageRead,
  listTickets,
  getTicket,
  createTicket,
  reply,
  uploadAttachment,
  fetchAttachment
}

export default appealAPI
