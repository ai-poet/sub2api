import { describe, expect, it, vi } from 'vitest'
import type { RouteLocationNormalized } from 'vue-router'

import { allowAnonymousCatalog } from '../publicCatalog'

type Store = Parameters<typeof allowAnonymousCatalog>[1]

const route = (publicCatalog?: boolean) => ({ meta: { publicCatalog } }) as unknown as RouteLocationNormalized

const store = (overrides: Record<string, unknown> = {}) =>
  ({
    publicSettingsLoaded: true,
    backendModeEnabled: false,
    cachedPublicSettings: { public_pricing_enabled: true },
    fetchPublicSettings: vi.fn(),
    ...overrides,
  }) as unknown as Store

describe('allowAnonymousCatalog', () => {
  it('lets anonymous visitors into the catalog while public pricing is on', async () => {
    expect(await allowAnonymousCatalog(route(true), store())).toBe(true)
  })

  it('keeps the login redirect when public pricing is off', async () => {
    const appStore = store({ cachedPublicSettings: { public_pricing_enabled: false } })
    expect(await allowAnonymousCatalog(route(true), appStore)).toBe(false)
  })

  it('keeps the login redirect in backend mode', async () => {
    expect(await allowAnonymousCatalog(route(true), store({ backendModeEnabled: true }))).toBe(false)
  })

  it('ignores routes that are not marked public', async () => {
    expect(await allowAnonymousCatalog(route(undefined), store())).toBe(false)
  })

  it('loads public settings first when they are missing', async () => {
    const appStore = store({ publicSettingsLoaded: false, cachedPublicSettings: null })
    expect(await allowAnonymousCatalog(route(true), appStore)).toBe(true)
    expect(appStore.fetchPublicSettings).toHaveBeenCalledOnce()
  })
})
