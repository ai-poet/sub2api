/**
 * 运维管理员写操作审批（fork 本地功能）的前端约定。
 *
 * 后端在认证层把 operator 的写请求排队并返回 202 + { approval_request_id }；
 * apiClient 把这种响应转成带 APPROVAL_PENDING 的拒绝并广播 `approval-queued` 事件，
 * 各弹窗只需在 catch 里用 isApprovalQueued() 识别后关闭自己，不再弹"成功"或"失败"。
 */

export const APPROVAL_PENDING_CODE = 'APPROVAL_PENDING'
export const APPROVAL_QUEUED_EVENT = 'approval-queued'

export interface ApprovalQueuedPayload {
  approval_request_id: number
  status: string
  action: string
  target_summary: string
  expires_at: string
}

export interface ApprovalQueuedError {
  status: 202
  code: typeof APPROVAL_PENDING_CODE
  message: string
  approval: ApprovalQueuedPayload
}

export function isApprovalQueuedPayload(value: unknown): value is ApprovalQueuedPayload {
  if (!value || typeof value !== 'object') return false
  const id = (value as { approval_request_id?: unknown }).approval_request_id
  return typeof id === 'number' && Number.isFinite(id) && id > 0
}

export function isApprovalQueued(err: unknown): err is ApprovalQueuedError {
  if (!err || typeof err !== 'object') return false
  return (err as { code?: unknown }).code === APPROVAL_PENDING_CODE && isApprovalQueuedPayload((err as { approval?: unknown }).approval)
}

/** 广播"已提交审批"事件，供 App.vue 统一弹提示并刷新角标。 */
export function dispatchApprovalQueued(payload: ApprovalQueuedPayload): void {
  if (typeof window === 'undefined' || typeof window.dispatchEvent !== 'function') return
  try {
    window.dispatchEvent(new CustomEvent(APPROVAL_QUEUED_EVENT, { detail: payload }))
  } catch {
    // 非浏览器环境（测试）没有 CustomEvent 时静默
  }
}

/** 审计动作名 → operator.approval.actionLabels.* 的键；未知动作返回 null（前端回退显示动作名）。 */
const APPROVAL_ACTION_LABEL_KEYS: Record<string, string> = {
  'admin.users.create': 'userCreate',
  'admin.users.update': 'userUpdate',
  'admin.users.balance.create': 'userBalance',
  'admin.users.replace_group.create': 'userReplaceGroup',
  'admin.users.batch_concurrency.create': 'userBatchConcurrency',
  'admin.users.batch_limits.create': 'userBatchLimits',
  'admin.users.platform_quotas.update': 'userPlatformQuotas',
  'admin.users.platform_quotas.reset.create': 'userPlatformQuotaReset',
  'admin.users.attributes.update': 'userAttributes',
  'admin.users.auth_identities.create': 'userAuthIdentity',
  'admin.api_keys.update': 'apiKeyUpdate',
  'admin.subscriptions.assign.create': 'subscriptionAssign',
  'admin.subscriptions.bulk_assign.create': 'subscriptionBulkAssign',
  'admin.subscriptions.extend.create': 'subscriptionExtend',
  'admin.subscriptions.reset_quota.create': 'subscriptionResetQuota',
  'admin.subscriptions.revoke.create': 'subscriptionRevoke',
  'admin.subscriptions.restore.create': 'subscriptionRestore',
  'admin.subscriptions.delete': 'subscriptionRevoke'
}

export function approvalActionLabelKey(action: string): string | null {
  const key = APPROVAL_ACTION_LABEL_KEYS[action]
  return key ? `operator.approval.actionLabels.${key}` : null
}
