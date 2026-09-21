import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import DingTalkOAuthSection from '@/components/auth/DingTalkOAuthSection.vue'
import OidcOAuthSection from '@/components/auth/OidcOAuthSection.vue'

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, unknown>
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('OAuth login sections', () => {
  beforeEach(() => {
    routeState.query = { redirect: '/billing?plan=pro', aff: 'AFF123' }
    window.sessionStorage.clear()
  })

  it.each([
    // LinuxDo 把邀请码随 start 请求带给后端（新用户直登分支不经过前端，靠后端 cookie 绑定推荐关系）
    ['linuxdo', LinuxDoOAuthSection, { redirect: '/billing?plan=pro', aff_code: 'AFF456' }],
    ['dingtalk', DingTalkOAuthSection, { redirect: '/billing?plan=pro' }],
    ['oidc', OidcOAuthSection, { redirect: '/billing?plan=pro' }]
  ] as const)('emits a %s start request from the original button', async (provider, component, params) => {
    const originalHref = window.location.href
    const wrapper = mount(component, { props: { affCode: 'AFF456' } })

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
      provider,
      params
    })
    expect(window.sessionStorage.getItem('oauth_aff_code')).toBe('AFF456')
    expect(window.location.href).toBe(originalHref)
  })

  it('resolves the LinuxDo referral code from the query when no prop is given', async () => {
    const wrapper = mount(LinuxDoOAuthSection)

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
      provider: 'linuxdo',
      params: { redirect: '/billing?plan=pro', aff_code: 'AFF123' }
    })
  })

  it('omits aff_code from the LinuxDo start request when there is no referral code', async () => {
    routeState.query = { redirect: '/dashboard' }
    window.localStorage.clear()
    const wrapper = mount(LinuxDoOAuthSection)

    await wrapper.get('button').trigger('click')

    expect(wrapper.emitted('start')?.[0]?.[0]).toEqual({
      provider: 'linuxdo',
      params: { redirect: '/dashboard' }
    })
    expect(window.sessionStorage.getItem('oauth_aff_code')).toBeNull()
  })
})
