import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const state = vi.hoisted(() => ({
  isAuthenticated: true,
  hasConsoleAccess: false,
  openCount: vi.fn(),
  unreadCount: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { tickets: { openCount: state.openCount } }
}))

vi.mock('@/api/tickets', () => ({
  default: { unreadCount: state.unreadCount }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isAuthenticated() {
      return state.isAuthenticated
    },
    get hasConsoleAccess() {
      return state.hasConsoleAccess
    }
  })
}))

import { useTicketsStore } from '@/stores/tickets'

describe('tickets store (fork)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    state.isAuthenticated = true
    state.hasConsoleAccess = false
  })

  it('polls the unread count for regular users', async () => {
    state.unreadCount.mockResolvedValue({ count: 3 })
    const store = useTicketsStore()
    await store.fetchCounts()
    expect(store.unreadCount).toBe(3)
    expect(store.openCount).toBe(0)
    expect(state.openCount).not.toHaveBeenCalled()
  })

  it('polls the open count for console roles (admin and operator alike)', async () => {
    state.hasConsoleAccess = true
    state.openCount.mockResolvedValue({ count: 7 })
    const store = useTicketsStore()
    await store.fetchCounts()
    expect(store.openCount).toBe(7)
    expect(state.unreadCount).not.toHaveBeenCalled()
  })

  it('swallows API failures and keeps the previous value', async () => {
    state.unreadCount.mockResolvedValueOnce({ count: 2 }).mockRejectedValueOnce(new Error('boom'))
    const store = useTicketsStore()
    await store.fetchCounts()
    await store.fetchCounts()
    expect(store.unreadCount).toBe(2)
  })

  it('clears counts when logged out and on reset', async () => {
    state.unreadCount.mockResolvedValue({ count: 4 })
    const store = useTicketsStore()
    await store.fetchCounts()
    expect(store.unreadCount).toBe(4)

    state.isAuthenticated = false
    await store.fetchCounts()
    expect(store.unreadCount).toBe(0)

    state.isAuthenticated = true
    await store.fetchCounts()
    store.reset()
    expect(store.unreadCount).toBe(0)
    expect(store.openCount).toBe(0)
  })
})
