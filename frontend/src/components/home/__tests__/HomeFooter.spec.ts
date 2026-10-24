import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeFooter from '../HomeFooter.vue'

const state = vi.hoisted(() => ({ contactInfo: '', qrCode: '', groupURL: '' }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    contactInfo: state.contactInfo,
    cachedPublicSettings: { community_qr_code: state.qrCode, community_group_url: state.groupURL }
  })
}))

vi.mock('@/composables/useContentTranslation', () => ({
  useContentTranslation: () => ({ tx: (text: string) => text })
}))

vi.mock('@/components/home/useHomeSectionNav', () => ({
  useHomeSectionNav: () => ({ goToSection: vi.fn() })
}))

function mountFooter() {
  return mount(HomeFooter, {
    props: { siteName: 'CheapRouter', docUrl: '', currentYear: 2026 },
    global: {
      stubs: {
        HomeBrandMark: true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
      }
    }
  })
}

describe('HomeFooter support links (fork)', () => {
  beforeEach(() => {
    state.contactInfo = ''
    state.qrCode = ''
    state.groupURL = ''
  })

  it('always shows the online support entry and the acceptable use policy', () => {
    const wrapper = mountFooter()
    expect(wrapper.find('[data-test="footer-support"]').attributes('href')).toBe('/tickets')
    expect(wrapper.find('[data-test="footer-aup"]').attributes('href')).toBe('/acceptable-use')
    expect(wrapper.find('[data-test="footer-contact"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="support-community"]').exists()).toBe(false)
  })

  it('reuses the community group QR code as a public support contact', () => {
    state.qrCode = 'data:image/png;base64,AAAA'
    state.groupURL = 'https://t.me/example'
    const wrapper = mountFooter()
    const column = wrapper.find('[data-test="footer-contact-column"]')
    expect(column.find('img').attributes('src')).toBe('data:image/png;base64,AAAA')
    expect(column.find('[data-test="support-community-join"]').attributes('href')).toBe('https://t.me/example')
  })

  it('drops a community group link that is not http(s)', () => {
    state.qrCode = 'data:image/png;base64,AAAA'
    state.groupURL = 'javascript:alert(1)'
    const wrapper = mountFooter()
    expect(wrapper.find('[data-test="support-community"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="support-community-join"]').exists()).toBe(false)
  })

  it('shows the configured support email as a mailto link', () => {
    state.contactInfo = ' support@example.com '
    const contact = mountFooter().find('[data-test="support-contact"]')
    expect(contact.attributes('href')).toBe('mailto:support@example.com')
    expect(contact.text()).toBe('support@example.com')
  })

  it('shows free-text contact info as plain text', () => {
    state.contactInfo = 'QQ: 123456789'
    const contact = mountFooter().find('[data-test="support-contact"]')
    expect(contact.element.tagName).toBe('SPAN')
    expect(contact.text()).toBe('QQ: 123456789')
  })
})
