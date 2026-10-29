import { computed, ref } from 'vue'
import { keysAPI } from '@/api/keys'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'

// 画布走 OpenAI / Grok 生图，与 Gemini 批量生图的入口开关不是同一组能力。
const loaded = ref(false)
const loading = ref(false)
const hasAllowedCanvasKey = ref(false)
let pendingLoad: Promise<boolean> | null = null
const pageSize = 100

function keyAllowsCanvas(key: ApiKey): boolean {
  return (
    key.status === 'active' &&
    (key.group?.platform === 'openai' || key.group?.platform === 'grok') &&
    key.group?.allow_image_generation === true
  )
}

async function loadCanvasAccess(force = false): Promise<boolean> {
  const authStore = useAuthStore()
  if (!authStore.isAuthenticated) {
    loaded.value = true
    hasAllowedCanvasKey.value = false
    return false
  }
  if (loaded.value && !force) return hasAllowedCanvasKey.value
  if (pendingLoad && !force) return pendingLoad

  loading.value = true
  pendingLoad = (async () => {
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, pageSize, {
        status: 'active',
        sort_by: 'created_at',
        sort_order: 'desc',
      })
      if ((response.items || []).some(keyAllowsCanvas)) {
        hasAllowedCanvasKey.value = true
        loaded.value = true
        return true
      }
      if (page >= response.pages || (response.items || []).length === 0) {
        hasAllowedCanvasKey.value = false
        loaded.value = true
        return false
      }
      page += 1
    }
  })()
    .catch(() => {
      hasAllowedCanvasKey.value = false
      loaded.value = false
      return false
    })
    .finally(() => {
      loading.value = false
      pendingLoad = null
    })

  return pendingLoad
}

export function useCanvasAccess() {
  return {
    canUseCanvas: computed(() => hasAllowedCanvasKey.value),
    canvasAccessLoaded: computed(() => loaded.value),
    canvasAccessLoading: computed(() => loading.value),
    refreshCanvasAccess: loadCanvasAccess,
  }
}
