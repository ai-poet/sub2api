import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpsDashboard from '../OpsDashboard.vue'
import OpsDashboardHeader from '../components/OpsDashboardHeader.vue'
import OpsRequestDetailsModal from '../components/OpsRequestDetailsModal.vue'
import OpsErrorDetailsModal from '../components/OpsErrorDetailsModal.vue'
import OpsErrorDetailModal from '../components/OpsErrorDetailModal.vue'
import OpsSettingsDialog from '../components/OpsSettingsDialog.vue'
import OpsAlertRulesCard from '../components/OpsAlertRulesCard.vue'

const state = vi.hoisted(() => ({
  isAdmin: true,
  routerReplace: vi.fn()
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    get isAdmin() {
      return state.isAdmin
    },
    get isOperator() {
      return !state.isAdmin
    }
  }),
  useAppStore: () => ({ showError: vi.fn(), showWarning: vi.fn(), showSuccess: vi.fn() }),
  useAdminSettingsStore: () => ({
    opsMonitoringEnabled: true,
    opsQueryModeDefault: 'auto',
    fetch: vi.fn().mockResolvedValue(undefined)
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace: state.routerReplace, push: vi.fn() })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/api/admin/ops', () => {
  const opsAPI = {
    getAdvancedSettings: vi.fn().mockResolvedValue({
      display_alert_events: false,
      display_openai_token_stats: false,
      auto_refresh_enabled: false,
      auto_refresh_interval_seconds: 30
    }),
    getMetricThresholds: vi.fn().mockResolvedValue(null),
    getDashboardSnapshotV2: vi.fn().mockResolvedValue({ overview: null, throughput_trend: null, error_trend: null }),
    getThroughputTrend: vi.fn().mockResolvedValue({ points: [] }),
    getLatencyHistogram: vi.fn().mockResolvedValue(null),
    getErrorDistribution: vi.fn().mockResolvedValue(null),
    getDashboardOverview: vi.fn().mockResolvedValue(null),
    getErrorTrend: vi.fn().mockResolvedValue(null)
  }
  // `@/api/admin/index.ts` 以默认导出方式引入 opsAPI，两种导出都要给。
  return { opsAPI, default: opsAPI }
})

async function mountDashboard() {
  const wrapper = shallowMount(OpsDashboard, {
    global: { renderStubDefaultSlot: true }
  })
  await flushPromises()
  return wrapper
}

// 运维管理员（operator）只读模式：设置 / 告警规则弹窗不挂载，但钻取弹窗必须挂载且能打开，
// 否则点「明细」既无弹窗也不发请求（后端白名单本来就放行了这些只读接口并做了 operator 投影）。
describe('OpsDashboard read-only (operator) mode', () => {
  beforeEach(() => {
    state.routerReplace.mockReset()
  })

  it('keeps drill-down modals mounted for operators and opens them on demand', async () => {
    state.isAdmin = false
    const wrapper = await mountDashboard()

    expect(wrapper.findComponent(OpsSettingsDialog).exists()).toBe(false)
    expect(wrapper.findComponent(OpsAlertRulesCard).exists()).toBe(false)

    expect(wrapper.findComponent(OpsRequestDetailsModal).exists()).toBe(true)
    expect(wrapper.findComponent(OpsErrorDetailsModal).exists()).toBe(true)
    expect(wrapper.findComponent(OpsErrorDetailModal).exists()).toBe(true)
    expect(wrapper.findComponent(OpsRequestDetailsModal).props('modelValue')).toBe(false)

    const header = wrapper.findComponent(OpsDashboardHeader)
    expect(header.exists()).toBe(true)

    header.vm.$emit('openRequestDetails')
    await flushPromises()
    expect(wrapper.findComponent(OpsRequestDetailsModal).props('modelValue')).toBe(true)

    header.vm.$emit('openErrorDetails', 'upstream')
    await flushPromises()
    expect(wrapper.findComponent(OpsRequestDetailsModal).props('modelValue')).toBe(false)
    const errorList = wrapper.findComponent(OpsErrorDetailsModal)
    expect(errorList.props('show')).toBe(true)
    expect(errorList.props('errorType')).toBe('upstream')

    errorList.vm.$emit('openErrorDetail', 42)
    await flushPromises()
    const detail = wrapper.findComponent(OpsErrorDetailModal)
    expect(detail.props('show')).toBe(true)
    expect(detail.props('errorId')).toBe(42)
    expect(detail.props('errorType')).toBe('upstream')
    expect(wrapper.findComponent(OpsErrorDetailsModal).props('show')).toBe(false)
  })

  it('still mounts settings and alert rules dialogs for admins', async () => {
    state.isAdmin = true
    const wrapper = await mountDashboard()

    expect(wrapper.findComponent(OpsSettingsDialog).exists()).toBe(true)
    expect(wrapper.findComponent(OpsAlertRulesCard).exists()).toBe(true)
    expect(wrapper.findComponent(OpsRequestDetailsModal).exists()).toBe(true)
  })
})
