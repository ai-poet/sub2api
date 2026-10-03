import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import FingerprintBenchmarkNotes from '../FingerprintBenchmarkNotes.vue'
import zhFork from '@/i18n/locales/zh/fork'
import enFork from '@/i18n/locales/en/fork'
import { FINGERPRINT_BENCHMARKS, MEOW_REPO_URL, MODELTRACE_REPO_URL, astraModelLabel } from '@/utils/groupStatus'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

describe('FingerprintBenchmarkNotes', () => {
  it('lists every family with its methods, models and source links', () => {
    const wrapper = mount(FingerprintBenchmarkNotes)

    const gpt = wrapper.find('[data-benchmark-family="gpt"]')
    expect(gpt.findAll('[data-benchmark-source]').map((li) => li.attributes('data-benchmark-source'))).toEqual(['gptMeow', 'gptModelTrace', 'gptJuice'])
    expect(gpt.find('[data-benchmark-source="gptMeow"]').text()).toContain('GPT-6 Sol / GPT-6 Astra')
    expect(gpt.find('[data-benchmark-source="gptModelTrace"]').text()).toContain('GPT-6.1 Sol')
    const claude = wrapper.find('[data-benchmark-family="claude"]')
    expect(claude.find('[data-benchmark-source="claudeModelTrace"]').text()).toContain('Claude Opus 5.5 / Claude Opus 5')
    expect(claude.find('[data-benchmark-source="claudeMeow"]').text()).toContain('Claude Fable 5.1')

    const meowLink = gpt.find('[data-benchmark-source="gptMeow"] [data-benchmark-repo]')
    expect(meowLink.attributes('href')).toBe(MEOW_REPO_URL)
    expect(meowLink.attributes('target')).toBe('_blank')
    expect(meowLink.attributes('rel')).toBe('noopener noreferrer')
    // 地址本身可读
    expect(meowLink.text()).toContain('chen-006/meow-llm-detector')
    expect(claude.find('[data-benchmark-source="claudeModelTrace"] [data-benchmark-repo]').attributes('href')).toBe(MODELTRACE_REPO_URL)

    // Juice 没有独立仓库，标为本站实现
    const juice = gpt.find('[data-benchmark-source="gptJuice"]')
    expect(juice.find('[data-benchmark-repo]').exists()).toBe(false)
    expect(juice.text()).toContain('modelStatus.benchmarks.selfImplemented')
    expect(wrapper.text()).toContain('modelStatus.benchmarks.disclaimer')
  })

  it('has readable model names and translations for every source', () => {
    for (const family of FINGERPRINT_BENCHMARKS) {
      expect(zhFork.modelStatus.benchmarks.families).toHaveProperty(family.family)
      expect(enFork.modelStatus.benchmarks.families).toHaveProperty(family.family)
      for (const source of family.sources) {
        expect(zhFork.modelStatus.benchmarks.sources).toHaveProperty(source.key)
        expect(enFork.modelStatus.benchmarks.sources).toHaveProperty(source.key)
        expect(zhFork.modelStatus.benchmarks.methods).toHaveProperty(source.method)
        expect(enFork.modelStatus.benchmarks.methods).toHaveProperty(source.method)
        for (const model of source.models) {
          expect(astraModelLabel(model)).not.toBe(model)
        }
        if (source.method !== 'sol_juice') {
          expect(source.repo).toMatch(/^https:\/\/github\.com\//)
        }
      }
    }
  })
})
