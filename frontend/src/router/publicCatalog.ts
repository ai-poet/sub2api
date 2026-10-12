// fork：模型广场对匿名访客开放。
// /models 仍标记 requiresAuth，路由守卫在「未登录 → 跳登录」之前调用这里：
// 公开定价开启（public_pricing_enabled，opt-out）且不在 backend mode 时放行，
// 页面用 /pricing/public 渲染匿名视图；否则维持原行为，跳登录并带回 redirect。

import type { RouteLocationNormalized } from 'vue-router'
import type { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

type AppStore = ReturnType<typeof useAppStore>

export async function allowAnonymousCatalog(to: RouteLocationNormalized, appStore: AppStore): Promise<boolean> {
  if (to.meta.publicCatalog !== true) return false
  if (!appStore.publicSettingsLoaded) {
    try {
      await appStore.fetchPublicSettings()
    } catch {
      // 设置拿不到时按开关的默认值处理（公开定价默认开启）。
    }
  }
  if (appStore.backendModeEnabled) return false
  return resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.publicPricing)
}
