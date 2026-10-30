import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { installFakeIdb, type FakeIdb } from '@/utils/__tests__/fakeIdb'
import {
  __resetCanvasStoreForTest,
  getWorkspaceId,
  listAssets,
  listRuns,
  loadAsset,
  loadRun,
  saveAsset,
  saveRun,
  saveSetting,
} from '@/utils/canvasLocalStore'
import { useAuthStore } from '@/stores/auth'
import { useCanvasStudio, __resetCanvasStudioForTest } from '../useCanvasStudio'
import { getCanvasTask, submitCanvasTask } from '@/api/canvas'
import { keysAPI } from '@/api/keys'
import { getCatalog } from '@/api/modelCatalog'
import type { ApiKey, Group, User } from '@/types'
import type { CanvasRun, CanvasScope } from '@/types/canvas'

vi.mock('@/api/canvas', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/canvas')>()
  return {
    ...actual,
    submitCanvasTask: vi.fn(),
    getCanvasTask: vi.fn(),
  }
})

vi.mock('@/api/keys', () => ({
  keysAPI: { list: vi.fn() },
}))

vi.mock('@/api/modelCatalog', () => ({
  getCatalog: vi.fn(),
}))

const submitMock = vi.mocked(submitCanvasTask)
const pollMock = vi.mocked(getCanvasTask)
const listKeysMock = vi.mocked(keysAPI.list)
const catalogMock = vi.mocked(getCatalog)

let fake: FakeIdb

function makeUser(id: number): User {
  return { id, username: `u${id}`, email: '', role: 'user' } as User
}

function makeKey(id: number, platform = 'openai'): ApiKey {
  return {
    id,
    user_id: 1,
    key: `sk-${id}`,
    name: `k${id}`,
    group_id: id,
    status: 'active',
    group: {
      id,
      name: `g${id}`,
      platform,
      allow_image_generation: true,
    } as Group,
  } as ApiKey
}

function keyPage(items: ApiKey[]) {
  return { items, page: 1, pages: 1, total: items.length }
}

/** catalogWith returns a catalog where each model is served by group groupId. */
function catalogWith(models: Array<{ model: string; groupId: number }>) {
  return {
    items: models.map(({ model, groupId }) => ({
      model,
      billing_mode: 'image',
      best_group: { id: groupId },
      other_groups: [],
    })),
  }
}

function makeRun(id: string, overrides: Partial<CanvasRun> = {}): CanvasRun {
  return {
    id,
    apiKeyId: 1,
    params: {
      apiKeyId: 1,
      model: 'gpt-image-1',
      prompt: 'seeded',
      size: '1024x1024',
      quality: '',
      count: 1,
      operation: 'generate',
    },
    status: 'processing',
    createdAt: 1,
    expiresAt: Date.now() + 60 * 60 * 1000,
    paused: false,
    outputs: [],
    ...overrides,
  }
}

function scopeFor(userId = 1): CanvasScope {
  return { userId, workspaceId: getWorkspaceId(userId) }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

/** flush settles pending microtasks without advancing timers. */
async function flush(): Promise<void> {
  for (let i = 0; i < 6; i += 1) {
    await vi.advanceTimersByTimeAsync(0)
  }
}

function networkError(): Error {
  return Object.assign(new Error('network down'), { status: 0 })
}

function completedTask(data: unknown[], expiresAt = 4_000_000_000) {
  return { status: 'completed', expires_at: expiresAt, result: { data } }
}

beforeEach(() => {
  setActivePinia(createPinia())
  fake = installFakeIdb()
  __resetCanvasStoreForTest()
  __resetCanvasStudioForTest()
  window.localStorage.clear()
  vi.useFakeTimers()
  useAuthStore().user = makeUser(1)
  listKeysMock.mockResolvedValue(keyPage([makeKey(1), makeKey(2)]))
  catalogMock.mockResolvedValue(
    catalogWith([{ model: 'gpt-image-1', groupId: 1 }]) as never,
  )
  submitMock.mockResolvedValue({
    id: 'imgtask_1',
    task_id: 'imgtask_1',
    status: 'processing',
    created_at: 1,
    expires_at: 4_000_000_000,
  })
  pollMock.mockResolvedValue({ status: 'processing' } as never)
})

afterEach(() => {
  __resetCanvasStudioForTest()
  __resetCanvasStoreForTest()
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('useCanvasStudio initialize', () => {
  it('restores the persisted selection into the param refs', async () => {
    const scope = scopeFor()
    catalogMock.mockResolvedValue(
      catalogWith([
        { model: 'gpt-image-1', groupId: 1 },
        { model: 'gpt-image-2', groupId: 2 },
      ]) as never,
    )
    await saveSetting(scope, 'selection', {
      apiKeyId: 2,
      model: 'gpt-image-2',
      prompt: 'kept prompt',
      size: '2048x2048',
      quality: 'high',
      count: 3,
      operation: 'edit',
    })
    const studio = useCanvasStudio()

    await studio.initialize()

    expect(studio.apiKeyId.value).toBe(2)
    expect(studio.model.value).toBe('gpt-image-2')
    expect(studio.prompt.value).toBe('kept prompt')
    expect(studio.size.value).toBe('1024x1024') // unknown tier normalizes to default
    expect(studio.quality.value).toBe('high')
    expect(studio.count.value).toBe(3)
    expect(studio.operation.value).toBe('edit')
    expect(studio.ready.value).toBe(true)
  })

  it('normalizes invalid persisted fields and unavailable key/model', async () => {
    const scope = scopeFor()
    await saveSetting(scope, 'selection', {
      apiKeyId: 99, // not an eligible key
      model: 'ghost-model', // absent from catalog
      prompt: 'keep me',
      size: 'bogus',
      quality: 'ultra',
      count: 9,
      operation: 'zzz',
    })
    const studio = useCanvasStudio()

    await studio.initialize()

    expect(studio.apiKeyId.value).toBe(1) // fell back to the default eligible key
    expect(studio.model.value).toBe('gpt-image-1')
    expect(studio.prompt.value).toBe('keep me')
    expect(studio.size.value).toBe('1024x1024')
    expect(studio.quality.value).toBe('')
    expect(studio.count.value).toBe(1)
    expect(studio.operation.value).toBe('generate')
  })

  it('is a no-op for an already-ready same user and workspace', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-live'))
    const studio = useCanvasStudio()
    await studio.initialize()
    const before = studio.history.value

    await studio.initialize()

    // Ordinary remounts must not replace live history objects trackers use.
    expect(studio.history.value).toBe(before)
  })

  it('resumes a stored run polling with its original key, not the selected one', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-old', { apiKeyId: 1 }))
    const studio = useCanvasStudio()

    await studio.initialize()

    expect(studio.apiKeyId.value).toBe(1)
    expect(pollMock).toHaveBeenCalledWith('sk-1', 'run-old', expect.anything())
    studio.apiKeyId.value = 2
    pollMock.mockClear()
    await vi.advanceTimersByTimeAsync(3000)
    expect(pollMock).toHaveBeenCalledWith('sk-1', 'run-old', expect.anything())
  })

  it('marks a run unavailable only when a loaded key list lacks its key', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-gone', { apiKeyId: 9 }))
    const studio = useCanvasStudio()

    await studio.initialize()

    const run = studio.history.value[0]
    expect(run.status).toBe('unavailable')
    expect(run.error).toBe('key-missing')
    expect(pollMock).not.toHaveBeenCalled()
  })

  it('pauses instead of judging when the key list fails to load', async () => {
    listKeysMock.mockRejectedValue(new Error('server error'))
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-pending', { apiKeyId: 1 }))
    const studio = useCanvasStudio()

    await studio.initialize()

    const run = studio.history.value[0]
    expect(run.status).toBe('processing')
    expect(run.paused).toBe(true)
    expect(pollMock).not.toHaveBeenCalled()

    listKeysMock.mockResolvedValue(keyPage([makeKey(1)]))
    await studio.resumeRun('run-pending')
    expect(pollMock).toHaveBeenCalledWith('sk-1', 'run-pending', expect.anything())
    expect(run.paused).toBe(false)
  })

  it('settles runs past their retention deadline as expired', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-expired', { expiresAt: Date.now() - 1000 }))
    const studio = useCanvasStudio()

    await studio.initialize()

    expect(studio.history.value[0].status).toBe('expired')
    expect(pollMock).not.toHaveBeenCalled()
  })

  it('refreshes expiresAt from poll results instead of the stale acceptance time', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-ttl', { expiresAt: Date.now() + 500 }))
    pollMock.mockResolvedValue(completedTask([{ b64_json: 'aGk=' }]) as never)
    const studio = useCanvasStudio()

    await studio.initialize()
    await flush()

    const run = studio.history.value[0]
    expect(run.status).toBe('completed')
    expect(run.expiresAt).toBe(4_000_000_000_000)
  })
})

describe('useCanvasStudio submit and polling', () => {
  async function readyStudio() {
    const studio = useCanvasStudio()
    await studio.initialize()
    studio.prompt.value = 'draw a boat'
    return studio
  }

  it('freezes params and never auto-retries a failed POST', async () => {
    const studio = await readyStudio()
    submitMock.mockRejectedValue(new Error('insufficient quota'))

    const run = await studio.submit({ images: [] })

    expect(run).toBeNull()
    expect(submitMock).toHaveBeenCalledTimes(1)
    expect(studio.errorMessage.value).toBe('insufficient quota')
    expect(studio.submitting.value).toBe(false)
  })

  it('refuses to submit when the model list is empty', async () => {
    catalogMock.mockResolvedValue({ items: [] } as never)
    const studio = await readyStudio()

    expect(await studio.submit()).toBeNull()
    expect(studio.errorMessage.value).toBe('no-models')
    expect(submitMock).not.toHaveBeenCalled()
  })

  it('fails validation locally: prompt, sources, mask, platform caps', async () => {
    const img = () => new Blob(['x'], { type: 'image/png' })
    listKeysMock.mockResolvedValue(
      keyPage([makeKey(1), makeKey(2), makeKey(3, 'grok')]),
    )
    catalogMock.mockResolvedValue(
      catalogWith([
        { model: 'gpt-image-1', groupId: 1 },
        { model: 'grok-image', groupId: 3 },
      ]) as never,
    )
    const studio = await readyStudio()

    studio.prompt.value = '   '
    expect(await studio.submit()).toBeNull()
    expect(studio.errorMessage.value).toBe('empty-prompt')

    studio.prompt.value = 'draw'
    studio.operation.value = 'edit'
    expect(await studio.submit({ images: [] })).toBeNull()
    expect(studio.errorMessage.value).toBe('missing-source')

    expect(await studio.submit({ images: [img()], mask: img() })).toBeNull()
    expect(studio.errorMessage.value).toBe('invalid-input')

    studio.operation.value = 'inpaint'
    expect(await studio.submit({ images: [img()] })).toBeNull()
    expect(studio.errorMessage.value).toBe('invalid-input')

    // Grok accepts at most 3 references.
    studio.apiKeyId.value = 3
    studio.operation.value = 'edit'
    studio.model.value = 'grok-image'
    expect(await studio.submit({ images: [img(), img(), img(), img()] })).toBeNull()
    expect(studio.errorMessage.value).toBe('too-many-images')
    expect(submitMock).not.toHaveBeenCalled()
  })

  it('keeps frozen params and original key after the selection changes', async () => {
    const studio = await readyStudio()
    studio.apiKeyId.value = 1
    studio.model.value = 'gpt-image-1'
    studio.prompt.value = 'p-old'

    const run = await studio.submit()
    expect(run?.params).toMatchObject({ apiKeyId: 1, model: 'gpt-image-1', prompt: 'p-old' })

    studio.apiKeyId.value = 2
    studio.model.value = 'm-new'
    studio.prompt.value = 'p-new'
    await vi.advanceTimersByTimeAsync(3000)

    expect(pollMock).toHaveBeenCalledWith('sk-1', 'imgtask_1', expect.anything())
    expect(run?.params.prompt).toBe('p-old')
  })

  it('harvests parallel completions independently by output index', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r1', { apiKeyId: 1 }))
    await saveRun(scope, makeRun('r2', { apiKeyId: 2, createdAt: 2 }))
    const first = deferred<never>()
    const second = deferred<never>()
    pollMock.mockImplementation(((_key: string, taskId: string) =>
      taskId === 'r1' ? first.promise : second.promise) as never)
    const studio = useCanvasStudio()
    await studio.initialize()

    second.resolve(completedTask([{ b64_json: 'aGk=' }]) as never)
    await flush()
    first.resolve(completedTask([{ b64_json: 'aGVsbG8=' }]) as never)
    await flush()

    const r1 = studio.history.value.find((run) => run.id === 'r1')!
    const r2 = studio.history.value.find((run) => run.id === 'r2')!
    expect(r1.status).toBe('completed')
    expect(r2.status).toBe('completed')
    expect(r1.outputs[0]).toMatchObject({ index: 0, state: 'local', assetKey: 'output:r1:0' })
    expect(r2.outputs[0]).toMatchObject({ index: 0, state: 'local', assetKey: 'output:r2:0' })
    expect((await loadAsset(scope, 'output:r1:0'))?.blob.size).toBe(5)
    expect((await loadAsset(scope, 'output:r2:0'))?.blob.size).toBe(2)
    expect(pollMock).toHaveBeenCalledWith('sk-1', 'r1', expect.anything())
    expect(pollMock).toHaveBeenCalledWith('sk-2', 'r2', expect.anything())
  })

  it('pauses after repeated transient failures and resumes manually', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-net', { apiKeyId: 1 }))
    pollMock.mockRejectedValue(networkError())
    const studio = useCanvasStudio()
    await studio.initialize()

    expect(pollMock).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(6000)
    expect(pollMock).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(12000)
    expect(pollMock).toHaveBeenCalledTimes(3)

    const run = studio.history.value[0]
    expect(run.paused).toBe(true)
    expect(run.status).toBe('processing')
    await vi.advanceTimersByTimeAsync(60000)
    expect(pollMock).toHaveBeenCalledTimes(3)

    pollMock.mockResolvedValue({ status: 'processing' } as never)
    await studio.resumeRun('run-net')
    expect(pollMock).toHaveBeenCalledTimes(4)
    expect(run.paused).toBe(false)
  })

  it('settles auth and gone answers without pausing', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('run-auth', { apiKeyId: 1 }))
    await saveRun(scope, makeRun('run-404', { apiKeyId: 2, createdAt: 2 }))
    pollMock.mockImplementation(((_key: string, taskId: string) =>
      taskId === 'run-auth'
        ? Promise.reject(Object.assign(new Error('denied'), { status: 401 }))
        : Promise.reject(Object.assign(new Error('gone'), { status: 404 }))) as never)
    const studio = useCanvasStudio()
    await studio.initialize()
    await flush()

    const auth = studio.history.value.find((run) => run.id === 'run-auth')!
    const gone = studio.history.value.find((run) => run.id === 'run-404')!
    expect(auth).toMatchObject({ status: 'unavailable', error: 'key-unauthorized' })
    expect(gone).toMatchObject({ status: 'unavailable', error: 'task-missing' })
    await vi.advanceTimersByTimeAsync(60000)
    expect(pollMock).toHaveBeenCalledTimes(2)
  })

  it('treats storage failure as a warning, not a task failure', async () => {
    const studio = await readyStudio()
    await vi.advanceTimersByTimeAsync(400) // flush the selection persist write
    fake.failNextRequest(new DOMException('full', 'QuotaExceededError'))

    const run = await studio.submit()
    await flush() // persistRun is queued; the warning lands async

    expect(run).not.toBeNull()
    expect(studio.storageWarning.value).toBe('quota')
    expect(pollMock).toHaveBeenCalledWith('sk-1', 'imgtask_1', expect.anything())
  })
})

describe('useCanvasStudio outputs', () => {
  it('saves outputs by index, keeps remote URLs on failure, dedups handler calls', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-out', { apiKeyId: 1 }))
    const fetchMock = vi.fn().mockRejectedValue(new Error('CORS'))
    vi.stubGlobal('fetch', fetchMock)
    const handler = vi.fn()
    const studio = useCanvasStudio()
    const unregister = studio.setOutputHandler(handler)
    const task = completedTask([{ b64_json: 'aGk=' }, {}, { url: 'https://cdn.example.com/x.png' }])
    pollMock.mockResolvedValue(task as never)
    await studio.initialize()
    await flush()

    const run = studio.history.value.find((item) => item.id === 'r-out')!
    expect(run.status).toBe('completed')
    expect(run.outputs).toEqual([
      { index: 0, state: 'local', assetKey: 'output:r-out:0', error: undefined, remoteUrl: undefined },
      { index: 1, state: 'missing', error: 'no-bytes', assetKey: undefined, remoteUrl: undefined },
      {
        index: 2,
        state: 'remote',
        remoteUrl: 'https://cdn.example.com/x.png',
        error: 'unreachable',
        assetKey: undefined,
      },
    ])
    expect(handler).toHaveBeenCalledTimes(1)
    expect(handler.mock.calls[0][0].id).toBe('r-out')
    expect(handler.mock.calls[0][1].index).toBe(0)
    expect((await listAssets(scope)).map((asset) => asset.key)).toEqual(['output:r-out:0'])

    // Retrying harvest never duplicates the placed output or the asset.
    expect(await studio.retryOutputs('r-out')).toBe(true)
    await flush()
    expect(handler).toHaveBeenCalledTimes(1)
    expect(await listAssets(scope)).toHaveLength(1)
    expect(run.outputs[2].state).toBe('remote')

    // A reachable remote URL promotes the output to a local asset on take.
    fetchMock.mockResolvedValue({
      ok: true,
      // Declared type keeps the download off the FileReader sniff path,
      // which jsdom does not complete under fake timers.
      blob: async () => new Blob(['png!'], { type: 'image/png' }),
    } as Response)
    const blob = await studio.takeOutput('r-out', 2)
    expect(blob?.size).toBe(4)
    expect(run.outputs[2].state).toBe('local')
    expect(await loadAsset(scope, 'output:r-out:2')).not.toBeNull()

    unregister()
  })

  it('reuses bytes persisted by a partial earlier harvest', async () => {
    const scope = scopeFor()
    await saveAsset(scope, {
      key: 'output:r-part:0',
      kind: 'output',
      blob: new Blob(['AAAA']),
      runId: 'r-part',
      outputIndex: 0,
      createdAt: 1,
    })
    await saveRun(
      scope,
      makeRun('r-part', {
        outputs: [
          { index: 0, state: 'pending' },
          { index: 1, state: 'pending' },
        ],
      }),
    )
    // The stored bytes survive; the task payload differs to prove reuse.
    pollMock.mockResolvedValue(
      completedTask([{ b64_json: 'QkJCQg==' }, { b64_json: 'aGk=' }]) as never,
    )
    const studio = useCanvasStudio()
    await studio.initialize()
    await flush()

    const run = studio.history.value[0]
    expect(run.outputs[0]).toMatchObject({ index: 0, state: 'local', assetKey: 'output:r-part:0' })
    expect(run.outputs[1]).toMatchObject({ index: 1, state: 'local', assetKey: 'output:r-part:1' })
    expect((await loadAsset(scope, 'output:r-part:0'))?.blob.size).toBe(4)
    expect((await loadAsset(scope, 'output:r-part:1'))?.blob.size).toBe(2)
  })

  it('keeps a paid output reachable in-session when storage is full', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-full', { apiKeyId: 1 }))
    const studio = useCanvasStudio()
    await studio.initialize()
    await vi.advanceTimersByTimeAsync(400)
    // Quota failures hit, in request order: the pre-harvest run write, the
    // prior-asset read inside harvest, and the asset write itself.
    fake.failNextRequest(new DOMException('full', 'QuotaExceededError'))
    fake.failNextRequest(new DOMException('full', 'QuotaExceededError'))
    fake.failNextRequest(new DOMException('full', 'QuotaExceededError'))
    pollMock.mockResolvedValue(completedTask([{ b64_json: 'aGk=' }]) as never)
    await flush()
    await vi.advanceTimersByTimeAsync(3000)
    await flush()

    const run = studio.history.value[0]
    expect(run.outputs[0]).toMatchObject({ index: 0, state: 'missing', error: 'storage' })
    expect(studio.storageWarning.value).toBe('quota')
    // The live blob is still returned this session instead of being lost.
    const blob = await studio.takeOutput('r-full', 0)
    expect(blob?.size).toBe(2)
    expect(await loadAsset(scope, 'output:r-full:0')).toBeNull()
  })

  it('delivers already-local outputs to a handler mounted after harvest', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-early', { apiKeyId: 1 }))
    pollMock.mockResolvedValue(completedTask([{ b64_json: 'aGk=' }]) as never)
    const studio = useCanvasStudio()
    await studio.initialize()
    await flush()
    expect(studio.history.value[0].outputs[0].state).toBe('local')

    const handler = vi.fn()
    studio.setOutputHandler(handler)
    await flush()

    expect(handler).toHaveBeenCalledTimes(1)
    expect(handler.mock.calls[0][1].index).toBe(0)
    expect(handler.mock.calls[0][2]?.size).toBe(2)
  })

  it('redelivers to a new handler after a failed placement', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-place', { apiKeyId: 1 }))
    pollMock.mockResolvedValue(completedTask([{ b64_json: 'aGk=' }]) as never)
    const failing = vi.fn().mockRejectedValue(new Error('editor busy'))
    const studio = useCanvasStudio()
    studio.setOutputHandler(failing)
    await studio.initialize()
    await flush()
    expect(failing).toHaveBeenCalledTimes(1)

    // The failed placement is not marked placed, so the next handler gets it.
    const next = vi.fn()
    studio.setOutputHandler(next)
    await flush()
    expect(next).toHaveBeenCalledTimes(1)
    expect(next.mock.calls[0][1].index).toBe(0)
  })

  it('harvests to storage but skips a disposed output handler', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-late', { apiKeyId: 1 }))
    const handler = vi.fn()
    const studio = useCanvasStudio()
    const unregister = studio.setOutputHandler(handler)
    unregister()
    pollMock.mockResolvedValue(completedTask([{ b64_json: 'aGk=' }]) as never)

    await studio.initialize()
    await flush()

    expect(handler).not.toHaveBeenCalled()
    expect((await loadAsset(scope, 'output:r-late:0'))?.blob.size).toBe(2)
    expect(studio.history.value[0].status).toBe('completed')
  })
})

describe('useCanvasStudio lifecycle', () => {
  it('drops results arriving after clearLocalData and rotates the workspace', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-stale', { apiKeyId: 1 }))
    const pending = deferred<never>()
    pollMock.mockReturnValue(pending.promise)
    const handler = vi.fn()
    const studio = useCanvasStudio()
    studio.setOutputHandler(handler)
    await studio.initialize()
    expect(pollMock).toHaveBeenCalledTimes(1)

    expect(await studio.clearLocalData()).toBe(true)
    pending.resolve(completedTask([{ b64_json: 'aGk=' }]) as never)
    await vi.advanceTimersByTimeAsync(30000)

    expect(studio.history.value).toEqual([])
    expect(handler).not.toHaveBeenCalled()
    expect(pollMock).toHaveBeenCalledTimes(1)
    expect(await listRuns(scope)).toEqual([])
    const next = studio.scope.value!
    expect(next.userId).toBe(1)
    expect(next.workspaceId).not.toBe(scope.workspaceId)
    expect(getWorkspaceId(1)).toBe(next.workspaceId)
  })

  it('publishes nothing when rotation lands mid-download', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-dl', { apiKeyId: 1 }))
    pollMock.mockResolvedValue(
      completedTask([{ url: 'https://cdn.example.com/x.png' }]) as never,
    )
    const download = deferred<Response>()
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(download.promise))
    const handler = vi.fn()
    const studio = useCanvasStudio()
    studio.setOutputHandler(handler)
    await studio.initialize()
    await flush() // harvest is now blocked inside the remote download

    expect(await studio.clearLocalData()).toBe(true)
    download.resolve({ ok: true, blob: async () => new Blob(['late']) } as Response)
    await flush()

    expect(handler).not.toHaveBeenCalled()
    expect(await listAssets(scope)).toEqual([])
    expect(await listAssets(studio.scope.value!)).toEqual([])
  })

  it('publishes nothing when rotation lands while the asset write commits', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-commit', { apiKeyId: 1 }))
    const pending = deferred<never>()
    pollMock.mockReturnValue(pending.promise)
    const handler = vi.fn()
    const studio = useCanvasStudio()
    studio.setOutputHandler(handler)
    await studio.initialize()
    await vi.advanceTimersByTimeAsync(400) // let the processing persist commit

    // Hold the two upcoming commits: the pre-harvest run save, then the asset.
    const releaseRun = fake.holdNextCommit()
    const releaseAsset = fake.holdNextCommit()
    pending.resolve(completedTask([{ b64_json: 'aGk=' }]) as never)
    await flush()
    releaseRun()
    await flush() // saveAsset's transaction is now held mid-commit

    const clearing = studio.clearLocalData()
    await flush()
    releaseAsset()
    expect(await clearing).toBe(true)
    await flush()

    expect(handler).not.toHaveBeenCalled()
    expect(await listAssets(scope)).toEqual([])
    expect(studio.scope.value!.workspaceId).not.toBe(scope.workspaceId)
  })

  it('surfaces a failed wipe instead of pretending success', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-fail', { status: 'completed', apiKeyId: 1 }))
    const studio = useCanvasStudio()
    await studio.initialize()
    await vi.advanceTimersByTimeAsync(400)
    fake.failNextRequest(new DOMException('io', 'UnknownError'))

    expect(await studio.clearLocalData()).toBe(false)
    expect(studio.errorMessage.value).toBe('clear-failed')
    // Rotation still happened; the unwiped records stay in the old workspace.
    expect(studio.scope.value!.workspaceId).not.toBe(scope.workspaceId)
    expect(await loadRun(scope, 'r-fail')).not.toBeNull()
  })

  it('does not publish another user\'s runs after an account switch', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-u1', { apiKeyId: 1 }))
    const studio = useCanvasStudio()
    await studio.initialize()
    expect(studio.history.value.map((run) => run.id)).toEqual(['r-u1'])

    useAuthStore().user = makeUser(2)
    await flush()

    expect(studio.scope.value?.userId).toBe(2)
    expect(studio.history.value).toEqual([])
    await vi.advanceTimersByTimeAsync(30000)
    expect(pollMock).toHaveBeenCalledTimes(1)
    expect(pollMock).toHaveBeenLastCalledWith('sk-1', 'r-u1', expect.anything())
  })

  it('discards a pending key load from the previous account', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-u1', { apiKeyId: 1 }))
    const pendingKeys = deferred<never>()
    listKeysMock
      .mockImplementationOnce(() => pendingKeys.promise)
      .mockResolvedValue(keyPage([makeKey(1), makeKey(2)]))
    const studio = useCanvasStudio()
    const firstInit = studio.initialize()

    // Switch accounts while the first user's key page is still in flight.
    useAuthStore().user = makeUser(2)
    pendingKeys.resolve(keyPage([makeKey(1)]) as never)
    await firstInit
    await flush()

    expect(studio.scope.value?.userId).toBe(2)
    expect(listKeysMock.mock.calls.length).toBeGreaterThanOrEqual(2)
    // The stale single-key page never overwrote the fresh list.
    expect(studio.apiKeys.value).toHaveLength(2)
    expect(studio.history.value).toEqual([])
    expect(pollMock).not.toHaveBeenCalled()
  })

  it('keeps the account watcher alive after the mounting scope is stopped', async () => {
    const viewScope = effectScope()
    const studio = useCanvasStudio()
    await viewScope.run(() => studio.initialize())
    viewScope.stop()

    useAuthStore().user = makeUser(2)
    await flush()

    expect(studio.scope.value?.userId).toBe(2)
    expect(studio.ready.value).toBe(true)
  })

  it('refuses to delete a run that is still processing', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-active', { apiKeyId: 1 }))
    listKeysMock.mockRejectedValue(new Error('keys down')) // leaves it paused
    const studio = useCanvasStudio()
    await studio.initialize()

    await studio.deleteRun('r-active')

    expect(studio.history.value.map((run) => run.id)).toEqual(['r-active'])
    expect(await loadRun(scope, 'r-active')).not.toBeNull()
  })

  it('applies a run\'s frozen params, retaining the valid current model/key', async () => {
    const scope = scopeFor()
    await saveRun(
      scope,
      makeRun('r-params', {
        status: 'completed',
        params: {
          apiKeyId: 2,
          model: 'gone-model',
          prompt: 'p2',
          size: 'bogus',
          quality: 'ultra',
          count: 4,
          operation: 'inpaint',
        },
      }),
    )
    const studio = useCanvasStudio()
    await studio.initialize()

    studio.applyRunParams(studio.history.value[0])

    expect(studio.apiKeyId.value).toBe(2) // still eligible
    expect(studio.model.value).toBe('gpt-image-1') // gone-model not in catalog
    expect(studio.prompt.value).toBe('p2')
    expect(studio.size.value).toBe('1024x1024')
    expect(studio.quality.value).toBe('')
    expect(studio.count.value).toBe(4)
    expect(studio.operation.value).toBe('inpaint')
  })

  it('deletes a terminal run from memory and storage', async () => {
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-del', { status: 'completed' }))
    const studio = useCanvasStudio()
    await studio.initialize()

    await studio.deleteRun('r-del')

    expect(studio.history.value).toEqual([])
    expect(await listRuns(scope)).toEqual([])
  })
})

describe('useCanvasStudio outputs via gateway proxy', () => {
  it('harvests an HTTP output through the gateway with the frozen run key', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(['remote-png'], { type: 'image/png' }),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)
    const scope = scopeFor()
    await saveRun(scope, makeRun('r-http', { apiKeyId: 1 }))
    const studio = useCanvasStudio()
    const gate = deferred<never>()
    pollMock.mockImplementation(() => gate.promise as never)
    await studio.initialize()
    // The panel key switches after the run started; the download must still
    // carry the key the task was submitted with.
    studio.apiKeyId.value = 2
    gate.resolve(completedTask([{ url: 'http://cdn.example.com/out.png' }]) as never)
    await vi.advanceTimersByTimeAsync(3000)
    await flush()

    const content = fetchMock.mock.calls.filter(([url]) =>
      String(url).includes('/v1/images/tasks/r-http/outputs/0/content'),
    )
    expect(content).toHaveLength(1)
    const [, options] = content[0] as [string, RequestInit]
    expect((options.headers as Record<string, string>).Authorization).toBe('Bearer sk-1')
    expect(
      fetchMock.mock.calls.every(([url]) => !String(url).includes('cdn.example.com')),
    ).toBe(true)
    expect(fetchMock.mock.calls.every(([url]) => !String(url).includes('sk-1'))).toBe(true)
    const r = studio.history.value.find((run) => run.id === 'r-http')!
    expect(r.outputs[0].state).toBe('local')
    expect((await loadAsset(scope, 'output:r-http:0'))?.blob.size).toBe(9)
  })

  it('retries a stored remote output through the gateway with the run key', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(['bytes'], { type: 'image/png' }),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)
    const scope = scopeFor()
    await saveRun(
      scope,
      makeRun('r-remote', {
        apiKeyId: 2,
        status: 'completed',
        outputs: [{ index: 1, state: 'remote', remoteUrl: 'http://cdn.example.com/x.png' }],
      }),
    )
    const studio = useCanvasStudio()
    await studio.initialize()
    pollMock.mockResolvedValue(
      completedTask([{ b64_json: 'aGk=' }, { url: 'http://cdn.example.com/x.png' }]) as never,
    )

    const ok = await studio.retryOutputs('r-remote')
    expect(ok).toBe(true)
    const content = fetchMock.mock.calls.filter(([url]) =>
      String(url).includes('/v1/images/tasks/r-remote/outputs/1/content'),
    )
    expect(content).toHaveLength(1)
    const [, options] = content[0] as [string, RequestInit]
    expect((options.headers as Record<string, string>).Authorization).toBe('Bearer sk-2')
    const r = studio.history.value.find((run) => run.id === 'r-remote')!
    expect(r.outputs[1].state).toBe('local')
  })

  it('takes a remote output through the gateway with the run key, and reports key-missing', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      blob: async () => new Blob(['bytes'], { type: 'image/png' }),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)
    const scope = scopeFor()
    await saveRun(
      scope,
      makeRun('r-take', {
        apiKeyId: 2,
        status: 'completed',
        outputs: [{ index: 0, state: 'remote', remoteUrl: 'http://cdn.example.com/x.png' }],
      }),
    )
    const studio = useCanvasStudio()
    await studio.initialize()

    const blob = await studio.takeOutput('r-take', 0)
    expect(blob?.size).toBe(5)
    const content = fetchMock.mock.calls.filter(([url]) =>
      String(url).includes('/v1/images/tasks/r-take/outputs/0/content'),
    )
    expect(content).toHaveLength(1)
    const [, options] = content[0] as [string, RequestInit]
    expect((options.headers as Record<string, string>).Authorization).toBe('Bearer sk-2')

    // Without the run's key the output is reported missing, never fetched with
    // the currently selected key.
    await saveRun(
      scope,
      makeRun('r-nokey', {
        apiKeyId: 9,
        status: 'completed',
        outputs: [{ index: 0, state: 'remote', remoteUrl: 'http://cdn.example.com/y.png' }],
      }),
    )
    const studio2 = useCanvasStudio()
    await studio2.initialize()
    fetchMock.mockClear()
    const missing = await studio2.takeOutput('r-nokey', 0)
    expect(missing).toBeNull()
    expect(studio2.errorMessage.value).toBe('key-missing')
    expect(fetchMock.mock.calls.every(([url]) => !String(url).includes('r-nokey'))).toBe(true)
  })
})
