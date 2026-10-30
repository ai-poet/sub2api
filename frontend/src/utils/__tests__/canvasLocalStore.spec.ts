import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { installFakeIdb, type FakeIdb } from './fakeIdb'
import {
  LocalStoreError,
  LocalStoreQuotaError,
  MAX_TERMINAL_RUNS,
  __resetCanvasStoreForTest,
  clearScope,
  enforceRetention,
  getWorkspaceId,
  listAssets,
  listRuns,
  listScenes,
  loadAsset,
  loadRun,
  loadScene,
  loadSetting,
  normalizeWorkspaceId,
  rotateWorkspaceId,
  saveAsset,
  saveRun,
  saveScene,
  saveSetting,
} from '../canvasLocalStore'
import type { CanvasRun, CanvasScope } from '@/types/canvas'

const DB_NAME = 'sub2api-canvas-studio'
const scopeA: CanvasScope = { userId: 1, workspaceId: 'ws-a' }
const scopeB: CanvasScope = { userId: 1, workspaceId: 'ws-b' }
const scopeOtherUser: CanvasScope = { userId: 2, workspaceId: 'ws-a' }

let fake: FakeIdb

beforeEach(() => {
  fake = installFakeIdb()
  __resetCanvasStoreForTest()
  window.localStorage.clear()
})

afterEach(() => {
  __resetCanvasStoreForTest()
  vi.unstubAllGlobals()
})

function makeRun(id: string, overrides: Partial<CanvasRun> = {}): CanvasRun {
  return {
    id,
    apiKeyId: 1,
    params: {
      apiKeyId: 1,
      model: 'gpt-image-1',
      prompt: 'p',
      size: '1024x1024',
      quality: '',
      count: 1,
      operation: 'generate',
    },
    status: 'completed',
    createdAt: 1,
    expiresAt: 0,
    paused: false,
    outputs: [],
    ...overrides,
  }
}

function bigBlob(size: number): Blob {
  const blob = new Blob(['x'])
  // Shadows the prototype getter so tests can simulate large assets without
  // allocating hundreds of MiB.
  Object.defineProperty(blob, 'size', { value: size })
  return blob
}

describe('canvasLocalStore scoped records', () => {
  it('leaves unowned v1 records intact and unclaimed after upgrade', async () => {
    fake.seedDatabase(DB_NAME, 1, {
      scenes: [{ key: 'default', json: '{"v":1}', updatedAt: 1 }],
      assets: [{ key: 'legacy', blob: new Blob(['old']), createdAt: 1 }],
    })

    // The v2 open upgrades in place: v1 records survive but carry no scope.
    expect(await loadScene(scopeA, 'default')).toBeNull()
    expect(await listScenes(scopeA)).toEqual([])
    expect(await loadAsset(scopeA, 'legacy')).toBeNull()

    await saveScene(scopeA, 'default', '{"v":2}')
    expect((await listScenes(scopeA)).map((scene) => scene.json)).toEqual(['{"v":2}'])

    const rawScenes = fake.inspect(DB_NAME, 'scenes')
    expect(rawScenes?.get('default')).toMatchObject({ json: '{"v":1}' })
    const rawAssets = fake.inspect(DB_NAME, 'assets')
    expect(rawAssets?.has('legacy')).toBe(true)
    // The v2 upgrade added the new stores without touching v1 data.
    expect(fake.inspect(DB_NAME, 'runs')).toBeDefined()
    expect(fake.inspect(DB_NAME, 'settings')).toBeDefined()
  })

  it('isolates every record type by user and workspace', async () => {
    await saveAsset(scopeA, {
      key: 'source:x',
      kind: 'source',
      blob: new Blob(['x']),
      createdAt: 1,
    })
    await saveScene(scopeA, 'default', '{}')
    await saveRun(scopeA, makeRun('r1'))
    await saveSetting(scopeA, 'selection', { model: 'm' })

    for (const other of [scopeB, scopeOtherUser]) {
      expect(await loadAsset(other, 'source:x')).toBeNull()
      expect(await listScenes(other)).toEqual([])
      expect(await loadRun(other, 'r1')).toBeNull()
      expect(await loadSetting(other, 'selection')).toBeNull()
    }

    expect((await loadAsset(scopeA, 'source:x'))?.kind).toBe('source')
    expect(await loadRun(scopeA, 'r1')).toMatchObject({ id: 'r1' })
    expect(await loadSetting(scopeA, 'selection')).toEqual({ model: 'm' })
  })

  it('detaches reactive inputs into plain snapshots at write time', async () => {
    // Real IndexedDB structured-clones records and rejects proxies; the store
    // must detach Vue reactive inputs so writes succeed and snapshot state.
    const reactiveScope = reactive({ userId: 1, workspaceId: 'ws-a' })
    const reactiveRun = reactive(makeRun('rx'))
    await saveRun(reactiveScope, reactiveRun)
    const reactiveSetting = reactive({ nested: { model: 'm' } })
    await saveSetting(reactiveScope, 'selection', reactiveSetting)
    const reactiveAsset = reactive({
      key: 'source:r',
      kind: 'source' as const,
      blob: new Blob(['r']),
      createdAt: 1,
    })
    await saveAsset(reactiveScope, reactiveAsset)

    // Mutating the originals after the write cannot alter what was stored.
    reactiveRun.status = 'failed'
    reactiveRun.error = 'changed'
    reactiveSetting.nested.model = 'changed'
    reactiveAsset.kind = 'output' as never

    expect((await loadRun(scopeA, 'rx'))?.status).toBe('completed')
    expect((await loadSetting(scopeA, 'selection'))).toEqual({ nested: { model: 'm' } })
    expect((await loadAsset(scopeA, 'source:r'))?.kind).toBe('source')
  })

  it('clears only the requested scope', async () => {
    fake.seedDatabase(DB_NAME, 1, {
      assets: [{ key: 'legacy', blob: new Blob(['old']), createdAt: 1 }],
    })
    await saveAsset(scopeA, { key: 'a', kind: 'source', blob: new Blob(['a']), createdAt: 1 })
    await saveAsset(scopeB, { key: 'b', kind: 'source', blob: new Blob(['b']), createdAt: 1 })

    await clearScope(scopeA)

    expect(await listAssets(scopeA)).toEqual([])
    expect((await listAssets(scopeB)).map((asset) => asset.key)).toEqual(['b'])
    expect(fake.inspect(DB_NAME, 'assets')?.has('legacy')).toBe(true)
  })
})

describe('canvasLocalStore transaction semantics', () => {
  it('rejects and rolls back when a request fails', async () => {
    await saveSetting(scopeA, 'before', 1)
    fake.failNextRequest(new DOMException('full', 'QuotaExceededError'))

    await expect(
      saveAsset(scopeA, { key: 'a', kind: 'source', blob: new Blob(['a']), createdAt: 1 }),
    ).rejects.toBeInstanceOf(LocalStoreQuotaError)
    expect(await loadAsset(scopeA, 'a')).toBeNull()
    // Earlier committed records survive the failed transaction.
    expect(await loadSetting(scopeA, 'before')).toBe(1)
  })

  it('resolves a write only after the transaction commits', async () => {
    fake.failNextCommit(new DOMException('commit failed', 'UnknownError'))

    // If the API only awaited request success, this promise would resolve.
    await expect(saveScene(scopeA, 'default', '{}')).rejects.toBeInstanceOf(LocalStoreError)
    expect(await loadScene(scopeA, 'default')).toBeNull()
  })
})

describe('canvasLocalStore retention', () => {
  it('caps terminal runs and never prunes active runs', async () => {
    let pruned: string[] = []
    for (let i = 0; i < MAX_TERMINAL_RUNS + 5; i += 1) {
      pruned = await saveRun(scopeA, makeRun(`r${i}`, { createdAt: i }))
    }
    expect(pruned).toEqual(['r4'])
    const runs = await listRuns(scopeA)
    expect(runs).toHaveLength(MAX_TERMINAL_RUNS)
    // Oldest terminal runs were pruned; the newest survive.
    expect(runs[0].id).toBe('r5')
    expect(runs[runs.length - 1].id).toBe(`r${MAX_TERMINAL_RUNS + 4}`)

    await saveRun(scopeA, makeRun('active', { status: 'processing', createdAt: 999 }))
    const after = await listRuns(scopeA)
    expect(after).toHaveLength(MAX_TERMINAL_RUNS + 1)
    expect(after.some((run) => run.id === 'active')).toBe(true)
  })

  it('evicts unprotected assets but protects scene and run references', async () => {
    const mib = 1024 * 1024
    await saveScene(scopeA, 'default', '{"src":"asset://source:scene"}')
    await saveRun(
      scopeA,
      makeRun('run1', {
        outputs: [{ index: 0, assetKey: 'output:run1:0', state: 'local' }],
      }),
    )
    // Protected: scene ref (200MiB) + run output ref (100MiB).
    await saveAsset(scopeA, { key: 'source:scene', kind: 'source', blob: bigBlob(200 * mib), createdAt: 1 })
    await saveAsset(scopeA, { key: 'output:run1:0', kind: 'output', blob: bigBlob(100 * mib), createdAt: 2 })
    // Unprotected: oldest evicted first.
    await saveAsset(scopeA, { key: 'source:old', kind: 'source', blob: bigBlob(50 * mib), createdAt: 3 })
    await saveAsset(scopeA, { key: 'source:new', kind: 'source', blob: bigBlob(50 * mib), createdAt: 4 })

    const report = await enforceRetention(scopeA)

    expect(report.evictedAssetKeys).toEqual(['source:old', 'source:new'])
    // Protected content alone exceeds the budget: warn, don't break the scene.
    expect(report.overBudget).toBe(true)
    expect(report.assetBytes).toBe(300 * mib)
    expect((await loadAsset(scopeA, 'source:scene'))?.key).toBe('source:scene')
    expect((await loadAsset(scopeA, 'output:run1:0'))?.key).toBe('output:run1:0')
    expect(await loadAsset(scopeA, 'source:old')).toBeNull()
  })

  it('stays within budget by evicting oldest unprotected assets first', async () => {
    const mib = 1024 * 1024
    await saveAsset(scopeA, { key: 'source:1', kind: 'source', blob: bigBlob(200 * mib), createdAt: 1 })
    await saveAsset(scopeA, { key: 'source:2', kind: 'source', blob: bigBlob(100 * mib), createdAt: 2 })

    const report = await enforceRetention(scopeA)

    expect(report.evictedAssetKeys).toEqual(['source:1'])
    expect(report.overBudget).toBe(false)
    expect(report.assetBytes).toBe(100 * mib)
    expect(await loadAsset(scopeA, 'source:1')).toBeNull()
    expect((await loadAsset(scopeA, 'source:2'))?.key).toBe('source:2')
  })
})

describe('workspace identity', () => {
  it('creates a stable per-user UUID and rotates on clear', () => {
    const first = getWorkspaceId(1)
    expect(normalizeWorkspaceId(first)).toBe(first)
    expect(getWorkspaceId(1)).toBe(first)
    expect(getWorkspaceId(2)).not.toBe(first)

    const rotated = rotateWorkspaceId(1)
    expect(rotated).not.toBe(first)
    expect(getWorkspaceId(1)).toBe(rotated)
  })

  it('regenerates an invalid stored workspace id', () => {
    window.localStorage.setItem('canvas:workspace:7', 'not-a-uuid')
    const id = getWorkspaceId(7)
    expect(normalizeWorkspaceId(id)).toBe(id)
  })
})
