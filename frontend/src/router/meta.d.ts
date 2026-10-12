/**
 * Type definitions for Vue Router meta fields
 * Extends the RouteMeta interface with custom properties
 */

import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /**
     * Whether this route requires authentication
     * @default true
     */
    requiresAuth?: boolean

    /**
     * Whether this route requires admin role
     * @default false
     */
    requiresAdmin?: boolean

    /**
     * 运维管理员（operator）是否可访问该管理路由。只在后端白名单覆盖的页面上标记，
     * 未标记的 requiresAdmin 路由对 operator 默认拒绝（与后端默认拒绝的白名单一致）。
     * @default false
     */
    operatorAllowed?: boolean

    /**
     * fork：多域名登录交接页的模式（见 views/auth/SessionHandoffView.vue）
     */
    handoffMode?: 'give' | 'complete' | 'pull'

    /**
     * fork：未登录时，公开定价开启则放行（见 router/publicCatalog.ts），页面渲染匿名视图
     */
    publicCatalog?: boolean

    /**
     * fork：页面自己在正文里显示大标题（TokenRouter 的仪表盘布局），顶栏不再重复显示标题和说明
     */
    pageHeadingInBody?: boolean

    /**
     * Page title for this route
     */
    title?: string

    /**
     * Optional breadcrumb items for navigation
     */
    breadcrumbs?: Array<{
      label: string
      to?: string
    }>

    /**
     * Icon name for this route (for sidebar navigation)
     */
    icon?: string

    /**
     * Whether to hide this route from navigation menu
     * @default false
     */
    hideInMenu?: boolean

    /**
     * @default false
     */

    /**
     * 是否要求风控中心功能开关已启用
     * @default false
     */
    requiresRiskControl?: boolean

    /**
     * 是否要求订阅功能开关（subscription_enabled，opt-out）未被显式关闭
     * @default false
     */
    requiresSubscription?: boolean

    /**
     * i18n key for the page title
     */
    titleKey?: string

    /**
     * i18n key for the page description
     */
    descriptionKey?: string
  }
}
