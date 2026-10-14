import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post }
}))

import { applyBatch, groups, list, usage } from '@/api/admin/userGroupRates'

describe('admin userGroupRates API (fork)', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('lists rates under /admin/users and drops empty filters', async () => {
    get.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 1, user_count: 0 } })
    const controller = new AbortController()

    await list({ page: 2, page_size: 50, search: '', group_id: 3, user_status: undefined, sort_by: 'discount', sort_order: 'asc' }, { signal: controller.signal })

    expect(get).toHaveBeenCalledWith('/admin/users/group-rates', {
      params: { page: 2, page_size: 50, group_id: 3, sort_by: 'discount', sort_order: 'asc' },
      signal: controller.signal
    })
  })

  it('loads group options and 30-day usage', async () => {
    get.mockResolvedValueOnce({ data: [{ id: 3 }] }).mockResolvedValueOnce({ data: [] })

    expect(await groups()).toEqual([{ id: 3 }])
    expect(get).toHaveBeenNthCalledWith(1, '/admin/users/group-rates/groups')

    await usage([1, 2])
    expect(get).toHaveBeenNthCalledWith(2, '/admin/users/group-rates/usage', { params: { user_ids: '1,2' } })
  })

  it('skips the usage request when there are no users', async () => {
    expect(await usage([])).toEqual([])
    expect(get).not.toHaveBeenCalled()
  })

  it('posts batch changes with explicit nulls for restored defaults', async () => {
    post.mockResolvedValue({ data: { updated: 1, cleared: 1 } })
    const items = [
      { user_id: 1, group_id: 3, rate_multiplier: 0.8 },
      { user_id: 2, group_id: 3, rate_multiplier: null }
    ]

    expect(await applyBatch(items)).toEqual({ updated: 1, cleared: 1 })
    expect(post).toHaveBeenCalledWith('/admin/users/batch-group-rates', { items })
  })
})
