import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { AdminGroup, GroupStatusAdminView } from '@/types'
import GroupRuntimeStatusDialog from '../GroupRuntimeStatusDialog.vue'

const mocks = vi.hoisted(() => ({
  getRuntimeStatus: vi.fn(),
  updateRuntimeStatus: vi.fn(),
  probeRuntimeStatus: vi.fn(),
  probeRuntimeStatusModelTrace: vi.fn(),
  probeRuntimeStatusAstraCheck: vi.fn()
}))
vi.mock('@/api/admin', () => ({ adminAPI: { groups: mocks } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
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

function buildView(platform: 'openai' | 'anthropic'): GroupStatusAdminView {
  const expected = platform === 'openai' ? 'gpt-5.6-sol' : 'claude-opus-5-5'
  const targets =
    platform === 'openai'
      ? [
          { id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol' },
          { id: 'gpt-6-sol', display_name: 'GPT-6 Sol' },
          { id: 'gpt-6-astra', display_name: 'GPT-6 Astra' }
        ]
      : [{ id: 'claude-opus-5-5', display_name: 'Claude Opus 5.5' }]
  return {
    group: { id: 5, name: 'G', platform } as any,
    config: {
      id: 1,
      group_id: 5,
      enabled: true,
      probe_model: expected,
      probe_prompt: 'ping',
      validation_mode: 'non_empty',
      expected_keywords: [],
      interval_seconds: 60,
      timeout_seconds: 30,
      slow_latency_ms: 15000,
      notify_enabled: true,
      modeltrace_enabled: true,
      modeltrace_expected_model: expected,
      modeltrace_request_model: '',
      modeltrace_interval_seconds: 3600,
      astra_check_enabled: false,
      astra_check_request_model: 'gpt-6-astra',
      astra_check_tier: 'low',
      astra_check_interval_seconds: 3600,
      created_at: '',
      updated_at: ''
    },
    summary: {
      group_id: 5,
      config_id: 1,
      enabled: true,
      probe_model: expected,
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
      modeltrace_enabled: true,
      modeltrace_expected_model: expected,
      modeltrace_request_model: '',
      modeltrace_interval_seconds: 3600,
      modeltrace_verdict: 'mismatch',
      modeltrace_stable_status: 'mismatch',
      modeltrace_run_expected_model: expected,
      modeltrace_top_model: 'claude-opus-5',
      modeltrace_top_probability: 0.97,
      modeltrace_expected_probability: 0.01,
      modeltrace_ranking: [
        { model: 'claude-opus-5', display_name: 'claude-opus-5', family: 'claude', family_name: 'Claude', probability: 0.97, conditional_probability: 0.98, score: 2 },
        { model: expected, display_name: expected, family: 'claude', family_name: 'Claude', probability: 0.01, conditional_probability: 0.01, score: 0.1 }
      ],
      modeltrace_reasons: [],
      modeltrace_detail: 'expected Claude Opus 5.5 1.0% · top Claude Opus 5 97.0%',
      modeltrace_checked_at: '2026-09-28T00:00:00Z',
      modeltrace_consecutive_mismatch: 2,
      modeltrace_valid_outputs: 3,
      modeltrace_input_tokens: 750,
      modeltrace_output_tokens: 3300,
      modeltrace_reasoning_tokens: 0,
      modeltrace_last_cost_usd: 0.08,
      modeltrace_running: false,
      modeltrace_bank_sha256: '1c2cb74d372f',
      modeltrace_bank_built_at: '2026-09-23T06:17:08Z',
      astra_check_enabled: false,
      astra_check_request_model: 'gpt-6-astra',
      astra_check_tier: 'low',
      astra_check_interval_seconds: 3600,
      astra_check_verdict: '',
      astra_check_stable_status: '',
      astra_check_winner: '',
      astra_check_matches: [],
      astra_check_reasons: [],
      astra_check_detail: '',
      astra_check_checked_at: null,
      astra_check_consecutive_mismatch: 0,
      astra_check_valid_samples: 0,
      astra_check_planned_samples: 0,
      astra_check_input_tokens: 0,
      astra_check_output_tokens: 0,
      astra_check_reasoning_tokens: 0,
      astra_check_last_cost_usd: 0,
      astra_check_running: false,
      astra_check_benchmark_version: '',
      astra_check_benchmark_models: [],
      astra_check_benchmark_tiers: []
    },
    modeltrace_targets: targets,
    modeltrace_last_run: {
      id: 9,
      group_id: 5,
      config_id: 1,
      platform,
      bank_sha256: '1c2cb74d',
      bank_built_at: '',
      expected_model: expected,
      request_model: expected,
      account_id: 3,
      account_type: 'apikey',
      round: 2,
      verdict: 'mismatch',
      outcome: 'difference_signal',
      top_model: 'claude-opus-5',
      top_probability: 0.97,
      expected_probability: 0.01,
      calibration_queries: 3,
      beta: 12,
      ranking: [],
      family_probabilities: [],
      reasons: [],
      outputs: [
        { seq: 1, challenge_id: 'probe-1', prompt: 'p', expected_count: 300, minimum_numbers: 165, parsed_numbers: 298, accepted: true, thinking_present: false, attempts: 1, latency_ms: 12000, input_tokens: 250, output_tokens: 1100, reasoning_tokens: 0, excerpt: '12, 7, 88', top_model: 'claude-opus-5', top_probability: 0.9 },
        { seq: 2, challenge_id: 'probe-2', prompt: 'p', expected_count: 310, minimum_numbers: 171, parsed_numbers: 40, accepted: false, rejection: 'max_tokens', stop_reason: 'max_tokens', thinking_present: false, attempts: 1, latency_ms: 30000, input_tokens: 250, output_tokens: 4096, reasoning_tokens: 0 }
      ],
      attempts_planned: 6,
      attempts_made: 4,
      valid_outputs: 3,
      input_tokens: 750,
      output_tokens: 3300,
      reasoning_tokens: 0,
      cost_usd: 0.08,
      latency_ms: 60000,
      http_code: 200,
      error_detail: '',
      started_at: '',
      finished_at: '',
      created_at: ''
    }
  } as unknown as GroupStatusAdminView
}

async function mountDialog(platform: 'openai' | 'anthropic') {
  mocks.getRuntimeStatus.mockResolvedValue(buildView(platform))
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

describe('GroupRuntimeStatusDialog ModelTrace card', () => {
  it('renders the Claude target, stable mismatch verdict, ranking and run details', async () => {
    const wrapper = await mountDialog('anthropic')
    const text = wrapper.text()

    expect(text).toContain('admin.groups.runtimeStatus.modelTrace.title')
    // Astra 只对 OpenAI 分组显示
    expect(text).not.toContain('admin.groups.runtimeStatus.astraCheck.title')

    const options = wrapper.findAll('select option').map((o) => o.text())
    expect(options).toContain('Claude Opus 5.5')
    expect(options).not.toContain('GPT-5.6 Sol')

    expect(text).toContain('admin.groups.runtimeStatus.modelTrace.statuses.mismatch')
    expect(text).toContain('"top":"Claude Opus 5"')
    expect(text).toContain('Claude Opus 5')
    expect(text).toContain('97.0%')
    expect(text).toContain('1c2cb74d372f')
    expect(text).toContain('max_tokens')
    expect(text).toContain('admin.groups.runtimeStatus.modelTrace.disclaimer')
  })

  it('offers the three GPT targets on OpenAI groups', async () => {
    const wrapper = await mountDialog('openai')
    const options = wrapper.findAll('select option').map((o) => o.text())
    expect(options).toEqual(expect.arrayContaining(['GPT-5.6 Sol', 'GPT-6 Sol', 'GPT-6 Astra']))
    expect(wrapper.text()).toContain('admin.groups.runtimeStatus.astraCheck.title')
  })

  it('saves the ModelTrace settings before starting an async check', async () => {
    const view = buildView('openai')
    const wrapper = await mountDialog('openai')
    mocks.updateRuntimeStatus.mockResolvedValue(view)
    mocks.probeRuntimeStatusModelTrace.mockResolvedValue({
      ...view,
      summary: { ...view.summary, modeltrace_running: false }
    })

    const select = wrapper.findAll('select').find((s) => s.findAll('option').some((o) => o.text() === 'GPT-6 Astra'))!
    await select.setValue('gpt-6-astra')
    const probe = wrapper.findAll('button').find((b) => b.text() === 'admin.groups.runtimeStatus.modelTrace.probeNow')!
    await probe.trigger('click')
    await flushPromises()

    expect(mocks.updateRuntimeStatus).toHaveBeenCalledWith(
      5,
      expect.objectContaining({
        modeltrace_enabled: true,
        modeltrace_expected_model: 'gpt-6-astra',
        modeltrace_request_model: '',
        modeltrace_interval_seconds: 3600
      })
    )
    expect(mocks.probeRuntimeStatusModelTrace).toHaveBeenCalledWith(5)
  })
})
