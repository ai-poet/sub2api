import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN'
}))

import { appealClient, getSession, createTicket } from '@/api/appeal'
import { storeAppealSession } from '@/utils/appeal'

describe('appeal API client (fork)', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
  })

  it('sends only the appeal header, never Authorization, even when signed in elsewhere', async () => {
    localStorage.setItem('auth_token', 'normal-jwt')
    storeAppealSession('apl_token', 7200)
    const adapter = vi.fn().mockResolvedValue({
      status: 200,
      data: { code: 0, data: { email: 'u@example.com', status: 'disabled', expires_at: '', has_active_appeal: false } },
      headers: {},
      config: {},
      statusText: 'OK'
    })
    appealClient.defaults.adapter = adapter

    const session = await getSession()
    expect(session.email).toBe('u@example.com')
    const config = adapter.mock.calls[0][0]
    expect(config.headers.get('X-Appeal-Token')).toBe('apl_token')
    expect(config.headers.get('Authorization')).toBeFalsy()
    expect(config.headers.get('Accept-Language')).toBe('zh-CN')
  })

  it('a 401 neither clears the normal login state nor redirects', async () => {
    localStorage.setItem('auth_token', 'normal-jwt')
    storeAppealSession('apl_token', 7200)
    appealClient.defaults.adapter = vi.fn().mockRejectedValue({
      response: { status: 401, data: { code: 401, reason: 'APPEAL_TOKEN_INVALID', message: 'appeal session is invalid or expired' } },
      config: { headers: {} },
      message: 'Request failed'
    })

    await expect(getSession()).rejects.toEqual(
      expect.objectContaining({ status: 401, reason: 'APPEAL_TOKEN_INVALID' })
    )
    expect(localStorage.getItem('auth_token')).toBe('normal-jwt')
    expect(window.location.pathname).not.toBe('/login')
  })

  it('surfaces business errors with their reason', async () => {
    storeAppealSession('apl_token', 7200)
    appealClient.defaults.adapter = vi.fn().mockRejectedValue({
      response: { status: 409, data: { code: 409, reason: 'TICKET_APPEAL_ACTIVE', message: 'an appeal ticket is already open' } },
      config: { headers: {} },
      message: 'Request failed'
    })
    await expect(createTicket('t', 'b')).rejects.toEqual(expect.objectContaining({ status: 409, reason: 'TICKET_APPEAL_ACTIVE' }))
  })
})
