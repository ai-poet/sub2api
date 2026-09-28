import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import PaseoBridgeView from '@/views/auth/PaseoBridgeView.vue'

// The `/auth/paseo` path is kept for the desktop client that still opens it,
// but nothing on the page may name Paseo any more.

const {
  localeState,
  routeState,
  locationState,
  routerReplaceMock,
  getAuthTokenMock,
  createDesktopSessionMock,
  createDesktopLoginCodeMock,
  copyToClipboardMock,
  keysListMock,
  keysCreateMock,
  getAvailableGroupsMock,
} = vi.hoisted(() => ({
  localeState: { current: 'en' as 'en' | 'zh' },
  routeState: {
    fullPath: '/auth/paseo?endpoint=https%3A%2F%2Fapi.example.com',
    query: { endpoint: 'https://api.example.com' } as Record<string, unknown>,
  },
  locationState: { current: { href: 'http://localhost/auth/paseo' } },
  routerReplaceMock: vi.fn(),
  getAuthTokenMock: vi.fn(),
  createDesktopSessionMock: vi.fn(),
  createDesktopLoginCodeMock: vi.fn(),
  copyToClipboardMock: vi.fn(),
  keysListMock: vi.fn(),
  keysCreateMock: vi.fn(),
  getAvailableGroupsMock: vi.fn(),
}))

const messages: Record<'en' | 'zh', unknown> = { en, zh }

function lookup(key: string, params?: Record<string, unknown>): string {
  const value = key
    .split('.')
    .reduce<unknown>(
      (node, part) => (node && typeof node === 'object' ? (node as Record<string, unknown>)[part] : undefined),
      messages[localeState.current]
    )
  if (typeof value !== 'string') {
    return key
  }
  return value.replace(/\{(\w+)\}/g, (placeholder, name: string) =>
    params && name in params ? String(params[name]) : placeholder
  )
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => lookup(key, params),
  }),
}))

vi.mock('@/composables/useClipboard', async () => {
  const { ref } = await import('vue')
  return {
    useClipboard: () => {
      const copied = ref(false)
      return {
        copied,
        copyToClipboard: async (...args: unknown[]) => {
          copyToClipboardMock(...args)
          copied.value = true
          return true
        },
      }
    },
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ replace: (...args: unknown[]) => routerReplaceMock(...args) }),
}))

vi.mock('@/components/layout', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    AuthLayout: defineComponent({
      setup: (_, { slots }) => () => h('div', slots.default?.()),
    }),
  }
})

vi.mock('@/api', () => ({
  keysAPI: {
    list: (...args: unknown[]) => keysListMock(...args),
    create: (...args: unknown[]) => keysCreateMock(...args),
  },
  userGroupsAPI: {
    getAvailable: (...args: unknown[]) => getAvailableGroupsMock(...args),
  },
}))

vi.mock('@/api/auth', () => ({
  authAPI: {
    createDesktopSession: (...args: unknown[]) => createDesktopSessionMock(...args),
    createDesktopLoginCode: (...args: unknown[]) => createDesktopLoginCodeMock(...args),
  },
  getAuthToken: () => getAuthTokenMock(),
}))

vi.mock('@/utils/auth-redirect', () => ({
  rememberOAuthReturnPath: vi.fn(),
  clearStoredOAuthReturnPath: vi.fn(),
}))

function group(id: number, platform: string) {
  return { id, platform, status: 'active' }
}

// A valid S256 challenge (RFC 7636 appendix B) and the desktop's listener.
const CHALLENGE = 'E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM'
const LOOPBACK = 'http://127.0.0.1:51789/callback'
const NOW = 1_710_000_000_000
const STORAGE_KEY = `sub2api_desktop_login:${CHALLENGE}`

describe.each(['en', 'zh'] as const)('PaseoBridgeView (%s)', (locale) => {
  const wrappers: VueWrapper[] = []

  afterEach(() => {
    wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
    vi.useRealTimers()
  })

  beforeEach(() => {
    localeState.current = locale
    routeState.query = { endpoint: 'https://api.example.com' }
    window.sessionStorage.clear()
    createDesktopLoginCodeMock.mockReset().mockResolvedValue({ code: 'K7QM-3XPD', expires_in: 600 })
    copyToClipboardMock.mockReset()
    locationState.current = { href: 'http://localhost/auth/paseo' }
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState.current,
    })
    routerReplaceMock.mockReset()
    getAuthTokenMock.mockReset().mockReturnValue('browser-access-token')
    createDesktopSessionMock.mockReset().mockResolvedValue({
      access_token: 'desktop-at',
      refresh_token: 'desktop-rt',
      expires_in: 900,
    })
    keysListMock.mockReset().mockResolvedValue({ items: [], pages: 1 })
    getAvailableGroupsMock.mockReset().mockResolvedValue([group(1, 'anthropic'), group(2, 'openai')])
    keysCreateMock.mockReset().mockImplementation(async (name: string, groupId?: number) => ({
      key: `sk-${groupId ?? 'default'}`,
      name,
      status: 'active',
      group: groupId ? group(groupId, groupId === 1 ? 'anthropic' : 'openai') : undefined,
    }))
  })

  it('sends a signed-out visitor to login and back to the same bridge link', async () => {
    getAuthTokenMock.mockReturnValue(null)

    mount(PaseoBridgeView)
    await flushPromises()

    expect(routerReplaceMock).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: routeState.fullPath },
    })
  })

  it('hands the session to the app without naming Paseo anywhere the user can see', async () => {
    const wrapper = mount(PaseoBridgeView)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain(lookup('auth.desktopBridge.title'))
    expect(text).toContain(lookup('auth.desktopBridge.opening'))
    expect(text).toContain(lookup('auth.desktopBridge.openApp'))
    expect(text).not.toMatch(/paseo/i)
    expect(text).not.toContain('auth.desktopBridge.')

    const keyNames = keysCreateMock.mock.calls.map(([name]) => name as string)
    expect(keyNames).toEqual(['Desktop App (Claude Code)', 'Desktop App (Codex)'])
    expect(locationState.current.href).toMatch(/^agentdesk:\/\/auth\/callback#/)
  })

  it('falls back to the translated failure message when the error carries none', async () => {
    createDesktopSessionMock.mockRejectedValue('network down')

    const wrapper = mount(PaseoBridgeView)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain(lookup('auth.desktopBridge.failed'))
    expect(text).toContain(lookup('auth.desktopBridge.failedStatus'))
    expect(text).not.toMatch(/paseo/i)
  })

  describe('one-time sign-in code', () => {
    function useFakeClock() {
      vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval', 'Date'] })
      vi.setSystemTime(NOW)
    }

    async function mountBridge() {
      const wrapper = mount(PaseoBridgeView)
      wrappers.push(wrapper)
      await flushPromises()
      return wrapper
    }

    function codeQuery(extra: Record<string, unknown> = {}) {
      return {
        endpoint: 'https://api.example.com/',
        redirect_to: LOOPBACK,
        code_challenge: CHALLENGE,
        code_challenge_method: 'S256',
        ...extra,
      }
    }

    const codeCallback = `${LOOPBACK}#code=K7QM-3XPD&endpoint=https%3A%2F%2Fapi.example.com`

    it('issues a code for the challenge, shows it, then hands it to the loopback listener', async () => {
      useFakeClock()
      routeState.query = codeQuery()

      const wrapper = await mountBridge()

      expect(createDesktopLoginCodeMock).toHaveBeenCalledTimes(1)
      expect(createDesktopLoginCodeMock).toHaveBeenCalledWith({
        code_challenge: CHALLENGE,
        code_challenge_method: 'S256',
        api_key: 'sk-1',
        claude_api_key: 'sk-1',
        codex_api_key: 'sk-2',
      })
      expect(createDesktopSessionMock).not.toHaveBeenCalled()

      const text = wrapper.text()
      expect(wrapper.get('[data-testid="desktop-login-code-value"]').text()).toBe('K7QM-3XPD')
      expect(text).toContain(lookup('auth.desktopBridge.code.label'))
      expect(text).toContain(lookup('auth.desktopBridge.code.hint'))
      expect(text).toContain(lookup('auth.desktopBridge.code.expiresIn', { time: '10:00' }))
      expect(text).not.toContain('auth.desktopBridge.')
      expect(text).not.toMatch(/paseo/i)
      const link = wrapper.get('[data-testid="desktop-login-code"] a')
      expect(link.attributes('href')).toBe(codeCallback)
      expect(link.text()).toBe(lookup('auth.desktopBridge.code.returnToApp'))
      expect(JSON.parse(window.sessionStorage.getItem(STORAGE_KEY) ?? 'null')).toEqual({
        code: 'K7QM-3XPD',
        expiresAt: NOW + 600_000,
        redirected: false,
      })

      // A beat to see the code first.
      vi.advanceTimersByTime(1499)
      expect(locationState.current.href).toBe('http://localhost/auth/paseo')

      vi.advanceTimersByTime(1)
      await nextTick()
      expect(locationState.current.href).toBe(codeCallback)
      expect(wrapper.text()).toContain(lookup('auth.desktopBridge.opening'))
      expect(JSON.parse(window.sessionStorage.getItem(STORAGE_KEY) ?? 'null')).toMatchObject({
        redirected: true,
      })
    })

    it('shows the code already issued in this tab without asking again or navigating', async () => {
      useFakeClock()
      routeState.query = codeQuery()
      window.sessionStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ code: 'ABCD-EFGH', expiresAt: NOW + 120_000, redirected: true })
      )

      const wrapper = await mountBridge()
      vi.advanceTimersByTime(5_000)
      await nextTick()

      expect(createDesktopLoginCodeMock).not.toHaveBeenCalled()
      expect(createDesktopSessionMock).not.toHaveBeenCalled()
      expect(keysListMock).not.toHaveBeenCalled()
      expect(locationState.current.href).toBe('http://localhost/auth/paseo')

      expect(wrapper.get('[data-testid="desktop-login-code-value"]').text()).toBe('ABCD-EFGH')
      expect(wrapper.text()).toContain(lookup('auth.desktopBridge.code.expiresIn', { time: '01:55' }))
      const link = wrapper.get('[data-testid="desktop-login-code"] a')
      expect(link.attributes('href')).toBe(
        `${LOOPBACK}#code=ABCD-EFGH&endpoint=https%3A%2F%2Fapi.example.com`
      )
      expect(link.text()).toBe(lookup('auth.desktopBridge.code.returnToApp'))
    })

    it('copies the code and says so', async () => {
      useFakeClock()
      routeState.query = codeQuery()

      const wrapper = await mountBridge()
      const button = wrapper.get('[data-testid="desktop-login-code"] button')
      expect(button.text()).toBe(lookup('auth.desktopBridge.code.copy'))

      await button.trigger('click')
      await flushPromises()

      expect(copyToClipboardMock).toHaveBeenCalledWith(
        'K7QM-3XPD',
        lookup('auth.desktopBridge.code.copied')
      )
      expect(button.text()).toBe(lookup('auth.desktopBridge.code.copied'))
    })

    it('counts down and replaces the code once it expires', async () => {
      useFakeClock()
      createDesktopLoginCodeMock.mockResolvedValue({ code: 'K7QM-3XPD', expires_in: 3 })
      // No listener to hand it to: the code is only shown, never sent to a
      // guessed app scheme.
      routeState.query = codeQuery({ redirect_to: undefined })

      const wrapper = await mountBridge()
      expect(wrapper.text()).toContain(lookup('auth.desktopBridge.code.expiresIn', { time: '00:03' }))
      expect(wrapper.find('a').exists()).toBe(false)

      vi.advanceTimersByTime(1_000)
      await nextTick()
      expect(wrapper.text()).toContain(lookup('auth.desktopBridge.code.expiresIn', { time: '00:02' }))

      vi.advanceTimersByTime(2_000)
      await nextTick()
      const text = wrapper.text()
      expect(text).toContain(lookup('auth.desktopBridge.code.expired'))
      expect(text).not.toContain('K7QM-3XPD')
      expect(locationState.current.href).toBe('http://localhost/auth/paseo')
    })

    it('reports a failure to issue the code like the legacy flow does', async () => {
      routeState.query = codeQuery()
      createDesktopLoginCodeMock.mockRejectedValue(new Error('code service unavailable'))

      const wrapper = await mountBridge()

      const text = wrapper.text()
      expect(text).toContain('code service unavailable')
      expect(text).toContain(lookup('auth.desktopBridge.failedStatus'))
      expect(wrapper.find('[data-testid="desktop-login-code"]').exists()).toBe(false)
      expect(locationState.current.href).toBe('http://localhost/auth/paseo')
    })

    it.each([
      ['no challenge', { code_challenge: undefined, code_challenge_method: undefined }],
      ['a malformed challenge', { code_challenge: CHALLENGE.slice(1) }],
      ['a method other than S256', { code_challenge_method: 'plain' }],
    ])('keeps the token-fragment flow with %s', async (_label, override) => {
      useFakeClock()
      routeState.query = codeQuery(override)

      const wrapper = await mountBridge()

      expect(createDesktopLoginCodeMock).not.toHaveBeenCalled()
      expect(createDesktopSessionMock).toHaveBeenCalledTimes(1)
      expect(wrapper.find('[data-testid="desktop-login-code"]').exists()).toBe(false)
      expect(locationState.current.href).toBe(
        `${LOOPBACK}#access_token=desktop-at&refresh_token=desktop-rt&expires_in=900&api_key=sk-1&claude_api_key=sk-1&codex_api_key=sk-2&endpoint=https%3A%2F%2Fapi.example.com`
      )
    })
  })
})
