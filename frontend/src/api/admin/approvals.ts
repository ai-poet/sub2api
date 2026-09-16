/**
 * 运维管理员写操作审批 API（fork 本地功能）。
 * 管理员：全部申请 + 一键通过 / 拒绝；运维管理员：只看得到自己的申请，可撤回。
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export type AdminApprovalStatus =
  | 'pending'
  | 'executing'
  | 'approved'
  | 'failed'
  | 'rejected'
  | 'cancelled'
  | 'expired'

export type ApprovalStatusFilter = 'pending' | 'processed' | AdminApprovalStatus

export interface AdminApprovalActorRef {
  id: number
  email: string
}

export interface AdminApprovalRequest {
  id: number
  status: AdminApprovalStatus
  action: string
  method: string
  route_template: string
  request_path: string
  request_query?: string
  target_type: string
  target_id?: number
  target_summary: string
  requester: AdminApprovalActorRef
  requester_ip?: string
  /** 脱敏后的请求体（JSON 字符串），原始 body 只在服务端解密重放 */
  request_body: string
  decided_by?: AdminApprovalActorRef
  decided_at?: string
  decision_reason?: string
  executed_at?: string
  result_status_code?: number
  result_body?: string
  result_error?: string
  expires_at: string
  created_at: string
  updated_at: string
}

export interface ApprovalReplayResult {
  status_code: number
  duration_ms: number
}

export interface ApprovalDecisionResponse {
  approval: AdminApprovalRequest
  replay?: ApprovalReplayResult
}

export interface ApprovalPendingCountResponse {
  pending: number
}

export async function list(
  page: number,
  pageSize: number,
  filters: { status?: ApprovalStatusFilter } = {},
  options: { signal?: AbortSignal } = {}
): Promise<PaginatedResponse<AdminApprovalRequest>> {
  const params: Record<string, unknown> = { page, page_size: pageSize }
  if (filters.status) params.status = filters.status
  const { data } = await apiClient.get<PaginatedResponse<AdminApprovalRequest>>('/admin/approvals', {
    params,
    signal: options.signal
  })
  return data
}

export async function get(id: number): Promise<AdminApprovalRequest> {
  const { data } = await apiClient.get<AdminApprovalRequest>(`/admin/approvals/${id}`)
  return data
}

export async function pendingCount(): Promise<ApprovalPendingCountResponse> {
  const { data } = await apiClient.get<ApprovalPendingCountResponse>('/admin/approvals/pending-count')
  return data
}

/** 一键通过：后端以管理员身份重放原请求，响应附带重放的 HTTP 状态。 */
export async function approve(id: number): Promise<ApprovalDecisionResponse> {
  const { data } = await apiClient.post<ApprovalDecisionResponse>(`/admin/approvals/${id}/approve`)
  return data
}

export async function reject(id: number, reason = ''): Promise<ApprovalDecisionResponse> {
  const { data } = await apiClient.post<ApprovalDecisionResponse>(`/admin/approvals/${id}/reject`, { reason })
  return data
}

export interface ApprovalBatchItem {
  id: number
  status: AdminApprovalStatus | 'skipped'
  error?: string
  http_status?: number
}

export interface ApprovalBatchResponse {
  results: ApprovalBatchItem[]
  approved: number
  failed: number
  skipped: number
}

/** 批量一键通过（最多 50 条）：逐条重放，单条失败不影响其它。 */
export async function batchApprove(ids: number[]): Promise<ApprovalBatchResponse> {
  const { data } = await apiClient.post<ApprovalBatchResponse>('/admin/approvals/batch-approve', { ids })
  return data
}

export async function cancel(id: number): Promise<ApprovalDecisionResponse> {
  const { data } = await apiClient.post<ApprovalDecisionResponse>(`/admin/approvals/${id}/cancel`)
  return data
}

export const approvalsAPI = {
  list,
  get,
  pendingCount,
  approve,
  batchApprove,
  reject,
  cancel
}

export default approvalsAPI
