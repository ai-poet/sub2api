// fork：排行榜（登录可见）。后端按固定时间窗缓存，用户榜不含用户 ID、邮箱和原始用户名。
import { apiClient } from './client'

export type LeaderboardWindow = '24h' | '7d' | '30d'
export type LeaderboardMetric = 'tokens' | 'requests' | 'cost'

export const LEADERBOARD_WINDOWS: LeaderboardWindow[] = ['24h', '7d', '30d']

export interface LeaderboardModelItem {
  rank: number
  model: string
  platform: string
  tokens: number
  requests: number
  share_percent: number
}

export interface LeaderboardModelsResponse {
  enabled: boolean
  window: LeaderboardWindow
  generated_at?: string
  total_tokens: number
  /** 榜单之外（含不公开的模型）的 token 占比 */
  other_share_percent: number
  items: LeaderboardModelItem[]
}

export interface LeaderboardUserItem {
  rank: number
  /** 脱敏后的名字；为空时显示「用户」 */
  display_name: string
  /** 头像化名种子 */
  avatar_seed: string
  tokens: number
  requests: number
  actual_cost?: number
  is_self: boolean
}

export interface LeaderboardUsersResponse {
  enabled: boolean
  window: LeaderboardWindow
  metric: LeaderboardMetric
  show_cost: boolean
  generated_at?: string
  /** 参与人数太少时不公布榜单 */
  insufficient_data: boolean
  items: LeaderboardUserItem[]
}

export async function getModelLeaderboard(window: LeaderboardWindow): Promise<LeaderboardModelsResponse> {
  const { data } = await apiClient.get<LeaderboardModelsResponse>('/leaderboard/models', { params: { window } })
  return { ...data, items: Array.isArray(data?.items) ? data.items : [] }
}

export async function getUserLeaderboard(
  window: LeaderboardWindow,
  metric: LeaderboardMetric = 'tokens',
): Promise<LeaderboardUsersResponse> {
  const { data } = await apiClient.get<LeaderboardUsersResponse>('/leaderboard/users', { params: { window, metric } })
  return { ...data, items: Array.isArray(data?.items) ? data.items : [] }
}

export const leaderboardAPI = { getModelLeaderboard, getUserLeaderboard }

export default leaderboardAPI
