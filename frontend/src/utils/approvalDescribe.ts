/**
 * 审批申请的文字化描述（fork 本地功能）。
 *
 * 后端只保存脱敏后的请求体（JSON），管理员审批时不该去读一串 JSON。这里按动作把请求体翻译成
 * 「一句话概述 + 逐项字段」，未识别的字段也会以「字段名：值」列出，保证不会有内容被藏起来。
 */

import type { AdminApprovalRequest } from '@/api/admin/approvals'

export type ApprovalTranslate = (key: string, params?: Record<string, unknown>) => string

export interface ApprovalDescribeContext {
  t: ApprovalTranslate
  /** 分组 id → 名称（拿不到时返回 undefined，展示为 #id） */
  groupName?: (id: number) => string | undefined
  /** 用户属性定义 id → 名称 */
  attributeName?: (id: number) => string | undefined
}

export interface ApprovalDescriptionLine {
  label: string
  value: string
}

export interface ApprovalDescription {
  /** 一句话概述，如「给 a@b.com 增加余额 10」 */
  summary: string
  /** 逐项字段说明 */
  lines: ApprovalDescriptionLine[]
}

type Body = Record<string, unknown>

const P = 'operator.approval.describe'
const REDACTED = '***'

function parseBody(raw?: string): Body {
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Body) : {}
  } catch {
    return {}
  }
}

function isPresent(v: unknown): boolean {
  return v !== undefined && v !== null && v !== ''
}

function num(v: unknown): number | null {
  if (typeof v === 'number' && Number.isFinite(v)) return v
  if (typeof v === 'string' && v.trim() !== '' && Number.isFinite(Number(v))) return Number(v)
  return null
}

function fmtNum(v: unknown): string {
  const n = num(v)
  if (n === null) return String(v ?? '')
  return Number.isInteger(n) ? String(n) : String(Math.round(n * 10000) / 10000)
}

function fmtRaw(v: unknown): string {
  if (v === null || v === undefined) return '-'
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  try {
    return JSON.stringify(v)
  } catch {
    return String(v)
  }
}

function idList(v: unknown): number[] {
  if (!Array.isArray(v)) return []
  return v.map((x) => num(x)).filter((n): n is number => n !== null)
}

export function describeApproval(req: AdminApprovalRequest, ctx: ApprovalDescribeContext): ApprovalDescription {
  const t = ctx.t
  const body = parseBody(req.request_body)
  const used = new Set<string>()
  const lines: ApprovalDescriptionLine[] = []
  const target = req.target_summary || '-'

  const take = (key: string): unknown => {
    used.add(key)
    return body[key]
  }
  const add = (labelKey: string, value: string) => {
    lines.push({ label: t(`${P}.fields.${labelKey}`), value })
  }
  const yesNo = (v: unknown) => t(`${P}.values.${v ? 'yes' : 'no'}`)
  const groupLabel = (id: number) => ctx.groupName?.(id) ?? `#${id}`
  const groupsLabel = (ids: number[]) => (ids.length === 0 ? t(`${P}.values.none`) : ids.map(groupLabel).join(', '))
  const scopeLabel = (): string => {
    const all = take('all')
    const ids = idList(take('user_ids'))
    if (all === true) return t(`${P}.values.scopeAll`)
    return t(`${P}.values.scopeUsers`, { count: ids.length, ids: ids.slice(0, 8).join(', ') + (ids.length > 8 ? '...' : '') })
  }
  const windowLabel = (window: string): string => {
    const key = window === 'daily' ? 'quotaDaily' : window === 'weekly' ? 'quotaWeekly' : window === 'monthly' ? 'quotaMonthly' : ''
    return key ? t(`${P}.values.${key}`) : window
  }

  let summary = ''

  switch (req.action) {
    case 'admin.users.create': {
      const email = fmtRaw(take('email'))
      summary = t(`${P}.summary.userCreate`, { email })
      take('password')
      const username = take('username')
      if (isPresent(username)) add('username', fmtRaw(username))
      const balance = take('balance')
      if (isPresent(balance)) add('initialBalance', fmtNum(balance))
      const concurrency = take('concurrency')
      if (num(concurrency)) add('concurrency', fmtNum(concurrency))
      const rpm = take('rpm_limit')
      if (num(rpm)) add('rpmLimit', fmtNum(rpm))
      const groups = take('allowed_groups')
      if (Array.isArray(groups) && groups.length > 0) add('allowedGroups', groupsLabel(idList(groups)))
      const restrict = take('restrict_public_groups')
      if (restrict === true) add('restrictPublicGroups', yesNo(true))
      const notes = take('notes')
      if (isPresent(notes)) add('notes', fmtRaw(notes))
      break
    }
    case 'admin.users.update': {
      summary = t(`${P}.summary.userUpdate`, { target })
      const email = take('email')
      if (isPresent(email)) add('email', fmtRaw(email))
      const password = take('password')
      if (isPresent(password)) add('resetPassword', yesNo(true))
      const username = take('username')
      if (username !== undefined && username !== null) add('username', fmtRaw(username))
      const status = take('status')
      if (isPresent(status)) {
        add(
          'status',
          status === 'active' ? t(`${P}.values.statusActive`) : status === 'disabled' ? t(`${P}.values.statusDisabled`) : fmtRaw(status)
        )
      }
      const balance = take('balance')
      if (isPresent(balance)) add('balance', fmtNum(balance))
      const concurrency = take('concurrency')
      if (isPresent(concurrency)) add('concurrency', fmtNum(concurrency))
      const rpm = take('rpm_limit')
      if (isPresent(rpm)) add('rpmLimit', fmtNum(rpm))
      const groups = take('allowed_groups')
      if (Array.isArray(groups)) add('allowedGroups', groupsLabel(idList(groups)))
      const restrict = take('restrict_public_groups')
      if (typeof restrict === 'boolean') add('restrictPublicGroups', yesNo(restrict))
      const rates = take('group_rates')
      if (rates && typeof rates === 'object' && !Array.isArray(rates)) {
        const parts = Object.entries(rates as Record<string, unknown>).map(([gid, rate]) => {
          const id = num(gid)
          const name = id === null ? gid : groupLabel(id)
          return rate === null || rate === undefined ? `${name}: ${t(`${P}.values.remove`)}` : `${name} x${fmtNum(rate)}`
        })
        if (parts.length > 0) add('groupRates', parts.join('; '))
      }
      const notes = take('notes')
      if (notes !== undefined && notes !== null) add('notes', fmtRaw(notes))
      const role = take('role')
      if (isPresent(role)) add('role', fmtRaw(role))
      break
    }
    case 'admin.users.balance.create': {
      const amount = fmtNum(take('balance'))
      const op = String(take('operation') ?? 'add')
      const key = op === 'subtract' ? 'balanceSubtract' : op === 'set' ? 'balanceSet' : 'balanceAdd'
      summary = t(`${P}.summary.${key}`, { target, amount })
      const notes = take('notes')
      if (isPresent(notes)) add('notes', fmtRaw(notes))
      break
    }
    case 'admin.users.replace_group.create': {
      const from = num(take('old_group_id'))
      const to = num(take('new_group_id'))
      summary = t(`${P}.summary.replaceGroup`, {
        target,
        from: from === null ? '-' : groupLabel(from),
        to: to === null ? '-' : groupLabel(to)
      })
      break
    }
    case 'admin.users.batch_concurrency.create': {
      const scope = scopeLabel()
      const value = fmtNum(take('concurrency'))
      const mode = String(take('mode') ?? 'set')
      summary = t(`${P}.summary.${mode === 'add' ? 'batchConcurrencyAdd' : 'batchConcurrencySet'}`, { scope, value })
      break
    }
    case 'admin.users.batch_limits.create': {
      const scope = scopeLabel()
      summary = t(`${P}.summary.batchLimits`, { scope })
      const concurrency = take('concurrency')
      if (isPresent(concurrency)) add('concurrency', fmtNum(concurrency))
      const rpm = take('rpm_limit')
      if (isPresent(rpm)) add('rpmLimit', fmtNum(rpm))
      break
    }
    case 'admin.users.platform_quotas.update': {
      summary = t(`${P}.summary.platformQuotas`, { target })
      const quotas = take('quotas')
      if (Array.isArray(quotas)) {
        for (const q of quotas as Body[]) {
          if (!q || typeof q !== 'object') continue
          const limit = (v: unknown) => (isPresent(v) ? fmtNum(v) : t(`${P}.values.unlimited`))
          lines.push({
            label: fmtRaw(q.platform ?? '-'),
            value: [
              `${t(`${P}.values.quotaDaily`)} ${limit(q.daily_limit_usd)}`,
              `${t(`${P}.values.quotaWeekly`)} ${limit(q.weekly_limit_usd)}`,
              `${t(`${P}.values.quotaMonthly`)} ${limit(q.monthly_limit_usd)}`
            ].join(' / ')
          })
        }
      }
      break
    }
    case 'admin.users.platform_quotas.reset.create': {
      const platform = fmtRaw(take('platform'))
      const window = String(take('window') ?? '')
      summary = t(`${P}.summary.platformQuotaReset`, { target, platform, window: windowLabel(window) })
      break
    }
    case 'admin.users.attributes.update': {
      summary = t(`${P}.summary.attributes`, { target })
      const values = take('values')
      if (values && typeof values === 'object' && !Array.isArray(values)) {
        for (const [id, value] of Object.entries(values as Record<string, unknown>)) {
          const n = num(id)
          lines.push({ label: (n !== null && ctx.attributeName?.(n)) || `#${id}`, value: fmtRaw(value) })
        }
      }
      break
    }
    case 'admin.users.auth_identities.create': {
      const provider = fmtRaw(take('provider_type') || '-')
      summary = t(`${P}.summary.authIdentity`, { target, provider })
      const key = take('provider_key')
      if (isPresent(key)) add('providerKey', fmtRaw(key))
      const subject = take('provider_subject')
      if (isPresent(subject)) add('providerSubject', fmtRaw(subject))
      take('issuer')
      take('metadata')
      take('channel')
      break
    }
    case 'admin.api_keys.update': {
      const gid = take('group_id')
      const reset = take('reset_rate_limit_usage')
      const g = num(gid)
      if (g !== null && g > 0) summary = t(`${P}.summary.apiKeyBind`, { target, group: groupLabel(g) })
      else if (g === 0) summary = t(`${P}.summary.apiKeyUnbind`, { target })
      else summary = t(`${P}.summary.apiKeyUpdate`, { target })
      if (reset === true) add('resetRateLimit', yesNo(true))
      break
    }
    case 'admin.subscriptions.assign.create': {
      summary = t(`${P}.summary.subscriptionAssign`, { target })
      take('user_id')
      const gid = num(take('group_id'))
      if (gid !== null) add('group', groupLabel(gid))
      const days = num(take('validity_days'))
      add('validity', days && days > 0 ? t(`${P}.values.days`, { n: days }) : t(`${P}.values.validityDefault`))
      const notes = take('notes')
      if (isPresent(notes)) add('notes', fmtRaw(notes))
      break
    }
    case 'admin.subscriptions.bulk_assign.create': {
      const ids = idList(take('user_ids'))
      const gid = num(take('group_id'))
      summary = t(`${P}.summary.subscriptionBulkAssign`, { count: ids.length, group: gid === null ? '-' : groupLabel(gid) })
      add('users', ids.slice(0, 20).join(', ') + (ids.length > 20 ? '...' : ''))
      const days = num(take('validity_days'))
      add('validity', days && days > 0 ? t(`${P}.values.days`, { n: days }) : t(`${P}.values.validityDefault`))
      const notes = take('notes')
      if (isPresent(notes)) add('notes', fmtRaw(notes))
      break
    }
    case 'admin.subscriptions.extend.create': {
      const days = num(take('days')) ?? 0
      summary = t(`${P}.summary.${days < 0 ? 'subscriptionShorten' : 'subscriptionExtend'}`, { target, days: Math.abs(days) })
      break
    }
    case 'admin.subscriptions.reset_quota.create': {
      const picked: string[] = []
      if (take('daily') === true) picked.push(t(`${P}.values.quotaDaily`))
      if (take('weekly') === true) picked.push(t(`${P}.values.quotaWeekly`))
      if (take('monthly') === true) picked.push(t(`${P}.values.quotaMonthly`))
      summary = t(`${P}.summary.subscriptionResetQuota`, {
        target,
        windows: picked.length > 0 ? picked.join(', ') : t(`${P}.values.allWindows`)
      })
      break
    }
    case 'admin.subscriptions.revoke.create':
      summary = t(`${P}.summary.subscriptionRevoke`, { target })
      break
    case 'admin.subscriptions.restore.create':
      summary = t(`${P}.summary.subscriptionRestore`, { target })
      break
    case 'admin.subscriptions.delete':
      summary = t(`${P}.summary.subscriptionDelete`, { target })
      break
    default:
      summary = t(`${P}.summary.unknown`, { action: req.action, target })
  }

  // 没识别的字段一律原样列出，不让任何内容藏在 JSON 里。
  for (const [key, value] of Object.entries(body)) {
    if (used.has(key)) continue
    lines.push({ label: key, value: value === REDACTED ? t(`${P}.values.redacted`) : fmtRaw(value) })
  }

  return { summary, lines }
}
