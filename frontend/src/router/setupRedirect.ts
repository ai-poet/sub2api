export function resolveCompletedSetupRedirectPath(
  isAuthenticated: boolean,
  isAdmin: boolean,
  isOperator = false
): string {
  if (!isAuthenticated) {
    return '/login'
  }

  if (isAdmin) return '/admin/dashboard'
  // 运维管理员（operator）没有管理仪表盘，落到运维监控。
  if (isOperator) return '/admin/ops'
  return '/dashboard'
}
