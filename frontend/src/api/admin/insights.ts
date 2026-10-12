// fork：排行榜 / 首页公开统计的后台配置（仅管理员）。
import { apiClient } from '../client'

export interface InsightsConfig {
  model_ranking_enabled: boolean
  usage_ranking_enabled: boolean
  /** 3..50 */
  usage_ranking_limit: number
  usage_ranking_show_cost: boolean
  public_stats_enabled: boolean
}

export async function getConfig(): Promise<InsightsConfig> {
  const { data } = await apiClient.get<InsightsConfig>('/admin/insights/config')
  return data
}

export async function updateConfig(config: InsightsConfig): Promise<InsightsConfig> {
  const { data } = await apiClient.put<InsightsConfig>('/admin/insights/config', config)
  return data
}

export const insightsAPI = { getConfig, updateConfig }

export default insightsAPI
