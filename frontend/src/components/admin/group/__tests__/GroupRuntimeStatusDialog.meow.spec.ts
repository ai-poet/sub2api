import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { AdminGroup, AstraCheckState, GroupStatusAdminView } from '@/types'
import GroupRuntimeStatusDialog from '../GroupRuntimeStatusDialog.vue'

const mocks = vi.hoisted(() => ({
  getRuntimeStatus: vi.fn(),
  updateRuntimeStatus: vi.fn(),
  probeRuntimeStatus: vi.fn(),
  probeRuntimeStatusAstraCheck: vi.fn()
}))
const toasts = vi.hoisted(() => ({ showSuccess: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { groups: mocks } }))
vi.mock('@/stores', () => ({ useAppStore: () => toasts }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key} ${JSON.stringify(params)}` : key),
      te: () => false
    })
  }
})
enableAutoUnmount(afterEach)

const gptTargets = [
  { id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol', platform: 'openai', default_request_model: 'gpt-5.6-sol', package_id: 'meow-gpt-other-cap98' },
  { id: 'gpt-6-sol', display_name: 'GPT-6 Sol', platform: 'openai', default_request_model: 'gpt-6-sol', package_id: 'meow-gpt-other-cap98-efficient' },
  { id: 'gpt-6-astra', display_name: 'GPT-6 Astra', platform: 'openai', default_request_model: 'gpt-6-astra', package_id: 'meow-gpt-other-cap98-efficient' }
]
const claudeTargets = [
  { id: 'claude-opus-5.5', display_name: 'Claude Opus 5.5', platform: 'anthropic', default_request_model: 'claude-opus-5-5', package_id: 'meow-claude-other-cap98-efficient' },
  { id: 'claude-fable-5.1', display_name: 'Claude Fable 5.1', platform: 'anthropic', default_request_model: 'claude-fable-5-1', package_id: 'meow-claude-other-cap98-efficient' }
]

function state(expected: string, display: string, extra: Partial<AstraCheckState> = {}): AstraCheckState {
  return {
    id: 0,
    group_id: 5,
    config_id: 1,
    expected_model: expected,
    display_name: display,
    verdict: '',
    stable_status: '',
    winner: '',
    matches: [],
    reasons: [],
    detail: '',
    checked_at: null,
    consecutive_mismatch: 0,
    valid_samples: 0,
    planned_samples: 0,
    input_tokens: 0,
    output_tokens: 0,
    reasoning_tokens: 0,
    last_cost_usd: 0,
    last_run_id: null,
    benchmark_package_id: '',
    benchmark_version: '',
    ...extra
  }
}

function buildView(platform: 'openai' | 'anthropic'): GroupStatusAdminView {
  const openai = platform === 'openai'
  const models = openai
    ? [{ expected_model: 'gpt-6-sol', request_model: '' }, { expected_model: 'gpt-6-astra', request_model: 'astra-alias' }]
    : [{ expected_model: 'claude-opus-5.5', request_model: '' }]
  const states = openai
    ? [
        state('gpt-6-sol', 'GPT-6 Sol', {
          verdict: 'mismatch',
          stable_status: 'mismatch',
          winner: 'gpt-6-astra',
          checked_at: '2026-09-28T00:00:00Z',
          consecutive_mismatch: 2,
          valid_samples: 32,
          planned_samples: 32,
          last_cost_usd: 0.02,
          benchmark_version: '4.5.4-predictive.20260924.2',
          detail: 'expected GPT-6 Sol · strongest GPT-6 Astra 97.1%/56.7%',
          matches: [
            { model: 'gpt-6-astra', name: 'GPT-6 Astra', score: -20, match: 0.971, threshold: 0.567, passed: true },
            { model: 'gpt-6-sol', name: 'GPT-6 Sol', score: -40, match: 0.02, threshold: 0.5, passed: false },
            { model: 'other_known_external', name: '其他模型', score: -60, match: 0.01, threshold: 0.56, passed: false }
          ],
          reasons: []
        }),
        state('gpt-6-astra', 'GPT-6 Astra')
      ]
    : [state('claude-opus-5.5', 'Claude Opus 5.5', { verdict: 'match', stable_status: 'pass', winner: 'claude-opus-5.5', checked_at: '2026-09-28T00:00:00Z' })]
  return {
    group: { id: 5, name: 'G', platform } as any,
    config: {
      id: 1,
      group_id: 5,
      enabled: true,
      probe_model: 'probe',
      probe_prompt: 'ping',
      validation_mode: 'non_empty',
      expected_keywords: [],
      interval_seconds: 60,
      timeout_seconds: 30,
      slow_latency_ms: 15000,
      notify_enabled: true,
      astra_check_enabled: true,
      astra_check_models: models,
      astra_check_tier: 'low',
      astra_check_interval_seconds: 3600,
      created_at: '',
      updated_at: ''
    },
    summary: {
      group_id: 5,
      config_id: 1,
      enabled: true,
      probe_model: 'probe',
      latest_status: 'up',
      stable_status: 'up',
      response_excerpt: '',
      latency_ms: 100,
      http_code: 200,
      sub_status: '',
      error_detail: '',
      observed_at: '2026-09-28T00:00:00Z',
      consecutive_down: 0,
      consecutive_non_down: 3,
      astra_check_enabled: true,
      astra_check_models: models,
      astra_check_tier: 'low',
      astra_check_interval_seconds: 3600,
      astra_check_states: states,
      astra_check_running: false,
      astra_check_benchmarks: [
        {
          package_id: 'meow-gpt-other-cap98-efficient',
          version: '4.5.4-predictive.20260924.2',
          mode: 'gpt',
          content_sha256: '',
          body_sha256: '',
          models: [{ id: 'gpt-6-astra', name: 'GPT-6 Astra', request_model: '' }],
          tiers: [{ tier: 'low', requests: 32, calibrated: true }]
        },
        {
          package_id: 'meow-claude-other-cap98-efficient',
          version: '4.5.4-predictive.20260924.1',
          mode: 'claude',
          content_sha256: '',
          body_sha256: '',
          models: [{ id: 'claude-opus-5.5', name: 'Claude Opus 5.5', request_model: '' }],
          tiers: [{ tier: 'low', requests: 48, calibrated: true }]
        }
      ]
    },
    astra_check_targets: openai ? gptTargets : claudeTargets,
    astra_check_last_runs: openai
      ? [
          {
            id: 41,
            group_id: 5,
            config_id: 1,
            platform: 'openai',
            expected_model: 'gpt-6-sol',
            round: 2,
            benchmark_package_id: 'meow-gpt-other-cap98-efficient',
            benchmark_version: '4.5.4-predictive.20260924.2',
            benchmark_sha256: '',
            scoring_version: 'meow-fingerprint-v3-predictive',
            request_model: 'gpt-6-sol',
            tier: 'low',
            account_id: 3,
            verdict: 'mismatch',
            winner_model: 'gpt-6-astra',
            matches: [],
            cells: [{ cell_id: 'gpt__screen023', planned: 2, total: 2, valid: 2, invalid: 0, minimum: 2, categories: { '43': 2 } }],
            reasons: [],
            samples: [
              { seq: 1, cell_id: 'gpt__screen023', attempt: 1, answer: '43', category: '43', outcome: 'valid', final: true, http_code: 200, latency_ms: 900, at: '' }
            ],
            requests_planned: 32,
            requests_completed: 32,
            valid_samples: 32,
            input_tokens: 0,
            output_tokens: 0,
            reasoning_tokens: 0,
            cost_usd: 0.02,
            latency_ms: 30000,
            http_code: null,
            error_detail: '',
            started_at: '',
            finished_at: '',
            created_at: ''
          }
        ]
      : []
  } as GroupStatusAdminView
}

async function mountDialog(platform: 'openai' | 'anthropic' | 'gemini') {
  if (platform !== 'gemini') {
    mocks.getRuntimeStatus.mockResolvedValue(buildView(platform))
  } else {
    const view = buildView('openai')
    view.astra_check_targets = []
    mocks.getRuntimeStatus.mockResolvedValue(view)
  }
  const wrapper = mount(GroupRuntimeStatusDialog, {
    props: { show: true, group: { id: 5, name: 'G', platform } as AdminGroup },
    global: {
      stubs: {
        BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
        Toggle: { props: ['modelValue'], template: '<input type="checkbox" :checked="modelValue" />' }
      }
    }
  })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('GroupRuntimeStatusDialog meow fingerprint card', () => {
  it('lists the platform targets with the configured selection and request models', async () => {
    const wrapper = await mountDialog('openai')
    const checked = (id: string) => (wrapper.find(`[data-astra-model="${id}"]`).element as HTMLInputElement).checked
    expect(checked('gpt-5.6-sol')).toBe(false)
    expect(checked('gpt-6-sol')).toBe(true)
    expect(checked('gpt-6-astra')).toBe(true)
    expect((wrapper.find('[data-astra-request-model="gpt-6-astra"]').element as HTMLInputElement).value).toBe('astra-alias')
    expect(wrapper.find('[data-astra-request-model="gpt-5.6-sol"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-astra-model="claude-opus-5.5"]').exists()).toBe(false)

    // 只列本平台用到的基准包
    const text = wrapper.text()
    expect(text).toContain('meow-gpt-other-cap98-efficient')
    expect(text).not.toContain('meow-claude-other-cap98-efficient')
  })

  it('renders one result block per configured model, including the not-yet-checked one', async () => {
    const wrapper = await mountDialog('openai')
    const sol = wrapper.find('[data-astra-state="gpt-6-sol"]')
    const astra = wrapper.find('[data-astra-state="gpt-6-astra"]')
    expect(sol.exists()).toBe(true)
    expect(astra.exists()).toBe(true)

    const solText = sol.text()
    expect(solText).toContain('admin.groups.runtimeStatus.astraCheck.statuses.mismatch')
    expect(solText).toContain('"winner":"GPT-6 Astra"')
    expect(solText).toContain('97.1%')
    expect(solText).toContain('admin.groups.runtimeStatus.astraCheck.otherModel')
    expect(solText).toContain('32 / 32')
    // 最近一次运行的样本与题目计数
    expect(solText).toContain('gpt__screen023')
    expect(solText).toContain('43×2')

    expect(astra.text()).toContain('admin.groups.runtimeStatus.astraCheck.notChecked')
  })

  it('saves the selected models before checking a single model', async () => {
    const view = buildView('openai')
    const wrapper = await mountDialog('openai')
    mocks.updateRuntimeStatus.mockResolvedValue(view)
    mocks.probeRuntimeStatusAstraCheck.mockResolvedValue(view)

    await wrapper.find('[data-astra-model="gpt-5.6-sol"]').setValue(true)
    await wrapper.find('[data-astra-request-model="gpt-5.6-sol"]').setValue('sol-old')
    await wrapper.find('[data-astra-probe-model="gpt-6-astra"]').trigger('click')
    await flushPromises()

    expect(mocks.updateRuntimeStatus).toHaveBeenCalledWith(
      5,
      expect.objectContaining({
        astra_check_enabled: true,
        astra_check_models: [
          { expected_model: 'gpt-5.6-sol', request_model: 'sol-old' },
          { expected_model: 'gpt-6-sol', request_model: '' },
          { expected_model: 'gpt-6-astra', request_model: 'astra-alias' }
        ],
        astra_check_tier: 'low',
        astra_check_interval_seconds: 3600
      })
    )
    expect(mocks.probeRuntimeStatusAstraCheck).toHaveBeenCalledWith(5, 'gpt-6-astra')
  })

  it('checks every selected model with the "check all" button', async () => {
    const view = buildView('anthropic')
    const wrapper = await mountDialog('anthropic')
    mocks.updateRuntimeStatus.mockResolvedValue(view)
    mocks.probeRuntimeStatusAstraCheck.mockResolvedValue(view)

    expect(wrapper.find('[data-astra-model="claude-fable-5.1"]').exists()).toBe(true)
    await wrapper.find('[data-astra-probe-all]').trigger('click')
    await flushPromises()

    expect(mocks.updateRuntimeStatus).toHaveBeenCalledWith(
      5,
      expect.objectContaining({ astra_check_models: [{ expected_model: 'claude-opus-5.5', request_model: '' }] })
    )
    expect(mocks.probeRuntimeStatusAstraCheck).toHaveBeenCalledWith(5, undefined)
  })

  it('refuses to save an enabled check without any model selected', async () => {
    const wrapper = await mountDialog('anthropic')
    await wrapper.find('[data-astra-model="claude-opus-5.5"]').setValue(false)
    expect(wrapper.find('[data-astra-probe-all]').attributes('disabled')).toBeDefined()

    const save = wrapper.findAll('button').find((b) => b.text() === 'admin.groups.runtimeStatus.save')!
    await save.trigger('click')
    await flushPromises()

    expect(mocks.updateRuntimeStatus).not.toHaveBeenCalled()
    expect(toasts.showError).toHaveBeenCalledWith('admin.groups.runtimeStatus.astraCheck.noModelSelected')
  })

  it('hides the card and leaves the fingerprint settings out of the payload on other platforms', async () => {
    const wrapper = await mountDialog('gemini')
    expect(wrapper.text()).not.toContain('admin.groups.runtimeStatus.astraCheck.title')

    mocks.updateRuntimeStatus.mockResolvedValue(buildView('openai'))
    const save = wrapper.findAll('button').find((b) => b.text() === 'admin.groups.runtimeStatus.save')!
    await save.trigger('click')
    await flushPromises()

    const payload = mocks.updateRuntimeStatus.mock.calls[0][1]
    expect(payload).not.toHaveProperty('astra_check_enabled')
    expect(payload).not.toHaveProperty('astra_check_models')
  })
})
