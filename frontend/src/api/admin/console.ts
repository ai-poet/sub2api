/**
 * Admin console session API
 * 控制台会话信息：当前角色、scope 与 ops 开关。
 * 运维管理员（operator）无权读取 GET /admin/settings，侧边栏与运维页用本接口决定要渲染什么。
 */

import { apiClient } from '../client'

export type ConsoleScope = '*' | 'ops:read' | 'usage:read'

export interface ConsoleSession {
  role: 'admin' | 'user' | 'operator'
  scopes: ConsoleScope[]
  ops_monitoring_enabled: boolean
  ops_realtime_monitoring_enabled: boolean
  ops_query_mode_default: string
}

export async function getSession(): Promise<ConsoleSession> {
  const { data } = await apiClient.get<ConsoleSession>('/admin/console/session')
  return data
}

export const consoleAPI = {
  getSession
}

export default consoleAPI
