import { describe, expect, it } from 'vitest'
import { describeApproval } from '../approvalDescribe'
import type { AdminApprovalRequest } from '@/api/admin/approvals'

const t = (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key)
const ctx = {
  t,
  groupName: (id: number) => ({ 3: 'Pro', 4: 'Basic' } as Record<number, string>)[id],
  attributeName: (id: number) => ({ 5: '公司' } as Record<number, string>)[id]
}

function req(action: string, body: unknown, extra: Partial<AdminApprovalRequest> = {}): AdminApprovalRequest {
  return {
    id: 1,
    status: 'pending',
    action,
    method: 'POST',
    route_template: '/x',
    request_path: '/x',
    target_type: 'user',
    target_summary: 'u@example.com',
    requester: { id: 2, email: 'ops@example.com' },
    request_body: typeof body === 'string' ? body : JSON.stringify(body),
    expires_at: '',
    created_at: '',
    updated_at: '',
    ...extra
  }
}

describe('describeApproval', () => {
  it('describes a balance top-up with the amount and target', () => {
    const d = describeApproval(req('admin.users.balance.create', { balance: 10.5, operation: 'add', notes: '充值' }), ctx)
    expect(d.summary).toBe('operator.approval.describe.summary.balanceAdd:{"target":"u@example.com","amount":"10.5"}')
    expect(d.lines).toEqual([{ label: 'operator.approval.describe.fields.notes', value: '充值' }])
  })

  it('describes a user update field by field, never leaking the redacted password', () => {
    const d = describeApproval(
      req('admin.users.update', {
        password: '***',
        status: 'disabled',
        concurrency: 5,
        allowed_groups: [3, 4],
        restrict_public_groups: true,
        group_rates: { '3': 0.8, '4': null },
        role: 'user'
      }),
      ctx
    )
    expect(d.summary).toContain('summary.userUpdate')
    const labels = d.lines.map((l) => l.label)
    expect(labels).toContain('operator.approval.describe.fields.resetPassword')
    expect(d.lines.find((l) => l.label.endsWith('.status'))?.value).toBe('operator.approval.describe.values.statusDisabled')
    expect(d.lines.find((l) => l.label.endsWith('.allowedGroups'))?.value).toBe('Pro, Basic')
    expect(d.lines.find((l) => l.label.endsWith('.groupRates'))?.value).toBe('Pro x0.8; Basic: operator.approval.describe.values.remove')
    expect(d.lines.find((l) => l.label.endsWith('.role'))?.value).toBe('user')
    expect(JSON.stringify(d)).not.toContain('***')
  })

  it('describes subscription assignment with group name and validity', () => {
    const d = describeApproval(req('admin.subscriptions.assign.create', { user_id: 7, group_id: 3, validity_days: 30 }), ctx)
    expect(d.lines).toEqual([
      { label: 'operator.approval.describe.fields.group', value: 'Pro' },
      { label: 'operator.approval.describe.fields.validity', value: 'operator.approval.describe.values.days:{"n":30}' }
    ])
    const dflt = describeApproval(req('admin.subscriptions.assign.create', { user_id: 7, group_id: 9 }), ctx)
    expect(dflt.lines[0]).toEqual({ label: 'operator.approval.describe.fields.group', value: '#9' })
    expect(dflt.lines[1].value).toBe('operator.approval.describe.values.validityDefault')
  })

  it('describes batch operations with their scope', () => {
    const all = describeApproval(req('admin.users.batch_concurrency.create', { all: true, concurrency: 3, mode: 'set' }), ctx)
    expect(all.summary).toBe(
      'operator.approval.describe.summary.batchConcurrencySet:{"scope":"operator.approval.describe.values.scopeAll","value":"3"}'
    )
    const some = describeApproval(req('admin.users.batch_limits.create', { user_ids: [1, 2, 3], rpm_limit: 60 }), ctx)
    expect(some.summary.replace(/\\"/g, '"')).toContain('scopeUsers:{"count":3,"ids":"1, 2, 3"}')
    expect(some.lines).toEqual([{ label: 'operator.approval.describe.fields.rpmLimit', value: '60' }])
  })

  it('describes platform quotas, quota resets and attribute updates', () => {
    const quotas = describeApproval(
      req('admin.users.platform_quotas.update', { quotas: [{ platform: 'openai', daily_limit_usd: 5, weekly_limit_usd: null }] }),
      ctx
    )
    expect(quotas.lines[0].label).toBe('openai')
    expect(quotas.lines[0].value).toContain('operator.approval.describe.values.unlimited')
    const reset = describeApproval(req('admin.subscriptions.reset_quota.create', { daily: true, monthly: true }), ctx)
    expect(reset.summary).toContain('operator.approval.describe.values.quotaDaily, operator.approval.describe.values.quotaMonthly')
    const attrs = describeApproval(req('admin.users.attributes.update', { values: { '5': 'ACME', '6': 'x' } }), ctx)
    expect(attrs.lines).toEqual([
      { label: '公司', value: 'ACME' },
      { label: '#6', value: 'x' }
    ])
  })

  it('describes api key group changes and extend/shorten', () => {
    expect(describeApproval(req('admin.api_keys.update', { group_id: 3 }), ctx).summary).toContain('"group":"Pro"')
    expect(describeApproval(req('admin.api_keys.update', { group_id: 0 }), ctx).summary).toContain('apiKeyUnbind')
    expect(describeApproval(req('admin.subscriptions.extend.create', { days: -7 }), ctx).summary).toContain('subscriptionShorten:{"target":"u@example.com","days":7}')
  })

  it('lists unknown fields verbatim so nothing hides behind the summary', () => {
    const d = describeApproval(req('admin.users.balance.create', { balance: 1, operation: 'set', surprise: { a: 1 }, token: '***' }), ctx)
    expect(d.lines).toEqual([
      { label: 'surprise', value: '{"a":1}' },
      { label: 'token', value: 'operator.approval.describe.values.redacted' }
    ])
    const unknown = describeApproval(req('admin.something.new', { k: 'v' }), ctx)
    expect(unknown.summary).toBe('operator.approval.describe.summary.unknown:{"action":"admin.something.new","target":"u@example.com"}')
    expect(unknown.lines).toEqual([{ label: 'k', value: 'v' }])
  })

  it('tolerates an empty or malformed body', () => {
    expect(describeApproval(req('admin.subscriptions.revoke.create', ''), ctx).lines).toEqual([])
    expect(describeApproval(req('admin.subscriptions.revoke.create', 'not json'), ctx).lines).toEqual([])
    expect(describeApproval(req('admin.users.create', '[1,2]'), ctx).lines).toEqual([])
  })
})
