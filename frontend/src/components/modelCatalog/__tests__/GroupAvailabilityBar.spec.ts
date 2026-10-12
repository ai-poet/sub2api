import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import GroupAvailabilityBar from '../GroupAvailabilityBar.vue'
import type { GroupStatusHistoryBucket } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${Object.values(params).join(',')}` : key),
  }),
}))

const bucket = (availability: number, total: number): GroupStatusHistoryBucket => ({
  bucket_start: '2026-10-12T00:00:00Z',
  bucket_end: '2026-10-12T01:00:00Z',
  availability,
  avg_latency_ms: null,
  total_count: total,
  down_count: 0,
  latest_status: total ? 'up' : '',
})

describe('GroupAvailabilityBar', () => {
  it('colours each bucket by availability band and grey when there is no data', () => {
    const wrapper = mount(GroupAvailabilityBar, {
      props: {
        buckets: [bucket(100, 4), bucket(98.5, 4), bucket(92, 4), bucket(50, 4), bucket(0, 0)],
        availability: 88.1234,
        totalCount: 16,
      },
    })
    const cells = wrapper.findAll('[data-availability]')
    expect(cells.map(cell => cell.classes().find(name => name.startsWith('bg-')))).toEqual([
      'bg-emerald-500',
      'bg-lime-500',
      'bg-amber-400',
      'bg-rose-400',
      'bg-gray-200',
    ])
    expect(wrapper.text()).toContain('88.12%')
  })

  it('says there is no data when nothing was probed', () => {
    const wrapper = mount(GroupAvailabilityBar, {
      props: { buckets: [bucket(0, 0)], availability: 0, totalCount: 0 },
    })
    expect(wrapper.text()).toContain('modelCatalog.availability.noData')
  })
})
