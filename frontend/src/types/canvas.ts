/**
 * canvas.ts defines the canvas-studio domain types shared by the API adapter,
 * the local store, and the useCanvasStudio composable.
 *
 * All timestamps are millisecond epoch values; server payloads that arrive in
 * seconds are converted at the API boundary. Scope fields carry both the owning
 * user and the per-user workspace UUID so every local record is partitioned by
 * account and by workspace generation.
 */

/**
 * CanvasOperation selects the explicit generation mode of a run. generate is
 * text-to-image without sources, edit sends one or more source images, and
 * inpaint sends exactly one source image plus a same-size mask.
 */
export type CanvasOperation = 'generate' | 'edit' | 'inpaint'

/**
 * CanvasScope partitions every local record by user and workspace. The
 * workspaceId is a per-user local UUID: rotating it orphans the previous
 * workspace's records, which is how "clear local data" invalidates in-flight
 * callbacks and other tabs without touching other users' data.
 */
export interface CanvasScope {
  /** Owning account id from the auth store. */
  userId: number
  /** Per-user workspace UUID stored in localStorage. */
  workspaceId: string
}

/**
 * CanvasSelection is the set of submission parameters frozen onto a run at
 * submit time. It stores the API key id, never the key secret: the secret is
 * resolved from the key list only for the lifetime of a request.
 */
export interface CanvasSelection {
  /** Id of the user API key used to submit. */
  apiKeyId: number
  /** Image model name. */
  model: string
  /** Prompt text. */
  prompt: string
  /** Size tier, e.g. 1024x1024. */
  size: string
  /** Quality tier; empty string means the gateway default. */
  quality: string
  /** Requested output count. */
  count: number
  /** Explicit operation mode. */
  operation: CanvasOperation
}

/**
 * CanvasAssetKind distinguishes user-imported source images from generated
 * outputs so eviction can reason about what produced a blob.
 */
export type CanvasAssetKind = 'source' | 'output'

/**
 * CanvasAsset is one stored image. The key is the logical asset identifier
 * inside a scope (for example output:<runId>:<index>) and is what scene
 * snapshots embed as asset:// references.
 */
export interface CanvasAsset {
  /** Logical key inside the owning scope. */
  key: string
  /** Whether the bytes are a user source or a generated output. */
  kind: CanvasAssetKind
  /** Image bytes. */
  blob: Blob
  /** Server run id that produced the image, for outputs. */
  runId?: string
  /** Original index in the run result array, for outputs. */
  outputIndex?: number
  /** Creation time in milliseconds. */
  createdAt: number
}

/**
 * CanvasRunStatus is the local lifecycle of a run. processing means the task
 * may still complete; completed/failed are server-terminal; expired means the
 * server retention window passed; unavailable means the run can no longer be
 * queried at all (deleted key, missing task).
 */
export type CanvasRunStatus =
  | 'processing'
  | 'completed'
  | 'failed'
  | 'expired'
  | 'unavailable'

/**
 * CanvasRunOutputState describes where one output's bytes live. pending means
 * not yet harvested, local means persisted as an asset, remote means only the
 * upstream URL is usable (for example after a CORS failure), and missing means
 * the bytes are unobtainable.
 */
export type CanvasRunOutputState = 'pending' | 'local' | 'remote' | 'missing'

/**
 * CanvasRunOutput tracks one result entry by its original index in the server
 * data array. Keeping the index stable lets earlier malformed entries not
 * shift the identity of later outputs.
 */
export interface CanvasRunOutput {
  /** Original index in the server result data array. */
  index: number
  /** Logical asset key once bytes are stored locally. */
  assetKey?: string
  /** Upstream URL kept when remote bytes could not be downloaded. */
  remoteUrl?: string
  /** Where the output bytes currently live. */
  state: CanvasRunOutputState
  /** Failure detail for missing/remote outputs. */
  error?: string
  /** True once the editor confirmed placing this output into the scene. */
  autoPlaced?: boolean
}

/**
 * CanvasRun is the persisted unit of submitted work. The params snapshot and
 * apiKeyId are frozen when the task is accepted; later selection changes and
 * key switches never rewrite an accepted run.
 */
export interface CanvasRun {
  /** Server task id. */
  id: string
  /** Id of the key that submitted the task. */
  apiKeyId: number
  /** Frozen submission parameters. */
  params: CanvasSelection
  /** Local lifecycle status. */
  status: CanvasRunStatus
  /** Acceptance time in milliseconds. */
  createdAt: number
  /** Server retention deadline in milliseconds; 0 when the server sent none. */
  expiresAt: number
  /** True while polling is paused after transient network failures. */
  paused: boolean
  /** Harvest state per result index. */
  outputs: CanvasRunOutput[]
  /** Machine-readable failure code or server message. */
  error?: string
}

/**
 * CanvasSceneRecord is one named scene snapshot. The JSON payload is opaque to
 * the store; the editor owns its schema and embeds asset:// references.
 */
export interface CanvasSceneRecord {
  /** Logical scene name inside the scope, e.g. default. */
  name: string
  /** Serialized scene snapshot. */
  json: string
  /** Update time in milliseconds. */
  updatedAt: number
}
