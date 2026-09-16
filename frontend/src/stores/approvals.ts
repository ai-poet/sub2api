import { defineStore } from 'pinia'
import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { useAuthStore } from './auth'

// 运维管理员写操作审批（fork 本地功能）：待审数量，供侧边栏角标使用。
// 管理员看到全站待审数，运维管理员看到自己的待审数（后端按角色决定口径）。
const POLL_INTERVAL_MS = 60 * 1000

export const useApprovalsStore = defineStore('approvals', () => {
  const pendingCount = ref(0)
  const loading = ref(false)
  let timer: number | null = null

  async function fetchPendingCount() {
    const authStore = useAuthStore()
    if (!authStore.hasConsoleAccess) {
      pendingCount.value = 0
      return
    }
    if (loading.value) return
    loading.value = true
    try {
      const res = await adminAPI.approvals.pendingCount()
      pendingCount.value = Number(res?.pending ?? 0) || 0
    } catch (err) {
      console.error('Failed to fetch pending approvals:', err)
    } finally {
      loading.value = false
    }
  }

  /** 启动轮询（幂等）：立即取一次，之后每分钟刷新；页面不可见时跳过。 */
  function start() {
    if (timer !== null) return
    void fetchPendingCount()
    timer = window.setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) return
      void fetchPendingCount()
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
    pendingCount.value = 0
  }

  return { pendingCount, loading, fetchPendingCount, start, stop, reset }
})
