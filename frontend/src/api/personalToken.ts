/**
 * 运维管理员个人令牌（fork 本地功能）：本人自助查看 / 生成 / 吊销。
 * 这组接口只认浏览器登录（JWT），令牌本身调不到这里；管理员视图见 api/admin/personalTokens.ts。
 */

import { apiClient } from './client'

export interface PersonalTokenInfo {
  /** 提示串，如 pat-1a2b3c...9f3c；明文只在生成响应里出现一次 */
  hint: string
  created_at: string
  /** null 表示永不过期 */
  expires_at: string | null
  expired: boolean
  last_used_at: string | null
  last_used_ip: string
}

export interface PersonalTokenStatus {
  /** 管理员是否打开了个人令牌功能 */
  feature_enabled: boolean
  /** 当前账号能否持有令牌（运维管理员且已激活） */
  eligible: boolean
  token: PersonalTokenInfo | null
  /** 可选有效期（天），0 表示永不过期 */
  expiry_options: number[]
}

export interface GeneratePersonalTokenRequest {
  password: string
  expires_in_days: number
}

export interface GeneratePersonalTokenResponse {
  /** 明文令牌，只返回这一次 */
  token: string
  info: PersonalTokenInfo
}

export async function getStatus(): Promise<PersonalTokenStatus> {
  const { data } = await apiClient.get<PersonalTokenStatus>('/user/personal-token')
  return data
}

export async function generate(
  payload: GeneratePersonalTokenRequest
): Promise<GeneratePersonalTokenResponse> {
  const { data } = await apiClient.post<GeneratePersonalTokenResponse>(
    '/user/personal-token',
    payload
  )
  return data
}

export async function revoke(): Promise<{ revoked: boolean }> {
  const { data } = await apiClient.delete<{ revoked: boolean }>('/user/personal-token')
  return data
}

export const personalTokenAPI = {
  getStatus,
  generate,
  revoke
}

export default personalTokenAPI
