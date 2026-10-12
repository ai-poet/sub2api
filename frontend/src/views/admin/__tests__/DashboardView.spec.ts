import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import DashboardView from '../DashboardView.vue'

// fork：管理员仪表盘整页在 components/admin/dashboard/fork/AdminDashboardPage.vue（行为见它的 spec），
// 本视图只负责套 AppLayout。
describe('admin DashboardView', () => {
  it('wraps the fork dashboard page in AppLayout', () => {
    const wrapper = mount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div data-testid="layout"><slot /></div>' },
          AdminDashboardPage: { template: '<div data-testid="admin-dashboard-page" />' },
        },
      },
    })
    expect(wrapper.find('[data-testid="layout"] [data-testid="admin-dashboard-page"]').exists()).toBe(true)
  })
})
