import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AcceptableUsePolicyView from '../AcceptableUsePolicyView.vue'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

// 测试环境用 vue-i18n 运行时版（不编译字符串消息），这里直接从真实语言包取文案并替换 {siteName}，
// 断言的就是线上会显示的条款内容。
const state = vi.hoisted(() => ({ messages: {} as Record<string, unknown> }))

function lookup(key: string, params?: Record<string, unknown>): string {
  let current: unknown = state.messages
  for (const segment of key.split('.')) {
    current = (current as Record<string, unknown> | undefined)?.[segment]
  }
  if (typeof current !== 'string') return key
  return current.replace(/\{(\w+)\}/g, (match, name: string) => String(params?.[name] ?? match))
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: lookup }) }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: { site_name: 'CheapRouter' }, siteName: 'CheapRouter', contactInfo: '' })
}))

function mountView(locale: 'zh' | 'en') {
  state.messages = locale === 'zh' ? zh : en
  return mount(AcceptableUsePolicyView, {
    global: {
      stubs: { RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } }
    }
  })
}

describe('AcceptableUsePolicyView (fork)', () => {
  it('lists the six prohibited categories required by payment review', () => {
    const text = mountView('zh').find('[data-test="aup-prohibited"]').text()
    for (const label of ['色情 / NSFW', '暴力 / 血腥', '仇恨言论', '儿童不安全内容（CSAM）', '深度伪造 / 冒充他人', '侵犯版权 / 商标']) {
      expect(text).toContain(label)
    }
  })

  it('describes moderation, reporting and enforcement, and links to the report ticket form', () => {
    const wrapper = mountView('en')
    expect(wrapper.find('[data-test="aup-moderation"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="aup-enforcement"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="aup-link-reporting"]').attributes('href')).toBe('/tickets?new=report')
    expect(wrapper.find('[data-test="legal-contact-tickets"]').attributes('href')).toBe('/tickets')
    expect(wrapper.text()).toContain('CheapRouter')
    expect(wrapper.text()).not.toContain('{siteName}')
  })
})
