import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeHeader from '../HomeHeader.vue'

const translations: Record<string, string> = {
  'home.docs': 'Docs',
  'home.navModels': 'Models',
  'home.navPricing': 'Pricing',
  'home.navChangelog': 'Changelog',
  'home.switchToLight': 'Switch to light',
  'home.switchToDark': 'Switch to dark',
  'home.dashboard': 'Dashboard',
  'home.loginConsole': 'Login',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] || key,
    }),
  }
})

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRouter: () => ({
      currentRoute: { value: { path: '/home' } },
      push: vi.fn().mockResolvedValue(undefined),
    }),
  }
})

function mountHeader(props: Partial<InstanceType<typeof HomeHeader>['$props']> = {}) {
  return mount(HomeHeader, {
    props: {
      siteName: 'Sub2API',
      docUrl: '',
      isDark: false,
      isAuthenticated: false,
      dashboardPath: '/dashboard',
      userInitial: '',
      ...props,
    },
    global: {
      stubs: {
        Icon: true,
        LocaleSwitcher: true,
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('HomeHeader', () => {
  it('shows the changelog nav link by default', () => {
    const wrapper = mountHeader()

    const changelogLink = wrapper.find('[data-test="nav-changelog"]')
    expect(changelogLink.exists()).toBe(true)
    expect(changelogLink.attributes('href')).toBe('/changelog')
  })

  it('hides the changelog nav link when showChangelog is false', () => {
    const wrapper = mountHeader({ showChangelog: false })

    expect(wrapper.find('[data-test="nav-changelog"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Changelog')
  })
})
