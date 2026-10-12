// fork：首页公开统计。匿名接口，后端从不返回 401；这里再兜一层，任何失败都当作「不可用」，从不抛错。
import { apiClient } from './client'

export interface PublicStats {
  enabled: boolean
  /** false 时数据暂时拿不到，首页隐藏计数 */
  available: boolean
  total_tokens: number
  today_tokens: number
  total_requests: number
  as_of?: string
}

const UNAVAILABLE: PublicStats = {
  enabled: false,
  available: false,
  total_tokens: 0,
  today_tokens: 0,
  total_requests: 0,
}

export async function getPublicStats(): Promise<PublicStats> {
  try {
    const { data } = await apiClient.get<PublicStats>('/stats/public')
    if (!data || typeof data !== 'object') return { ...UNAVAILABLE }
    return { ...UNAVAILABLE, ...data }
  } catch {
    return { ...UNAVAILABLE }
  }
}

export default { getPublicStats }
