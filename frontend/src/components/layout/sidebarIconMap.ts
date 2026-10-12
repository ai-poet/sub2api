// fork：侧栏路径 → 动态图标名（components/motion-icons/registry.ts）。
// 只用 type 导入，不在主包里引入 motion-v；没有登记的路径（含自定义菜单）保持静态图标。
// 新增侧栏入口时在这里登记，sidebarIconMap.spec.ts 会检查 AppSidebar 里的每个路径。
import type { AnimatedIconName } from '@/components/motion-icons/registry'

export const SIDEBAR_ANIMATED_ICONS: Readonly<Record<string, AnimatedIconName>> = {
  // 用户菜单
  '/dashboard': 'dashboard',
  '/keys': 'key',
  '/batch-image': 'batchImage',
  '/usage': 'chart',
  '/available-channels': 'layers',
  '/subscriptions': 'creditCard',
  '/purchase': 'recharge',
  '/redeem': 'gift',
  '/integration-guide': 'book',
  '/tickets': 'chat',
  '/models': 'modelCatalog',
  '/leaderboard': 'ranking',
  '/model-status': 'server',
  '/referral': 'affiliate',
  '/profile': 'userCircle',

  // 管理菜单
  '/admin/dashboard': 'dashboard',
  '/admin/ops': 'monitor',
  '/admin/usage': 'chart',
  '/admin/risk-control': 'shieldCheck',
  '/admin/users': 'users',
  '/admin/subscriptions': 'creditCard',
  '/admin/approvals': 'badge',
  '/admin/tickets': 'chat',
  '/admin/groups': 'folder',
  '/admin/accounts': 'globe',
  '/admin/plugins': 'cube',
  '/admin/announcements': 'bell',
  '/admin/proxies': 'server',
  '/admin/security-audit': 'shield',
  '/admin/prompt-audit': 'shield',
  '/admin/redeem': 'ticket',
  '/admin/promo-codes': 'gift',
  '/admin/payment-management': 'order',
  '/admin/audit-logs': 'history',
  '/admin/settings': 'cog',

  // 底部操作
  'theme:sun': 'sun',
  'theme:moon': 'moon',
}

export function resolveSidebarAnimatedIcon(path: string): AnimatedIconName | undefined {
  return SIDEBAR_ANIMATED_ICONS[path]
}
