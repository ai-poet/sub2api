// fork：用量排行（登录可见）。按日期范围查询（YYYY-MM-DD，服务器时区，含首尾，最长 90 天），后端按范围缓存；
// 用户榜对普通用户不含用户 ID、邮箱和原始用户名；管理员 / 运维管理员拿到实名（identified）。排名指标与条数由后台设置决定。
import { apiClient } from './client'

export type LeaderboardMetric = 'tokens' | 'requests' | 'cost'

/** 日期范围最长天数（含首尾），与后端 insightsMaxRangeDays 一致。 */
export const LEADERBOARD_MAX_RANGE_DAYS = 90

export interface LeaderboardRange {
  start_date: string
  end_date: string
}

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
  start_date: string
  end_date: string
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
  /** 以下三项只对管理员和运维管理员下发 */
  user_id?: number
  email?: string
  username?: string
}

export interface LeaderboardUsersResponse {
  enabled: boolean
  start_date: string
  end_date: string
  /** 后台设定的排名指标，前端当主指标显示 */
  sort_by: LeaderboardMetric
  show_cost: boolean
  /** 后台设定的榜单条数 */
  limit: number
  generated_at?: string
  /** 参与人数太少时不公布榜单 */
  insufficient_data: boolean
  /** 实名视角（管理员 / 运维管理员） */
  identified?: boolean
  items: LeaderboardUserItem[]
}

export async function getModelLeaderboard(range: LeaderboardRange): Promise<LeaderboardModelsResponse> {
  const { data } = await apiClient.get<LeaderboardModelsResponse>('/leaderboard/models', { params: range })
  return { ...data, items: Array.isArray(data?.items) ? data.items : [] }
}

export async function getUserLeaderboard(range: LeaderboardRange): Promise<LeaderboardUsersResponse> {
  const { data } = await apiClient.get<LeaderboardUsersResponse>('/leaderboard/users', { params: range })
  return { ...data, items: Array.isArray(data?.items) ? data.items : [] }
}

export const leaderboardAPI = { getModelLeaderboard, getUserLeaderboard }

export default leaderboardAPI
