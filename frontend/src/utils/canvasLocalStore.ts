/**
 * canvasLocalStore.ts 在浏览器 IndexedDB 里保存画布快照与本地作品。
 *
 * 两类记录分开放：
 * - scenes：画布的 Fabric JSON 快照，关掉页面再回来能恢复现场；
 * - assets：生成结果的图片字节，供历史列表在没有网络时也能看图。
 *
 * 数据库名带版本，升级时靠 object store 的名字判断该建什么。
 */

/** 数据库名。与 TokenRouter 的同名库互不相干，各自定义前缀。 */
const DB_NAME = 'sub2api-canvas-studio'
const DB_VERSION = 1

/** 画布快照的 object store。 */
const STORE_SCENES = 'scenes'

/** 图片字节的 object store。 */
const STORE_ASSETS = 'assets'

/**
 * CanvasSceneRecord 是一份画布快照。
 */
export interface CanvasSceneRecord {
  /** 快照标识，同时也是场景名。 */
  key: string
  /** 序列化后的 Fabric 场景 JSON。 */
  json: string
  /** 更新时间（毫秒时间戳）。 */
  updatedAt: number
}

/**
 * CanvasAsset 是一份本地图片记录。
 */
export interface CanvasAsset {
  /** 本地记录标识。 */
  key: string
  /** 图片字节。 */
  blob: Blob
  /** 关联的服务端任务 ID，用于和历史列表对齐。 */
  runId?: string
  /** 该图片在任务输出中的序号。 */
  outputIndex?: number
  /** 创建时间（毫秒时间戳）。 */
  createdAt: number
}

/**
 * LocalStoreErrorType 是本地存储失败的分类。
 */
export type LocalStoreErrorType = 'quota' | 'unavailable' | 'unknown'

/**
 * LocalStoreError 携带失败分类，便于 UI 给出不同的提示。
 */
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

/** 打开数据库的共享 promise。多次并发调用只触发一次 open。 */
let dbPromise: Promise<IDBDatabase> | null = null

/**
 * classifyError 把 IndexedDB 的异常翻译成可读的失败分类。
 */
function classifyError(error: unknown): LocalStoreError {
  const name = error instanceof DOMException ? error.name : ''
  if (name === 'QuotaExceededError') {
    return new LocalStoreError('quota', '本地存储空间不足', error)
  }
  if (name === 'NotSupportedError' || name === 'InvalidStateError') {
    return new LocalStoreError('unavailable', '当前环境不支持本地存储', error)
  }
  const message = error instanceof Error ? error.message : String(error)
  return new LocalStoreError('unknown', message || '本地存储操作失败', error)
}

/**
 * idbRequest 把一个 IndexedDB 请求包装成 promise，并按失败类型分类。
 */
function idbRequest<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(classifyError(request.error))
  })
}

/**
 * openDatabase 打开（或升级）本地数据库。
 *
 * onupgradeneeded 里按名字建 store，重复升级不会重复建，
 * 因此新版本只要换 store 名字即可，不需要迁移已有数据。
 */
function openDatabase(): Promise<IDBDatabase> {
  return new Promise<IDBDatabase>((resolve, reject) => {
    if (typeof indexedDB === 'undefined') {
      reject(new LocalStoreError('unavailable', '当前环境不支持 IndexedDB'))
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
    }
    request.onsuccess = () => {
      const db = request.result
      // 另一个标签页升级了数据库时，本连接会被关闭；此时清掉缓存，
      // 下次调用会重新打开，避免一直拿着一个失效的连接。
      db.onversionchange = () => {
        db.close()
        dbPromise = null
      }
      resolve(db)
    }
    request.onerror = () => reject(classifyError(request.error))
    request.onblocked = () => reject(new LocalStoreError('unavailable', '本地存储被其他标签页占用'))
  })
}

/**
 * database 返回共享的数据库连接。
 * 打开失败时清掉缓存，让下一次调用可以重试。
 */
async function database(): Promise<IDBDatabase> {
  if (!dbPromise) {
    dbPromise = openDatabase().catch((error) => {
      dbPromise = null
      throw error
    })
  }
  return dbPromise
}

/**
 * saveScene 保存一份画布快照。相同 key 会覆盖旧快照。
 */
export async function saveScene(key: string, json: string): Promise<void> {
  const db = await database()
  const tx = db.transaction(STORE_SCENES, 'readwrite')
  const record: CanvasSceneRecord = { key, json, updatedAt: Date.now() }
  await idbRequest(tx.objectStore(STORE_SCENES).put(record))
}

/**
 * loadScene 读取一份画布快照。��存在时返回 null。
 */
export async function loadScene(key: string): Promise<CanvasSceneRecord | null> {
  const db = await database()
  const tx = db.transaction(STORE_SCENES, 'readonly')
  const record = await idbRequest<CanvasSceneRecord | undefined>(
    tx.objectStore(STORE_SCENES).get(key) as IDBRequest<CanvasSceneRecord | undefined>,
  )
  return record ?? null
}

/**
 * listScenes 列出所有画布快照，按更新时间倒序。
 */
export async function listScenes(): Promise<CanvasSceneRecord[]> {
  const db = await database()
  const tx = db.transaction(STORE_SCENES, 'readonly')
  const records = await idbRequest<CanvasSceneRecord[]>(
    tx.objectStore(STORE_SCENES).getAll() as IDBRequest<CanvasSceneRecord[]>,
  )
  return records.sort((a, b) => b.updatedAt - a.updatedAt)
}

/**
 * deleteScene 删除一份画布快照。
 */
export async function deleteScene(key: string): Promise<void> {
  const db = await database()
  const tx = db.transaction(STORE_SCENES, 'readwrite')
  await idbRequest(tx.objectStore(STORE_SCENES).delete(key))
}

/**
 * saveAsset 保存一份本地图片记录。
 */
export async function saveAsset(asset: CanvasAsset): Promise<void> {
  const db = await database()
  const tx = db.transaction(STORE_ASSETS, 'readwrite')
  await idbRequest(tx.objectStore(STORE_ASSETS).put(asset))
}

/**
 * loadAsset 读取一份本地图片记录。不存在时返回 null。
 */
export async function loadAsset(key: string): Promise<CanvasAsset | null> {
  const db = await database()
  const tx = db.transaction(STORE_ASSETS, 'readonly')
  const record = await idbRequest<CanvasAsset | undefined>(
    tx.objectStore(STORE_ASSETS).get(key) as IDBRequest<CanvasAsset | undefined>,
  )
  return record ?? null
}

/**
 * listAssets 列出全部本地图片记录，按创建时间倒序。
 */
export async function listAssets(): Promise<CanvasAsset[]> {
  const db = await database()
  const tx = db.transaction(STORE_ASSETS, 'readonly')
  const records = await idbRequest<CanvasAsset[]>(
    tx.objectStore(STORE_ASSETS).getAll() as IDBRequest<CanvasAsset[]>,
  )
  return records.sort((a, b) => b.createdAt - a.createdAt)
}

/**
 * deleteAsset 删除一份本地图片记录。
 */
export async function deleteAsset(key: string): Promise<void> {
  const db = await database()
  const tx = db.transaction(STORE_ASSETS, 'readwrite')
  await idbRequest(tx.objectStore(STORE_ASSETS).delete(key))
}

/**
 * clearLocalData 清空画布快照和本地图片。
 */
export async function clearLocalData(): Promise<void> {
  const db = await database()
  await idbRequest(db.transaction(STORE_SCENES, 'readwrite').objectStore(STORE_SCENES).clear())
  await idbRequest(db.transaction(STORE_ASSETS, 'readwrite').objectStore(STORE_ASSETS).clear())
}
