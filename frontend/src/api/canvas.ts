/**
 * canvas.ts submits canvas generation tasks to the async image gateway and
 * polls their results.
 *
 * The operation is explicit: generate posts JSON to
 * /v1/images/generations/async, edit posts multipart with the source image
 * bytes (one image under "image", several under "image[]"), and inpaint posts
 * exactly one image plus a non-empty mask. Submissions are never retried
 * because a duplicate POST would bill twice.
 */

import { buildGatewayUrl } from './client'
import type { CanvasOperation } from '@/types/canvas'

/** Status reported by the async image task endpoints. */
export type CanvasTaskStatus = 'processing' | 'completed' | 'failed' | string

/** Maximum source images accepted per edit request. */
export const CANVAS_MAX_EDIT_IMAGES = 16
/** Per-image download bound so a stalled upstream cannot hang a harvest. */
export const OUTPUT_FETCH_TIMEOUT_MS = 60_000

/**
 * CanvasSubmitResult is the task handle returned by the async endpoints.
 * Timestamps arrive as seconds and are converted to milliseconds by callers.
 */
export interface CanvasSubmitResult {
  /** Task id used for polling. */
  id: string
  /** Task object name; equals id on this gateway. */
  task_id: string
  /** Initial task status. */
  status: CanvasTaskStatus
  /** Creation time (seconds). */
  created_at: number
  /** Retention deadline (seconds). */
  expires_at: number
  /** Optional server-supplied poll URL; unused by the client. */
  poll_url?: string
}

/**
 * CanvasTask is the polled task state. The result body follows the OpenAI
 * images shape: a data array whose entries carry url or b64_json.
 */
export interface CanvasTask {
  id: string
  task_id: string
  status: CanvasTaskStatus
  http_status?: number
  /** Upstream response body on success. */
  result?: { data?: Array<{ url?: string; b64_json?: string }> } | null
  /** Error body on failure. */
  error?: { code?: string; message?: string; type?: string } | null
  created_at: number
  completed_at?: number | null
  expires_at: number
}

/**
 * CanvasSubmitRequest describes one submission. images holds the original
 * source bytes; the operation alone decides whether they are sent and how.
 */
export interface CanvasSubmitRequest {
  /** Explicit operation mode. */
  operation: CanvasOperation
  /** Source image bytes for edit/inpaint; must be empty for generate. */
  images: Blob[]
  /** Mask bytes for inpaint; must be null for generate/edit. */
  mask: Blob | null
  /** Image model name. */
  model: string
  /** Prompt text. */
  prompt: string
  /** Size tier, e.g. 1024x1024. */
  size: string
  /** Requested output count; defaults to 1. */
  count?: number
  /** Optional quality tier. */
  quality?: string
  /** Optional background mode. */
  background?: string
}

/**
 * CanvasInputError rejects a request whose inputs contradict its operation
 * before any network call is made, so invalid submits can never be billed.
 */
export class CanvasInputError extends Error {
  readonly code = 'invalid_input'

  constructor(message: string) {
    super(message)
    this.name = 'CanvasInputError'
  }
}

/**
 * parseCanvasError translates a gateway error response into an Error carrying
 * the upstream code and HTTP status for the composable's classification.
 */
async function parseCanvasError(response: Response): Promise<Error> {
  let message = response.statusText || `HTTP ${response.status}`
  let code: string | number = response.status
  try {
    const body = await response.json()
    const err = body?.error
    if (typeof err?.message === 'string' && err.message) {
      message = err.message
    } else if (typeof body?.message === 'string' && body.message) {
      message = body.message
    }
    if (err?.code) {
      code = err.code
    }
  } catch {
    // Non-JSON error bodies keep the status text.
  }
  const error = new Error(message)
  ;(error as { code?: string | number }).code = code
  ;(error as { status?: number }).status = response.status
  return error
}

function authHeaders(apiKey: string, extra?: HeadersInit): HeadersInit {
  return {
    Authorization: `Bearer ${apiKey}`,
    ...extra,
  }
}

/**
 * validateSubmitRequest enforces the input contract of each operation before
 * any request leaves the browser.
 */
function validateSubmitRequest(request: CanvasSubmitRequest): void {
  if (typeof request.model !== 'string' || request.model.trim() === '') {
    throw new CanvasInputError('model is required')
  }
  if (typeof request.prompt !== 'string' || request.prompt.trim() === '') {
    throw new CanvasInputError('prompt is required')
  }
  const count = request.count ?? 1
  if (!Number.isInteger(count) || count < 1 || count > 4) {
    throw new CanvasInputError('count must be an integer between 1 and 4')
  }
  const images = request.images
  if (images.some((image) => image.size === 0)) {
    throw new CanvasInputError('source images must not be empty')
  }
  switch (request.operation) {
    case 'generate':
      if (images.length > 0 || request.mask) {
        throw new CanvasInputError('generate does not accept source images or a mask')
      }
      break
    case 'edit':
      if (images.length === 0) {
        throw new CanvasInputError('edit requires at least one source image')
      }
      if (images.length > CANVAS_MAX_EDIT_IMAGES) {
        throw new CanvasInputError(`edit accepts at most ${CANVAS_MAX_EDIT_IMAGES} images`)
      }
      if (request.mask) {
        throw new CanvasInputError('edit does not accept a mask; use inpaint')
      }
      break
    case 'inpaint':
      if (images.length !== 1) {
        throw new CanvasInputError('inpaint requires exactly one source image')
      }
      if (!request.mask || request.mask.size === 0) {
        throw new CanvasInputError('inpaint requires a non-empty mask')
      }
      break
    default:
      throw new CanvasInputError(`unsupported canvas operation: ${String(request.operation)}`)
  }
}

/** EXT_BY_MIME maps supported image MIME types to upload file extensions. */
const EXT_BY_MIME: Record<string, string> = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/webp': 'webp',
}

/**
 * sniffImageType identifies PNG/JPEG/WebP bytes by their magic headers and
 * falls back to PNG, matching the gateway's accepted formats.
 */
function sniffImageType(bytes: Uint8Array): string {
  if (
    bytes.length >= 8 &&
    bytes[0] === 0x89 &&
    bytes[1] === 0x50 &&
    bytes[2] === 0x4e &&
    bytes[3] === 0x47
  ) {
    return 'image/png'
  }
  if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) {
    return 'image/jpeg'
  }
  if (
    bytes.length >= 12 &&
    bytes[0] === 0x52 && // R
    bytes[1] === 0x49 && // I
    bytes[2] === 0x46 && // F
    bytes[3] === 0x46 && // F
    bytes[8] === 0x57 && // W
    bytes[9] === 0x45 && // E
    bytes[10] === 0x42 && // B
    bytes[11] === 0x50 // P
  ) {
    return 'image/webp'
  }
  return 'image/png'
}

/**
 * imageFileName picks an upload filename whose extension matches the actual
 * bytes, preferring the declared Blob type and sniffing the header otherwise
 * so a JPEG source is not mislabeled as .png.
 */
/**
 * readBlobRange reads a Blob slice through FileReader, which is available in
 * every environment that can display a Blob (unlike Blob.arrayBuffer, which
 * some test DOMs do not implement).
 */
function readBlobRange(blob: Blob, start: number, end: number): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as ArrayBuffer)
    reader.onerror = () => reject(reader.error ?? new Error('blob read failed'))
    reader.readAsArrayBuffer(blob.slice(start, end))
  })
}

/** blobHead reads the first bytes of a Blob for magic sniffing. */
async function blobHead(blob: Blob): Promise<Uint8Array> {
  return new Uint8Array(await readBlobRange(blob, 0, 12))
}

async function imageFileName(blob: Blob, base: string): Promise<string> {
  let mime = EXT_BY_MIME[blob.type] ? blob.type : null
  if (!mime && blob.size > 0) {
    mime = sniffImageType(await blobHead(blob))
  }
  return `${base}.${EXT_BY_MIME[mime ?? 'image/png'] ?? 'png'}`
}

/**
 * submitCanvasTask posts one generation task. Generate uses a JSON body;
 * edit/inpaint use multipart with original bytes, one image under "image" and
 * several under "image[]" to match the gateway parser. The caller-owned signal
 * aborts the request; there is no retry because a repost could bill twice.
 */
export async function submitCanvasTask(
  apiKey: string,
  request: CanvasSubmitRequest,
  signal?: AbortSignal,
): Promise<CanvasSubmitResult> {
  validateSubmitRequest(request)
  const count = request.count ?? 1

  let response: Response
  if (request.operation === 'generate') {
    response = await fetch(buildGatewayUrl('/v1/images/generations/async'), {
      method: 'POST',
      headers: authHeaders(apiKey, { 'Content-Type': 'application/json' }),
      body: JSON.stringify({
        model: request.model,
        prompt: request.prompt,
        size: request.size,
        n: count,
        ...(request.quality ? { quality: request.quality } : {}),
        ...(request.background ? { background: request.background } : {}),
      }),
      signal,
    })
  } else {
    const form = new FormData()
    form.append('model', request.model)
    form.append('prompt', request.prompt)
    form.append('size', request.size)
    form.append('n', String(count))
    if (request.quality) {
      form.append('quality', request.quality)
    }
    if (request.background) {
      form.append('background', request.background)
    }
    if (request.images.length === 1) {
      form.append('image', request.images[0], await imageFileName(request.images[0], 'canvas-0'))
    } else {
      for (const [index, image] of request.images.entries()) {
        form.append('image[]', image, await imageFileName(image, `canvas-${index}`))
      }
    }
    if (request.operation === 'inpaint' && request.mask) {
      form.append('mask', request.mask, await imageFileName(request.mask, 'mask'))
    }
    response = await fetch(buildGatewayUrl('/v1/images/edits/async'), {
      method: 'POST',
      headers: authHeaders(apiKey),
      body: form,
      signal,
    })
  }
  if (!response.ok) {
    throw await parseCanvasError(response)
  }
  return response.json()
}

/**
 * getCanvasTask polls one task by id. The signal aborts this single poll; the
 * caller decides whether the run keeps tracking afterwards. A bounded
 * internal timeout keeps a stalled gateway from pinning the poll in flight.
 */
export async function getCanvasTask(
  apiKey: string,
  taskId: string,
  signal?: AbortSignal,
): Promise<CanvasTask> {
  const controller = new AbortController()
  const onAbort = () => controller.abort(signal?.reason)
  if (signal) {
    if (signal.aborted) controller.abort(signal.reason)
    else signal.addEventListener('abort', onAbort, { once: true })
  }
  const timer = setTimeout(
    () => controller.abort(new DOMException('poll timeout', 'TimeoutError')),
    15_000,
  )
  try {
    const response = await fetch(
      buildGatewayUrl(`/v1/images/tasks/${encodeURIComponent(taskId)}`),
      { headers: authHeaders(apiKey), signal: controller.signal },
    )
    if (!response.ok) {
      throw await parseCanvasError(response)
    }
    return await response.json()
  } finally {
    clearTimeout(timer)
    signal?.removeEventListener('abort', onAbort)
  }
}

/**
 * CanvasTaskOutput is one raw result entry. The index is the entry's position
 * in the server data array, so malformed earlier entries cannot shift the
 * identity of later outputs.
 */
export interface CanvasTaskOutput {
  /** Original index in the result data array. */
  index: number
  /** Upstream URL when present. */
  url?: string
  /** Base64 payload when present. */
  b64?: string
}

/**
 * extractTaskOutputs maps the task result into one entry per data index.
 * Entries with neither url nor b64_json stay in the list (with both fields
 * undefined) so downstream harvest keeps the original indexes stable. A
 * single-object url result is also accepted for upstreams that return one.
 */
export function extractTaskOutputs(task: CanvasTask): CanvasTaskOutput[] {
  const data = task.result?.data
  if (Array.isArray(data)) {
    return data.map((item, index) => ({
      index,
      url: typeof item?.url === 'string' && item.url ? item.url : undefined,
      b64: typeof item?.b64_json === 'string' && item.b64_json ? item.b64_json : undefined,
    }))
  }
  const single = (task.result as { url?: string } | null)?.url
  return typeof single === 'string' && single ? [{ index: 0, url: single }] : []
}

/**
 * isSafeOutputUrl limits remote output fetches to credential-free http(s),
 * rejecting javascript:, data:, and embedded userinfo before they reach fetch.
 */
export function isSafeOutputUrl(url: string): boolean {
  try {
    const parsed = new URL(url)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return false
    return parsed.username === '' && parsed.password === ''
  } catch {
    return false
  }
}

/**
 * decodeBase64Blob turns a base64 payload into a Blob without a network call.
 * The MIME type is sniffed from the decoded bytes so a WebP output is not
 * mislabeled as PNG.
 */
function decodeBase64Blob(b64: string): Blob | null {
  try {
    const binary = atob(b64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i)
    }
    return new Blob([bytes], { type: sniffImageType(bytes) })
  } catch {
    return null
  }
}


/**
 * fetchTaskOutputBlob resolves one output's bytes. Base64 is preferred because
 * it needs no second request; a failed decode falls back to a safe http(s)
 * URL fetch bounded by a per-image timeout linked to the caller's signal.
 * Caller aborts propagate as AbortError so harvest can tell "cancelled by
 * rotation" apart from a genuinely unreachable URL; timeouts and network
 * failures return null.
 */
export async function fetchTaskOutputBlob(
  output: Pick<CanvasTaskOutput, 'url' | 'b64'>,
  signal?: AbortSignal,
  source?: CanvasTaskOutputSource,
): Promise<Blob | null> {
  if (signal?.aborted) {
    throw signal.reason ?? new DOMException('aborted', 'AbortError')
  }
  if (output.b64) {
    const blob = decodeBase64Blob(output.b64)
    if (blob) {
      return blob
    }
  }
  if (source !== undefined) {
    // A task source pins the download to the gateway content endpoint with the
    // task's original key. Whether it succeeds or fails we never touch the
    // external URL: HTTPS pages cannot reach plain-HTTP storage and the API key
    // must never be sent off-origin.
    return fetchGatewayOutput(source, signal)
  }
  if (output.url && isSafeOutputUrl(output.url)) {
    const controller = new AbortController()
    const onAbort = () => controller.abort(signal?.reason)
    if (signal) {
      if (signal.aborted) controller.abort(signal.reason)
      else signal.addEventListener('abort', onAbort, { once: true })
    }
    const timer = setTimeout(
      () => controller.abort(new DOMException('output fetch timeout', 'TimeoutError')),
      OUTPUT_FETCH_TIMEOUT_MS,
    )
    try {
      const response = await fetch(output.url, { signal: controller.signal })
      if (response.ok) {
        const blob = await response.blob()
        if (blob.type) return blob
        // No declared type: sniff the header and re-wrap with the real MIME.
        const buffer = await readBlobRange(blob, 0, blob.size)
        return new Blob([buffer], { type: sniffImageType(new Uint8Array(buffer.slice(0, 12))) })
      }
      return null
    } catch (error) {
      // Only a caller-side abort propagates; a local timeout or any other
      // failure reports null so the output is marked remote/missing.
      if (signal?.aborted) throw error
      return null
    } finally {
      clearTimeout(timer)
      signal?.removeEventListener('abort', onAbort)
    }
  }
  return null
}

/**
 * CanvasTaskOutputSource identifies which stored task output a remote URL came
 * from so fetchTaskOutputBlob can fetch it through the same-origin gateway
 * proxy (the backend downloads the origin bytes under the SSRF guard).
 */
export interface CanvasTaskOutputSource {
  /** The frozen key that submitted the run — never the currently selected key. */
  apiKey: string
  taskId: string
  index: number
}

function isValidOutputSource(source: CanvasTaskOutputSource): boolean {
  return (
    typeof source.apiKey === 'string' &&
    source.apiKey.length > 0 &&
    typeof source.taskId === 'string' &&
    source.taskId.trim().length > 0 &&
    Number.isInteger(source.index) &&
    source.index >= 0 &&
    source.index <= Number.MAX_SAFE_INTEGER
  )
}

async function fetchGatewayOutput(
  source: CanvasTaskOutputSource,
  signal?: AbortSignal,
): Promise<Blob | null> {
  if (!isValidOutputSource(source)) {
    return null
  }
  const controller = new AbortController()
  const onAbort = () => controller.abort(signal?.reason)
  if (signal) {
    if (signal.aborted) controller.abort(signal.reason)
    else signal.addEventListener('abort', onAbort, { once: true })
  }
  const timer = setTimeout(
    () => controller.abort(new DOMException('output fetch timeout', 'TimeoutError')),
    OUTPUT_FETCH_TIMEOUT_MS,
  )
  try {
    const response = await fetch(
      buildGatewayUrl(`/v1/images/tasks/${encodeURIComponent(source.taskId)}/outputs/${source.index}/content`),
      {
        headers: authHeaders(source.apiKey, { Accept: 'image/*' }),
        signal: controller.signal,
      },
    )
    if (!response.ok) {
      return null
    }
    return await response.blob()
  } catch (error) {
    // Caller aborts still propagate; local timeouts and failures report null.
    if (signal?.aborted) throw error
    return null
  } finally {
    clearTimeout(timer)
    signal?.removeEventListener('abort', onAbort)
  }
}
