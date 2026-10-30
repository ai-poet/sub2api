/**
 * fakeIdb.ts is a focused IndexedDB test double for the canvas local store.
 *
 * It models the semantics the store relies on — scoped key ranges, request
 * success events, transaction commit/abort, and write rollback — without
 * pretending to be a complete implementation. Requests are processed on
 * microtasks, and a transaction commits only once its pending request count
 * drains, which lets tests prove writes await tx completion. Injected request
 * and commit errors exercise the abort path.
 */

import { vi } from 'vitest'
import { isProxy } from 'vue'

type StoredValue = Record<string, unknown> | undefined

interface StoreData {
  keyPath: string | null
  records: Map<IDBValidKey, StoredValue>
}

interface DbData {
  version: number
  stores: Map<string, StoreData>
}

interface FakeRequest<T = unknown> {
  result: T
  error: unknown
  onsuccess: ((event: { target: FakeRequest<T> }) => void) | null
  onerror: ((event: { target: FakeRequest<T> }) => void) | null
  onupgradeneeded: ((event: { target: FakeRequest }) => void) | null
  onblocked: ((event: { target: FakeRequest }) => void) | null
}

const ABSENT = Symbol('absent')

function domError(name: string, message = name): DOMException {
  return new DOMException(message, name)
}

/**
 * FakeKeyRange matches the IDBKeyRange.bound usage of the store: inclusive
 * string bounds over record keys.
 */
class FakeKeyRange {
  constructor(
    readonly lower: IDBValidKey,
    readonly upper: IDBValidKey,
  ) {}

  static bound(lower: IDBValidKey, upper: IDBValidKey): FakeKeyRange {
    return new FakeKeyRange(lower, upper)
  }

  includes(key: IDBValidKey): boolean {
    return key >= this.lower && key <= this.upper
  }
}

type KeyQuery = FakeKeyRange | IDBValidKey | undefined

function keyMatches(query: KeyQuery, key: IDBValidKey): boolean {
  if (query === undefined || query === null) return true
  if (query instanceof FakeKeyRange) return query.includes(key)
  return key === query
}

export interface FakeIdbEnvironment {
  databases: Map<string, DbData>
  requestErrors: unknown[]
  commitErrors: unknown[]
  commitHolds: Array<{ promise: Promise<void> }>
}

/**
 * cloneStored approximates real IndexedDB structured-clone semantics: plain
 * objects and arrays are cloned at put and again at read, a Vue reactive
 * proxy inside a record throws DataCloneError instead of being silently
 * stored (which is what makes detach bugs visible to tests), and non-plain
 * values such as Blobs pass through — real engines clone them, but tests do
 * not rely on clone identity there, and a plain structuredClone would quietly
 * strip a jsdom Blob into a plain object without size or type.
 */
function cloneStored<T>(value: T): T {
  if (value === null || typeof value !== 'object') return value
  if (isProxy(value)) {
    throw new DOMException('The object could not be cloned.', 'DataCloneError')
  }
  if (Array.isArray(value)) {
    return value.map((item) => cloneStored(item)) as T
  }
  const proto = Object.getPrototypeOf(value)
  if (proto !== Object.prototype && proto !== null) return value
  const copy: Record<string, unknown> = {}
  for (const [key, item] of Object.entries(value)) {
    copy[key] = cloneStored(item)
  }
  return copy as T
}

class FakeTransaction {
  error: unknown = null
  oncomplete: (() => void) | null = null
  onerror: (() => void) | null = null
  onabort: (() => void) | null = null
  private pending = 0
  private settled = false
  private undo: Array<() => void> = []

  constructor(
    private readonly env: FakeIdbEnvironment,
    private readonly db: DbData,
    private readonly storeNames: string[],
  ) {}

  objectStore(name: string): FakeObjectStore {
    const store = this.db.stores.get(name)
    if (!store || !this.storeNames.includes(name)) {
      throw domError('NotFoundError', `object store ${name} not in transaction`)
    }
    return new FakeObjectStore(this, store)
  }

  /** issue schedules one request inside the transaction's pending count. */
  issue<T>(work: () => T): FakeRequest<T> {
    if (this.settled) {
      throw domError('InvalidStateError', 'transaction already finished')
    }
    this.pending += 1
    const request: FakeRequest<T> = {
      result: undefined as T,
      error: null,
      onsuccess: null,
      onerror: null,
      onupgradeneeded: null,
      onblocked: null,
    }
    queueMicrotask(() => {
      try {
        const injected = this.env.requestErrors.shift()
        if (injected !== undefined) throw injected
        request.result = work()
      } catch (error) {
        request.error = error
      }
      if (request.error !== null && request.error !== undefined) {
        request.onerror?.({ target: request })
        this.pending -= 1
        // Default IDB behavior: an unhandled request error aborts the tx.
        this.abortInternal(request.error)
        return
      }
      request.onsuccess?.({ target: request })
      this.pending -= 1
      this.scheduleFinish()
    })
    return request
  }

  recordUndo(restore: () => void): void {
    this.undo.push(restore)
  }

  abort(): void {
    this.abortInternal(domError('AbortError'))
  }

  private abortInternal(error: unknown): void {
    if (this.settled) return
    this.settled = true
    this.error = this.error ?? error
    // Undo replays newest-first so later writes are rolled back before the
    // earlier ones they overwrote.
    for (const restore of this.undo.splice(0).reverse()) restore()
    queueMicrotask(() => {
      this.onerror?.()
      this.onabort?.()
    })
  }

  private scheduleFinish(): void {
    queueMicrotask(() => {
      if (this.settled || this.pending > 0) return
      const hold = this.env.commitHolds.shift()
      if (hold) {
        // A held commit lets tests interleave other work mid-flight.
        void hold.promise.then(() => this.finishCommit())
        return
      }
      this.finishCommit()
    })
  }

  private finishCommit(): void {
    if (this.settled) return
    if (this.pending > 0) {
      this.scheduleFinish()
      return
    }
    const injected = this.env.commitErrors.shift()
    if (injected !== undefined) {
      this.abortInternal(injected)
      return
    }
    this.settled = true
    this.undo.length = 0
    queueMicrotask(() => this.oncomplete?.())
  }
}

class FakeObjectStore {
  constructor(
    private readonly tx: FakeTransaction,
    private readonly store: StoreData,
  ) {}

  get indexNames(): { contains: (name: string) => boolean } {
    return { contains: () => false }
  }

  private recordKey(record: unknown): IDBValidKey {
    const keyPath = this.store.keyPath
    if (!keyPath) throw domError('DataError', 'store has no keyPath')
    const key = (record as Record<string, IDBValidKey>)[keyPath]
    if (key === undefined) throw domError('DataError', 'record lacks key')
    return key
  }

  get(query: IDBValidKey): FakeRequest<StoredValue> {
    return this.tx.issue(() => {
      const value = this.store.records.get(query)
      return value === undefined ? undefined : cloneStored(value)
    })
  }

  put(record: StoredValue): FakeRequest<IDBValidKey> {
    return this.tx.issue(() => {
      const key = this.recordKey(record)
      // Clone at put time like real IDB: DataCloneError on proxies, and later
      // mutations of the caller's object cannot leak into storage.
      const stored = cloneStored(record)
      const previous = this.store.records.has(key)
        ? this.store.records.get(key)
        : ABSENT
      this.tx.recordUndo(() => {
        if (previous === ABSENT) this.store.records.delete(key)
        else this.store.records.set(key, previous)
      })
      this.store.records.set(key, stored)
      return key
    })
  }

  delete(query: IDBValidKey): FakeRequest<undefined> {
    return this.tx.issue(() => {
      const previous = this.store.records.has(query)
        ? this.store.records.get(query)
        : ABSENT
      this.tx.recordUndo(() => {
        if (previous !== ABSENT) this.store.records.set(query, previous)
      })
      this.store.records.delete(query)
      return undefined
    })
  }

  clear(): FakeRequest<undefined> {
    return this.tx.issue(() => {
      const snapshot = new Map(this.store.records)
      this.tx.recordUndo(() => {
        this.store.records.clear()
        for (const [key, value] of snapshot) this.store.records.set(key, value)
      })
      this.store.records.clear()
      return undefined
    })
  }

  getAll(query?: KeyQuery): FakeRequest<StoredValue[]> {
    return this.tx.issue(() =>
      [...this.store.records.entries()]
        .filter(([key]) => keyMatches(query, key))
        .map(([, value]) => cloneStored(value)),
    )
  }

  getAllKeys(query?: KeyQuery): FakeRequest<IDBValidKey[]> {
    return this.tx.issue(() =>
      [...this.store.records.keys()].filter((key) => keyMatches(query, key)),
    )
  }
}

class FakeDatabase {
  onversionchange: (() => void) | null = null

  constructor(
    private readonly env: FakeIdbEnvironment,
    private readonly data: DbData,
  ) {}

  get objectStoreNames(): { contains: (name: string) => boolean } {
    const stores = this.data.stores
    return { contains: (name: string) => stores.has(name) }
  }

  createObjectStore(name: string, options?: { keyPath?: string | null }): FakeObjectStore {
    if (this.data.stores.has(name)) {
      throw domError('ConstraintError', `object store ${name} exists`)
    }
    const store: StoreData = { keyPath: options?.keyPath ?? null, records: new Map() }
    this.data.stores.set(name, store)
    // The upgrade path only creates stores, so a detached accessor suffices.
    return new FakeObjectStore(
      new FakeTransaction(this.env, this.data, [name]),
      store,
    )
  }

  transaction(storeNames: string | string[]): FakeTransaction {
    const names = Array.isArray(storeNames) ? storeNames : [storeNames]
    for (const name of names) {
      if (!this.data.stores.has(name)) {
        throw domError('NotFoundError', `object store ${name} missing`)
      }
    }
    return new FakeTransaction(this.env, this.data, names)
  }

  close(): void {
    this.onversionchange?.()
  }
}

export interface FakeIdb {
  /** Injected into vi.stubGlobal('indexedDB', ...). */
  indexedDB: { open: (name: string, version?: number) => FakeRequest<FakeDatabase> }
  /** Injected into vi.stubGlobal('IDBKeyRange', ...). */
  IDBKeyRange: typeof FakeKeyRange
  /** Fails the next request processed by any transaction. */
  failNextRequest(error: unknown): void
  /** Fails the next transaction commit. */
  failNextCommit(error: unknown): void
  /** Holds the next transaction commit until the returned function runs. */
  holdNextCommit(): () => void
  /** Seeds a database directly, e.g. a v1 database before upgrade. */
  seedDatabase(
    name: string,
    version: number,
    stores: Record<string, Array<Record<string, unknown>>>,
    keyPath?: string,
  ): void
  /** Raw store records for assertions on unowned v1 data. */
  inspect(name: string, storeName: string): Map<IDBValidKey, StoredValue> | undefined
}

/**
 * installFakeIdb stubs indexedDB and IDBKeyRange on globalThis and returns the
 * inspection/injection handles. Call __resetCanvasStoreForTest after install
 * so the store picks up the new backend.
 */
export function installFakeIdb(): FakeIdb {
  const env: FakeIdbEnvironment = {
    databases: new Map(),
    requestErrors: [],
    commitErrors: [],
    commitHolds: [],
  }

  const fake: FakeIdb = {
    indexedDB: {
      open(name: string, version?: number): FakeRequest<FakeDatabase> {
        const request: FakeRequest<FakeDatabase> = {
          result: undefined as unknown as FakeDatabase,
          error: null,
          onsuccess: null,
          onerror: null,
          onupgradeneeded: null,
          onblocked: null,
        }
        queueMicrotask(() => {
          let data = env.databases.get(name)
          if (data && version !== undefined && data.version > version) {
            request.error = domError('VersionError')
            request.onerror?.({ target: request as unknown as FakeRequest })
            return
          }
          if (!data) {
            // A brand-new database still goes through onupgradeneeded, so it
            // starts at version 0 and upgrades to the requested version.
            data = { version: 0, stores: new Map() }
            env.databases.set(name, data)
          }
          request.result = new FakeDatabase(env, data)
          if (version !== undefined && data.version < version) {
            data.version = version
            request.onupgradeneeded?.({ target: request as unknown as FakeRequest })
          }
          request.onsuccess?.({ target: request as unknown as FakeRequest })
        })
        return request
      },
    },
    IDBKeyRange: FakeKeyRange,
    failNextRequest(error: unknown): void {
      env.requestErrors.push(error)
    },
    failNextCommit(error: unknown): void {
      env.commitErrors.push(error)
    },
    holdNextCommit(): () => void {
      let release!: () => void
      env.commitHolds.push({
        promise: new Promise<void>((resolve) => {
          release = resolve
        }),
      })
      return release
    },
    seedDatabase(name, version, stores, keyPath = 'key'): void {
      const data: DbData = { version, stores: new Map() }
      for (const [storeName, records] of Object.entries(stores)) {
        const map = new Map<IDBValidKey, StoredValue>()
        for (const record of records) {
          map.set(record[keyPath] as IDBValidKey, record)
        }
        data.stores.set(storeName, { keyPath, records: map })
      }
      env.databases.set(name, data)
    },
    inspect(name: string, storeName: string) {
      return env.databases.get(name)?.stores.get(storeName)?.records
    },
  }

  vi.stubGlobal('indexedDB', fake.indexedDB)
  vi.stubGlobal('IDBKeyRange', fake.IDBKeyRange)
  return fake
}
