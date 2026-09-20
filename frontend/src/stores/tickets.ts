import { defineStore } from 'pinia'
import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import ticketsAPI from '@/api/tickets'
import { useAuthStore } from './auth'

// 工单（fork 本地功能）：侧边栏角标。
// 客服（admin / operator）看全站待处理数，普通用户看自己有未读客服回复的工单数。
const POLL_INTERVAL_MS = 60 * 1000

export const useTicketsStore = defineStore('tickets', () => {
  /** 客服侧：status=open 的工单数 */
  const openCount = ref(0)
  /** 用户侧：user_unread=true 的工单数 */
  const unreadCount = ref(0)
  const loading = ref(false)
  let timer: number | null = null

  async function fetchCounts() {
    const authStore = useAuthStore()
    if (!authStore.isAuthenticated) {
      openCount.value = 0
      unreadCount.value = 0
      return
    }
    if (loading.value) return
    loading.value = true
    try {
      if (authStore.hasConsoleAccess) {
        const res = await adminAPI.tickets.openCount()
        openCount.value = Number(res?.count ?? 0) || 0
      } else {
        const res = await ticketsAPI.unreadCount()
        unreadCount.value = Number(res?.count ?? 0) || 0
      }
    } catch (err) {
      console.error('Failed to fetch ticket counts:', err)
    } finally {
      loading.value = false
    }
  }

  /** 启动轮询（幂等）：立即取一次，之后每分钟刷新；页面不可见时跳过。 */
  function start() {
    if (timer !== null) return
    void fetchCounts()
    timer = window.setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) return
      void fetchCounts()
    }, POLL_INTERVAL_MS)
  }

  function stop() {
    if (timer !== null) {
      window.clearInterval(timer)
      timer = null
    }
  }

  function reset() {
    stop()
    openCount.value = 0
    unreadCount.value = 0
  }

  return { openCount, unreadCount, loading, fetchCounts, start, stop, reset }
})
