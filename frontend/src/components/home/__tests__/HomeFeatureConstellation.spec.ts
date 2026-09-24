import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeFeatureConstellation from '../HomeFeatureConstellation.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

function mountFeatures(mode: 'client' | 'api') {
  return mount(HomeFeatureConstellation, {
    props: { mode, siteName: 'CheapRouter' },
    global: { stubs: { Icon: true, HomeBrandMark: true } },
  })
}

function cardKeys(column: ReturnType<ReturnType<typeof mountFeatures>['find']>) {
  return column
    .findAll('h3')
    .map((title) => title.text().replace('home.landing.features.cards.', '').replace('.title', ''))
}

describe('HomeFeatureConstellation', () => {
  it('puts relay quality on the left and the client on the right when a client exists', () => {
    const wrapper = mountFeatures('client')

    const left = wrapper.find('[data-test="features-left"]')
    const right = wrapper.find('[data-test="features-right"]')
    expect(left.text()).toContain('home.landing.features.columns.quality')
    expect(right.text()).toContain('home.landing.features.columns.client')
    expect(cardKeys(left)).toEqual(['quality', 'probe', 'failover'])
    expect(cardKeys(right)).toEqual(['builtinAgent', 'allAgents', 'images'])
  })

  it('keeps relay quality on the left and access on the right without a client', () => {
    const wrapper = mountFeatures('api')

    const left = wrapper.find('[data-test="features-left"]')
    const right = wrapper.find('[data-test="features-right"]')
    expect(wrapper.text()).not.toContain('home.landing.features.columns.')
    expect(cardKeys(left)).toEqual(['quality', 'probe', 'failover'])
    expect(cardKeys(right)).toEqual(['compatible', 'metered', 'invoice'])
  })
})
