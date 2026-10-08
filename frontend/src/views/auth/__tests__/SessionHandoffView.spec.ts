import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SessionHandoffView from '../SessionHandoffView.vue'

const LOGIN = 'https://cheaprouter.cc'
const ALIAS = 'https://cdn.cheaprouter.cc'
const CHALLENGE = 'A'.repeat(43)
const VERIFIER = 'v'.repeat(43)

const mocks = vi.hoisted(() => ({
  route: { query: {} as Record<string, unknown>, meta: {} as Record<string, unknown> },
  replace: vi.fn(),
  createSessionHandoffCode: vi.fn(),
  exchangeSessionHandoffCode: vi.fn(),
  persistOAuthTokenContext: vi.fn(),
  setToken: vi.fn(),
  auth: { isAuthenticated: false, homePath: '/dashboard' },
  app: {
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    cachedPublicSettings: {} as Record<string, unknown>,
  },
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ replace: mocks.replace }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/api/sessionHandoff', () => ({
  createSessionHandoffCode: mocks.createSessionHandoffCode,
  exchangeSessionHandoffCode: mocks.exchangeSessionHandoffCode,
}))

vi.mock('@/api/auth', () => ({
  persistOAuthTokenContext: mocks.persistOAuthTokenContext,
  buildOAuthLoginStartURL: (request: { provider: string; params: Record<string, string> }) =>
    `/api/v1/auth/oauth/${request.provider}/start?${new URLSearchParams(request.params).toString()}`,
}))

vi.mock('@/stores', () => ({
  useAppStore: () => mocks.app,
  useAuthStore: () => ({ ...mocks.auth, setToken: mocks.setToken }),
}))

const stubs = {
  AuthLayout: { template: '<div><slot /></div>' },
  Icon: true,
  'router-link': { template: '<a><slot /></a>' },
}

const originalLocation = window.location

function stubLocation(origin: string, hash = '') {
  const location = {
    origin,
    hash,
    pathname: '/auth/handoff/complete',
    href: `${origin}/auth/handoff`,
    replace: vi.fn(),
    assign: vi.fn(),
  }
  Object.defineProperty(window, 'location', { value: location, writable: true })
  return location
}

function mountAs(mode: 'give' | 'complete' | 'pull', query: Record<string, unknown> = {}) {
  mocks.route.meta = { handoffMode: mode }
  mocks.route.query = query
  return mount(SessionHandoffView, { global: { stubs } })
}

describe('SessionHandoffView', () => {
  beforeEach(() => {
    sessionStorage.clear()
    vi.clearAllMocks()
    mocks.auth.isAuthenticated = false
    mocks.app.cachedPublicSettings = {
      session_handoff_login_origin: LOGIN,
      session_handoff_alias_origins: [ALIAS],
    }
    mocks.setToken.mockResolvedValue({})
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', { value: originalLocation, writable: true })
  })

  it('hands a signed-in session to the alias with a one-time code', async () => {
    const location = stubLocation(LOGIN)
    mocks.auth.isAuthenticated = true
    mocks.createSessionHandoffCode.mockResolvedValue({ code: 'CODE', target_origin: ALIAS, expires_in: 120 })

    mountAs('give', { origin: ALIAS, challenge: CHALLENGE, redirect: '/keys' })
    await flushPromises()

    expect(mocks.createSessionHandoffCode).toHaveBeenCalledWith({
      code_challenge: CHALLENGE,
      code_challenge_method: 'S256',
      target_origin: ALIAS,
    })
    expect(location.replace).toHaveBeenCalledWith(`${ALIAS}/auth/handoff/complete#code=CODE&redirect=%2Fkeys`)
    expect(sessionStorage.getItem('sub2api_session_handoff_give')).toBeNull()
  })

  it('starts the chosen OAuth provider on the login origin once, then falls back to the login page', async () => {
    const location = stubLocation(LOGIN)

    mountAs('give', { origin: ALIAS, challenge: CHALLENGE, redirect: '/keys', provider: 'github', aff_code: 'AFF' })
    await flushPromises()

    expect(location.href).toBe('/api/v1/auth/oauth/github/start?aff_code=AFF&redirect=%2Fauth%2Fhandoff')
    expect(sessionStorage.getItem('email_oauth_pending_provider')).toBe('github')
    expect(JSON.parse(sessionStorage.getItem('sub2api_session_handoff_give')!).oauthStarted).toBe(true)
    expect(mocks.createSessionHandoffCode).not.toHaveBeenCalled()

    // 授权回来仍未登录（例如用户取消）：不再自动发起，改去登录页
    mountAs('give')
    await flushPromises()
    expect(mocks.replace).toHaveBeenLastCalledWith({ path: '/login', query: { redirect: '/auth/handoff', aff_code: 'AFF' } })
  })

  it('refuses to hand off to an origin that is not configured', async () => {
    stubLocation(LOGIN)
    mocks.auth.isAuthenticated = true

    const wrapper = mountAs('give', { origin: 'https://evil.example', challenge: CHALLENGE })
    await flushPromises()

    expect(mocks.createSessionHandoffCode).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="session-handoff-error"]').text()).toContain('auth.sessionHandoff.originNotAllowed')
  })

  it('redeems the code with the local verifier and signs in on the alias', async () => {
    stubLocation(ALIAS, '#code=CODE&redirect=%2Fkeys')
    const replaceState = vi.spyOn(window.history, 'replaceState')
    sessionStorage.setItem('sub2api_session_handoff_receive', JSON.stringify({ verifier: VERIFIER, from: LOGIN, redirect: '/usage', t: Date.now() }))
    mocks.exchangeSessionHandoffCode.mockResolvedValue({ access_token: 'access', refresh_token: 'refresh', expires_in: 3600, token_type: 'Bearer' })

    mountAs('complete')
    await flushPromises()

    expect(replaceState).toHaveBeenCalled()
    expect(mocks.exchangeSessionHandoffCode).toHaveBeenCalledWith('CODE', VERIFIER)
    expect(mocks.persistOAuthTokenContext).toHaveBeenCalledWith({ refresh_token: 'refresh', expires_in: 3600 })
    expect(mocks.setToken).toHaveBeenCalledWith('access')
    expect(mocks.replace).toHaveBeenCalledWith('/keys')
    expect(sessionStorage.getItem('sub2api_session_handoff_receive')).toBeNull()
    replaceState.mockRestore()
  })

  it('does not redeem a code it never asked for', async () => {
    stubLocation(ALIAS, '#code=FOREIGN')

    const wrapper = mountAs('complete')
    await flushPromises()

    expect(mocks.exchangeSessionHandoffCode).not.toHaveBeenCalled()
    expect(mocks.setToken).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="session-handoff-error"]').text()).toContain('auth.sessionHandoff.expired')
  })

  it('pulls a session only from a configured origin', async () => {
    const location = stubLocation(LOGIN)

    const wrapper = mountAs('pull', { from: 'https://evil.example', redirect: '/profile' })
    await flushPromises()
    expect(location.assign).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="session-handoff-error"]').exists()).toBe(true)

    mountAs('pull', { from: ALIAS, redirect: '/profile' })
    await flushPromises()
    const target = new URL(location.assign.mock.calls[0][0] as string)
    expect(target.origin).toBe(ALIAS)
    expect(target.pathname).toBe('/auth/handoff')
    expect(target.searchParams.get('origin')).toBe(LOGIN)
    expect(target.searchParams.get('redirect')).toBe('/profile')
  })

  it('shows the disabled notice when handoff is not configured', async () => {
    stubLocation(LOGIN)
    mocks.app.cachedPublicSettings = {}

    const wrapper = mountAs('give', { origin: ALIAS, challenge: CHALLENGE })
    await flushPromises()

    expect(wrapper.find('[data-test="session-handoff-error"]').text()).toContain('auth.sessionHandoff.disabled')
  })
})
