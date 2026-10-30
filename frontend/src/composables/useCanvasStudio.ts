/**
 * useCanvasStudio.ts owns the canvas-studio data and task layer: key and
 * model catalogs, persisted selection, task submission, independent polling,
 * output harvest, and local history.
 *
 * Module-level state keeps paid work alive across page remounts, while user
 * switches are handled by a detached watcher. Every async continuation
 * re-validates its captured (generation, scope) pair before mutating state,
 * writing, or publishing — so a clearLocalData rotation, a user switch, or a
 * disposed view can never resurrect data or publish into another workspace.
 *
 * Privacy and billing rules: the API key secret is only resolved from the
 * in-memory key list for the lifetime of a request and is never persisted,
 * submissions are never retried, and a run always polls with the key id
 * frozen at submit time.
 */

import { computed, effectScope, ref, watch, type EffectScope } from 'vue'
import { keysAPI } from '@/api/keys'
import { getCatalog, type ModelCatalogItem } from '@/api/modelCatalog'
import {
  extractTaskOutputs,
  fetchTaskOutputBlob,
  getCanvasTask,
  isSafeOutputUrl,
  submitCanvasTask,
  CANVAS_MAX_EDIT_IMAGES,
  type CanvasSubmitResult,
  type CanvasTask,
  type CanvasTaskOutput,
} from '@/api/canvas'
import {
  clearScope,
  deleteRun as deleteStoredRun,
  enforceRetention,
  getWorkspaceId,
  listRuns,
  loadAsset,
  loadSetting,
  normalizeWorkspaceId,
  onWorkspaceChanged,
  rotateWorkspaceId,
  saveAsset,
  saveRun,
  saveSetting,
  type LocalStoreErrorType,
} from '@/utils/canvasLocalStore'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'
import type {
  CanvasOperation,
  CanvasRun,
  CanvasRunOutput,
  CanvasScope,
  CanvasSelection,
} from '@/types/canvas'

/** Base delay between polls of one run. */
const POLL_INTERVAL_MS = 3000
/** Consecutive transient failures before a run pauses for manual resume. */
const MAX_TRANSIENT_FAILURES = 3
/** Ceiling for the transient-failure backoff delay. */
const MAX_BACKOFF_MS = 30000
/** Debounce for persisting the selection into local settings. */
const SELECTION_SAVE_DELAY_MS = 300
/** Settings key for the persisted selection. */
const SELECTION_SETTING_KEY = 'selection'
/** Grok keys accept at most this many edit references. */
const MAX_IMAGES_GROK = 3
/** Bound on in-session blobs kept when storage writes fail. */
const SESSION_BLOB_CACHE_MAX = 32

/** Size tiers the panel offers; anything else normalizes to the first tier. */
const CANVAS_SIZES = ['1024x1024', '1024x1536', '1536x1024'] as const
/** Quality tiers; the empty tier means the gateway default. */
const CANVAS_QUALITIES = ['', 'low', 'medium', 'high'] as const
const CANVAS_MIN_COUNT = 1
const CANVAS_MAX_COUNT = 4
const CANVAS_OPERATIONS: readonly CanvasOperation[] = ['generate', 'edit', 'inpaint']

/**
 * CanvasSubmitInput carries the source bytes prepared by the editor. The
 * current selection refs supply the remaining parameters.
 */
export interface CanvasSubmitInput {
  /** Source images for edit/inpaint; must be empty for generate. */
  images?: Blob[]
  /** Mask bytes for inpaint; must be null otherwise. */
  mask?: Blob | null
}

/**
 * CanvasOutputHandler receives each newly harvested output at most once per
 * (run, index) identity, and only after the returned promise settles is the
 * output marked placed. The editor registers it for auto placement.
 */
export type CanvasOutputHandler = (
  run: CanvasRun,
  output: CanvasRunOutput,
  blob: Blob | null,
) => void | Promise<void>

/**
 * RunTracker is the per-run polling lifecycle: its own timer, in-flight
 * guard, failure counter, and abort controller. The apiKey string is frozen
 * here so later key-list changes or panel switches cannot redirect a run.
 */
interface RunTracker {
  runId: string
  apiKey: string
  generation: number
  scope: CanvasScope
  timer: ReturnType<typeof setTimeout> | null
  inFlight: boolean
  failures: number
  controller: AbortController
}

/**
 * HarvestCtx carries the staleness token and abort signal through a harvest
 * or retry. Every await rechecks it before writing or publishing.
 */
interface HarvestCtx {
  generation: number
  scope: CanvasScope
  signal: AbortSignal
  /** Detaches the ctx's controller from the abort registry when present. */
  release?: () => void
}

type StoreWriteResult<T> =
  | { status: 'ok'; value: T }
  | { status: 'skipped' }
  | { status: 'error'; error: unknown }

const scope = ref<CanvasScope | null>(null)
const ready = ref(false)
const submitting = ref(false)
const history = ref<CanvasRun[]>([])
const errorMessage = ref('')
const storageWarning = ref('')

const apiKeyId = ref<number | null>(null)
const model = ref('')
const prompt = ref('')
const size = ref('1024x1024')
const quality = ref('')
const count = ref(1)
const operation = ref<CanvasOperation>('generate')

const apiKeys = ref<ApiKey[]>([])
const catalogItems = ref<ModelCatalogItem[]>([])
/** ok means the key list was fetched at least once this session. */
const keysLoadState = ref<'idle' | 'ok' | 'error'>('idle')

let initializedUserId: number | null = null
let initPromise: Promise<void> | null = null
let generation = 0
const trackers = new Map<string, RunTracker>()
const placedOutputs = new Set<string>()
const placingOutputs = new Set<string>()
const sessionBlobCache = new Map<string, Blob>()
const harvestControllers = new Set<AbortController>()
let outputHandler: CanvasOutputHandler | null = null
let submitController: AbortController | null = null
let unsubscribeWorkspace: (() => void) | null = null
let internalScope: EffectScope | null = null
let userWatchStop: (() => void) | null = null
let selectionSaveTimer: ReturnType<typeof setTimeout> | null = null
let selectionSaveGeneration = 0
let hydrating = 0
let clearing = false
let lifecycleBound = false
let storeWriteChain: Promise<unknown> = Promise.resolve()

const eligibleKeys = computed(() =>
  apiKeys.value.filter(
    (key) =>
      key.status === 'active' &&
      (key.group?.platform === 'openai' || key.group?.platform === 'grok') &&
      key.group?.allow_image_generation === true,
  ),
)

const selectedGroupId = computed(() => {
  const key = eligibleKeys.value.find((item) => item.id === apiKeyId.value)
  return key?.group?.id ?? key?.group_id ?? null
})

/** imageModels lists the image models the selected key's group exposes. */
const imageModels = computed(() => {
  const names = new Set<string>()
  for (const item of catalogItems.value) {
    if (item.billing_mode !== 'image' || !item.model) continue
    if (selectedGroupId.value !== null) {
      const inBestGroup = item.best_group?.id === selectedGroupId.value
      const inOtherGroups = (item.other_groups || []).some(
        (companion) => companion.group?.id === selectedGroupId.value,
      )
      if (!inBestGroup && !inOtherGroups) continue
    }
    names.add(item.model)
  }
  return [...names].sort()
})

const activeRunCount = computed(
  () => history.value.filter((run) => run.status === 'processing').length,
)

function scopesEqual(a: CanvasScope | null, b: CanvasScope | null): boolean {
  return !!a && !!b && a.userId === b.userId && a.workspaceId === b.workspaceId
}

/**
 * isCurrentGeneration gates every async continuation: a stale generation or
 * scope means the work belonged to a cleared workspace or another user.
 */
function isCurrentGeneration(gen: number, captured: CanvasScope | null): boolean {
  return gen === generation && scopesEqual(captured, scope.value)
}

function isCtxCurrent(ctx: HarvestCtx): boolean {
  return !ctx.signal.aborted && isCurrentGeneration(ctx.generation, ctx.scope)
}

function isTrackerCurrent(tracker: RunTracker): boolean {
  return (
    trackers.get(tracker.runId) === tracker &&
    isCurrentGeneration(tracker.generation, tracker.scope)
  )
}

function isAbortError(error: unknown): boolean {
  return (error as { name?: string })?.name === 'AbortError'
}

/** warnStorage records a visible storage warning without failing work. */
function warnStorage(error: unknown): void {
  const type = (error as { type?: LocalStoreErrorType } | undefined)?.type
  storageWarning.value = type ?? 'unknown'
}

/**
 * enqueueStoreWrite serializes store mutations in the order they were
 * requested and executes each only if its captured generation is still live.
 * That ordering keeps an older snapshot (e.g. processing) from overwriting a
 * newer one (e.g. completed), and the generation guard keeps queued writes
 * from recreating records after a workspace rotation.
 */
function enqueueStoreWrite<T>(
  gen: number,
  scopeSnap: CanvasScope,
  work: () => Promise<T>,
): Promise<StoreWriteResult<T>> {
  const result = storeWriteChain.then(async (): Promise<StoreWriteResult<T>> => {
    if (!isCurrentGeneration(gen, scopeSnap)) return { status: 'skipped' }
    try {
      return { status: 'ok', value: await work() }
    } catch (error) {
      return { status: 'error', error }
    }
  })
  storeWriteChain = result
  return result
}

/**
 * detachRun snapshots a run into plain data synchronously, so a queued write
 * stores the state captured at call time, not whatever the run mutates into
 * while waiting in the write chain.
 */
function detachRun(run: CanvasRun): CanvasRun {
  return {
    id: run.id,
    apiKeyId: run.apiKeyId,
    params: { ...run.params },
    status: run.status,
    createdAt: run.createdAt,
    expiresAt: run.expiresAt,
    paused: run.paused,
    outputs: run.outputs.map((output) => ({ ...output })),
    ...(run.error !== undefined ? { error: run.error } : {}),
  }
}

/**
 * persistRun queues a run snapshot write. Pruned terminal ids reported by the
 * store are dropped from the in-memory history so the terminal cap holds in
 * both places; storage failure warns without failing the run.
 */
function persistRun(run: CanvasRun): void {
  const current = scope.value
  if (!current) return
  const gen = generation
  const scopeSnap = { ...current }
  const snapshot = detachRun(run)
  void enqueueStoreWrite(gen, scopeSnap, () => saveRun(scopeSnap, snapshot)).then((result) => {
    if (result.status === 'error') {
      // Warnings belong to the scope that issued the write, not whatever
      // scope is current when the failure lands.
      if (isCurrentGeneration(gen, scopeSnap)) warnStorage(result.error)
      return
    }
    if (
      result.status === 'ok' &&
      result.value.length > 0 &&
      isCurrentGeneration(gen, scopeSnap)
    ) {
      history.value = history.value.filter(
        (item) => !result.value.includes(item.id),
      )
    }
  })
}

function persistSelection(): void {
  const current = scope.value
  if (!current) return
  const gen = generation
  const scopeSnap = { ...current }
  const selection: CanvasSelection = {
    apiKeyId: apiKeyId.value ?? 0,
    model: model.value,
    prompt: prompt.value,
    size: size.value,
    quality: quality.value,
    count: count.value,
    operation: operation.value,
  }
  void enqueueStoreWrite(gen, scopeSnap, () =>
    saveSetting(scopeSnap, SELECTION_SETTING_KEY, selection),
  ).then((result) => {
    if (result.status === 'error' && isCurrentGeneration(gen, scopeSnap)) {
      warnStorage(result.error)
    }
  })
}

function flushSelectionSave(): void {
  if (selectionSaveTimer !== null) {
    clearTimeout(selectionSaveTimer)
    selectionSaveTimer = null
  }
  persistSelection()
}

/**
 * scheduleSelectionSave debounces selection persistence. The captured
 * generation is rechecked when the timer fires so a save queued before a
 * clear cannot repopulate the rotated workspace.
 */
function scheduleSelectionSave(): void {
  // Hydration suppression keeps defaults from overwriting a stored selection
  // before restoreSelection has applied it.
  if (!scope.value || hydrating > 0) return
  selectionSaveGeneration = generation
  if (selectionSaveTimer !== null) clearTimeout(selectionSaveTimer)
  selectionSaveTimer = setTimeout(() => {
    selectionSaveTimer = null
    if (selectionSaveGeneration !== generation) return
    persistSelection()
  }, SELECTION_SAVE_DELAY_MS)
}

watch([apiKeyId, model, prompt, size, quality, count, operation], scheduleSelectionSave)

watch(imageModels, (models) => {
  if (!models.includes(model.value)) model.value = models[0] ?? ''
})

/** bindLifecycleFlush persists the selection best-effort on unload/hidden. */
function bindLifecycleFlush(): void {
  if (lifecycleBound || typeof window === 'undefined') return
  lifecycleBound = true
  window.addEventListener('pagehide', flushSelectionSave)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') flushSelectionSave()
  })
}

/**
 * runRetention applies the asset budget and reports pruned runs; protected
 * over-budget content only warns instead of breaking the scene.
 */
async function runRetention(current: CanvasScope, gen: number): Promise<void> {
  const result = await enqueueStoreWrite(gen, { ...current }, () =>
    enforceRetention(current),
  )
  if (result.status === 'error') {
    warnStorage(result.error)
    return
  }
  if (result.status !== 'ok' || !isCurrentGeneration(gen, current)) return
  const report = result.value
  if (report.prunedRunIds.length > 0) {
    history.value = history.value.filter(
      (run) => !report.prunedRunIds.includes(run.id),
    )
  }
  if (report.overBudget) {
    storageWarning.value = 'budget'
  }
}

function stopTracker(runId: string): void {
  const tracker = trackers.get(runId)
  if (!tracker) return
  trackers.delete(runId)
  if (tracker.timer !== null) clearTimeout(tracker.timer)
  tracker.controller.abort()
}

/**
 * newHarvestCtx creates a staleness context for a standalone operation
 * (retryOutputs, takeOutput, handler delivery). Its controller is aborted by
 * disposeScopeWork alongside the trackers; release() detaches it when the
 * operation ends.
 */
function newHarvestCtx(current: CanvasScope): HarvestCtx {
  const controller = new AbortController()
  harvestControllers.add(controller)
  return {
    generation,
    scope: { ...current },
    signal: controller.signal,
    release: () => {
      harvestControllers.delete(controller)
    },
  }
}

/**
 * disposeScopeWork aborts every tracker, in-flight harvest, and submit, and
 * bumps the generation so queued writes and late continuations go stale.
 */
function disposeScopeWork(): void {
  generation += 1
  for (const tracker of trackers.values()) {
    if (tracker.timer !== null) clearTimeout(tracker.timer)
    tracker.controller.abort()
  }
  trackers.clear()
  for (const controller of harvestControllers) controller.abort()
  harvestControllers.clear()
  sessionBlobCache.clear()
  submitController?.abort()
  submitController = null
  placedOutputs.clear()
  placingOutputs.clear()
}

function findRun(runId: string): CanvasRun | undefined {
  return history.value.find((run) => run.id === runId)
}

/**
 * classifyPollError separates definitive answers from transient failures.
 * Auth and missing-task answers settle the run; anything else (network,
 * 5xx, rate limiting) only pauses tracking.
 */
function classifyPollError(
  error: unknown,
): 'abort' | 'auth' | 'gone' | 'expired' | 'transient' {
  if (isAbortError(error)) return 'abort'
  const status = (error as { status?: number })?.status ?? 0
  if (status === 401 || status === 403) return 'auth'
  if (status === 410) return 'expired'
  if (status === 404) return 'gone'
  return 'transient'
}

function scheduleTick(tracker: RunTracker, delay: number): void {
  tracker.timer = setTimeout(() => {
    tracker.timer = null
    void tick(tracker)
  }, delay)
}

/**
 * refreshExpiresAt keeps the run's retention deadline in sync with the
 * server, which extends expires_at at completion; an acceptance-time value
 * must not expire a run whose task is still being reported.
 */
function refreshExpiresAt(run: CanvasRun, task: CanvasTask): void {
  if (typeof task.expires_at === 'number' && task.expires_at > 0) {
    run.expiresAt = task.expires_at * 1000
  }
}

/**
 * tick performs one poll. Each run owns its timer independently, so parallel
 * runs neither share a refresh cycle nor block each other. Transient errors
 * back off; three in a row pause the run for manual resume rather than
 * reporting a paid generation as failed.
 */
async function tick(tracker: RunTracker): Promise<void> {
  if (tracker.inFlight || !isTrackerCurrent(tracker)) return
  const run = findRun(tracker.runId)
  if (!run) {
    stopTracker(tracker.runId)
    return
  }
  tracker.inFlight = true
  try {
    const task = await getCanvasTask(tracker.apiKey, run.id, tracker.controller.signal)
    if (!isTrackerCurrent(tracker)) return
    tracker.failures = 0
    refreshExpiresAt(run, task)
    if (task.status === 'completed') {
      await harvestRun(tracker, run, task)
      return
    }
    if (task.status === 'failed') {
      run.status = 'failed'
      run.paused = false
      run.error = task.error?.message || 'task-failed'
      persistRun(run)
      stopTracker(run.id)
      return
    }
    if (run.expiresAt > 0 && Date.now() > run.expiresAt) {
      run.status = 'expired'
      run.paused = false
      persistRun(run)
      stopTracker(run.id)
      return
    }
    scheduleTick(tracker, POLL_INTERVAL_MS)
  } catch (error) {
    if (!isTrackerCurrent(tracker)) return
    const kind = classifyPollError(error)
    if (kind === 'abort') return
    if (kind === 'auth') {
      run.status = 'unavailable'
      run.paused = false
      run.error = 'key-unauthorized'
      persistRun(run)
      stopTracker(run.id)
      return
    }
    if (kind === 'expired' || kind === 'gone') {
      // A 404 past the retention deadline means expired, not just missing.
      const expired = kind === 'expired' || (run.expiresAt > 0 && Date.now() > run.expiresAt)
      run.status = expired ? 'expired' : 'unavailable'
      run.paused = false
      run.error = expired ? 'task-expired' : 'task-missing'
      persistRun(run)
      stopTracker(run.id)
      return
    }
    tracker.failures += 1
    if (tracker.failures >= MAX_TRANSIENT_FAILURES) {
      run.paused = true
      persistRun(run)
      stopTracker(run.id)
      return
    }
    scheduleTick(tracker, Math.min(POLL_INTERVAL_MS * 2 ** tracker.failures, MAX_BACKOFF_MS))
  } finally {
    tracker.inFlight = false
  }
}

/**
 * startPolling registers an independent tracker for a run. The key secret is
 * captured once — polls of this run can never drift onto another key.
 */
function startPolling(run: CanvasRun, apiKey: string): void {
  const current = scope.value
  if (!current || trackers.has(run.id)) return
  const tracker: RunTracker = {
    runId: run.id,
    apiKey,
    generation,
    scope: { ...current },
    timer: null,
    inFlight: false,
    failures: 0,
    controller: new AbortController(),
  }
  trackers.set(run.id, tracker)
  void tick(tracker)
}

/** cacheSessionBlob keeps a blob obtainable this session when storage is full. */
function cacheSessionBlob(identity: string, blob: Blob): void {
  if (!sessionBlobCache.has(identity) && sessionBlobCache.size >= SESSION_BLOB_CACHE_MAX) {
    const oldest = sessionBlobCache.keys().next().value
    if (oldest !== undefined) sessionBlobCache.delete(oldest)
  }
  sessionBlobCache.set(identity, blob)
}

/**
 * invokeOutputHandler delivers a harvested output to the registered placer.
 * An in-flight delivery dedups concurrent callers; the (run, index) identity
 * is marked placed only after the handler's promise resolves, so a failed
 * placement can be delivered again to a later handler.
 */
function invokeOutputHandler(
  ctx: HarvestCtx,
  run: CanvasRun,
  output: CanvasRunOutput,
  blob: Blob,
): void {
  const handler = outputHandler
  const gen = ctx.generation
  const scopeSnap = ctx.scope
  if (!handler || !isCurrentGeneration(ctx.generation, ctx.scope)) return
  const identity = `${run.id}:${output.index}`
  if (placedOutputs.has(identity) || placingOutputs.has(identity)) return
  placingOutputs.add(identity)
  // The .then wrapper catches synchronous handler throws. Placement dedup is
  // gated on generation/scope validity only — a finished poll tracker aborts
  // its own signal and must not un-mark a successful placement.
  void Promise.resolve()
    .then(() => handler(run, output, blob))
    .then(() => {
      placingOutputs.delete(identity)
      if (isCurrentGeneration(gen, scopeSnap) && outputHandler === handler) {
        placedOutputs.add(identity)
      }
    })
    .catch(() => {
      // Failed placement stays unplaced so a retry or remount can deliver it.
      placingOutputs.delete(identity)
    })
}

/**
 * harvestOutput resolves one result entry into an output state. It first
 * checks for bytes persisted by a partial earlier harvest, then fetches with
 * the ctx signal; after every await the ctx is rechecked before any write or
 * publication so a rotation mid-download cannot leak the previous user's data.
 */
async function harvestOutput(
  ctx: HarvestCtx,
  run: CanvasRun,
  item: CanvasTaskOutput,
): Promise<void> {
  if (!isCtxCurrent(ctx)) return
  let output = run.outputs[item.index]
  if (!output) {
    output = { index: item.index, state: 'pending' }
    run.outputs[item.index] = output
  }
  const assetKey = output.assetKey ?? `output:${run.id}:${item.index}`

  // A crash between asset write and run persist leaves durable bytes behind;
  // reuse them instead of re-fetching. A 'local' marker whose asset is gone
  // falls through and becomes retryable instead of silently complete.
  const prior = await loadAsset(ctx.scope, assetKey).catch(() => null)
  if (!isCtxCurrent(ctx)) return
  if (prior) {
    output.state = 'local'
    output.assetKey = assetKey
    output.error = undefined
    invokeOutputHandler(ctx, run, output, prior.blob)
    return
  }

  let blob: Blob | null = null
  try {
    blob = await fetchTaskOutputBlob(item, ctx.signal)
  } catch (error) {
    if (isAbortError(error)) return // rotation/user switch: leave state alone
    blob = null
  }
  if (!isCtxCurrent(ctx)) return

  if (blob) {
    const meta = {
      key: assetKey,
      kind: 'output' as const,
      blob,
      runId: run.id,
      outputIndex: item.index,
      createdAt: Date.now(),
    }
    const written = await enqueueStoreWrite(ctx.generation, ctx.scope, () =>
      saveAsset(ctx.scope, meta),
    )
    if (!isCtxCurrent(ctx)) return
    if (written.status === 'skipped') return
    if (written.status === 'ok') {
      output.state = 'local'
      output.assetKey = assetKey
      output.error = undefined
    } else {
      warnStorage(written.error)
      // Keep the paid bytes reachable this session even though the disk is
      // full; takeOutput and the handler still get the live blob.
      cacheSessionBlob(`${run.id}:${item.index}`, blob)
      if (item.url && isSafeOutputUrl(item.url)) {
        output.state = 'remote'
        output.remoteUrl = item.url
      } else {
        output.state = 'missing'
      }
      output.error = 'storage'
    }
    invokeOutputHandler(ctx, run, output, blob)
    return
  }

  if (item.url && isSafeOutputUrl(item.url)) {
    // CORS or network failure: keep the URL for preview/download instead of
    // pretending the image is importable.
    output.state = 'remote'
    output.remoteUrl = item.url
    output.error = 'unreachable'
    return
  }
  output.state = 'missing'
  output.error = 'no-bytes'
}

/**
 * harvestRun extracts a completed task's outputs and resolves their bytes.
 * Existing output entries are merged by original index — already-local
 * outputs are never reset to pending — and staleness is rechecked inside the
 * loop so a late completion cannot write into a rotated workspace.
 */
async function harvestRun(tracker: RunTracker, run: CanvasRun, task: CanvasTask): Promise<void> {
  const ctx: HarvestCtx = {
    generation: tracker.generation,
    scope: tracker.scope,
    signal: tracker.controller.signal,
  }
  const items = extractTaskOutputs(task)
  const existing = new Map(run.outputs.map((output) => [output.index, output]))
  run.outputs = items.map(
    (item) => existing.get(item.index) ?? { index: item.index, state: 'pending' as const },
  )
  persistRun(run)
  for (const item of items) {
    if (!isCtxCurrent(ctx)) return
    await harvestOutput(ctx, run, item)
  }
  if (!isCtxCurrent(ctx)) return
  run.status = 'completed'
  run.paused = false
  persistRun(run)
  stopTracker(run.id)
  void runRetention(ctx.scope, ctx.generation)
}

/**
 * resumeStoredRun decides whether a stored processing run keeps polling,
 * pauses on missing catalog data, or settles as expired/unavailable. A failed
 * key-list load never proves a key was deleted, so it pauses instead.
 */
function resumeStoredRun(run: CanvasRun): void {
  const current = scope.value
  if (!current || run.status !== 'processing' || trackers.has(run.id)) return
  if (run.expiresAt > 0 && run.expiresAt <= Date.now()) {
    run.status = 'expired'
    run.paused = false
    run.error = 'task-expired'
    persistRun(run)
    return
  }
  const key = apiKeys.value.find((item) => item.id === run.apiKeyId)
  if (keysLoadState.value === 'ok' && !key) {
    run.status = 'unavailable'
    run.paused = false
    run.error = 'key-missing'
    persistRun(run)
    return
  }
  if (key) {
    run.paused = false
    persistRun(run)
    startPolling(run, key.key)
    return
  }
  // Key list not loaded yet: keep the run resumable rather than judging it.
  run.paused = true
  persistRun(run)
}

/**
 * loadApiKeys pages the key list. The captured generation/scope is rechecked
 * after every page so a stale user's keys — including their secrets — can
 * never overwrite a newer session's list.
 */
async function loadApiKeys(gen: number, captured: CanvasScope): Promise<void> {
  try {
    const items: ApiKey[] = []
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, 100, {
        status: 'active',
        sort_by: 'created_at',
        sort_order: 'desc',
      })
      if (!isCurrentGeneration(gen, captured)) return
      items.push(...(response.items || []))
      if (page >= response.pages || (response.items || []).length === 0) break
      page += 1
    }
    apiKeys.value = items
    keysLoadState.value = 'ok'
    if (!eligibleKeys.value.some((key) => key.id === apiKeyId.value)) {
      apiKeyId.value = eligibleKeys.value[0]?.id ?? null
    }
  } catch {
    if (!isCurrentGeneration(gen, captured)) return
    // Keep any previously fetched keys: a stale list still resolves run key
    // secrets, and a load failure must not be read as "key deleted".
    keysLoadState.value = apiKeys.value.length > 0 ? 'ok' : 'error'
  }
}

async function loadCatalog(gen: number, captured: CanvasScope): Promise<void> {
  try {
    const catalog = await getCatalog()
    if (!isCurrentGeneration(gen, captured)) return
    catalogItems.value = catalog.items || []
  } catch {
    // Catalog failure only empties the model list; runs keep tracking.
  }
}

function normalizeSize(value: unknown): string {
  return CANVAS_SIZES.includes(value as (typeof CANVAS_SIZES)[number])
    ? (value as string)
    : CANVAS_SIZES[0]
}

function normalizeQuality(value: unknown): string {
  return CANVAS_QUALITIES.includes(value as (typeof CANVAS_QUALITIES)[number])
    ? (value as string)
    : ''
}

function normalizeCount(value: unknown): number {
  const parsed = Number(value)
  if (Number.isInteger(parsed) && parsed >= CANVAS_MIN_COUNT && parsed <= CANVAS_MAX_COUNT) {
    return parsed
  }
  return CANVAS_MIN_COUNT
}

function normalizeOperation(value: unknown): CanvasOperation {
  return CANVAS_OPERATIONS.includes(value as CanvasOperation)
    ? (value as CanvasOperation)
    : 'generate'
}

/**
 * restoreSelection applies a persisted selection with per-field
 * normalization. A stored key or model that is no longer valid falls back to
 * the current/default selection instead of leaving the panel inconsistent.
 */
async function restoreSelection(current: CanvasScope): Promise<void> {
  const gen = generation
  try {
    const stored = await loadSetting<Partial<CanvasSelection>>(current, SELECTION_SETTING_KEY)
    if (!stored || !isCurrentGeneration(gen, current)) return
    if (
      typeof stored.apiKeyId === 'number' &&
      eligibleKeys.value.some((key) => key.id === stored.apiKeyId)
    ) {
      apiKeyId.value = stored.apiKeyId
    }
    if (typeof stored.prompt === 'string') prompt.value = stored.prompt
    size.value = normalizeSize(stored.size)
    quality.value = normalizeQuality(stored.quality)
    count.value = normalizeCount(stored.count)
    operation.value = normalizeOperation(stored.operation)
    if (typeof stored.model === 'string' && imageModels.value.includes(stored.model)) {
      model.value = stored.model
    } else if (!imageModels.value.includes(model.value)) {
      model.value = imageModels.value[0] ?? ''
    }
  } catch (error) {
    warnStorage(error)
  }
}

async function loadLocalRuns(current: CanvasScope): Promise<void> {
  const gen = generation
  try {
    const runs = await listRuns(current)
    if (!isCurrentGeneration(gen, current)) return
    history.value = runs.sort((a, b) => b.createdAt - a.createdAt)
  } catch (error) {
    warnStorage(error)
  }
}

/**
 * storedWorkspaceId reads the persisted workspace UUID without generating
 * one, used to detect external rotation.
 */
function storedWorkspaceId(userId: number): string | null {
  try {
    return normalizeWorkspaceId(
      window.localStorage.getItem(`canvas:workspace:${userId}`),
    )
  } catch {
    return null
  }
}

/**
 * adoptWorkspace drops stale work and rebinds to a workspace id after an
 * external rotation (or invalidation). The new partition starts empty.
 */
function adoptWorkspace(userId: number, workspaceId: string): void {
  disposeScopeWork()
  history.value = []
  scope.value = { userId, workspaceId }
  const gen = generation
  const current = scope.value
  void loadLocalRuns(current).then(() => {
    if (isCurrentGeneration(gen, current)) {
      for (const run of history.value) resumeStoredRun(run)
      if (outputHandler) void deliverPendingOutputs(outputHandler)
    }
  })
}

/**
 * onExternalRotation reacts to a workspace-id change in another tab. A null
 * or invalid value still invalidates this tab's partition and adopts a fresh
 * workspace rather than clinging to stale data.
 */
function onExternalRotation(userId: number, workspaceId: string | null): void {
  if (!scope.value || scope.value.userId !== userId) return
  let next: string | null
  if (workspaceId) {
    next = workspaceId
  } else {
    // The key was removed or corrupted in another tab: adopt whatever valid
    // workspace is stored now, or generate a fresh one.
    next = storedWorkspaceId(userId) ?? getWorkspaceId(userId)
  }
  if (!next || next === scope.value.workspaceId) return
  adoptWorkspace(userId, next)
}

/**
 * teardownAll stops all work and resets session state — including the
 * selection refs, so one user's prompt/parameters can never be persisted
 * into another user's settings. Persisted data is untouched.
 */
function teardownAll(): void {
  disposeScopeWork()
  unsubscribeWorkspace?.()
  unsubscribeWorkspace = null
  if (selectionSaveTimer !== null) {
    clearTimeout(selectionSaveTimer)
    selectionSaveTimer = null
  }
  outputHandler = null
  sessionBlobCache.clear()
  history.value = []
  scope.value = null
  ready.value = false
  submitting.value = false
  errorMessage.value = ''
  storageWarning.value = ''
  apiKeys.value = []
  catalogItems.value = []
  keysLoadState.value = 'idle'
  apiKeyId.value = null
  model.value = ''
  prompt.value = ''
  size.value = CANVAS_SIZES[0]
  quality.value = ''
  count.value = 1
  operation.value = 'generate'
  initializedUserId = null
  initPromise = null
}

/**
 * ensureUserWatch registers the account-switch watcher inside a detached
 * effect scope. The composable's work deliberately survives view unmounts, so
 * the watcher must not be bound to whichever component called initialize
 * first — a normal watch created there would be stopped with that component.
 */
function ensureUserWatch(): void {
  if (userWatchStop) return
  internalScope ??= effectScope(true)
  const authStore = useAuthStore()
  internalScope.run(() => {
    userWatchStop = watch(
      () => authStore.user?.id ?? null,
      (next, prev) => {
        if (!prev || next === prev) return
        // A different account must never see the previous user's runs or let
        // their callbacks publish into the new session.
        teardownAll()
        void initialize()
      },
    )
  })
}

/**
 * initialize binds the composable to the current user: resolves the per-user
 * workspace UUID, loads catalogs and stored runs, restores the selection, and
 * resumes polling of unfinished runs with their original keys. Calling it
 * again for an already-ready same user and workspace is a no-op so remounts
 * do not replace live history.
 */
async function initialize(): Promise<void> {
  const authStore = useAuthStore()
  const userId = authStore.user?.id ?? null
  if (!userId) {
    teardownAll()
    errorMessage.value = 'not-authenticated'
    return
  }
  if (initializedUserId === userId && initPromise) {
    return initPromise
  }
  if (initializedUserId !== null && initializedUserId !== userId) {
    teardownAll()
  }
  if (
    initializedUserId === userId &&
    ready.value &&
    scope.value &&
    scope.value.workspaceId === storedWorkspaceId(userId)
  ) {
    return
  }
  const job = (async () => {
    if (
      initializedUserId === userId &&
      scope.value &&
      scope.value.workspaceId !== storedWorkspaceId(userId)
    ) {
      // External rotation happened before our storage listener ran.
      adoptWorkspace(userId, storedWorkspaceId(userId) ?? getWorkspaceId(userId))
      return
    }
    initializedUserId = userId
    ready.value = false
    let workspaceId: string
    try {
      workspaceId = getWorkspaceId(userId)
    } catch (error) {
      warnStorage(error)
      errorMessage.value = 'storage-unavailable'
      return
    }
    scope.value = { userId, workspaceId }
    const gen = generation
    const current = scope.value
    ensureUserWatch()
    bindLifecycleFlush()
    unsubscribeWorkspace?.()
    unsubscribeWorkspace = onWorkspaceChanged(userId, (next) =>
      onExternalRotation(userId, next),
    )
    // Gate the selection watcher until restoreSelection has applied stored
    // values; otherwise defaults would persist over the stored selection.
    hydrating += 1
    try {
      await Promise.all([
        loadApiKeys(gen, current),
        loadCatalog(gen, current),
        loadLocalRuns(current),
      ])
      if (!isCurrentGeneration(gen, current)) return
      await restoreSelection(current)
    } finally {
      hydrating -= 1
    }
    if (!isCurrentGeneration(gen, current)) return
    for (const run of history.value) {
      resumeStoredRun(run)
    }
    // Completed runs whose outputs never reached local storage get one free
    // re-harvest attempt on this mount — a GET, never a billed re-submit.
    for (const run of history.value) {
      if (
        run.status === 'completed' &&
        (run.outputs.length === 0 || run.outputs.some((o) => o.state !== 'local'))
      ) {
        void retryOutputs(run.id)
      }
    }
    void runRetention(current, gen)
    ready.value = true
    // A handler registered while initialization was in flight still gets its
    // pending local outputs.
    if (outputHandler) void deliverPendingOutputs(outputHandler)
  })()
  const tracked = job.finally(() => {
    if (initPromise === tracked) initPromise = null
  })
  initPromise = tracked
  return tracked
}

/**
 * validateSubmit enforces the panel-level contract: a resolvable model from
 * the loaded catalog, a non-empty prompt, and per-platform image limits.
 * API-side validation stays generic; the Grok cap lives here.
 */
function validateSubmit(
  key: ApiKey,
  images: Blob[],
  mask: Blob | null,
): string | null {
  if (imageModels.value.length === 0) return 'no-models'
  if (!imageModels.value.includes(model.value)) return 'invalid-model'
  if (prompt.value.trim() === '') return 'empty-prompt'
  const maxImages = key.group?.platform === 'grok' ? MAX_IMAGES_GROK : CANVAS_MAX_EDIT_IMAGES
  switch (operation.value) {
    case 'generate':
      if (images.length > 0 || mask) return 'invalid-input'
      break
    case 'edit':
      if (images.length === 0) return 'missing-source'
      if (images.length > maxImages) return 'too-many-images'
      if (mask) return 'invalid-input'
      break
    case 'inpaint':
      if (images.length !== 1 || !mask) return 'invalid-input'
      break
    default:
      return 'invalid-input'
  }
  if (images.some((image) => image.size === 0)) return 'empty-source'
  if (mask && mask.size === 0) return 'empty-source'
  return null
}

/**
 * submit freezes the current selection and key onto a new run. The submitting
 * lock covers only the POST — execution tracking is per-run — and the run is
 * persisted as soon as the server accepts it so a refresh cannot lose it.
 */
async function submit(input: CanvasSubmitInput = {}): Promise<CanvasRun | null> {
  errorMessage.value = ''
  const current = scope.value
  const gen = generation
  if (!current || !ready.value || clearing) {
    errorMessage.value = 'not-ready'
    return null
  }
  if (submitting.value) {
    errorMessage.value = 'busy'
    return null
  }
  const key = eligibleKeys.value.find((item) => item.id === apiKeyId.value)
  if (!key) {
    errorMessage.value = 'no-api-key'
    return null
  }
  const images = input.images ?? []
  const mask = input.mask ?? null
  const invalid = validateSubmit(key, images, mask)
  if (invalid) {
    errorMessage.value = invalid
    return null
  }
  const frozen: CanvasSelection = {
    apiKeyId: key.id,
    model: model.value,
    prompt: prompt.value,
    size: normalizeSize(size.value),
    quality: normalizeQuality(quality.value),
    count: normalizeCount(count.value),
    operation: operation.value,
  }
  submitController?.abort()
  submitController = new AbortController()
  const controller = submitController
  submitting.value = true
  try {
    const result: CanvasSubmitResult = await submitCanvasTask(
      key.key,
      {
        operation: frozen.operation,
        images,
        mask,
        model: frozen.model,
        prompt: frozen.prompt,
        size: frozen.size,
        count: frozen.count,
        quality: frozen.quality || undefined,
      },
      controller.signal,
    )
    if (!isCurrentGeneration(gen, current)) return null
    const run: CanvasRun = {
      id: result.id,
      apiKeyId: frozen.apiKeyId,
      params: frozen,
      status: 'processing',
      createdAt: (result.created_at || 0) * 1000 || Date.now(),
      expiresAt: (result.expires_at || 0) * 1000,
      paused: false,
      outputs: [],
    }
    history.value = [run, ...history.value]
    persistRun(run)
    startPolling(run, key.key)
    return run
  } catch (error) {
    if (!isAbortError(error) && isCurrentGeneration(gen, current)) {
      errorMessage.value = error instanceof Error ? error.message : String(error)
    }
    return null
  } finally {
    if (isCurrentGeneration(gen, current)) submitting.value = false
  }
}

/**
 * resumeRun restarts tracking of a paused or restored processing run with its
 * frozen key id. The current panel selection is never substituted.
 */
async function resumeRun(runId: string): Promise<void> {
  const run = findRun(runId)
  const current = scope.value
  const gen = generation
  if (!run || !current || run.status !== 'processing') return
  if (keysLoadState.value !== 'ok') {
    await loadApiKeys(gen, current)
    if (!isCurrentGeneration(gen, current)) return
  }
  resumeStoredRun(run)
}

/**
 * retryOutputs re-harvests outputs of a completed run that never reached
 * local storage, including records that have no output metadata at all. The
 * task is re-fetched (results are not persisted in full); only non-local
 * outputs are retried so placed outputs never duplicate.
 */
async function retryOutputs(runId: string): Promise<boolean> {
  const run = findRun(runId)
  const current = scope.value
  if (!run || !current || run.status !== 'completed') return false
  const gen = generation
  const ctx = newHarvestCtx(current)
  try {
    const pending = run.outputs.filter((output) => output.state !== 'local')
    if (pending.length === 0 && run.outputs.length > 0) return true
    if (keysLoadState.value !== 'ok') {
      await loadApiKeys(gen, current)
      if (!isCtxCurrent(ctx)) return false
    }
    const key = apiKeys.value.find((item) => item.id === run.apiKeyId)
    if (!key) {
      errorMessage.value = 'key-missing'
      return false
    }
    const task = await getCanvasTask(key.key, run.id, ctx.signal)
    if (!isCtxCurrent(ctx)) return false
    refreshExpiresAt(run, task)
    const items = extractTaskOutputs(task)
    const existing = new Map(run.outputs.map((output) => [output.index, output]))
    run.outputs = items.map(
      (item) => existing.get(item.index) ?? { index: item.index, state: 'pending' as const },
    )
    for (const item of items) {
      if (!isCtxCurrent(ctx)) return false
      await harvestOutput(ctx, run, item)
    }
    persistRun(run)
    return true
  } catch (error) {
    if (!isCtxCurrent(ctx)) return false
    const kind = classifyPollError(error)
    if (kind === 'expired' || kind === 'gone') {
      run.status = 'expired'
      run.error = 'task-expired'
      persistRun(run)
    } else if (kind === 'auth') {
      run.error = 'key-unauthorized'
      persistRun(run)
    } else if (kind !== 'abort') {
      errorMessage.value = error instanceof Error ? error.message : String(error)
    }
    return false
  } finally {
    ctx.release?.()
  }
}

/**
 * takeOutput resolves one output's bytes for manual placement or download.
 * Lookup order is session cache (storage-failed harvests), local asset, then
 * remote URL; a remote blob that finally downloads is promoted to a local
 * asset. Returns null once the calling context has gone stale.
 */
async function takeOutput(runId: string, index: number): Promise<Blob | null> {
  const run = findRun(runId)
  const current = scope.value
  if (!run || !current) return null
  const output = run.outputs.find((item) => item.index === index)
  if (!output) return null
  const ctx = newHarvestCtx(current)
  const identity = `${runId}:${index}`
  try {
    const cached = sessionBlobCache.get(identity)
    if (cached) return cached
    if (output.state === 'local' && output.assetKey) {
      const asset = await loadAsset(current, output.assetKey).catch(() => null)
      if (!isCtxCurrent(ctx)) return null
      if (asset) return asset.blob
    }
    if (output.state === 'remote' && output.remoteUrl) {
      let blob: Blob | null = null
      try {
        blob = await fetchTaskOutputBlob({ url: output.remoteUrl }, ctx.signal)
      } catch {
        return null
      }
      if (!blob || !isCtxCurrent(ctx)) return null
      const assetKey = output.assetKey ?? `output:${run.id}:${index}`
      const written = await enqueueStoreWrite(ctx.generation, ctx.scope, () =>
        saveAsset(ctx.scope, {
          key: assetKey,
          kind: 'output',
          blob,
          runId: run.id,
          outputIndex: index,
          createdAt: Date.now(),
        }),
      )
      if (!isCtxCurrent(ctx)) return null
      if (written.status === 'ok') {
        output.state = 'local'
        output.assetKey = assetKey
        output.error = undefined
        persistRun(run)
      } else if (written.status === 'error') {
        warnStorage(written.error)
        cacheSessionBlob(identity, blob)
      }
      return blob
    }
    return null
  } finally {
    ctx.release?.()
  }
}

/**
 * applyRunParams restores a run's frozen parameters into the selection refs.
 * Fields that are no longer valid (deleted key, model absent from the
 * catalog) keep the current valid selection instead of going stale.
 */
function applyRunParams(run: CanvasRun): void {
  const params = run.params
  if (!params) return
  if (eligibleKeys.value.some((key) => key.id === params.apiKeyId)) {
    apiKeyId.value = params.apiKeyId
  }
  if (imageModels.value.includes(params.model)) {
    model.value = params.model
  }
  prompt.value = params.prompt ?? prompt.value
  size.value = normalizeSize(params.size)
  quality.value = normalizeQuality(params.quality)
  count.value = normalizeCount(params.count)
  operation.value = normalizeOperation(params.operation)
}

/**
 * deleteRun removes a terminal run from local history; processing runs are
 * refused because deleting tracked paid work would orphan its outputs.
 * Retention keeps scene-referenced assets afterwards.
 */
async function deleteRun(runId: string): Promise<void> {
  const run = findRun(runId)
  if (!run || run.status === 'processing') return
  const current = scope.value
  const gen = generation
  stopTracker(runId)
  history.value = history.value.filter((item) => item.id !== runId)
  if (!current) return
  const result = await enqueueStoreWrite(gen, { ...current }, () =>
    deleteStoredRun(current, runId),
  )
  if (result.status === 'error') warnStorage(result.error)
  if (isCurrentGeneration(gen, current)) {
    void runRetention(current, gen)
  }
}

/**
 * clearLocalData wipes the current scope's records and rotates the workspace
 * UUID. Rotation happens before the wipe so any queued write goes stale via
 * the generation guard instead of recreating cleared records; the clear
 * itself waits for the write chain to drain first. Other tabs holding the old
 * id are invalidated through the storage event. Returns false when a step
 * failed — confirmation and success UI belong to the caller.
 */
async function clearLocalData(): Promise<boolean> {
  const oldScope = scope.value ? { ...scope.value } : null
  clearing = true
  if (selectionSaveTimer !== null) {
    clearTimeout(selectionSaveTimer)
    selectionSaveTimer = null
  }
  try {
    return await clearLocalDataInner(oldScope)
  } finally {
    clearing = false
  }
}

async function clearLocalDataInner(oldScope: CanvasScope | null): Promise<boolean> {
  disposeScopeWork()
  submitting.value = false
  history.value = []
  errorMessage.value = ''
  sessionBlobCache.clear()
  if (!oldScope) return true
  let ok = true
  try {
    const next = rotateWorkspaceId(oldScope.userId)
    scope.value = { userId: oldScope.userId, workspaceId: next }
  } catch (error) {
    warnStorage(error)
    ok = false
  }
  // Drain queued writes: stale ones skip, in-flight ones settle before the
  // wipe so nothing recreates records afterwards.
  await storeWriteChain.catch(() => {})
  try {
    await clearScope(oldScope)
  } catch (error) {
    warnStorage(error)
    ok = false
  }
  if (!ok) errorMessage.value = 'clear-failed'
  return ok
}

/**
 * deliverPendingOutputs offers already-local, never-placed outputs to a newly
 * registered handler — e.g. a remounted editor after harvest completed while
 * no view was attached.
 */
async function deliverPendingOutputs(handler: CanvasOutputHandler): Promise<void> {
  const current = scope.value
  if (!current) return
  const ctx = newHarvestCtx(current)
  try {
    for (const run of history.value) {
      if (outputHandler !== handler || !isCtxCurrent(ctx)) return
      if (run.status !== 'completed') continue
      for (const output of run.outputs) {
        if (outputHandler !== handler || !isCtxCurrent(ctx)) return
        if (output.state !== 'local' || !output.assetKey) continue
        const identity = `${run.id}:${output.index}`
        // Outputs the editor already confirmed on the scene are delivered.
        if (output.autoPlaced) continue
        if (placedOutputs.has(identity) || placingOutputs.has(identity)) continue
        const blob =
          sessionBlobCache.get(identity) ??
          (await loadAsset(current, output.assetKey).catch(() => null))?.blob
        if (!blob) continue
        invokeOutputHandler(ctx, run, output, blob)
      }
    }
  } finally {
    ctx.release?.()
  }
}

/**
 * markOutputPlaced records durable placement of one output — called by the
 * view after the editor confirmed the image is in the scene and flushed.
 * The flag is persisted so remounts never re-place outputs the user deleted.
 */
function markOutputPlaced(runId: string, index: number): void {
  const run = findRun(runId)
  const output = run?.outputs.find((item) => item.index === index)
  if (!run || !output || output.autoPlaced) return
  output.autoPlaced = true
  placedOutputs.add(`${runId}:${index}`)
  persistRun(run)
}

/**
 * setOutputHandler registers the auto-placement callback and returns an
 * unregister function. A disposed view unregisters so late completions keep
 * harvesting to storage but never publish onto a dead editor.
 */
function setOutputHandler(handler: CanvasOutputHandler | null): () => void {
  outputHandler = handler
  if (handler) void deliverPendingOutputs(handler)
  return () => {
    if (outputHandler === handler) outputHandler = null
  }
}

/** __resetCanvasStudioForTest restores module state between isolated tests. */
export function __resetCanvasStudioForTest(): void {
  teardownAll()
  userWatchStop?.()
  userWatchStop = null
  internalScope?.stop()
  internalScope = null
}

export function useCanvasStudio() {
  return {
    scope,
    ready,
    submitting,
    history,
    errorMessage,
    storageWarning,
    activeRunCount,
    apiKeyId,
    model,
    prompt,
    size,
    quality,
    count,
    operation,
    apiKeys,
    eligibleKeys,
    imageModels,
    keysLoadState,
    initialize,
    submit,
    resumeRun,
    retryOutputs,
    applyRunParams,
    deleteRun,
    clearLocalData,
    takeOutput,
    setOutputHandler,
    markOutputPlaced,
  }
}
