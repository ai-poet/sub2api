import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserDashboardHeatmap from '../UserDashboardHeatmap.vue'
import UserDashboardTopModels from '../UserDashboardTopModels.vue'
import { formatDateLocalInput } from '@/utils/format'
import type { ModelStat } from '@/types'

const { getDashboardTrend } = vi.hoisted(() => ({ getDashboardTrend: vi.fn() }))

vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardTrend } }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${Object.values(params).join(',')}` : key),
      locale: { value: 'en' },
    }),
  }
})

const daysAgo = (n: number) => {
  const date = new Date()
  date.setHours(0, 0, 0, 0)
  date.setDate(date.getDate() - n)
  return formatDateLocalInput(date)
}

describe('UserDashboardHeatmap', () => {
  beforeEach(() => {
    getDashboardTrend.mockReset()
  })

  it('asks for 13 weeks of daily trend and shades days by quartile', async () => {
    getDashboardTrend.mockResolvedValue({
      trend: [
        { date: daysAgo(0), requests: 4, total_tokens: 4000 },
        { date: daysAgo(1), requests: 1, total_tokens: 100 },
        { date: daysAgo(2), requests: 2, total_tokens: 2000 },
        { date: daysAgo(3), requests: 3, total_tokens: 3000 },
      ],
    })
    const wrapper = mount(UserDashboardHeatmap, { global: { stubs: { ModelIcon: true } } })
    await flushPromises()

    expect(getDashboardTrend).toHaveBeenCalledTimes(1)
    const params = getDashboardTrend.mock.calls[0][0]
    expect(params.granularity).toBe('day')
    expect(params.end_date).toBe(daysAgo(0))
    const span = (new Date(params.end_date).getTime() - new Date(params.start_date).getTime()) / 86400000
    expect(span).toBeGreaterThanOrEqual(84)
    expect(span).toBeLessThanOrEqual(90)

    expect(wrapper.findAll('[data-date]')).toHaveLength(13 * 7)
    expect(wrapper.get(`[data-date="${daysAgo(0)}"]`).attributes('data-level')).toBe('4')
    expect(wrapper.get(`[data-date="${daysAgo(1)}"]`).attributes('data-level')).toBe('1')
    expect(wrapper.get(`[data-date="${daysAgo(10)}"]`).attributes('data-level')).toBe('0')
    expect(wrapper.text()).toContain('dashboard.fork.heatmap.summary:4')
  })

  it('caps the column wave delay', async () => {
    getDashboardTrend.mockResolvedValue({ trend: [] })
    const wrapper = mount(UserDashboardHeatmap)
    await flushPromises()
    const delays = wrapper.findAll('[data-date]').map(cell => parseInt((cell.attributes('style') ?? '').match(/--wave-delay:\s*(\d+)ms/)?.[1] ?? '0', 10))
    expect(Math.max(...delays)).toBeLessThanOrEqual(480)
  })
})

describe('UserDashboardTopModels', () => {
  const model = (name: string, tokens: number): ModelStat => ({
    model: name,
    requests: 1,
    input_tokens: 0,
    output_tokens: 0,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: tokens,
    cost: 0,
    actual_cost: 0,
  })

  it('lists the top five by tokens with share bars', () => {
    const wrapper = mount(UserDashboardTopModels, {
      props: {
        models: [model('a', 10), model('b', 50), model('c', 20), model('d', 5), model('e', 5), model('f', 10)],
      },
      global: { stubs: { ModelIcon: true } },
    })
    const rows = wrapper.findAll('li')
    expect(rows).toHaveLength(5)
    expect(rows[0].text()).toContain('b')
    expect(rows[0].attributes('style')).toContain('--bar-w: 50%')
  })

  it('says so when there is no usage', () => {
    const wrapper = mount(UserDashboardTopModels, { props: { models: [] } })
    expect(wrapper.text()).toContain('dashboard.fork.topModels.empty')
  })
})
