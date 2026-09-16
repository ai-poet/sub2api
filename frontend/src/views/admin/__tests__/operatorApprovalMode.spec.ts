import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

// 运维管理员在用户管理 / 订阅管理页的只读适配（写操作由后端排队等待审批）。
// 页面与弹窗依赖大量子组件与接口，这里按仓库惯例用源码断言锁定关键分支，
// 避免上游合并悄悄去掉某个 readonly 判断。
function readSource(path: string): string {
  return readFileSync(resolve(path), 'utf8')
}

describe('operator approval mode — UsersView', () => {
  const source = readSource('src/views/admin/UsersView.vue')

  it('derives readonly from the auth store and hides destructive actions for operators', () => {
    expect(source).toContain("import { useAuthStore } from '@/stores/auth'")
    expect(source).toContain('const readonly = computed(() => !authStore.isAdmin)')
    expect(source).toContain('v-if="selectedCount > 0 && !readonly"')
    expect(source).toContain(`v-if="user.role !== 'admin' && !readonly"`)
  })

  it('skips dashboard usage loading and hides usage/group columns for operators', () => {
    expect(source).toContain('if (hasVisibleUsageColumn.value && !readonly.value) {')
    expect(source).toContain('OPERATOR_HIDDEN_COLUMN_KEYS')
    expect(source).toContain('].filter((col) => !readonly.value || !OPERATOR_HIDDEN_COLUMN_KEYS.has(col.key)))')
  })

  it('loads groups from the usage filter endpoint for operators', () => {
    expect(source).toContain('adminAPI.usage.listFilterGroups()')
    expect(source).toContain('readonly.value ? await loadOperatorGroups() : await adminAPI.groups.getAll()')
    expect(source).toContain('readonly.value ? await loadOperatorGroups() : await adminAPI.groups.getAllIncludingInactive()')
  })

  it('passes readonly down to the modals and swallows the queued state on toggle', () => {
    expect(source).toContain('<UserCreateModal :show="showCreateModal" :readonly="readonly"')
    expect(source).toContain('<UserEditModal :show="showEditModal" :user="editingUser" :readonly="readonly"')
    expect(source).toContain('<UserApiKeysModal :show="showApiKeysModal" :user="viewingUser" :readonly="readonly"')
    expect(source).toContain('<UserAllowedGroupsModal :show="showAllowedGroupsModal" :user="allowedGroupsUser" :readonly="readonly"')
    expect(source).toContain('if (isApprovalQueued(error)) return')
  })
})

describe('operator approval mode — user modals', () => {
  it('create/edit modals hide the role field and never send a role for operators', () => {
    const create = readSource('src/components/admin/user/UserCreateModal.vue')
    expect(create).toContain('<div v-if="!readonly">')
    expect(create).toContain("if (props.readonly) payload.role = 'user'")
    expect(create).toContain('if (isApprovalQueued(e)) {')

    const edit = readSource('src/components/admin/user/UserEditModal.vue')
    expect(edit).toContain('<div v-if="!readonly">')
    expect(edit).toContain('if (!props.readonly) data.role = form.role')
    expect(edit).toContain('if (isApprovalQueued(e)) {')
  })

  it('every write modal closes on the queued state instead of reporting an error', () => {
    for (const file of [
      'src/components/admin/user/UserBalanceModal.vue',
      'src/components/admin/user/UserPlatformQuotaModal.vue',
      'src/components/admin/user/BulkEditUserModal.vue',
      'src/components/admin/user/UserAllowedGroupsModal.vue',
      'src/components/admin/user/GroupReplaceModal.vue',
      'src/components/admin/user/UserApiKeysModal.vue'
    ]) {
      const source = readSource(file)
      expect(source, file).toContain("import { isApprovalQueued } from '@/utils/approval'")
      expect(source, file).toMatch(/isApprovalQueued\((e|error|attrErr)\)/)
    }
  })

  it('the api keys modal renders the masked key when the backend withholds the secret', () => {
    const source = readSource('src/components/admin/user/UserApiKeysModal.vue')
    expect(source).toContain("return key.key_masked || '••••'")
    expect(source).toContain('{{ keyDisplay(key) }}')
  })
})

describe('operator approval mode — SubscriptionsView', () => {
  const source = readSource('src/views/admin/SubscriptionsView.vue')

  it('loads groups from the usage filter endpoint for operators', () => {
    expect(source).toContain("import { useAuthStore } from '@/stores/auth'")
    expect(source).toContain('const readonly = computed(() => !authStore.isAdmin)')
    expect(source).toContain('(await adminAPI.usage.listFilterGroups()).map(')
  })

  it('all five write handlers close their dialog on the queued state', () => {
    expect(source.match(/if \(isApprovalQueued\(error\)\) \{/g)?.length).toBe(5)
    expect(source).not.toContain('error.response?.data?.detail')
  })
})

describe('operator approval mode — routing and navigation', () => {
  it('marks users, subscriptions and approvals as operator-allowed routes', () => {
    const router = readSource('src/router/index.ts')
    const usersBlock = router.slice(router.indexOf("path: '/admin/users'"), router.indexOf("path: '/admin/users'") + 400)
    expect(usersBlock).toContain('operatorAllowed: true')
    const subsBlock = router.slice(router.indexOf("path: '/admin/subscriptions'"), router.indexOf("path: '/admin/subscriptions'") + 400)
    expect(subsBlock).toContain('operatorAllowed: true')
    expect(router).toContain("path: '/admin/approvals'")
    expect(router).toContain("component: () => import('@/views/admin/ApprovalsView.vue')")
  })

  it('App.vue starts the approvals badge polling for console users and listens for queued writes', () => {
    const app = readSource('src/App.vue')
    expect(app).toContain('approvalsStore.start()')
    expect(app).toContain('approvalsStore.reset()')
    expect(app).toContain('window.addEventListener(APPROVAL_QUEUED_EVENT, onApprovalQueued)')
    expect(app).toContain("t('operator.approval.queuedToast'")
  })
})
