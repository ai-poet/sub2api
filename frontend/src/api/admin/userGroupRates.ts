/**
 * 专属倍率用户管理 API（fork 本地功能）。
 *
 * 跨用户列出 / 调整 user_group_rate_multipliers 里的专属计费倍率。管理员直接生效；
 * 运维管理员的批量调整后端返回 202（进入审批），apiClient 会把它变成 isApprovalQueued 可识别的 rejection。
 */

import { apiClient } from '../client'
import type { GroupPlatform, PaginatedResponse, SubscriptionType } from '@/types'

export interface UserGroupRateRow {
  user_id: number
  user_email: string
  username: string
  user_notes: string
  user_status: string
  group_id: number
  group_name: string
  group_platform: GroupPlatform
  group_status: string
  /** 分组默认倍率 */
  group_rate_multiplier: number
  /** 用户在该分组的专属倍率 */
  rate_multiplier: number
  /** 专属 RPM 上限（只读展示，由分组的 RPM 覆盖弹窗维护） */
  rpm_override: number | null
  updated_at: string
}

export interface UserGroupRateListResponse extends PaginatedResponse<UserGroupRateRow> {
  /** 当前筛选下的去重用户数 */
  user_count: number
}

export type UserGroupRateSortField = 'updated_at' | 'rate_multiplier' | 'discount' | 'user_id' | 'group_name'

export interface UserGroupRateListQuery {
  page?: number
  page_size?: number
  search?: string
  group_id?: number
  user_status?: string
  sort_by?: UserGroupRateSortField
  sort_order?: 'asc' | 'desc'
}

export interface UserGroupRateGroupOption {
  id: number
  name: string
  platform: GroupPlatform
  status: string
  subscription_type: SubscriptionType
  is_exclusive: boolean
  rate_multiplier: number
}

export interface UserGroupRateUsage {
  user_id: number
  group_id: number
  requests: number
  actual_cost: number
}

export interface UserGroupRateBatchItem {
  user_id: number
  group_id: number
  /** 数字：设为该专属倍率；null：恢复分组默认 */
  rate_multiplier: number | null
}

export interface UserGroupRateBatchResult {
  updated: number
  cleared: number
}

/** 单次批量调整上限，与后端 UserGroupRateBatchMaxItems 一致 */
export const USER_GROUP_RATE_BATCH_MAX = 200

export async function list(query: UserGroupRateListQuery = {}, options?: { signal?: AbortSignal }): Promise<UserGroupRateListResponse> {
  const params: Record<string, string | number> = {}
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== '') params[key] = value as string | number
  }
  const { data } = await apiClient.get<UserGroupRateListResponse>('/admin/users/group-rates', {
    params,
    signal: options?.signal
  })
  return data
}

export async function groups(): Promise<UserGroupRateGroupOption[]> {
  const { data } = await apiClient.get<UserGroupRateGroupOption[]>('/admin/users/group-rates/groups')
  return data
}

/** 近 30 天按（用户, 分组）汇总的请求数与实际扣费，最多 200 个用户 */
export async function usage(userIds: number[]): Promise<UserGroupRateUsage[]> {
  if (userIds.length === 0) return []
  const { data } = await apiClient.get<UserGroupRateUsage[]>('/admin/users/group-rates/usage', {
    params: { user_ids: userIds.join(',') }
  })
  return data
}

export async function applyBatch(items: UserGroupRateBatchItem[]): Promise<UserGroupRateBatchResult> {
  const { data } = await apiClient.post<UserGroupRateBatchResult>('/admin/users/batch-group-rates', { items })
  return data
}

export const userGroupRatesAPI = {
  list,
  groups,
  usage,
  applyBatch
}

export default userGroupRatesAPI
