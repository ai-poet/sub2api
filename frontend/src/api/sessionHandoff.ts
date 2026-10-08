/**
 * 多域名登录交接（fork 本地功能）。
 *
 * 已登录的一方用当前会话申请一个一次性交接码（绑定接收方的 PKCE challenge 与 origin），
 * 接收方拿码 + verifier 兑换一套自己的 token。见 backend service/session_handoff.go。
 */

import { apiClient } from './client'

export interface SessionHandoffCodeRequest {
  /** base64url(SHA-256(verifier))，无填充，43 个字符。 */
  code_challenge: string
  code_challenge_method: 'S256'
  /** 接收方 origin，必须是 session_handoff 配置里的域名。 */
  target_origin: string
}

export interface SessionHandoffCodeResponse {
  code: string
  /** 规范化后的接收方 origin。 */
  target_origin: string
  expires_in: number
}

export interface SessionHandoffExchangeResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export async function createSessionHandoffCode(
  body: SessionHandoffCodeRequest
): Promise<SessionHandoffCodeResponse> {
  const { data } = await apiClient.post<SessionHandoffCodeResponse>('/auth/session-handoff/code', body)
  return data
}

export async function exchangeSessionHandoffCode(
  code: string,
  codeVerifier: string
): Promise<SessionHandoffExchangeResponse> {
  const { data } = await apiClient.post<SessionHandoffExchangeResponse>('/auth/session-handoff/exchange', {
    code,
    code_verifier: codeVerifier,
  })
  return data
}
