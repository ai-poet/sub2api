import { describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: null }) }))

import { ForkFeatureFlags, isCanvasStudioEnabled, isLeaderboardEnabled } from '../forkFeatureFlags'
import { resolveFeatureFlag } from '../featureFlags'

describe('forkFeatureFlags', () => {
  it('defaults: model ranking on, user ranking and public stats off', () => {
    expect(resolveFeatureFlag(null, ForkFeatureFlags.modelRanking)).toBe(true)
    expect(resolveFeatureFlag(null, ForkFeatureFlags.usageRanking)).toBe(false)
    expect(resolveFeatureFlag(null, ForkFeatureFlags.publicStats)).toBe(false)
  })

  it('keeps canvas hidden unless the admin explicitly enables it', () => {
    expect(isCanvasStudioEnabled(undefined)).toBe(false)
    expect(isCanvasStudioEnabled({})).toBe(false)
    expect(isCanvasStudioEnabled({ canvas_studio_enabled: false })).toBe(false)
    expect(isCanvasStudioEnabled({ canvas_studio_enabled: true })).toBe(true)
  })

  it('shows the leaderboard entry while either board is on', () => {
    expect(isLeaderboardEnabled({ model_ranking_enabled: false, usage_ranking_enabled: false })).toBe(false)
    expect(isLeaderboardEnabled({ model_ranking_enabled: false, usage_ranking_enabled: true })).toBe(true)
    expect(isLeaderboardEnabled({ model_ranking_enabled: true, usage_ranking_enabled: false })).toBe(true)
    expect(isLeaderboardEnabled(undefined)).toBe(true)
  })
})
