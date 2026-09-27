/**
 * 运维管理员个人令牌（fork 本地功能）：管理员查看全部令牌并吊销任意一个。
 * 只允许 admin（后端 AdminOnly + 不在运维白名单）；响应只含提示串，没有明文也没有哈希。
 */

import { apiClient } from '../client'

/** active 可用；其余都会被后端拒绝 */
export type PersonalTokenState =
  | 'active'
  | 'expired'
  | 'revoked'
  | 'not_eligible'
  | 'user_inactive'
  | 'user_missing'

export interface AdminPersonalTokenItem {
  user_id: number
  user_email: string
  username: string
  user_role: string
  state: PersonalTokenState
  hint: string
  created_at: string
  expires_at: string | null
  last_used_at: string | null
  last_used_ip: string
  created_ip: string
}

export interface AdminPersonalTokenList {
  feature_enabled: boolean
  items: AdminPersonalTokenItem[]
}

export async function list(): Promise<AdminPersonalTokenList> {
  const { data } = await apiClient.get<AdminPersonalTokenList>('/admin/personal-tokens')
  return data
}

export async function revoke(userId: number): Promise<{ revoked: boolean }> {
  const { data } = await apiClient.delete<{ revoked: boolean }>(`/admin/personal-tokens/${userId}`)
  return data
}

export const personalTokensAPI = {
  list,
  revoke
}

export default personalTokensAPI
