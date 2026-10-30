<template>
  <div
    ref="rootEl"
    class="canvas-editor"
    :class="{ 'is-panning': panning }"
    @dragover.prevent="onDragOver"
    @drop.prevent="onDrop"
    @contextmenu.prevent
  >
    <canvas ref="mainCanvasEl" />
    <canvas ref="overlayEl" class="mask-layer" />
    <input ref="fileInput" type="file" class="hidden" multiple accept="image/png,image/jpeg,image/webp" @change="onFilesPicked" />
  </div>
</template>

<script setup lang="ts">
/**
 * CanvasEditor owns the infinite Fabric workspace: multi-image scene,
 * per-image masks in natural pixel coordinates, undo/redo, scoped scene
 * persistence, and import/export. Fabric internals never leave the file.
 *
 * Mask invariant: strokes are sampled points in natural image coordinates.
 * The native overlay maps client pixels through
 *   inverse(camera * imageWorldMatrix * translate(-w/2, -h/2))
 * so strokes follow their anchor through move/scale/rotate, and export
 * renders untouched-opaque / painted-transparent at source resolution.
 */
import { onBeforeUnmount, onMounted, ref, watch, type PropType } from 'vue'
import {
  util,
  Canvas as FabricCanvas,
  FabricImage,
  IText,
  type FabricObject,
  type TPointerEventInfo,
} from 'fabric'
import {
  getWorkspaceId,
  loadAsset,
  loadScene,
  saveAsset,
  saveScene,
} from '@/utils/canvasLocalStore'
import {
  applyMatrix,
  clientToScene,
  emptyBounds,
  extendBounds,
  fitViewport,
  invertMatrix,
  isEditableTarget,
  maskViewportMatrix,
  multiplyMatrices,
  translation,
  zoomAtClient,
  CANVAS_ZOOM_MIN,
  CANVAS_ZOOM_MAX,
  type Bounds,
  type Mat6,
  type Vec2,
} from '@/utils/canvasGeometry'
import {
  CANVAS_SCENE_VERSION,
  decodeSceneSnapshot,
  encodeSceneSnapshot,
  maskHasCoverage,
  placementIdentity,
  pointInImageLocal,
  type SceneObject,
  type SceneSnapshot,
} from '@/utils/canvasScene'
import {
  hasRedrawArea,
  renderMaskExport,
  renderMaskPreview,
} from '@/utils/canvasMask'
import type {
  CanvasBrushShape,
  CanvasScope,
  CanvasTool,
  CropRequestPayload,
  EditorStateSnapshot,
  ExportInputResult,
  PlaceOutputInput,
} from './types'
import type { CanvasOperation } from '@/types/canvas'

const SCENE_KEY = 'default'
const MAX_FILE_BYTES = 50 * 1024 * 1024
const MAX_IMAGE_PIXELS = 40_000_000
const MAX_IMAGE_EDGE = 16_384
const MASK_ALPHA = 0.45
const UNDO_LIMIT = 50
const SAVE_DEBOUNCE_MS = 600
const IMAGE_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])
const EXT_BY_MIME: Record<string, string> = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/webp': 'webp',
}
const PLACE_GAP = 40
/** Minimum pointer distance between recorded mask samples, natural px. */
const STROKE_DECIMATE = 0.5

const props = defineProps({
  scope: { type: Object as PropType<CanvasScope | null>, required: true },
  operation: { type: String as PropType<CanvasOperation>, default: 'generate' },
  tool: { type: String as PropType<CanvasTool>, default: 'select' },
  brushSize: { type: Number, default: 40 },
  brushShape: { type: String as PropType<CanvasBrushShape>, default: 'round' },
  textColor: { type: String, default: '#1f2937' },
  textSize: { type: Number, default: 24 },
  /** Locale string for new IText labels (localized by the view). */
  defaultText: { type: String, default: 'Text' },
  /** Optional crop-queue seam; when set, picked files are forwarded, not imported. */
  cropHandler: { type: Function as PropType<((payload: CropRequestPayload) => void) | null>, default: null },
})

const emit = defineEmits<{
  state: [snapshot: EditorStateSnapshot]
  error: [code: string]
  'update:tool': [tool: CanvasTool]
  'crop-request': [payload: CropRequestPayload]
}>()

const rootEl = ref<HTMLDivElement | null>(null)
const mainCanvasEl = ref<HTMLCanvasElement | null>(null)
const overlayEl = ref<HTMLCanvasElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const panning = ref(false)

interface Entry {
  /** Live metadata reference stored on the Fabric object — never a copy. */
  data: SceneObject
  obj: FabricObject
  /** Owning group (ActiveSelection) for world-matrix resolution. */
  group: FabricObject | null
}

let canvas: FabricCanvas | null = null
let overlayCtx: CanvasRenderingContext2D | null = null
let maskBuffer: HTMLCanvasElement | null = null
let resizeObserver: ResizeObserver | null = null
let generation = 0
let ready = false
let restoring = false
/** Set when a stored scene failed to load/decode: blocks every write path
 * (autosave, flush, unmount, pagehide) until reset() clears it. */
let saveBlocked = false
let sceneRev = 0
let savedRev = -1
let opQueue: Promise<unknown> = Promise.resolve()
let saveTimer: ReturnType<typeof setTimeout> | null = null
let undoStack: string[] = []
let redoStack: string[] = []
let undoBusy = false
let preGestureSnapshot: string | null = null
let gestureSeen = false
let textEditSnapshot: string | null = null
interface ActiveStroke {
  /** Anchor object reference — entries re-resolve on every use because an
   * Entry's group matrix is stale after selection changes. */
  anchorObj: FabricObject
  points: Vec2[]
  pointerId: number
  mode: 'add' | 'erase'
  shape: CanvasBrushShape
  width: number
}
let activeStroke: ActiveStroke | null = null

/** entryFor re-resolves a live entry for a Fabric object. */
function entryFor(obj: FabricObject): Entry | null {
  return collectEntries().find((e) => e.obj === obj) ?? null
}
let spaceDown = false
let panPointer: Vec2 | null = null
let pinch: { d: number; mid: Vec2 } | null = null
const assetBlobs = new Map<string, Blob>()
const assetUrls = new Map<string, string>()

function newId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `o-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

function emitError(code: string): void {
  emit('error', code)
}

/**
 * collectEntries flattens the canvas into live metadata entries. Grouped
 * (multi-selected) objects keep their own data plus a group reference for
 * world-matrix resolution — mutation through `entry.data` reaches the real
 * object, never a snapshot copy.
 */
function collectEntries(): Entry[] {
  if (!canvas) return []
  const out: Entry[] = []
  const walk = (obj: FabricObject, group: FabricObject | null) => {
    const data = (obj as unknown as { editorData?: SceneObject }).editorData
    if (data) out.push({ data, obj, group })
    const children = (obj as unknown as { getObjects?: () => FabricObject[] }).getObjects?.()
    if (children) for (const child of children) walk(child, group ?? obj)
  }
  for (const obj of canvas.getObjects()) walk(obj, null)
  return out
}

function imageEntries(): Entry[] {
  return collectEntries().filter((e) => e.data.role === 'image')
}

/** worldMatrix returns the scene-space transform including any group. */
function worldMatrix(entry: Entry): Mat6 {
  const own = entry.obj.calcTransformMatrix() as Mat6
  if (!entry.group) return own
  return multiplyMatrices(entry.group.calcTransformMatrix() as Mat6, own)
}

function attachData(obj: FabricObject, data: SceneObject): void {
  const host = obj as unknown as { editorData?: SceneObject }
  host.editorData = data
}

function maskAnchorEntry(): Entry | null {
  if (!canvas) return null
  const actives = canvas.getActiveObjects()
  const hits = imageEntries().filter((e) => actives.includes(e.obj as never))
  return hits.length === 1 ? hits[0] : null
}

/* ---------------- serialization ---------------- */

function serializeScene(): SceneSnapshot {
  if (!canvas) return { version: CANVAS_SCENE_VERSION, objects: [] }
  const objects = collectEntries().map(({ data }) => {
    const snapshot: SceneObject = {
      ...data,
      transform: undefined,
      maskStrokes: data.maskStrokes?.map((s) => ({ ...s, points: s.points.map((p) => ({ ...p })) })),
    }
    return snapshot
  })
  // World transforms are assigned per live entry, not via the spread copy.
  collectEntries().forEach((entry, i) => {
    objects[i].transform = [...worldMatrix(entry)] as Mat6
    if (entry.data.role === 'image' && entry.data.assetKey) {
      objects[i].src = `asset://${entry.data.assetKey}`
    }
  })
  return {
    version: CANVAS_SCENE_VERSION,
    viewport: [...(canvas.viewportTransform as Mat6)] as Mat6,
    objects,
  }
}

function pushUndo(): void {
  undoStack.push(encodeSceneSnapshot(serializeScene()))
  if (undoStack.length > UNDO_LIMIT) undoStack.shift()
  redoStack = []
  emitState()
}

/** enqueue serializes scene mutations; a failure cannot wedge the queue. */
function enqueue<T>(work: () => Promise<T> | T): Promise<T> {
  const job = opQueue.then(work)
  opQueue = job.catch(() => {})
  return job
}

function markDirty(): void {
  if (restoring) return
  sceneRev += 1
  emitState()
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => void persistScene(), SAVE_DEBOUNCE_MS)
}

/**
 * persistSceneNow performs the write inline — it must only run inside an
 * already-serialized context (queue job or post-queue), never nested inside
 * enqueue, or it would deadlock the chain it waits on.
 */
async function persistSceneNow(scope: CanvasScope, gen: number): Promise<void> {
  if (gen !== generation || !canvas || saveBlocked) return
  const rev = sceneRev
  const snapshot = encodeSceneSnapshot(serializeScene())
  try {
    await saveScene(scope, SCENE_KEY, snapshot)
  } catch {
    if (gen === generation) emitError('scene-save-failed')
    return
  }
  if (gen === generation && sceneRev === rev) savedRev = rev
}

/** persistScene captures the snapshot AFTER queued mutations settle. */
function persistScene(): Promise<void> {
  const scope = props.scope
  if (!scope) return Promise.resolve()
  const gen = generation
  return enqueue(() => persistSceneNow(scope, gen))
}

function sceneDirty(): boolean {
  return sceneRev !== savedRev
}

/* ---------------- asset blobs / urls ---------------- */

function assetUrl(assetKey: string, blob: Blob): string {
  const existing = assetUrls.get(assetKey)
  if (existing) return existing
  const url = URL.createObjectURL(blob)
  assetUrls.set(assetKey, url)
  assetBlobs.set(assetKey, blob)
  return url
}

function referencedAssetKeys(): Set<string> {
  const keys = new Set<string>()
  for (const e of imageEntries()) if (e.data.assetKey) keys.add(e.data.assetKey)
  for (const raw of [...undoStack, ...redoStack]) {
    const snap = decodeSceneSnapshot(raw)
    if (snap) for (const o of snap.objects) if (o.assetKey) keys.add(o.assetKey)
  }
  return keys
}

/** pruneAssetUrls revokes URLs no longer referenced by scene or history. */
function pruneAssetUrls(): void {
  const live = referencedAssetKeys()
  for (const [key, url] of assetUrls) {
    if (!live.has(key)) {
      URL.revokeObjectURL(url)
      assetUrls.delete(key)
      assetBlobs.delete(key)
    }
  }
}

async function blobForAsset(assetKey: string): Promise<Blob | null> {
  const cached = assetBlobs.get(assetKey)
  if (cached) return cached
  const scope = props.scope
  if (!scope) return null
  const asset = await loadAsset(scope, assetKey).catch(() => null)
  if (!asset) return null
  assetBlobs.set(assetKey, asset.blob)
  return asset.blob
}

/* ---------------- images ---------------- */

async function addImageObject(
  blob: Blob,
  opts: Partial<SceneObject> & { at?: Vec2 | null; select?: boolean } = {},
): Promise<Entry> {
  if (!canvas) throw new Error('disposed')
  const gen = generation
  const bitmap = await createImageBitmap(blob)
  const naturalW = bitmap.width
  const naturalH = bitmap.height
  if (
    !Number.isFinite(naturalW) ||
    !Number.isFinite(naturalH) ||
    naturalW <= 0 ||
    naturalH <= 0 ||
    naturalW * naturalH > MAX_IMAGE_PIXELS ||
    Math.max(naturalW, naturalH) > MAX_IMAGE_EDGE
  ) {
    bitmap.close()
    throw new Error('image-limits')
  }
  if (gen !== generation || !canvas) {
    bitmap.close()
    throw new Error('stale')
  }
  const scope = props.scope
  let assetKey = opts.assetKey
  if (!assetKey && scope) {
    assetKey = `source:${newId()}`
    try {
      await saveAsset(scope, { key: assetKey, kind: 'source', blob, createdAt: Date.now() })
    } catch {
      // Quota failure keeps the blob in the session cache so the scene stays
      // editable and undoable; persistence protection is best-effort only.
      emitError('asset-save-failed')
    }
  }
  if (gen !== generation || !canvas) {
    bitmap.close()
    throw new Error('stale')
  }
  if (assetKey) assetBlobs.set(assetKey, blob)
  const url = assetUrl(assetKey ?? newId(), blob)
  const img = await FabricImage.fromURL(url)
  bitmap.close()
  if (gen !== generation || !canvas) throw new Error('stale')
  const data: SceneObject = {
    role: 'image',
    objectId: opts.objectId ?? newId(),
    assetKey,
    src: assetKey ? `asset://${assetKey}` : undefined,
    runId: opts.runId,
    outputIndex: opts.outputIndex,
    left: opts.left ?? 0,
    top: opts.top ?? 0,
    scaleX: opts.scaleX ?? 1,
    scaleY: opts.scaleY ?? 1,
    angle: opts.angle ?? 0,
    width: naturalW,
    height: naturalH,
    maskStrokes: opts.maskStrokes ?? [],
  }
  if (opts.transform) {
    util.applyTransformToObject(img, opts.transform as Mat6)
    img.setCoords()
  } else {
    const viewW = canvas.getWidth()
    const viewH = canvas.getHeight()
    const fit = Math.min((0.7 * viewW) / naturalW, (0.7 * viewH) / naturalH, 1)
    const center =
      opts.at ??
      clientToScene(canvas.viewportTransform as Mat6, { x: viewW / 2, y: viewH / 2 })
    // Fabric uses center origins: left/top name the object's centre point.
    img.set({
      left: center.x,
      top: center.y,
      scaleX: fit,
      scaleY: fit,
    })
  }
  attachData(img, data)
  canvas.add(img)
  if (opts.select !== false) canvas.setActiveObject(img)
  canvas.requestRenderAll()
  markDirty()
  return { data, obj: img, group: null }
}

function addTextObject(at?: Vec2, from?: Partial<SceneObject>): void {
  if (!canvas || (!from && (!ready || restoring))) return
  const center =
    at ??
    clientToScene(canvas.viewportTransform as Mat6, {
      x: canvas.getWidth() / 2,
      y: canvas.getHeight() / 2,
    })
  const text = new IText(from?.text ?? props.defaultText, {
    fontSize: from?.fontSize ?? props.textSize,
    fill: from?.fill ?? props.textColor,
    fontFamily: 'system-ui, sans-serif',
  })
  const data: SceneObject = {
    role: 'text',
    objectId: from?.objectId ?? newId(),
    left: center.x,
    top: center.y,
    scaleX: 1,
    scaleY: 1,
    angle: 0,
    text: text.text,
    fontSize: text.fontSize,
    fill: typeof text.fill === 'string' ? text.fill : props.textColor,
  }
  text.set({ left: data.left, top: data.top })
  if (from?.transform) util.applyTransformToObject(text, from.transform as Mat6)
  attachData(text, data)
  if (!from) pushUndo()
  canvas.add(text)
  canvas.setActiveObject(text)
  if (!from) text.enterEditing()
  canvas.requestRenderAll()
  markDirty()
}

/* ---------------- scene restore ---------------- */

function validNumber(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

async function applySnapshot(snapshot: SceneSnapshot, gen: number): Promise<void> {
  if (!canvas) return
  const failures: string[] = []
  for (const obj of snapshot.objects) {
    if (gen !== generation || !canvas) return
    try {
      if (obj.role === 'image' && obj.assetKey) {
        const blob = await blobForAsset(obj.assetKey)
        if (!blob || gen !== generation || !canvas) {
          // Missing/corrupt bytes: skip the object, keep the rest of the scene.
          if (!blob) failures.push(obj.assetKey)
          continue
        }
        const entry = await addImageObject(blob, {
          objectId: obj.objectId,
          assetKey: obj.assetKey,
          runId: obj.runId,
          outputIndex: obj.outputIndex,
          maskStrokes: obj.maskStrokes,
          transform: obj.transform ?? [
            1, 0, 0, 1,
            validNumber(obj.left, 0),
            validNumber(obj.top, 0),
          ],
          select: false,
        })
        if (!obj.transform) {
          // Legacy record: keep decomposed fields.
          entry.obj.set({
            left: validNumber(obj.left, 0),
            top: validNumber(obj.top, 0),
            scaleX: validNumber(obj.scaleX, 1),
            scaleY: validNumber(obj.scaleY, 1),
            angle: validNumber(obj.angle, 0),
          })
        }
      } else if (obj.role === 'text' && typeof obj.text === 'string') {
        addTextObject(
          { x: validNumber(obj.left, 0), y: validNumber(obj.top, 0) },
          { ...obj, transform: obj.transform },
        )
      }
    } catch {
      failures.push(obj.objectId)
    }
  }
  if (failures.length > 0 && gen === generation) emitError('asset-missing')
  canvas?.requestRenderAll()
}

async function restoreScene(): Promise<void> {
  const scope = props.scope
  if (!scope || !canvas) return
  const gen = generation
  await enqueue(async () => {
    if (gen !== generation || !canvas) return
    restoring = true
    try {
      const record = await loadScene(scope, SCENE_KEY)
      if (gen !== generation || !canvas) return
      const snapshot = record ? decodeSceneSnapshot(record.json) : null
      if (record && !snapshot) {
        // A corrupt record must never be overwritten by autosave.
        saveBlocked = true
        emitError('scene-load-failed')
      }
      if (snapshot) {
        await applySnapshot(snapshot, gen)
        if (gen === generation && canvas && snapshot.viewport) {
          canvas.setViewportTransform(snapshot.viewport)
        }
      }
      if (gen === generation) {
        savedRev = sceneRev // restored state is the durable baseline
        ready = true
      }
    } catch {
      // A corrupt/unreadable record never overwrites the saved scene.
      if (gen === generation) {
        saveBlocked = true
        ready = true
        emitError('scene-load-failed')
      }
    } finally {
      restoring = false
      canvas?.requestRenderAll()
      emitState()
    }
  })
}

/* ---------------- mask: native overlay ---------------- */

function maskBufferCtx(w: number, h: number): CanvasRenderingContext2D | null {
  if (!maskBuffer || maskBuffer.width !== w || maskBuffer.height !== h) {
    maskBuffer = document.createElement('canvas')
    maskBuffer.width = w
    maskBuffer.height = h
  }
  return maskBuffer.getContext('2d')
}

/** scheduleMaskRender repaints the overlay at the next frame. */
function scheduleMaskRender(): void {
  requestAnimationFrame(renderOverlay)
}

function renderOverlay(): void {
  if (!overlayEl.value || !canvas || !overlayCtx) return
  const el = overlayEl.value
  const dpr = window.devicePixelRatio || 1
  overlayCtx.setTransform(dpr, 0, 0, dpr, 0, 0)
  overlayCtx.clearRect(0, 0, el.clientWidth, el.clientHeight)
  const anchor = maskAnchorEntry()
  if (!anchor?.data.width || !anchor.data.height) return
  renderMaskBuffer(anchor.data)
  if (!maskBuffer) return
  const vpt = maskViewportMatrix(
    canvas.viewportTransform as Mat6,
    worldMatrix(anchor),
    anchor.data.width,
    anchor.data.height,
  )
  overlayCtx.save()
  overlayCtx.globalAlpha = MASK_ALPHA
  overlayCtx.transform(
    dpr * vpt[0], dpr * vpt[1],
    dpr * vpt[2], dpr * vpt[3],
    dpr * vpt[4], dpr * vpt[5],
  )
  overlayCtx.drawImage(maskBuffer, 0, 0)
  overlayCtx.restore()
}

function renderMaskBuffer(data: SceneObject): void {
  const w = data.width ?? 0
  const h = data.height ?? 0
  const ctx = w > 0 && h > 0 ? maskBufferCtx(w, h) : null
  if (ctx) renderMaskPreview(ctx, data.maskStrokes ?? [], w, h)
}

/** toNatural maps a client point into the anchor's natural pixel space. */
function toNatural(entry: Entry, client: Vec2): Vec2 {
  const scene = clientToScene(canvas!.viewportTransform as Mat6, client)
  const toNat = invertMatrix(
    multiplyMatrices(worldMatrix(entry), translation(-entry.data.width! / 2, -entry.data.height! / 2)),
  )
  return applyMatrix(toNat, scene)
}

function clientPoint(e: PointerEvent | MouseEvent): Vec2 {
  const rect = rootEl.value!.getBoundingClientRect()
  return { x: e.clientX - rect.left, y: e.clientY - rect.top }
}

function pickImageAt(client: Vec2): Entry | null {
  const scene = clientToScene(canvas!.viewportTransform as Mat6, client)
  for (const entry of [...imageEntries()].reverse()) {
    const w = entry.data.width
    const h = entry.data.height
    if (!w || !h) continue
    const toNat = invertMatrix(
      multiplyMatrices(worldMatrix(entry), translation(-w / 2, -h / 2)),
    )
    if (pointInImageLocal(applyMatrix(toNat, scene), w, h)) return entry
  }
  return null
}

function onOverlayPointerDown(e: PointerEvent): void {
  if (!canvas) return
  // The overlay also owns panning: space, middle button, or a press that
  // misses every image pans instead of brushing — pan works in any tool.
  const wantPan = spaceDown || e.button === 1 || props.tool === 'pan'
  const hit =
    props.tool === 'brush' || props.tool === 'eraser' ? pickImageAt(clientPoint(e)) : null
  if (wantPan || !hit) {
    panning.value = true
    panPointer = { x: e.clientX, y: e.clientY }
    const overlay = overlayEl.value
    if (overlay && typeof overlay.setPointerCapture === 'function') {
      overlay.setPointerCapture(e.pointerId)
    }
    e.preventDefault()
    return
  }
  e.preventDefault()
  canvas.discardActiveObject()
  canvas.setActiveObject(hit.obj)
  canvas.requestRenderAll()
  // Re-resolve AFTER the discard: the old entry's group matrix is stale.
  const fresh = entryFor(hit.obj) ?? hit
  pushUndo()
  activeStroke = {
    anchorObj: fresh.obj,
    points: [toNatural(fresh, clientPoint(e))],
    pointerId: e.pointerId,
    mode: props.tool === 'eraser' ? 'erase' : 'add',
    shape: props.brushShape,
    width: props.brushSize,
  }
  const overlay = overlayEl.value
  if (overlay && typeof overlay.setPointerCapture === 'function') {
    overlay.setPointerCapture(e.pointerId)
  }
}

function onOverlayPointerMove(e: PointerEvent): void {
  if (panning.value && panPointer) {
    const dx = e.clientX - panPointer.x
    const dy = e.clientY - panPointer.y
    panPointer = { x: e.clientX, y: e.clientY }
    const vpt = [...canvas!.viewportTransform] as Mat6
    vpt[4] += dx
    vpt[5] += dy
    canvas!.setViewportTransform(vpt)
    canvas!.requestRenderAll()
    scheduleMaskRender()
    return
  }
  const stroke = activeStroke
  if (!stroke || e.pointerId !== stroke.pointerId) return
  const anchor = entryFor(stroke.anchorObj)
  if (!anchor) return
  const p = toNatural(anchor, clientPoint(e))
  const last = stroke.points[stroke.points.length - 1]
  if (Math.hypot(p.x - last.x, p.y - last.y) >= STROKE_DECIMATE) {
    stroke.points.push(p)
    // Live preview: tentative stroke included.
    const ctx = maskBufferCtx(anchor.data.width!, anchor.data.height!)
    if (ctx) {
      renderMaskPreview(
        ctx,
        [
          ...(anchor.data.maskStrokes ?? []),
          { mode: stroke.mode, shape: stroke.shape, width: stroke.width, points: stroke.points },
        ],
        anchor.data.width!,
        anchor.data.height!,
      )
    }
    scheduleMaskRender()
  }
}

function onOverlayPointerEnd(e: PointerEvent): void {
  if (panning.value) {
    panning.value = false
    panPointer = null
    markDirty()
    return
  }
  const stroke = activeStroke
  if (!stroke || e.pointerId !== stroke.pointerId) return
  activeStroke = null
  const anchor = entryFor(stroke.anchorObj)
  if (!anchor) return
  anchor.data.maskStrokes = [
    ...(anchor.data.maskStrokes ?? []),
    { mode: stroke.mode, shape: stroke.shape, width: stroke.width, points: stroke.points },
  ]
  scheduleMaskRender()
  markDirty()
}

function updateOverlayEvents(): void {
  const el = overlayEl.value
  if (!el) return
  el.style.pointerEvents = props.tool === 'brush' || props.tool === 'eraser' ? 'auto' : 'none'
}

/* ---------------- import ---------------- */

function openFilePicker(): void {
  fileInput.value?.click()
}

function onFilesPicked(e: Event): void {
  const input = e.target as HTMLInputElement
  const files = [...(input.files ?? [])]
  input.value = ''
  void routeImport(files.map((f) => f as Blob), null, true)
}

function validateBlob(blob: Blob): boolean {
  if (!IMAGE_TYPES.has(blob.type)) {
    emitError('import-type')
    return false
  }
  if (blob.size > MAX_FILE_BYTES) {
    emitError('file-too-large')
    return false
  }
  return true
}

async function routeImport(blobs: Blob[], at: Vec2 | null, viaCrop = false): Promise<void> {
  const valid = blobs.filter(validateBlob)
  if (valid.length === 0) return
  if (viaCrop) {
    // The file-picker path routes through the crop queue (handler owns the
    // blobs when set, otherwise the event is the seam); drops and pastes
    // never reach this — they import straight below.
    emit('crop-request', { blobs: valid, at })
    if (props.cropHandler) {
      props.cropHandler({ blobs: valid, at })
      return
    }
  }
  await importBlobs(valid, at)
}

async function importBlobs(blobs: Blob[], at: Vec2 | null = null): Promise<void> {
  const gen = generation
  await enqueue(async () => {
    if (gen !== generation || !canvas) return
    pushUndo()
    for (const blob of blobs) {
      if (gen !== generation || !canvas) return
      try {
        await addImageObject(blob, { at })
      } catch (error) {
        if (gen === generation) {
          emitError((error as Error).message === 'image-limits' ? 'file-too-large' : 'import-failed')
        }
      }
    }
  })
}

/* ---------------- placement / export ---------------- */

async function placeOutput(input: PlaceOutputInput): Promise<void> {
  const gen = generation
  const sc = props.scope
  await enqueue(async () => {
    if (gen !== generation || !canvas || !sc) throw new Error('stale')
    const identity = `${input.runId}:${input.outputIndex}`
    // Dedup inside the queue: concurrent deliveries cannot both pass.
    if (imageEntries().some((e) => placementIdentity(e.data) === identity)) return
    try {
      const bounds = sceneBounds()
      const at =
        input.at ??
        (bounds ? { x: bounds.maxX + PLACE_GAP, y: bounds.minY } : null)
      const brushing = props.tool === 'brush' || props.tool === 'eraser'
      await addImageObject(input.blob, {
        at,
        assetKey: input.assetKey,
        runId: input.runId,
        outputIndex: input.outputIndex,
        // Keep the user's active mask anchor selected while brushing.
        select: !brushing,
      })
      await persistSceneNow(sc, gen)
      if (sceneDirty()) throw new Error('scene-unsaved')
    } catch (error) {
      emitError('place-failed')
      throw error
    }
  })
}

function sceneBounds(): Bounds | null {
  const b = collectEntries().reduce((acc, entry) => {
    const r = entry.obj.getBoundingRect()
    const g = entry.group?.getBoundingRect()
    const rect = g ?? r
    return extendBounds(acc, {
      minX: rect.left,
      minY: rect.top,
      maxX: rect.left + rect.width,
      maxY: rect.top + rect.height,
    })
  }, emptyBounds() as Bounds)
  return Number.isFinite(b.minX) ? b : null
}

function selectedEntries(role?: string): Entry[] {
  if (!canvas) return []
  const actives = canvas.getActiveObjects()
  return collectEntries().filter(
    (e) => actives.includes(e.obj as never) && (!role || e.data.role === role),
  )
}

async function exportInput(operation: CanvasOperation): Promise<ExportInputResult> {
  const gen = generation
  return enqueue(async () => {
    if (gen !== generation || !canvas || !ready) throw new Error('stale')
    if (operation === 'generate') return { images: [], mask: null }
    // Explicit edit/inpaint consumes the SELECTED sources only — no silent
    // fallback to every image on the board.
    const targets = selectedEntries('image')
    if (operation === 'edit' && targets.length === 0) {
      emitError('missing-source')
      throw new Error('missing-source')
    }
    if (operation === 'inpaint' && targets.length !== 1) {
      emitError('inpaint-needs-one')
      throw new Error('inpaint-needs-one')
    }
    const images: Blob[] = []
    for (const entry of targets) {
      if (!entry.data.assetKey) {
        emitError('missing-asset')
        throw new Error('missing-asset')
      }
      const blob = await blobForAsset(entry.data.assetKey)
      if (!blob || gen !== generation) {
        emitError('missing-asset')
        throw new Error('missing-asset')
      }
      images.push(blob)
    }
    let mask: Blob | null = null
    if (operation === 'inpaint') {
      const anchor = targets[0].data
      const w = anchor.width ?? 0
      const h = anchor.height ?? 0
      if (!maskHasCoverage(anchor.maskStrokes) || !w || !h) {
        emitError('mask-required')
        throw new Error('mask-required')
      }
      const buffer = document.createElement('canvas')
      buffer.width = w
      buffer.height = h
      const ctx = buffer.getContext('2d')
      if (!ctx) {
        emitError('mask-export-failed')
        throw new Error('mask-export-failed')
      }
      renderMaskExport(ctx, anchor.maskStrokes ?? [], w, h)
      // Fully-erased masks (no surviving painted pixel) reject locally.
      if (!hasRedrawArea(ctx, w, h)) {
        emitError('mask-required')
        throw new Error('mask-required')
      }
      mask = await new Promise<Blob | null>((res) => buffer.toBlob(res, 'image/png'))
      if (!mask) {
        emitError('mask-export-failed')
        throw new Error('mask-export-failed')
      }
    }
    return { images, mask }
  })
}

/* ---------------- object actions ---------------- */

async function removeSelected(): Promise<void> {
  if (!canvas || !ready || restoring) return
  const gen = generation
  await enqueue(async () => {
    if (!canvas || gen !== generation) return
    const targets = canvas.getActiveObjects()
    if (targets.length === 0) return
    pushUndo()
    for (const obj of targets) canvas.remove(obj)
    canvas.discardActiveObject()
    canvas.requestRenderAll()
    markDirty()
    pruneAssetUrls()
  })
}

function clearMask(): void {
  if (!canvas || !ready || restoring) return
  const gen = generation
  void enqueue(async () => {
    if (!canvas || gen !== generation) return
    const anchor = maskAnchorEntry()
    if (!anchor) return
    pushUndo()
    anchor.data.maskStrokes = []
    scheduleMaskRender()
    markDirty()
  })
}

async function flattenSelected(): Promise<void> {
  if (!canvas) return
  const gen = generation
  await enqueue(async () => {
    if (gen !== generation || !canvas) return
    const targets = selectedEntries()
    if (targets.length === 0) return
    const bounds = targets.reduce((acc, entry) => {
      const m = worldMatrix(entry)
      const w = entry.data.width ?? entry.obj.width * entry.obj.scaleX
      const h = entry.data.height ?? entry.obj.height * entry.obj.scaleY
      let b = acc
      for (const p of [
        { x: -w / 2, y: -h / 2 }, { x: w / 2, y: -h / 2 },
        { x: -w / 2, y: h / 2 }, { x: w / 2, y: h / 2 },
      ]) {
        const s = applyMatrix(m, p)
        b = extendBounds(b, { minX: s.x, minY: s.y, maxX: s.x, maxY: s.y })
      }
      return b
    }, emptyBounds() as Bounds)
    const w = Math.ceil(bounds.maxX - bounds.minX)
    const h = Math.ceil(bounds.maxY - bounds.minY)
    if (w <= 0 || h <= 0 || w * h > MAX_IMAGE_PIXELS || Math.max(w, h) > MAX_IMAGE_EDGE) {
      emitError('flatten-too-large')
      return
    }
    const off = document.createElement('canvas')
    off.width = w
    off.height = h
    const ctx = off.getContext('2d')
    if (!ctx) return
    // Render each selected object at its world transform — the live camera
    // is never touched, and nothing outside the bounds is captured.
    ctx.setTransform(1, 0, 0, 1, -bounds.minX, -bounds.minY)
    for (const entry of targets) {
      ctx.save()
      const m = worldMatrix(entry)
      ctx.transform(m[0], m[1], m[2], m[3], m[4], m[5])
      ;(entry.obj as unknown as { _render: (c: CanvasRenderingContext2D) => void })._render(ctx)
      ctx.restore()
    }
    const blob = await new Promise<Blob | null>((res) => off.toBlob(res, 'image/png'))
    if (!blob || gen !== generation || !canvas) {
      emitError('flatten-failed')
      return
    }
    pushUndo()
    for (const entry of targets) canvas.remove(entry.obj)
    canvas.discardActiveObject()
    await addImageObject(blob, {
      at: { x: (bounds.minX + bounds.maxX) / 2, y: (bounds.minY + bounds.maxY) / 2 },
      scaleX: 1,
      scaleY: 1,
    })
  })
}

async function downloadSelected(): Promise<void> {
  for (const entry of selectedEntries('image')) {
    if (!entry.data.assetKey) continue
    const blob = await blobForAsset(entry.data.assetKey)
    if (!blob) continue
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `canvas-${entry.data.objectId.slice(0, 8)}.${EXT_BY_MIME[blob.type] ?? 'png'}`
    a.click()
    setTimeout(() => URL.revokeObjectURL(url), 10_000)
  }
}

/* ---------------- undo ---------------- */

async function applyUndoSnapshot(raw: string): Promise<void> {
  const snapshot = decodeSceneSnapshot(raw)
  if (!snapshot || !canvas) return
  const gen = generation
  restoring = true
  try {
    const live = collectEntries()
    for (const entry of live) canvas.remove(entry.obj)
    canvas.discardActiveObject()
    await applySnapshot(snapshot, gen)
  } finally {
    restoring = false
    canvas.requestRenderAll()
  }
  scheduleMaskRender()
  markDirty()
}

async function undo(): Promise<void> {
  if (undoBusy || undoStack.length === 0) return
  undoBusy = true
  try {
    await enqueue(async () => {
      const raw = undoStack.pop()!
      redoStack.push(encodeSceneSnapshot(serializeScene()))
      await applyUndoSnapshot(raw)
    })
  } finally {
    undoBusy = false
    emitState()
  }
}

async function redo(): Promise<void> {
  if (undoBusy || redoStack.length === 0) return
  undoBusy = true
  try {
    await enqueue(async () => {
      const raw = redoStack.pop()!
      undoStack.push(encodeSceneSnapshot(serializeScene()))
      await applyUndoSnapshot(raw)
    })
  } finally {
    undoBusy = false
    emitState()
  }
}

/* ---------------- camera ---------------- */

function fitToContent(): void {
  if (!canvas || !ready || restoring) return
  const bounds = sceneBounds()
  if (!bounds) return
  canvas.setViewportTransform(fitViewport(bounds, canvas.getWidth(), canvas.getHeight()))
  canvas.requestRenderAll()
  scheduleMaskRender()
  markDirty()
}

function zoomBy(factor: number): void {
  if (!canvas || !ready || restoring) return
  const center = { x: canvas.getWidth() / 2, y: canvas.getHeight() / 2 }
  canvas.setViewportTransform(zoomAtClient(canvas.viewportTransform as Mat6, center, factor))
  canvas.requestRenderAll()
  scheduleMaskRender()
  emitState()
}

function reset(): void {
  if (!canvas || !ready || restoring) return
  const gen = generation
  void enqueue(async () => {
    if (!canvas || gen !== generation) return
    saveBlocked = false
    pushUndo()
    for (const entry of collectEntries()) canvas.remove(entry.obj)
    canvas.discardActiveObject()
    canvas.setViewportTransform([1, 0, 0, 1, 0, 0])
    canvas.requestRenderAll()
    scheduleMaskRender()
    markDirty()
    pruneAssetUrls()
  })
}

async function flush(): Promise<void> {
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }
  await opQueue
  // Only write when this editor still owns the live workspace partition —
  // a stale editor must never recreate records after a clear/rotation.
  const scope = props.scope
  if (scope && canvas && getWorkspaceId(scope.userId) === scope.workspaceId && sceneDirty()) {
    await persistScene()
  }
  await opQueue
}

/* ---------------- events ---------------- */

function onDragOver(e: DragEvent): void {
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
}

function onDrop(e: DragEvent): void {
  if (!canvas || !e.dataTransfer || !rootEl.value) return
  const client = clientPoint(e)
  const at = clientToScene(canvas.viewportTransform as Mat6, client)
  const custom = e.dataTransfer.getData('application/x-canvas-asset')
  if (custom) {
    try {
      const payload = JSON.parse(custom) as { assetKey?: string }
      if (payload.assetKey && props.scope) {
        const gen = generation
        void enqueue(async () => {
          if (gen !== generation || !canvas) return
          const blob = await blobForAsset(payload.assetKey!)
          if (!blob || gen !== generation || !canvas) return
          await addImageObject(blob, { at, assetKey: payload.assetKey })
        })
      }
    } catch { /* malformed drag payload: ignore */ }
    return
  }
  const files = [...(e.dataTransfer.files ?? [])]
  if (files.length) void routeImport(files.map((f) => f as Blob), at)
}

function onPaste(e: ClipboardEvent): void {
  if (isEditableTarget(e.target) || modalOpen()) return
  const files = [...(e.clipboardData?.files ?? [])].filter((f) => IMAGE_TYPES.has(f.type))
  if (files.length) {
    e.preventDefault()
    void routeImport(files.map((f) => f as Blob), null)
  }
}

function modalOpen(): boolean {
  return !!document.querySelector('[role="dialog"], [data-canvas-dialog]')
}

function onKeyDown(e: KeyboardEvent): void {
  if (!canvas || isEditableTarget(e.target) || modalOpen()) return
  const active = canvas.getActiveObject() as unknown as { isEditing?: boolean } | null
  if (active?.isEditing) return
  const mod = e.metaKey || e.ctrlKey
  if (mod && e.key.toLowerCase() === 'z') {
    e.preventDefault()
    void (e.shiftKey ? redo() : undo())
    return
  }
  if (mod || e.altKey) return
  switch (e.key.toLowerCase()) {
    case 'v': emit('update:tool', 'select'); break
    case 'b': emit('update:tool', 'brush'); break
    case 'e': emit('update:tool', 'eraser'); break
    case 't':
      // T arms the text tool AND drops a label at the viewport center.
      emit('update:tool', 'text')
      addTextObject()
      break
    case 'h':
    case 'p': emit('update:tool', 'pan'); break
    case ' ': {
      // Only when focus is on the page/editor itself — toolbar buttons and
      // other focusable elements must keep their keyboard activation.
      const target = e.target as HTMLElement | null
      if (
        target &&
        target !== document.body &&
        !(rootEl.value && rootEl.value.contains(target))
      ) {
        break
      }
      spaceDown = true
      if (canvas) canvas.defaultCursor = 'grab'
      e.preventDefault()
      break
    }
    case 'delete':
    case 'backspace':
      e.preventDefault()
      void removeSelected()
      break
  }
}

function onKeyUp(e: KeyboardEvent): void {
  if (e.key === ' ') spaceDown = false
}

function resetPointerState(): void {
  panning.value = false
  panPointer = null
  pinch = null
  spaceDown = false
}

function onWheel(e: WheelEvent): void {
  if (!canvas) return
  e.preventDefault()
  // Normalize line/page delta modes into pixels.
  const unit = e.deltaMode === 1 ? 16 : e.deltaMode === 2 ? canvas.getHeight() : 1
  const dX = e.deltaX * unit
  const dY = e.deltaY * unit
  const vpt = [...canvas.viewportTransform] as Mat6
  if (e.ctrlKey || e.metaKey) {
    canvas.setViewportTransform(
      zoomAtClient(vpt, clientPoint(e), Math.exp(-dY * 0.0015)),
    )
  } else {
    vpt[4] -= dX
    vpt[5] -= dY
    canvas.setViewportTransform(vpt)
  }
  canvas.requestRenderAll()
  scheduleMaskRender()
  emitState()
  // Wheel pans/zooms move the camera — persist like any other viewport edit.
  markDirty()
}

function onCanvasMouseDown(e: TPointerEventInfo): void {
  if (!canvas) return
  const native = e.e as MouseEvent
  // Blank-drag pans; Shift+drag keeps Fabric's box selection.
  const overObject = !!e.target
  const wantPan =
    props.tool === 'pan' ||
    spaceDown ||
    native.button === 1 ||
    (!overObject && props.tool !== 'text' && !native.shiftKey)
  if (wantPan) {
    panning.value = true
    panPointer = { x: native.clientX, y: native.clientY }
    canvas.defaultCursor = 'grabbing'
    // Cancel the selection box Fabric may have armed for the empty area.
    ;(canvas as unknown as { _groupSelector?: unknown })._groupSelector = null
    return
  }
  if (props.tool === 'text' && native.button === 0 && !overObject) {
    addTextObject(clientToScene(canvas.viewportTransform as Mat6, clientPoint(native)))
    return
  }
  if (props.tool === 'select' && overObject) {
    // Capture the pre-gesture state so object:modified pushes ONE undo step.
    preGestureSnapshot = encodeSceneSnapshot(serializeScene())
    gestureSeen = false
  }
}

function onCanvasMouseMove(e: TPointerEventInfo): void {
  if (!panning.value || !panPointer || !canvas) return
  const native = e.e as MouseEvent
  const dx = native.clientX - panPointer.x
  const dy = native.clientY - panPointer.y
  panPointer = { x: native.clientX, y: native.clientY }
  const vpt = [...canvas.viewportTransform] as Mat6
  vpt[4] += dx
  vpt[5] += dy
  canvas.setViewportTransform(vpt)
  canvas.requestRenderAll()
  scheduleMaskRender()
}

function onPointerUp(): void {
  if (panning.value) markDirty()
  panning.value = false
  panPointer = null
  pinch = null
  if (canvas) canvas.defaultCursor = props.tool === 'pan' ? 'grab' : 'default'
}

function onGesture(): void {
  gestureSeen = true
  scheduleMaskRender()
}

function onObjectModified(): void {
  if (gestureSeen && preGestureSnapshot) {
    undoStack.push(preGestureSnapshot)
    if (undoStack.length > UNDO_LIMIT) undoStack.shift()
    redoStack = []
  }
  preGestureSnapshot = null
  gestureSeen = false
  // Metadata snapshots read world matrices at serialize time, so a group
  // transform needs no per-member sync here.
  markDirty()
  scheduleMaskRender()
}

function onTouchMove(e: TouchEvent): void {
  if (!canvas || e.touches.length !== 2 || !rootEl.value) return
  e.preventDefault()
  const [a, b] = [e.touches[0], e.touches[1]]
  const d = Math.hypot(a.clientX - b.clientX, a.clientY - b.clientY)
  const mid = {
    x: (a.clientX + b.clientX) / 2,
    y: (a.clientY + b.clientY) / 2,
  }
  const midClient = clientPoint({ clientX: mid.x, clientY: mid.y } as MouseEvent)
  if (pinch) {
    canvas.setViewportTransform(
      zoomAtClient(canvas.viewportTransform as Mat6, midClient, d / pinch.d),
    )
    const vpt = [...canvas.viewportTransform] as Mat6
    vpt[4] += midClient.x - pinch.mid.x
    vpt[5] += midClient.y - pinch.mid.y
    canvas.setViewportTransform(vpt)
    canvas.requestRenderAll()
    scheduleMaskRender()
  }
  pinch = { d, mid: midClient }
}

function onTouchEnd(e: TouchEvent): void {
  if (e.touches.length < 2) pinch = null
}

function applyToolMode(): void {
  if (!canvas) return
  const t = props.tool
  canvas.isDrawingMode = false
  canvas.selection = t === 'select' || t === 'text'
  canvas.skipTargetFind = t === 'pan' || t === 'brush' || t === 'eraser'
  canvas.defaultCursor = t === 'pan' ? 'grab' : t === 'text' ? 'text' : 'default'
  for (const obj of canvas.getObjects()) {
    obj.selectable = t === 'select' || t === 'text'
    obj.evented = t !== 'pan'
  }
  updateOverlayEvents()
  scheduleMaskRender()
  canvas.requestRenderAll()
}

watch(() => [props.tool, props.brushSize, props.brushShape], () => {
  applyToolMode()
  emitState()
})

watch(
  () => [props.textColor, props.textSize],
  () => {
    if (!canvas) return
    const active = canvas.getActiveObject() as IText | null
    if (active && (active as unknown as { editorData?: SceneObject }).editorData?.role === 'text') {
      pushUndo()
      active.set({ fill: props.textColor, fontSize: props.textSize })
      const data = (active as unknown as { editorData?: SceneObject }).editorData!
      data.fill = props.textColor
      data.fontSize = props.textSize
      canvas.requestRenderAll()
      markDirty()
    }
  },
)

watch(
  () => props.scope,
  (next, prev) => {
    if (!next || !prev) return
    if (next.userId !== prev.userId || next.workspaceId !== prev.workspaceId) {
      generation += 1
      ready = false
      undoStack = []
      redoStack = []
      canvas?.discardActiveObject()
      for (const entry of collectEntries()) canvas?.remove(entry.obj)
      void restoreScene()
    }
  },
)

function emitState(): void {
  if (!canvas) return
  const actives = canvas.getActiveObjects()
  emit('state', {
    ready,
    imageCount: imageEntries().length,
    selectedCount: actives.length,
    selectedImageCount: selectedEntries('image').length,
    hasMask: maskHasCoverage(maskAnchorEntry()?.data.maskStrokes),
    canUndo: undoStack.length > 0,
    canRedo: redoStack.length > 0,
    saveBlocked,
    zoom: canvas ? Math.min(CANVAS_ZOOM_MAX, Math.max(CANVAS_ZOOM_MIN, canvas.getZoom())) : 1,
  })
}

/* ---------------- lifecycle ---------------- */

function onPageHide(): void {
  if (saveTimer) {
    clearTimeout(saveTimer)
    saveTimer = null
  }
  // Capture synchronously while the canvas is alive, then write the snapshot
  // directly — queued ops may never settle during unload.
  persistSnapshotSync()
}

/** persistSnapshotSync writes a live snapshot without touching the queue. */
function persistSnapshotSync(): void {
  const sc = props.scope
  if (!sc || !canvas || saveBlocked) return
  if (getWorkspaceId(sc.userId) !== sc.workspaceId) return
  const snapshot = serializeScene()
  // An empty board never creates a new record on teardown.
  if (snapshot.objects.length === 0) return
  void saveScene(sc, SCENE_KEY, encodeSceneSnapshot(snapshot)).catch(() => {})
}

function onVisibilityChange(): void {
  if (document.visibilityState === 'hidden') onPageHide()
}

onMounted(async () => {
  const el = rootEl.value!
  canvas = new FabricCanvas(mainCanvasEl.value!, {
    preserveObjectStacking: true,
    stopContextMenu: true,
    fireRightClick: false,
    selectionColor: 'rgba(20,184,166,0.08)',
    selectionBorderColor: '#14b8a6',
    selectionLineWidth: 1,
  })
  overlayCtx = overlayEl.value!.getContext('2d')
  overlayEl.value!.style.pointerEvents = 'none'

  const resize = () => {
    const w = el.clientWidth
    const h = el.clientHeight
    const dpr = window.devicePixelRatio || 1
    canvas!.setDimensions({ width: w, height: h })
    overlayEl.value!.width = w * dpr
    overlayEl.value!.height = h * dpr
    overlayEl.value!.style.width = `${w}px`
    overlayEl.value!.style.height = `${h}px`
    canvas!.requestRenderAll()
    scheduleMaskRender()
  }
  resizeObserver = new ResizeObserver(resize)
  resizeObserver.observe(el)
  resize()

  canvas.on('mouse:down', onCanvasMouseDown)
  canvas.on('mouse:move', onCanvasMouseMove)
  canvas.on('mouse:up', onPointerUp)
  canvas.on('object:moving', onGesture)
  canvas.on('object:scaling', onGesture)
  canvas.on('object:rotating', onGesture)
  canvas.on('object:modified', onObjectModified)
  canvas.on('selection:created', () => { applyToolMode(); scheduleMaskRender(); emitState() })
  canvas.on('selection:updated', () => { scheduleMaskRender(); emitState() })
  canvas.on('selection:cleared', () => { scheduleMaskRender(); emitState() })
  canvas.on('object:added', () => emitState())
  canvas.on('object:removed', () => emitState())
  canvas.on('text:editing:entered', () => {
    textEditSnapshot = encodeSceneSnapshot(serializeScene())
  })
  canvas.on('text:editing:exited', (e) => {
    const obj = e.target as IText | undefined
    const data = obj && (obj as unknown as { editorData?: SceneObject }).editorData
    if (data && obj) data.text = obj.text
    if (textEditSnapshot) {
      undoStack.push(textEditSnapshot)
      if (undoStack.length > UNDO_LIMIT) undoStack.shift()
      redoStack = []
      textEditSnapshot = null
    }
    markDirty()
  })

  const overlay = overlayEl.value!
  overlay.addEventListener('pointerdown', onOverlayPointerDown)
  overlay.addEventListener('pointermove', onOverlayPointerMove)
  overlay.addEventListener('pointerup', onOverlayPointerEnd)
  overlay.addEventListener('pointercancel', onOverlayPointerEnd)

  // Pan/zoom/touch live on the root so they work above any active layer.
  el.addEventListener('wheel', onWheel, { passive: false })
  el.addEventListener('pointerup', onPointerUp)
  el.addEventListener('pointercancel', resetPointerState)
  el.addEventListener('touchmove', onTouchMove, { passive: false })
  el.addEventListener('touchend', onTouchEnd)
  window.addEventListener('keydown', onKeyDown)
  window.addEventListener('keyup', onKeyUp)
  window.addEventListener('blur', resetPointerState)
  window.addEventListener('paste', onPaste)
  window.addEventListener('pagehide', onPageHide)
  document.addEventListener('visibilitychange', onVisibilityChange)

  applyToolMode()
  await restoreScene()
})

onBeforeUnmount(() => {
  generation += 1
  if (saveTimer) clearTimeout(saveTimer)
  // Snapshot now, while the canvas is still alive — flush() after dispose
  // would see a null canvas and write nothing.
  persistSnapshotSync()
  const el = rootEl.value
  if (el) {
    el.removeEventListener('wheel', onWheel)
    el.removeEventListener('pointerup', onPointerUp)
    el.removeEventListener('pointercancel', resetPointerState)
    el.removeEventListener('touchmove', onTouchMove)
    el.removeEventListener('touchend', onTouchEnd)
  }
  const overlay = overlayEl.value
  if (overlay) {
    overlay.removeEventListener('pointerdown', onOverlayPointerDown)
    overlay.removeEventListener('pointermove', onOverlayPointerMove)
    overlay.removeEventListener('pointerup', onOverlayPointerEnd)
    overlay.removeEventListener('pointercancel', onOverlayPointerEnd)
  }
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
  window.removeEventListener('blur', resetPointerState)
  window.removeEventListener('paste', onPaste)
  window.removeEventListener('pagehide', onPageHide)
  document.removeEventListener('visibilitychange', onVisibilityChange)
  resizeObserver?.disconnect()
  for (const url of assetUrls.values()) URL.revokeObjectURL(url)
  assetUrls.clear()
  assetBlobs.clear()
  canvas?.dispose()
  canvas = null
  maskBuffer = null
  overlayCtx = null
})

defineExpose({
  openFilePicker,
  placeOutput,
  exportInput,
  flush,
  undo,
  redo,
  removeSelected,
  clearMask,
  reset,
  fitToContent,
  zoomBy,
  addText: () => addTextObject(),
  flattenSelected,
  downloadSelected,
  importBlobs,
})
</script>

<style scoped>
.canvas-editor {
  position: absolute;
  inset: 0;
  overflow: hidden;
  background:
    radial-gradient(circle, rgba(20, 184, 166, 0.12) 1px, transparent 1px) 0 0 / 22px 22px,
    var(--color-gray-50, #f9fafb);
}
.dark .canvas-editor {
  background:
    radial-gradient(circle, rgba(129, 140, 248, 0.16) 1px, transparent 1px) 0 0 / 22px 22px,
    var(--color-dark-900, #0b1120);
}
.canvas-editor.is-panning {
  cursor: grabbing;
}
.mask-layer {
  position: absolute;
  inset: 0;
}
.hidden {
  display: none;
}
</style>
