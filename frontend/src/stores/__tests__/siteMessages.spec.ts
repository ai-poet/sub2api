import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { flushPromises } from '@vue/test-utils'

const state = vi.hoisted(() => ({
  isAuthenticated: true,
  userId: 7 as number | undefined,
  unreadCount: vi.fn(),
  list: vi.fn(),
  markRead: vi.fn(),
  markAllRead: vi.fn()
}))

vi.mock('@/api/siteMessages', () => ({
  default: {
    unreadCount: state.unreadCount,
    list: state.list,
    markRead: state.markRead,
    markAllRead: state.markAllRead
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isAuthenticated() {
      return state.isAuthenticated
    },
    get user() {
      return state.userId ? { id: state.userId } : null
    }
  })
}))

import { useSiteMessagesStore, SITE_MESSAGE_POPPED_KEY_PREFIX } from '@/stores/siteMessages'

function message(id: number, extra: Record<string, unknown> = {}) {
  return {
    id,
    category: 'admin',
    title: `t${id}`,
    content: 'c',
    from: 'staff',
    read_at: null,
    created_at: '2026-10-01T00:00:00Z',
    ...extra
  }
}

describe('site messages store (fork)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    sessionStorage.clear()
    state.isAuthenticated = true
    state.userId = 7
    state.markRead.mockResolvedValue(undefined)
  })

  it('pops unread messages after login, oldest first', async () => {
    state.unreadCount.mockResolvedValue({ count: 2 })
    // 接口按时间倒序返回：3 比 2 新
    state.list.mockResolvedValue({ items: [message(3), message(2)], total: 2 })
    const store = useSiteMessagesStore()

    await store.fetchUnreadCount()
    await flushPromises()

    expect(store.unreadCount).toBe(2)
    expect(state.list).toHaveBeenCalledWith(1, 10, { unreadOnly: true })
    expect(store.currentPopup?.id).toBe(2)
    expect(store.popupQueue.map((m) => m.id)).toEqual([3])
  })

  it('does not pop anything when there is nothing unread', async () => {
    state.unreadCount.mockResolvedValue({ count: 0 })
    const store = useSiteMessagesStore()
    await store.fetchUnreadCount()
    await flushPromises()
    expect(state.list).not.toHaveBeenCalled()
    expect(store.currentPopup).toBeNull()
  })

  it('acknowledging marks read, decrements the badge and shows the next one', async () => {
    vi.useFakeTimers()
    try {
      state.unreadCount.mockResolvedValue({ count: 2 })
      state.list.mockResolvedValue({ items: [message(3), message(2)], total: 2 })
      const store = useSiteMessagesStore()
      await store.fetchUnreadCount()
      await flushPromises()

      await store.acknowledgePopup()
      expect(state.markRead).toHaveBeenCalledWith(2)
      expect(store.unreadCount).toBe(1)
      expect(store.currentPopup).toBeNull()

      vi.advanceTimersByTime(300)
      expect(store.currentPopup?.id).toBe(3)
    } finally {
      vi.useRealTimers()
    }
  })

  it('never pops the same message twice in one session, and re-pops when new ones arrive', async () => {
    state.unreadCount.mockResolvedValueOnce({ count: 1 })
    state.list.mockResolvedValueOnce({ items: [message(1)], total: 1 })
    const store = useSiteMessagesStore()
    await store.fetchUnreadCount()
    await flushPromises()
    expect(store.currentPopup?.id).toBe(1)
    expect(JSON.parse(sessionStorage.getItem(`${SITE_MESSAGE_POPPED_KEY_PREFIX}7`) ?? '[]')).toEqual([1])

    // 关掉弹窗但没标已读（查看全部）：同一会话里不会再弹
    store.openInboxFromPopup()
    expect(store.inboxOpen).toBe(true)

    // 页面刷新：新的 store 实例，sessionStorage 里记着弹过 1
    setActivePinia(createPinia())
    const refreshed = useSiteMessagesStore()
    state.unreadCount.mockResolvedValueOnce({ count: 1 })
    state.list.mockResolvedValueOnce({ items: [message(1)], total: 1 })
    await refreshed.fetchUnreadCount()
    await flushPromises()
    expect(refreshed.currentPopup).toBeNull()

    // 新消息到达：未读数上涨，只弹新的那条
    state.unreadCount.mockResolvedValueOnce({ count: 2 })
    state.list.mockResolvedValueOnce({ items: [message(2), message(1)], total: 2 })
    await refreshed.fetchUnreadCount()
    await flushPromises()
    expect(refreshed.currentPopup?.id).toBe(2)
    expect(refreshed.popupQueue).toEqual([])
  })

  it('mark all read clears the queue and the badge', async () => {
    state.unreadCount.mockResolvedValue({ count: 2 })
    state.list.mockResolvedValue({ items: [message(3), message(2)], total: 2 })
    state.markAllRead.mockResolvedValue({ updated: 2 })
    const store = useSiteMessagesStore()
    await store.fetchUnreadCount()
    await flushPromises()

    await store.markAllRead()
    expect(store.unreadCount).toBe(0)
    expect(store.currentPopup).toBeNull()
    expect(store.popupQueue).toEqual([])
  })

  it('reset stops everything and forgets popped ids', async () => {
    state.unreadCount.mockResolvedValue({ count: 1 })
    state.list.mockResolvedValue({ items: [message(1)], total: 1 })
    const store = useSiteMessagesStore()
    await store.fetchUnreadCount()
    await flushPromises()
    expect(sessionStorage.getItem(`${SITE_MESSAGE_POPPED_KEY_PREFIX}7`)).not.toBeNull()

    store.reset()
    expect(store.unreadCount).toBe(0)
    expect(store.currentPopup).toBeNull()
    expect(sessionStorage.getItem(`${SITE_MESSAGE_POPPED_KEY_PREFIX}7`)).toBeNull()
  })

  it('skips fetching when signed out', async () => {
    state.isAuthenticated = false
    const store = useSiteMessagesStore()
    await store.fetchUnreadCount()
    expect(state.unreadCount).not.toHaveBeenCalled()
    expect(store.unreadCount).toBe(0)
  })

  it('start is idempotent', () => {
    vi.useFakeTimers()
    try {
      state.unreadCount.mockResolvedValue({ count: 0 })
      const store = useSiteMessagesStore()
      store.start()
      store.start()
      expect(state.unreadCount).toHaveBeenCalledTimes(1)
      store.stop()
    } finally {
      vi.useRealTimers()
    }
  })
})
