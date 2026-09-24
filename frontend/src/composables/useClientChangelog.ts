import { computed, ref } from 'vue'
import { getClientChangelog, type ClientChangelogEntry } from '@/api/changelog'

// 首页（最新版本徽标）和 /changelog 页共用一次请求结果，在两页之间来回切换不重复拉取。
// 请求失败不记为已加载，下次进入页面会重试。
const entries = ref<ClientChangelogEntry[]>([])
const loaded = ref(false)
const loading = ref(false)
let pending: Promise<void> | null = null

export function useClientChangelog() {
  function load(): Promise<void> {
    if (loaded.value) return Promise.resolve()
    if (!pending) {
      loading.value = true
      pending = getClientChangelog()
        .then((list) => {
          entries.value = list
          loaded.value = true
        })
        .catch(() => {
          entries.value = []
        })
        .finally(() => {
          loading.value = false
          pending = null
        })
    }
    return pending
  }

  const latestVersion = computed(() => entries.value[0]?.version ?? '')

  return { entries, loaded, loading, latestVersion, load }
}

/** 仅供测试：清掉模块级缓存。 */
export function resetClientChangelogCache() {
  entries.value = []
  loaded.value = false
  loading.value = false
  pending = null
}
