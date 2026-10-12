import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import {
  adminDashboardDataSource,
  provideDashboardDataSource,
  useDashboardDataSource,
  userDashboardDataSource,
} from '../dashboardDataSource'

const mocks = vi.hoisted(() => ({
  getDashboardTrend: vi.fn(),
  getDashboardModels: vi.fn(),
  getDashboardStats: vi.fn(),
  keysList: vi.fn(),
  getAvailable: vi.fn(),
  getUsageTrend: vi.fn(),
  getModelStats: vi.fn(),
  getStats: vi.fn(),
  getAll: vi.fn(),
}))

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardTrend: mocks.getDashboardTrend,
    getDashboardModels: mocks.getDashboardModels,
    getDashboardStats: mocks.getDashboardStats,
  },
}))
vi.mock('@/api/keys', () => ({ keysAPI: { list: mocks.keysList } }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getAvailable: mocks.getAvailable } }))
vi.mock('@/api/admin/dashboard', () => ({
  getUsageTrend: mocks.getUsageTrend,
  getModelStats: mocks.getModelStats,
  getStats: mocks.getStats,
}))
vi.mock('@/api/admin/groups', () => ({ getAll: mocks.getAll }))

const query = { start_date: '2026-10-01', end_date: '2026-10-07', granularity: 'day' as const }

describe('dashboardDataSource', () => {
  it('user scope reads the user dashboard endpoints and the user’s own keys and groups', async () => {
    mocks.getDashboardTrend.mockResolvedValue({ trend: [{ date: '2026-10-01' }] })
    mocks.getDashboardModels.mockResolvedValue({ models: [{ model: 'm' }] })
    mocks.keysList
      .mockResolvedValueOnce({ items: [{ id: 1, name: 'a' }], pages: 2 })
      .mockResolvedValueOnce({ items: [{ id: 2, name: 'b' }], pages: 2 })
    mocks.getAvailable.mockResolvedValue([{ id: 5, name: 'g' }])

    expect(await userDashboardDataSource.getTrend(query)).toEqual([{ date: '2026-10-01' }])
    expect(mocks.getDashboardTrend).toHaveBeenCalledWith(query)
    expect(await userDashboardDataSource.getModels(query)).toEqual([{ model: 'm' }])
    expect(await userDashboardDataSource.loadApiKeys?.()).toEqual([{ id: 1, name: 'a' }, { id: 2, name: 'b' }])
    expect(await userDashboardDataSource.loadGroups()).toEqual([{ id: 5, name: 'g' }])
    expect(userDashboardDataSource.usagePath).toBe('/usage')
  })

  it('admin scope reads the site-wide admin endpoints and has no API key filter', async () => {
    mocks.getUsageTrend.mockResolvedValue({ trend: [{ date: '2026-10-02' }] })
    mocks.getModelStats.mockResolvedValue({ models: [] })
    mocks.getStats.mockResolvedValue({ rpm: 1 })
    mocks.getAll.mockResolvedValue([{ id: 7, name: 'all', platform: 'openai' }])

    expect(await adminDashboardDataSource.getTrend(query)).toEqual([{ date: '2026-10-02' }])
    expect(mocks.getUsageTrend).toHaveBeenCalledWith(query)
    expect(await adminDashboardDataSource.getModels(query)).toEqual([])
    expect(await adminDashboardDataSource.getLiveStats()).toEqual({ rpm: 1 })
    expect(await adminDashboardDataSource.loadGroups()).toEqual([{ id: 7, name: 'all' }])
    expect(adminDashboardDataSource.loadApiKeys).toBeUndefined()
    expect(adminDashboardDataSource.usagePath).toBe('/admin/usage')
    expect(mocks.getDashboardTrend).not.toHaveBeenCalledWith(query, expect.anything())
  })

  it('components get the page’s source, and the user source when none is provided', () => {
    const seen: string[] = []
    const Child = defineComponent({ setup: () => { seen.push(useDashboardDataSource().scope); return () => h('i') } })
    mount(Child)
    mount(defineComponent({
      setup: () => { provideDashboardDataSource(adminDashboardDataSource); return () => h(Child) },
    }))
    expect(seen).toEqual(['user', 'admin'])
  })
})
