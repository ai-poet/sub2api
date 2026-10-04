import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeComparisonSection from '../HomeComparisonSection.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

function mountSection(mode?: 'client' | 'api') {
  return mount(HomeComparisonSection, { props: { siteName: 'CheapRouter', ...(mode ? { mode } : {}) } })
}

describe('HomeComparisonSection', () => {
  it('describes the client set-up and agent teams when a client exists', () => {
    const text = mountSection('client').text()

    expect(text).toContain('home.comparison.items.pricing.usClient')
    expect(text).toContain('home.comparison.items.stability.usClient')
    expect(text).toContain('home.comparison.items.models.us')
  })

  it('keeps to API access without a client', () => {
    for (const text of [mountSection('api').text(), mountSection().text()]) {
      expect(text).toContain('home.comparison.items.pricing.us')
      expect(text).toContain('home.comparison.items.stability.us')
      expect(text).not.toContain('usClient')
    }
  })
})
