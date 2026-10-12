import { inject, provide, type InjectionKey } from 'vue'
import { getModelStats, getStats, getUsageTrend } from '@/api/admin/dashboard'
import { getAll as getAllAdminGroups } from '@/api/admin/groups'
import { userGroupsAPI } from '@/api/groups'
import { keysAPI } from '@/api/keys'
import { usageAPI } from '@/api/usage'
import type { ModelStat, TrendDataPoint } from '@/types'

// fork：仪表盘的数据源。用户仪表盘和管理员仪表盘用同一套组件（TokenRouter 用户仪表盘的设计），
// 只是取数口径不同：用户看自己的用量，管理员看全站。页面用 provideDashboardDataSource 指定，
// 没有指定时按用户口径，组件单独挂载（例如测试）时行为不变。

// DashboardQuery 是趋势和模型统计的查询条件：整天日期（结束日期包含在内）、粒度和筛选条件。
export type DashboardQuery = {
  start_date: string
  end_date: string
  granularity?: 'day' | 'hour'
} & Record<string, unknown>

// DashboardLiveStats 是标题下实时状态条用到的字段，用户统计和管理员统计都有。
export interface DashboardLiveStats {
  rpm: number
  tpm: number
  average_duration_ms: number
  today_actual_cost: number
}

export interface DashboardOption {
  id: number
  name: string
}

export interface DashboardDataSource {
  // scope 区分口径：user 只统计自己，admin 统计全站
  scope: 'user' | 'admin'
  getTrend: (query: DashboardQuery) => Promise<TrendDataPoint[]>
  getModels: (query: DashboardQuery) => Promise<ModelStat[]>
  getLiveStats: () => Promise<DashboardLiveStats>
  // 只有用户口径能按自己的 API 密钥筛选；管理员口径没有这个筛选项
  loadApiKeys?: () => Promise<DashboardOption[]>
  loadGroups: () => Promise<DashboardOption[]>
  // 模型排行「查看全部」跳到的使用记录页
  usagePath: string
}

// 密钥候选最多翻这么多页（每页 100 个），与使用记录页的上限相当。
const MAX_KEY_PAGES = 10

export const userDashboardDataSource: DashboardDataSource = {
  scope: 'user',
  getTrend: async (query) => (await usageAPI.getDashboardTrend(query)).trend || [],
  getModels: async (query) => (await usageAPI.getDashboardModels(query)).models || [],
  getLiveStats: () => usageAPI.getDashboardStats(),
  loadApiKeys: async () => {
    const first = await keysAPI.list(1, 100)
    const keys = [...(first.items || [])]
    const pages = Math.min(first.pages || 1, MAX_KEY_PAGES)
    for (let page = 2; page <= pages; page++) {
      const next = await keysAPI.list(page, 100)
      if (!next.items?.length) break
      keys.push(...next.items)
    }
    return keys.map((key) => ({ id: key.id, name: key.name }))
  },
  loadGroups: async () => (await userGroupsAPI.getAvailable() || []).map((group) => ({ id: group.id, name: group.name })),
  usagePath: '/usage',
}

// 管理员口径：全站趋势、模型统计和实时统计。不带筛选条件时后端走预聚合表，长时间范围也很轻。
export const adminDashboardDataSource: DashboardDataSource = {
  scope: 'admin',
  getTrend: async (query) => (await getUsageTrend(query)).trend || [],
  getModels: async (query) => (await getModelStats(query)).models || [],
  getLiveStats: () => getStats(),
  loadGroups: async () => (await getAllAdminGroups() || []).map((group) => ({ id: group.id, name: group.name })),
  usagePath: '/admin/usage',
}

const DASHBOARD_DATA_SOURCE_KEY: InjectionKey<DashboardDataSource> = Symbol('forkDashboardDataSource')

// provideDashboardDataSource 由页面调用，指定本页仪表盘组件的取数口径。
export const provideDashboardDataSource = (source: DashboardDataSource): DashboardDataSource => {
  provide(DASHBOARD_DATA_SOURCE_KEY, source)
  return source
}

// useDashboardDataSource 读取页面指定的数据源，没有指定时按用户口径。
export const useDashboardDataSource = (): DashboardDataSource =>
  inject(DASHBOARD_DATA_SOURCE_KEY, userDashboardDataSource)
