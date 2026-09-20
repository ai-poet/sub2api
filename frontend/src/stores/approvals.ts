import { defineStore } from 'pinia'
import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { useAuthStore } from './auth'

// 运维管理员写操作审批（fork 本地功能）：待审数量，供侧边栏角标使用。
// 管理员看到全站待审数，运维管理员看到自己的待审数（后端按角色决定口径）。
const POLL_INTERVAL_MS = 60 * 1000

// 后端未返回上限（旧版本）时的兜底，与 service 默认常量一致。
const DEFAULT_PENDING_LIMIT = 20
const DEFAULT_BATCH_LIMIT = 50

const positiveIntOr = (value: unknown, fallback: number): number => {
  const n = Number(value)
  return Number.isFinite(n) && n >= 1 ? Math.floor(n) : fallback
}

export const useApprovalsStore = defineStore('approvals', () => {
  const pendingCount = ref(0)
  // 生效中的数量上限（站点设置可调）：pending-count 接口顺带返回，审批页按 batchLimit 分批。
  const pendingLimit = ref(DEFAULT_PENDING_LIMIT)
  const batchLimit = ref(DEFAULT_BATCH_LIMIT)
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
      pendingLimit.value = positiveIntOr(res?.pending_limit, DEFAULT_PENDING_LIMIT)
      batchLimit.value = positiveIntOr(res?.batch_limit, DEFAULT_BATCH_LIMIT)
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

  return { pendingCount, pendingLimit, batchLimit, loading, fetchPendingCount, start, stop, reset }
})
