/**
 * CanvasEditor.spec.ts exercises the editor's runtime behavior with a focused
 * Fabric mock (jsdom has no canvas 2d): scene serialization ordering, undo
 * through queued mutations, mask strokes on live metadata, placement dedup,
 * deferred-decode disposal, world-transform persistence, and selected-only
 * export. The local store and fake IndexedDB are real.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

// ---- Fabric mock ----------------------------------------------------------
// vi.mock factories hoist above top-level declarations, so shared classes
// live inside vi.hoisted and the factory only references them.

const mocks = vi.hoisted(() => {
  class MockObject {
  left = 0
  top = 0
  scaleX = 1
  scaleY = 1
  angle = 0
  width = 0
  height = 0
  selectable = true
  evented = true
  isEditing = false
  _world: number[] | null = null
  editorData?: unknown
  set(props: Record<string, unknown>) { Object.assign(this, props); return this }
  setCoords() {}
  getBoundingRect() {
    return { left: this.left, top: this.top, width: this.width * this.scaleX, height: this.height * this.scaleY }
  }
  calcTransformMatrix() {
    return this._world ?? [1, 0, 0, 1, this.left, this.top]
  }
  _render() {}
}

class MockImage extends MockObject {
  static size = { width: 100, height: 100 }
  static async fromURL() {
    const img = new MockImage()
    img.width = MockImage.size.width
    img.height = MockImage.size.height
    return img
  }
}

class MockText extends MockObject {
  text: string
  fontSize: number
  fill: string
  constructor(t: string, o: Record<string, unknown>) {
    super()
    this.text = t
    this.fontSize = (o.fontSize as number) ?? 16
    this.fill = (o.fill as string) ?? '#000'
  }
  enterEditing() { this.isEditing = true }
}

class MockCanvas {
  _objects: MockObject[] = []
  _active: MockObject[] = []
  _events = new Map<string, Array<(e: unknown) => void>>()
  viewportTransform = [1, 0, 0, 1, 0, 0]
  upperCanvasEl = document.createElement('canvas')
  lowerCanvasEl = document.createElement('canvas')
  isDrawingMode = false
  selection = true
  skipTargetFind = false
  defaultCursor = ''
  constructor(el: unknown, _opts?: unknown) {
    const host = el as { __mockCanvas?: MockCanvas }
    host.__mockCanvas = this
  }
  on(name: string, fn: (e: unknown) => void) {
    this._events.set(name, [...(this._events.get(name) ?? []), fn])
  }
  fire(name: string, e?: unknown) {
    for (const fn of this._events.get(name) ?? []) fn(e)
  }
  getObjects() { return this._objects }
  add(o: MockObject) { this._objects.push(o) }
  remove(o: MockObject) { this._objects = this._objects.filter((x) => x !== o) }
  clear() { this._objects = [] }
  getActiveObjects() { return this._active }
  getActiveObject() { return this._active[0] ?? null }
  setActiveObject(o: MockObject) { this._active = [o] }
  discardActiveObject() { this._active = [] }
  requestRenderAll() {}
  setViewportTransform(v: number[]) { this.viewportTransform = v }
  setDimensions() {}
  getWidth() { return 800 }
  getHeight() { return 600 }
  getZoom() { return this.viewportTransform[0] }
  dispose() {}
}

  return { MockObject, MockImage, MockText, MockCanvas }
})

const { MockObject, MockImage, MockCanvas } = mocks

vi.mock('fabric', () => ({
  Canvas: mocks.MockCanvas,
  FabricImage: mocks.MockImage,
  IText: mocks.MockText,
  util: {
    applyTransformToObject: (obj: InstanceType<typeof mocks.MockObject>, m: number[]) => {
      obj._world = m
      obj.left = m[4]
      obj.top = m[5]
    },
  },
}))

// ---------------------------------------------------------------------------

import { installFakeIdb, type FakeIdb } from '@/utils/__tests__/fakeIdb'
import {
  __resetCanvasStoreForTest,
  getWorkspaceId,
  listAssets,
  listScenes,
  loadScene,
  saveScene,
} from '@/utils/canvasLocalStore'
import CanvasEditor from '../CanvasEditor.vue'
import type { CanvasScope } from '@/types/canvas'
import { decodeSceneSnapshot, type SceneObject } from '@/utils/canvasScene'
import { hasRedrawArea, renderMaskExport, renderMaskPreview } from '@/utils/canvasMask'
import type { MaskStroke } from '@/utils/canvasScene'

let fake: FakeIdb
let scope: CanvasScope
let imageSize = { width: 100, height: 100 }

interface EditorApi {
  importBlobs(blobs: Blob[], at?: { x: number; y: number } | null): Promise<void>
  placeOutput(input: { blob: Blob; runId: string; outputIndex: number; at?: { x: number; y: number } | null }): Promise<void>
  exportInput(op: string): Promise<{ images: Blob[]; mask: Blob | null }>
  flush(): Promise<void>
  undo(): Promise<void>
  redo(): Promise<void>
  clearMask(): void
}
function api(wrapper: ReturnType<typeof mountEditor>): EditorApi {
  return wrapper.vm as unknown as EditorApi
}

function scopeFor(): CanvasScope {
  return { userId: 1, workspaceId: getWorkspaceId(1) }
}

async function flushAll() {
  for (let i = 0; i < 8; i += 1) await Promise.resolve()
  await new Promise((r) => setTimeout(r, 0))
}

function mountEditor(extra: Record<string, unknown> = {}) {
  return mount(CanvasEditor, {
    props: { scope, tool: 'select', ...extra },
    attachTo: document.body,
  })
}

function editorCanvas(wrapper: ReturnType<typeof mountEditor>): InstanceType<typeof MockCanvas> {
  const el = wrapper.find('canvas')
  return (el.element as unknown as { __mockCanvas: InstanceType<typeof MockCanvas> }).__mockCanvas
}

beforeEach(() => {
  fake = installFakeIdb()
  __resetCanvasStoreForTest()
  window.localStorage.clear()
  scope = scopeFor()
  imageSize = { width: 100, height: 100 }
  MockImage.size = imageSize
  vi.stubGlobal('createImageBitmap', vi.fn(async (blob: Blob) => ({
    width: imageSize.width,
    height: imageSize.height,
    size: blob.size,
    close: () => {},
  })))
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => `blob:mock-${Math.random()}`),
    revokeObjectURL: vi.fn(),
  })
  class RO { observe() {} unobserve() {} disconnect() {} }
  vi.stubGlobal('ResizeObserver', RO)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('CanvasEditor runtime', () => {
  it('imports an image, survives quota failure in undo/redo via blob cache', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    // Inject AFTER mount so the quota error hits the asset write, not the
    // initial scene read.
    fake.failNextRequest(new DOMException('full', 'QuotaExceededError'))

    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    await flushAll()
    const entry = c._objects[0] as MockObject & { editorData: SceneObject }
    expect(entry.editorData.role).toBe('image')
    // Quota failure still leaves the blob reachable for undo.
    expect(wrapper.emitted('error')?.flat().join(',')).toContain('asset-save-failed')
    expect(c._objects).toHaveLength(1)

    await api(wrapper).undo()
    await flushAll()
    expect(c._objects).toHaveLength(0)
    await api(wrapper).redo()
    await flushAll()
    expect(c._objects).toHaveLength(1)
    wrapper.unmount()
  })

  it('records strokes on live metadata; clearMask and undo restore them', async () => {
    const wrapper = mountEditor({ tool: 'brush' })
    await flushAll()
    const c = editorCanvas(wrapper)
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    await flushAll()
    const obj = c._objects[0] as MockObject & { editorData: SceneObject }

    // Simulate a pointer stroke at the image's center on the overlay canvas.
    const overlay = wrapper.element.querySelector('canvas.mask-layer') as HTMLCanvasElement
    overlay.getBoundingClientRect = () =>
      ({ left: 0, top: 0, width: 800, height: 600 }) as DOMRect
    // The imported image sits at scene 300..400 x 200..300 with the mock
    // transforms, so (370,270) hits it and (50,50) would not.
    const pe = (type: string, x: number, y: number) =>
      Object.assign(new MouseEvent(type, { clientX: x, clientY: y, bubbles: true }), { pointerId: 1 })
    overlay.dispatchEvent(pe('pointerdown', 370, 270))
    overlay.dispatchEvent(pe('pointermove', 380, 275))
    overlay.dispatchEvent(pe('pointerup', 380, 275))
    await flushAll()

    expect(obj.editorData.maskStrokes).toHaveLength(1)
    expect(obj.editorData.maskStrokes![0].mode).toBe('add')

    await api(wrapper).clearMask()
    expect(obj.editorData.maskStrokes).toHaveLength(0)
    await api(wrapper).undo()
    await flushAll()
    // Undo restores a NEW object entry; read the live metadata again.
    const restored = c._objects[0] as MockObject & { editorData: SceneObject }
    expect(restored.editorData.maskStrokes).toHaveLength(1)
    wrapper.unmount()
  })

  it('serializes the post-op scene, not the pre-op snapshot', async () => {
    const wrapper = mountEditor()
    await flushAll()
    await api(wrapper).importBlobs([
      new Blob(['a'], { type: 'image/png' }),
      new Blob(['b'], { type: 'image/png' }),
    ])
    await flushAll()
    await api(wrapper).flush()
    const record = await loadScene(scope, 'default')
    const snap = decodeSceneSnapshot(record!.json)!
    expect(snap.objects).toHaveLength(2)
    expect(snap.objects[0].src).toMatch(/^asset:\/\/source:/)
    wrapper.unmount()
  })

  it('a deferred decode resolved after unmount writes nothing', async () => {
    const wrapper = mountEditor()
    await flushAll()
    let resolveBitmap!: (v: unknown) => void
    vi.stubGlobal('createImageBitmap', vi.fn(() => new Promise((r) => { resolveBitmap = r })))
    const pending = api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    // Let the queued job reach the deferred decode before unmounting.
    await Promise.resolve()
    await Promise.resolve()
    wrapper.unmount()
    resolveBitmap({ width: 10, height: 10, close: () => {} })
    await pending.catch(() => {})
    await flushAll()
    expect(await listScenes(scope)).toHaveLength(0)
    expect(await listAssets(scope)).toHaveLength(0)
  })

  it('dedups concurrent placeOutput by run:index inside the queue', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    const blob = new Blob(['img'], { type: 'image/png' })
    await Promise.all([
      api(wrapper).placeOutput({ blob, runId: 'r1', outputIndex: 0 }),
      api(wrapper).placeOutput({ blob, runId: 'r1', outputIndex: 0 }),
    ])
    await flushAll()
    expect(c._objects).toHaveLength(1)
    expect((c._objects[0] as { editorData: SceneObject }).editorData.runId).toBe('r1')
    wrapper.unmount()
  })

  it('rejects a failed placement so the view can retry', async () => {
    const wrapper = mountEditor()
    await flushAll()
    vi.stubGlobal('createImageBitmap', vi.fn(() => Promise.reject(new Error('decode'))))
    await expect(
      api(wrapper).placeOutput({ blob: new Blob(['x'], { type: 'image/png' }), runId: 'r9', outputIndex: 0 }),
    ).rejects.toThrow()
    expect(wrapper.emitted('error')?.flat().join(',')).toContain('place-failed')
    wrapper.unmount()
  })

  it('persists group world transforms for multi-selected members', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    await flushAll()
    const img = c._objects[0] as MockObject & { editorData: SceneObject }
    // Wrap the image in a fake active-selection group offset by (10,20).
    const group = new MockObject()
    group._world = [1, 0, 0, 1, 10, 20]
    img._world = [1, 0, 0, 1, 5, 5]
    ;(group as unknown as { getObjects: () => MockObject[] }).getObjects = () => [img]
    c._objects = [group as never]
    await api(wrapper).flush()
    const snap = decodeSceneSnapshot((await loadScene(scope, 'default'))!.json)!
    expect(snap.objects[0].transform).toEqual([1, 0, 0, 1, 15, 25])
    wrapper.unmount()
  })

  it('pans via the overlay while a mask tool is armed', async () => {
    const wrapper = mountEditor({ tool: 'brush' })
    await flushAll()
    const c = editorCanvas(wrapper)
    const overlay = wrapper.element.querySelector('canvas.mask-layer') as HTMLCanvasElement
    const pe = (type: string, x: number, y: number, button = 0) =>
      Object.assign(new MouseEvent(type, { clientX: x, clientY: y, button, bubbles: true }), { pointerId: 1 })
    // Middle-button drag on the overlay pans even though it captures pointers.
    overlay.dispatchEvent(pe('pointerdown', 100, 100, 1))
    overlay.dispatchEvent(pe('pointermove', 140, 120, 1))
    overlay.dispatchEvent(pe('pointerup', 140, 120, 1))
    await flushAll()
    expect(c.viewportTransform[4]).toBe(40)
    expect(c.viewportTransform[5]).toBe(20)
    wrapper.unmount()
  })

  it('writes the live snapshot on immediate unmount', async () => {
    const wrapper = mountEditor()
    await flushAll()
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    // Unmount before the 600ms debounce: the edit must still be stored.
    wrapper.unmount()
    await flushAll()
    const record = await loadScene(scope, 'default')
    expect(decodeSceneSnapshot(record!.json)!.objects).toHaveLength(1)
  })

  it('blocks every write after a corrupt scene load', async () => {
    await saveScene(scope, 'default', '{corrupt')
    const wrapper = mountEditor()
    await flushAll()
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    await api(wrapper).flush()
    await flushAll()
    // The corrupt record is preserved untouched — no overwrite happened.
    expect((await loadScene(scope, 'default'))!.json).toBe('{corrupt')
    expect(wrapper.emitted('error')?.flat().join(',')).toContain('scene-load-failed')
    wrapper.unmount()
  })

  it('exportInput consumes only selected sources', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    await api(wrapper).importBlobs([
      new Blob(['one'], { type: 'image/png' }),
      new Blob(['two'], { type: 'image/png' }),
    ])
    await flushAll()
    c.setActiveObject(c._objects[1] as MockObject)
    const out = await api(wrapper).exportInput('edit')
    expect(out.images).toHaveLength(1)
    expect(out.images[0].size).toBe(3) // 'two'
    // No selection at all rejects rather than silently using everything.
    c.discardActiveObject()
    await expect(api(wrapper).exportInput('edit')).rejects.toThrow()
    wrapper.unmount()
  })
})

// ---- exact mask pixels with a software ctx --------------------------------

class PixelCtx {
  width: number
  height: number
  data: Uint8ClampedArray
  fillStyle = '#000'
  strokeStyle = '#000'
  lineWidth = 1
  globalCompositeOperation = 'source-over'
  private path: Array<[number, number, number]> = []
  constructor(w: number, h: number) {
    this.width = w
    this.height = h
    this.data = new Uint8ClampedArray(w * h * 4)
  }
  private put(x: number, y: number, on: boolean) {
    const xi = Math.round(x)
    const yi = Math.round(y)
    if (xi < 0 || yi < 0 || xi >= this.width || yi >= this.height) return
    const i = (yi * this.width + xi) * 4
    if (this.globalCompositeOperation === 'destination-out') {
      this.data[i + 3] = 0
    } else {
      this.data[i] = 255
      this.data[i + 1] = 255
      this.data[i + 2] = 255
      this.data[i + 3] = on ? 255 : this.data[i + 3]
    }
  }
  beginPath() { this.path = [] }
  moveTo(x: number, y: number) { this.path.push([x, y, 0]) }
  lineTo(x: number, y: number) { this.path.push([x, y, 1]) }
  arc(x: number, y: number, r: number) { this.path.push([x, y, r]) }
  fill() {
    for (const [x, y, r] of this.path) {
      for (let dy = -r; dy <= r; dy += 1)
        for (let dx = -r; dx <= r; dx += 1)
          if (dx * dx + dy * dy <= r * r) this.put(x + dx, y + dy, true)
    }
    this.path = []
  }
  stroke() {
    const half = this.lineWidth / 2
    const pts = this.path
    for (let i = 0; i < pts.length; i += 1) {
      const [x, y] = pts[i]
      if (i === 0) this.put(x, y, true)
      else {
        const [px, py] = pts[i - 1]
        const steps = Math.max(1, Math.ceil(Math.hypot(x - px, y - py)))
        for (let s = 0; s <= steps; s += 1) {
          const cx = px + ((x - px) * s) / steps
          const cy = py + ((y - py) * s) / steps
          for (let dy = -half; dy <= half; dy += 1)
            for (let dx = -half; dx <= half; dx += 1)
              if (dx * dx + dy * dy <= half * half) this.put(cx + dx, cy + dy, true)
        }
      }
    }
    this.path = []
  }
  fillRect(x: number, y: number, w: number, h: number) {
    for (let yy = Math.floor(y); yy < y + h; yy += 1)
      for (let xx = Math.floor(x); xx < x + w; xx += 1) this.put(xx, yy, true)
  }
  clearRect(_x: number, _y: number, w: number, h: number) {
    for (let i = 0; i < w * h * 4; i += 1) this.data[i] = 0
  }
  save() {}
  restore() {}
  getImageData(_x: number, _y: number, w: number, h: number) {
    return { data: this.data, width: w, height: h }
  }
}

describe('canvasMask pixel semantics', () => {
  const strokes: MaskStroke[] = [
    { mode: 'add', shape: 'round', width: 2, points: [{ x: 4, y: 4 }] },
  ]

  it('export mask: untouched opaque white, painted fully transparent', () => {
    const ctx = new PixelCtx(8, 8)
    renderMaskExport(ctx as never, strokes, 8, 8)
    const d = ctx.getImageData(0, 0, 8, 8).data
    // Center of the painted dot: alpha 0.
    expect(d[(4 * 8 + 4) * 4 + 3]).toBe(0)
    // Untouched corner: opaque white.
    const corner = (0 * 8 + 0) * 4
    expect(d[corner + 3]).toBe(255)
    expect(d[corner]).toBe(255)
    expect(hasRedrawArea(ctx as never, 8, 8)).toBe(true)
  })

  it('a fully-erased mask reports no redraw area', () => {
    const ctx = new PixelCtx(8, 8)
    renderMaskExport(ctx as never, [], 8, 8)
    expect(hasRedrawArea(ctx as never, 8, 8)).toBe(false)
  })

  it('erase strokes restore alpha after an add', () => {
    const ctx = new PixelCtx(8, 8)
    renderMaskExport(
      ctx as never,
      [
        { mode: 'add', shape: 'round', width: 4, points: [{ x: 4, y: 4 }] },
        { mode: 'erase', shape: 'round', width: 4, points: [{ x: 4, y: 4 }] },
      ],
      8,
      8,
    )
    const d = ctx.getImageData(0, 0, 8, 8).data
    expect(d[(4 * 8 + 4) * 4 + 3]).toBe(255)
    expect(hasRedrawArea(ctx as never, 8, 8)).toBe(false)
  })

  it('preview paints add strokes and cuts erase strokes', () => {
    const ctx = new PixelCtx(8, 8)
    renderMaskPreview(
      ctx as never,
      [
        { mode: 'add', shape: 'round', width: 3, points: [{ x: 4, y: 4 }] },
        { mode: 'erase', shape: 'round', width: 3, points: [{ x: 4, y: 4 }] },
      ],
      8,
      8,
    )
    const d = ctx.getImageData(0, 0, 8, 8).data
    expect(d[(4 * 8 + 4) * 4 + 3]).toBe(0)
  })

  it('square stamps cover the segment uniformly', () => {
    const ctx = new PixelCtx(16, 8)
    renderMaskPreview(
      ctx as never,
      [{ mode: 'add', shape: 'square', width: 4, points: [{ x: 2, y: 4 }, { x: 10, y: 4 }] }],
      16,
      8,
    )
    const d = ctx.getImageData(0, 0, 16, 8).data
    expect(d[(4 * 16 + 2) * 4 + 3]).toBe(255)
    expect(d[(4 * 16 + 10) * 4 + 3]).toBe(255)
    expect(d[(4 * 16 + 6) * 4 + 3]).toBe(255) // midpoint covered by stamping
  })
})

// ---- F1/F2/F3 regression tests ---------------------------------------------

describe('placement origin (F1)', () => {
  it('import with `at` centres the object on the point', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })], { x: 500, y: 300 })
    await flushAll()
    const t = c._objects[0].calcTransformMatrix()
    expect(t[4]).toBe(500)
    expect(t[5]).toBe(300)
    wrapper.unmount()
  })

  it('default import centres on the viewport centre', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })])
    await flushAll()
    const t = c._objects[0].calcTransformMatrix()
    expect(t[4]).toBe(400) // getWidth()/2
    expect(t[5]).toBe(300)
    wrapper.unmount()
  })

  it('placeOutput lands right of existing bounds with a gap', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const c = editorCanvas(wrapper)
    await api(wrapper).importBlobs([new Blob(['x'], { type: 'image/png' })], { x: 100, y: 100 })
    await flushAll()
    await api(wrapper).placeOutput({ blob: new Blob(['o'], { type: 'image/png' }), runId: 'r1', outputIndex: 0 })
    await flushAll()
    expect(c._objects).toHaveLength(2)
    const t = c._objects[1].calcTransformMatrix()
    // New image's left edge (centre minus half width) clears old bounds + gap.
    expect(t[4] - 50).toBeGreaterThanOrEqual(150 + 40 - 0.001)
    // dedup still holds inside the queue
    await api(wrapper).placeOutput({ blob: new Blob(['o'], { type: 'image/png' }), runId: 'r1', outputIndex: 0 })
    await flushAll()
    expect(c._objects).toHaveLength(2)
    wrapper.unmount()
  })
})

describe('import routing + modal guards (F2/F3)', () => {
  it('drop and paste bypass crop; the picker path uses it', async () => {
    const crop = vi.fn()
    const wrapper = mountEditor({ cropHandler: crop })
    await flushAll()
    const c = editorCanvas(wrapper)
    const blob = new Blob(['x'], { type: 'image/png' })
    // simulated OS drop
    const dt = { getData: () => '', files: [blob], dropEffect: '' }
    wrapper.element.dispatchEvent(Object.assign(new Event('drop', { bubbles: true }), { dataTransfer: dt, clientX: 700, clientY: 400, preventDefault() {} }))
    await flushAll()
    expect(crop).not.toHaveBeenCalled()
    expect(wrapper.emitted('crop-request')).toBeUndefined()
    expect(c._objects.length).toBe(1)
    // file picker -> cropHandler owns the blobs
    wrapper.vm.openFilePicker?.()
    const input = wrapper.element.querySelector('input[type=file]') as HTMLInputElement
    Object.defineProperty(input, 'files', { value: [blob], configurable: true })
    input.dispatchEvent(new Event('change'))
    await flushAll()
    expect(crop).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('keyboard/paste ignored while a dialog is open', async () => {
    const wrapper = mountEditor()
    await flushAll()
    const dlg = document.createElement('div')
    dlg.setAttribute('role', 'dialog')
    dlg.setAttribute('aria-modal', 'true')
    document.body.appendChild(dlg)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'b' }))
    expect(wrapper.emitted('update:tool')).toBeUndefined()
    window.dispatchEvent(new Event('paste'))
    document.body.removeChild(dlg)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'b' }))
    expect(wrapper.emitted('update:tool')?.length).toBe(1)
    wrapper.unmount()
  })
})
