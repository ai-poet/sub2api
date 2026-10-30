/**
 * canvasLocalStore.ts persists canvas-studio data in IndexedDB: scene
 * snapshots, image asset bytes, submitted runs, and scoped settings.
 *
 * Database version 2 keeps the v1 scenes/assets stores untouched and adds
 * runs/settings stores. Every record written by this version is scoped: the
 * storage key carries a `u<userId>|w<workspaceId>|` prefix and the record
 * carries the scope itself. Scoped key-range queries therefore never see v1
 * records or another scope's records, and unowned v1 data is left intact and
 * unclaimed rather than attributed to the current user. Note that v1 scenes
 * referenced images through blob: object URLs, which do not survive reloads;
 * those records stay on disk but cannot restore their pictures.
 *
 * Writes await transaction completion, not just request success, so a
 * committed write is durable when its promise resolves and an aborted
 * transaction reports failure instead of silently losing records.
 */

import type {
  CanvasAsset,
  CanvasAssetKind,
  CanvasRun,
  CanvasSceneRecord,
  CanvasScope,
} from '@/types/canvas'

/** Database name shared with the v1 canvas store; data is upgraded in place. */
const DB_NAME = 'sub2api-canvas-studio'
/** Version 2 adds the runs and settings object stores. */
const DB_VERSION = 2

const STORE_SCENES = 'scenes'
const STORE_ASSETS = 'assets'
const STORE_RUNS = 'runs'
const STORE_SETTINGS = 'settings'

/** Terminal run history kept per scope. */
export const MAX_TERMINAL_RUNS = 200
/** Soft byte budget for evictable image assets per scope. */
export const ASSET_BYTE_BUDGET = 256 * 1024 * 1024

/**
 * LocalStoreErrorType selects quota, unavailable-storage, or unknown recovery
 * handling at the UI layer.
 */
export type LocalStoreErrorType = 'quota' | 'unavailable' | 'unknown'

/** LocalStoreError carries a classified browser storage failure. */
export class LocalStoreError extends Error {
  readonly type: LocalStoreErrorType

  constructor(type: LocalStoreErrorType, message: string, cause?: unknown) {
    super(message)
    this.name = 'LocalStoreError'
    this.type = type
    if (cause !== undefined) {
      Object.assign(this, { cause })
    }
  }
}

/** LocalStoreQuotaError marks exhausted browser storage for warning UI. */
export class LocalStoreQuotaError extends LocalStoreError {
  constructor(message = 'Local storage quota exceeded', cause?: unknown) {
    super('quota', message, cause)
    this.name = 'LocalStoreQuotaError'
  }
}

/**
 * toLocalStoreError maps browser errors onto the storage categories. Code 22
 * is the legacy quota error number seen on older engines.
 */
function toLocalStoreError(error: unknown): LocalStoreError {
  if (error instanceof LocalStoreError) return error
  const name = (error as DOMException | null)?.name || ''
  if (name === 'QuotaExceededError' || (error as DOMException)?.code === 22) {
    return new LocalStoreQuotaError('Local storage quota exceeded', error)
  }
  if (
    name === 'InvalidStateError' ||
    name === 'SecurityError' ||
    name === 'NotSupportedError' ||
    name === 'UnknownError'
  ) {
    return new LocalStoreError('unavailable', 'Local storage unavailable', error)
  }
  const message = error instanceof Error ? error.message : String(error)
  return new LocalStoreError('unknown', message || 'Local storage operation failed', error)
}

/**
 * scopePrefix is the storage-key prefix isolating one (user, workspace)
 * partition inside each object store.
 */
function scopePrefix(scope: CanvasScope): string {
  return `u${scope.userId}|w${scope.workspaceId}|`
}

/** scopedKey builds the storage key for a logical record key. */
function scopedKey(scope: CanvasScope, logicalKey: string): string {
  return `${scopePrefix(scope)}${logicalKey}`
}

/**
 * scopeRange selects exactly the storage keys of one scope, so list queries
 * never touch v1 records or other users' data.
 */
function scopeRange(scope: CanvasScope): IDBKeyRange {
  const prefix = scopePrefix(scope)
  return IDBKeyRange.bound(prefix, `${prefix}\uffff`)
}

/**
 * matchesScope verifies a stored record really belongs to the requested scope,
 * defending reads against records whose key prefix was forged or migrated.
 */
function matchesScope(recordScope: CanvasScope | undefined, scope: CanvasScope): boolean {
  return (
    !!recordScope &&
    recordScope.userId === scope.userId &&
    recordScope.workspaceId === scope.workspaceId
  )
}

/**
 * detachPlain deep-copies plain objects and arrays into fresh values. Reading
 * through a Vue reactive proxy yields the underlying values, so this both
 * produces a write-time snapshot (later mutations cannot leak into storage)
 * and strips proxies that structured cloning would reject. Blobs, class
 * instances, and primitives pass through untouched; functions and other
 * non-cloneable leaves are the caller's responsibility.
 */
function detachPlain<T>(value: T): T {
  if (value === null || typeof value !== 'object') return value
  if (Array.isArray(value)) {
    return value.map((item) => detachPlain(item)) as T
  }
  const proto = Object.getPrototypeOf(value)
  if (proto !== Object.prototype && proto !== null) return value
  const copy: Record<string, unknown> = {}
  for (const [key, item] of Object.entries(value)) {
    copy[key] = detachPlain(item)
  }
  return copy as T
}

/**
 * detachScope snapshots the two scope primitives so a later scope mutation
 * cannot redirect a write already in flight.
 */
function detachScope(scope: CanvasScope): CanvasScope {
  return { userId: scope.userId, workspaceId: scope.workspaceId }
}

/** Shared open promise so concurrent callers reuse one connection. */
let dbPromise: Promise<IDBDatabase> | null = null

/**
 * openDatabase opens or upgrades the store. The v2 upgrade creates the runs
 * and settings stores when missing; existing v1 records are never rewritten,
 * so nothing unowned is claimed on upgrade.
 */
function openDatabase(): Promise<IDBDatabase> {
  return new Promise<IDBDatabase>((resolve, reject) => {
    if (typeof indexedDB === 'undefined') {
      reject(new LocalStoreError('unavailable', 'IndexedDB is not supported in this environment'))
      return
    }
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE_SCENES)) {
        db.createObjectStore(STORE_SCENES, { keyPath: 'key' })
      }
      if (!db.objectStoreNames.contains(STORE_ASSETS)) {
        db.createObjectStore(STORE_ASSETS, { keyPath: 'key' })
      }
      if (!db.objectStoreNames.contains(STORE_RUNS)) {
        db.createObjectStore(STORE_RUNS, { keyPath: 'key' })
      }
      if (!db.objectStoreNames.contains(STORE_SETTINGS)) {
        db.createObjectStore(STORE_SETTINGS, { keyPath: 'key' })
      }
    }
    request.onsuccess = () => {
      const db = request.result
      // Another tab may upgrade the database and close this connection; drop
      // the cached promise so the next call reopens instead of reusing it.
      db.onversionchange = () => {
        db.close()
        dbPromise = null
      }
      resolve(db)
    }
    request.onerror = () => {
      dbPromise = null
      reject(toLocalStoreError(request.error))
    }
    request.onblocked = () => {
      dbPromise = null
      reject(new LocalStoreError('unavailable', 'Local storage is blocked by another session'))
    }
  })
}

/**
 * database returns the shared connection. A failed open clears the cache so a
 * later call retries instead of pinning a rejected promise.
 */
async function database(): Promise<IDBDatabase> {
  if (!dbPromise) {
    dbPromise = openDatabase()
  }
  return dbPromise
}

/** idbRequest adapts a single request to a promise with classified errors. */
function idbRequest<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(toLocalStoreError(request.error))
  })
}

/**
 * txFinished resolves when a transaction commits. Writes must await this:
 * request success alone does not prove durability.
 */
function txFinished(tx: IDBTransaction): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(toLocalStoreError(tx.error))
    tx.onabort = () =>
      reject(toLocalStoreError(tx.error ?? new Error('transaction aborted')))
  })
}

/**
 * transact runs work inside one readwrite transaction and resolves only after
 * commit. If work rejects, the transaction is aborted so partial writes roll
 * back before the classified error propagates.
 */
async function transact<T>(
  storeNames: string[],
  work: (tx: IDBTransaction) => Promise<T>,
): Promise<T> {
  const db = await database()
  const tx = db.transaction(storeNames, 'readwrite')
  const finished = txFinished(tx)
  try {
    const result = await work(tx)
    await finished
    return result
  } catch (error) {
    try {
      tx.abort()
    } catch {
      // The transaction already completed or aborted on its own.
    }
    try {
      await finished
    } catch {
      // The abort already reported the failure; keep the original error.
    }
    throw toLocalStoreError(error)
  }
}

/**
 * readOne runs a single readonly request. Reads resolve on request success
 * because a failed read has nothing to roll back.
 */
async function readOne<T>(
  storeName: string,
  make: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T> {
  const db = await database()
  try {
    return await idbRequest(make(db.transaction(storeName, 'readonly').objectStore(storeName)))
  } catch (error) {
    throw toLocalStoreError(error)
  }
}

/**
 * SceneRecord is the stored shape of a scene snapshot. The storage key embeds
 * the scope prefix while name stays the logical scene identifier.
 */
interface SceneRecord extends CanvasSceneRecord {
  key: string
  scope: CanvasScope
}

/**
 * saveScene writes one scoped scene snapshot. Same-name writes overwrite, and
 * the promise resolves only after the transaction commits.
 */
export async function saveScene(
  scope: CanvasScope,
  name: string,
  json: string,
): Promise<void> {
  const record: SceneRecord = {
    key: scopedKey(scope, name),
    scope: detachScope(scope),
    name,
    json,
    updatedAt: Date.now(),
  }
  await transact([STORE_SCENES], async (tx) => {
    await idbRequest(tx.objectStore(STORE_SCENES).put(record))
  })
}

/** loadScene reads one scoped scene snapshot; absent or foreign data is null. */
export async function loadScene(
  scope: CanvasScope,
  name: string,
): Promise<CanvasSceneRecord | null> {
  const record = await readOne(
    STORE_SCENES,
    (store) => store.get(scopedKey(scope, name)) as IDBRequest<SceneRecord | undefined>,
  )
  if (!record || !matchesScope(record.scope, scope)) return null
  return { name: record.name, json: record.json, updatedAt: record.updatedAt }
}

/** listScenes returns a scope's snapshots, newest first. */
export async function listScenes(scope: CanvasScope): Promise<CanvasSceneRecord[]> {
  const records = await readOne(
    STORE_SCENES,
    (store) => store.getAll(scopeRange(scope)) as IDBRequest<SceneRecord[]>,
  )
  return records
    .filter((record) => matchesScope(record.scope, scope))
    .map((record) => ({ name: record.name, json: record.json, updatedAt: record.updatedAt }))
    .sort((a, b) => b.updatedAt - a.updatedAt)
}

/** deleteScene removes one scoped snapshot. */
export async function deleteScene(scope: CanvasScope, name: string): Promise<void> {
  await transact([STORE_SCENES], async (tx) => {
    await idbRequest(tx.objectStore(STORE_SCENES).delete(scopedKey(scope, name)))
  })
}

/**
 * AssetRecord is the stored asset shape: the storage key holds the scope
 * prefix while id keeps the logical key the domain uses in asset:// refs.
 */
interface AssetRecord {
  key: string
  scope: CanvasScope
  id: string
  kind: CanvasAssetKind
  blob: Blob
  runId?: string
  outputIndex?: number
  createdAt: number
}

function toAsset(record: AssetRecord): CanvasAsset {
  return {
    key: record.id,
    kind: record.kind,
    blob: record.blob,
    runId: record.runId,
    outputIndex: record.outputIndex,
    createdAt: record.createdAt,
  }
}

/**
 * saveAsset writes image bytes under the asset's logical key inside the scope.
 * Repeating a key overwrites; the asset's bytes become durable on resolve.
 */
export async function saveAsset(scope: CanvasScope, asset: CanvasAsset): Promise<void> {
  const record: AssetRecord = {
    key: scopedKey(scope, asset.key),
    scope: detachScope(scope),
    id: asset.key,
    kind: asset.kind,
    blob: asset.blob,
    runId: asset.runId,
    outputIndex: asset.outputIndex,
    createdAt: asset.createdAt,
  }
  await transact([STORE_ASSETS], async (tx) => {
    await idbRequest(tx.objectStore(STORE_ASSETS).put(record))
  })
}

/** loadAsset reads one asset by logical key; foreign or absent keys are null. */
export async function loadAsset(scope: CanvasScope, key: string): Promise<CanvasAsset | null> {
  const record = await readOne(
    STORE_ASSETS,
    (store) => store.get(scopedKey(scope, key)) as IDBRequest<AssetRecord | undefined>,
  )
  if (!record || !matchesScope(record.scope, scope)) return null
  return toAsset(record)
}

/** listAssets returns a scope's assets, optionally filtered by kind. */
export async function listAssets(
  scope: CanvasScope,
  kind?: CanvasAssetKind,
): Promise<CanvasAsset[]> {
  const records = await readOne(
    STORE_ASSETS,
    (store) => store.getAll(scopeRange(scope)) as IDBRequest<AssetRecord[]>,
  )
  return records
    .filter((record) => matchesScope(record.scope, scope))
    .filter((record) => !kind || record.kind === kind)
    .map(toAsset)
    .sort((a, b) => a.createdAt - b.createdAt)
}

/** deleteAsset removes one asset by logical key. */
export async function deleteAsset(scope: CanvasScope, key: string): Promise<void> {
  await transact([STORE_ASSETS], async (tx) => {
    await idbRequest(tx.objectStore(STORE_ASSETS).delete(scopedKey(scope, key)))
  })
}

/**
 * RunRecord stores a run under its scope-prefixed id. The nested run object
 * keeps the domain shape untouched so persisted runs round-trip exactly.
 */
interface RunRecord {
  key: string
  scope: CanvasScope
  run: CanvasRun
  updatedAt: number
}

function toRun(record: RunRecord): CanvasRun {
  return record.run
}

/** terminalRun reports whether a run no longer needs polling protection. */
function terminalRun(run: CanvasRun): boolean {
  return run.status !== 'processing'
}

/**
 * saveRun persists one run and, when the run is terminal, prunes the scope's
 * terminal history down to MAX_TERMINAL_RUNS inside the same transaction so
 * the cap always holds after the write commits. Active runs are never
 * candidates for pruning. Returns the ids of runs pruned by this write so the
 * caller can drop them from its in-memory history.
 */
export async function saveRun(scope: CanvasScope, run: CanvasRun): Promise<string[]> {
  const runSnapshot = detachPlain(run)
  const record: RunRecord = {
    key: scopedKey(scope, runSnapshot.id),
    scope: detachScope(scope),
    run: runSnapshot,
    updatedAt: Date.now(),
  }
  const scopeSnapshot = detachScope(scope)
  return transact([STORE_RUNS], async (tx) => {
    const store = tx.objectStore(STORE_RUNS)
    await idbRequest(store.put(record))
    if (!terminalRun(runSnapshot)) return []
    const records = await idbRequest(store.getAll(scopeRange(scope)) as IDBRequest<RunRecord[]>)
    const terminal = records
      .filter((item) => matchesScope(item.scope, scopeSnapshot) && terminalRun(item.run))
      .sort((a, b) => a.run.createdAt - b.run.createdAt)
    const prunedRunIds: string[] = []
    const excess = terminal.length - MAX_TERMINAL_RUNS
    for (let i = 0; i < excess; i += 1) {
      await idbRequest(store.delete(terminal[i].key))
      prunedRunIds.push(terminal[i].run.id)
    }
    return prunedRunIds
  })
}

/** loadRun reads one run by id; absent or foreign runs are null. */
export async function loadRun(scope: CanvasScope, runId: string): Promise<CanvasRun | null> {
  const record = await readOne(
    STORE_RUNS,
    (store) => store.get(scopedKey(scope, runId)) as IDBRequest<RunRecord | undefined>,
  )
  if (!record || !matchesScope(record.scope, scope)) return null
  return toRun(record)
}

/** listRuns returns a scope's runs, oldest first. */
export async function listRuns(scope: CanvasScope): Promise<CanvasRun[]> {
  const records = await readOne(
    STORE_RUNS,
    (store) => store.getAll(scopeRange(scope)) as IDBRequest<RunRecord[]>,
  )
  return records
    .filter((item) => matchesScope(item.scope, scope))
    .map(toRun)
    .sort((a, b) => a.createdAt - b.createdAt)
}

/** deleteRun removes one run record. */
export async function deleteRun(scope: CanvasScope, runId: string): Promise<void> {
  await transact([STORE_RUNS], async (tx) => {
    await idbRequest(tx.objectStore(STORE_RUNS).delete(scopedKey(scope, runId)))
  })
}

/**
 * SettingRecord stores a structured-cloneable scoped value under a logical
 * settings key. Values must be JSON-shaped data (objects, arrays,
 * primitives); functions and class instances are not supported.
 */
interface SettingRecord {
  key: string
  scope: CanvasScope
  id: string
  value: unknown
  updatedAt: number
}

/** saveSetting writes one scoped setting. */
export async function saveSetting(
  scope: CanvasScope,
  key: string,
  value: unknown,
): Promise<void> {
  const record: SettingRecord = {
    key: scopedKey(scope, key),
    scope: detachScope(scope),
    id: key,
    value: detachPlain(value),
    updatedAt: Date.now(),
  }
  await transact([STORE_SETTINGS], async (tx) => {
    await idbRequest(tx.objectStore(STORE_SETTINGS).put(record))
  })
}

/** loadSetting reads one scoped setting; missing records are null. */
export async function loadSetting<T = unknown>(
  scope: CanvasScope,
  key: string,
): Promise<T | null> {
  const record = await readOne(
    STORE_SETTINGS,
    (store) => store.get(scopedKey(scope, key)) as IDBRequest<SettingRecord | undefined>,
  )
  if (!record || !matchesScope(record.scope, scope)) return null
  return record.value as T
}

/**
 * clearScope deletes every record of one scope across all stores in a single
 * transaction. Other scopes and unowned v1 records are outside the key range
 * and stay untouched.
 */
export async function clearScope(scope: CanvasScope): Promise<void> {
  await transact([STORE_SCENES, STORE_ASSETS, STORE_RUNS, STORE_SETTINGS], async (tx) => {
    for (const storeName of [STORE_SCENES, STORE_ASSETS, STORE_RUNS, STORE_SETTINGS]) {
      const store = tx.objectStore(storeName)
      const keys = await idbRequest(store.getAllKeys(scopeRange(scope)))
      for (const key of keys) {
        await idbRequest(store.delete(key))
      }
    }
  })
}

/**
 * SCENE_ASSET_REF_RE finds asset:// references inside serialized scenes so
 * retention can protect the images a scene still points at. Logical asset
 * keys are restricted to word characters, dots, dashes, and colons.
 */
const SCENE_ASSET_REF_RE = /asset:\/\/([\w:.-]+)/g

/**
 * collectSceneAssetRefs extracts the logical asset keys referenced by any of
 * the scope's stored scenes.
 */
function collectSceneAssetRefs(scenes: CanvasSceneRecord[]): Set<string> {
  const refs = new Set<string>()
  for (const scene of scenes) {
    for (const match of scene.json.matchAll(SCENE_ASSET_REF_RE)) {
      refs.add(match[1])
    }
  }
  return refs
}

/** CanvasRetentionReport describes one retention pass for the caller's UI. */
export interface CanvasRetentionReport {
  /** Terminal run ids pruned by the history cap. */
  prunedRunIds: string[]
  /** Logical asset keys evicted by the byte budget. */
  evictedAssetKeys: string[]
  /** Remaining asset bytes after eviction. */
  assetBytes: number
  /** True when protected content alone exceeds the byte budget. */
  overBudget: boolean
}

/**
 * enforceRetention applies the retention policy to one scope: terminal runs
 * beyond MAX_TERMINAL_RUNS are pruned oldest-first, then unprotected assets
 * are evicted oldest-first until the byte budget holds. Assets referenced by a
 * stored scene (asset:// refs) or by a surviving run's outputs are protected,
 * as is every record of an active run. When protected content alone exceeds
 * the budget nothing is force-evicted; the report flags overBudget so the UI
 * can warn instead of breaking the scene.
 */
export async function enforceRetention(scope: CanvasScope): Promise<CanvasRetentionReport> {
  return transact([STORE_SCENES, STORE_ASSETS, STORE_RUNS], async (tx) => {
    const runStore = tx.objectStore(STORE_RUNS)
    const sceneStore = tx.objectStore(STORE_SCENES)
    const assetStore = tx.objectStore(STORE_ASSETS)

    const runRecords = await idbRequest(
      runStore.getAll(scopeRange(scope)) as IDBRequest<RunRecord[]>,
    )
    const scopedRuns = runRecords.filter((item) => matchesScope(item.scope, scope))
    const terminal = scopedRuns
      .filter((item) => terminalRun(item.run))
      .sort((a, b) => a.run.createdAt - b.run.createdAt)
    const prunedRunIds: string[] = []
    const excess = terminal.length - MAX_TERMINAL_RUNS
    for (let i = 0; i < excess; i += 1) {
      await idbRequest(runStore.delete(terminal[i].key))
      prunedRunIds.push(terminal[i].run.id)
    }
    const survivingRuns = scopedRuns.filter((item) => !prunedRunIds.includes(item.run.id))

    const sceneRecords = await idbRequest(
      sceneStore.getAll(scopeRange(scope)) as IDBRequest<SceneRecord[]>,
    )
    const sceneRefs = collectSceneAssetRefs(
      sceneRecords
        .filter((item) => matchesScope(item.scope, scope))
        .map((item) => ({ name: item.name, json: item.json, updatedAt: item.updatedAt })),
    )

    // Outputs still owned by a surviving run stay downloadable from history.
    const protectedKeys = new Set<string>(sceneRefs)
    for (const item of survivingRuns) {
      for (const output of item.run.outputs) {
        if (output.assetKey) protectedKeys.add(output.assetKey)
      }
    }

    const assetRecords = await idbRequest(
      assetStore.getAll(scopeRange(scope)) as IDBRequest<AssetRecord[]>,
    )
    const assets = assetRecords
      .filter((item) => matchesScope(item.scope, scope))
      .sort((a, b) => a.createdAt - b.createdAt)
    let assetBytes = assets.reduce((sum, item) => sum + (item.blob?.size ?? 0), 0)
    const evictedAssetKeys: string[] = []
    for (const asset of assets) {
      if (assetBytes <= ASSET_BYTE_BUDGET) break
      if (protectedKeys.has(asset.id)) continue
      await idbRequest(assetStore.delete(asset.key))
      evictedAssetKeys.push(asset.id)
      assetBytes -= asset.blob?.size ?? 0
    }
    return {
      prunedRunIds,
      evictedAssetKeys,
      assetBytes,
      overBudget: assetBytes > ASSET_BYTE_BUDGET,
    }
  })
}

/**
 * WORKSPACE_STORAGE_PREFIX namespaces the per-user workspace UUID in
 * localStorage. Per-user keys keep one account's clear from rotating another
 * account's workspace on a shared browser.
 */
const WORKSPACE_STORAGE_PREFIX = 'canvas:workspace:'

const WORKSPACE_UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

/** workspaceStorageKey maps a user id to its localStorage key. */
function workspaceStorageKey(userId: number): string {
  return `${WORKSPACE_STORAGE_PREFIX}${userId}`
}

/**
 * normalizeWorkspaceId accepts only a well-formed UUID, so a corrupted or
 * injected localStorage value cannot merge two workspaces.
 */
export function normalizeWorkspaceId(value: unknown): string | null {
  if (typeof value !== 'string') return null
  const normalized = value.trim().toLowerCase()
  return WORKSPACE_UUID_RE.test(normalized) ? normalized : null
}

/** generateWorkspaceId creates a lowercase v4 UUID from crypto randomness. */
function generateWorkspaceId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID().toLowerCase()
  }
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    const bytes = new Uint8Array(16)
    crypto.getRandomValues(bytes)
    bytes[6] = (bytes[6] & 0x0f) | 0x40
    bytes[8] = (bytes[8] & 0x3f) | 0x80
    const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
  }
  throw new LocalStoreError('unavailable', 'Workspace identity is unavailable')
}

function workspaceStorage(): Storage {
  if (typeof window === 'undefined' || !window.localStorage) {
    throw new LocalStoreError('unavailable', 'Browser local storage is unavailable')
  }
  return window.localStorage
}

/**
 * getWorkspaceId returns the persisted workspace UUID for a user, generating
 * one when absent or invalid. Storage failures surface as LocalStoreError so
 * callers stop instead of silently sharing a partition.
 */
export function getWorkspaceId(userId: number): string {
  const storage = workspaceStorage()
  try {
    const current = normalizeWorkspaceId(storage.getItem(workspaceStorageKey(userId)))
    if (current) return current
    const generated = generateWorkspaceId()
    storage.setItem(workspaceStorageKey(userId), generated)
    return generated
  } catch (error) {
    if (error instanceof LocalStoreError) throw error
    throw new LocalStoreError('unavailable', 'Browser local storage is unavailable', error)
  }
}

/**
 * rotateWorkspaceId stores a fresh workspace UUID for a user and returns it.
 * Records under the old workspace stay on disk but become unreachable, which
 * is what makes late async callbacks and stale tabs unable to resurrect data.
 */
export function rotateWorkspaceId(userId: number): string {
  const storage = workspaceStorage()
  try {
    const generated = generateWorkspaceId()
    storage.setItem(workspaceStorageKey(userId), generated)
    return generated
  } catch (error) {
    if (error instanceof LocalStoreError) throw error
    throw new LocalStoreError('unavailable', 'Browser local storage is unavailable', error)
  }
}

/**
 * onWorkspaceChanged subscribes to cross-tab changes of a user's workspace
 * UUID. The callback receives the new normalized id (or null when the key was
 * removed or set to garbage) and returns an unsubscribe function. The storage
 * event only fires in other tabs, so same-tab rotation is reported by
 * rotateWorkspaceId directly.
 */
export function onWorkspaceChanged(
  userId: number,
  callback: (workspaceId: string | null) => void,
): () => void {
  if (typeof window === 'undefined') return () => {}
  const key = workspaceStorageKey(userId)
  const listener = (event: StorageEvent) => {
    if (event.storageArea !== window.localStorage) return
    if (event.key !== key) return
    callback(normalizeWorkspaceId(event.newValue))
  }
  window.addEventListener('storage', listener)
  return () => window.removeEventListener('storage', listener)
}

/**
 * __resetCanvasStoreForTest drops the cached connection promise between
 * isolated tests. It does not close connections or delete records.
 */
export function __resetCanvasStoreForTest(): void {
  dbPromise = null
}
