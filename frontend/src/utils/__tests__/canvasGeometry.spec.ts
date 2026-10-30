import { describe, expect, it } from 'vitest'
import {
  applyMatrix,
  clientToScene,
  fitViewport,
  invertMatrix,
  isEditableTarget,
  maskViewportMatrix,
  multiplyMatrices,
  pathDataFromPoints,
  sceneToClient,
  zoomAtClient,
  CANVAS_ZOOM_MIN,
  CANVAS_ZOOM_MAX,
  type Mat6,
} from '../canvasGeometry'
import {
  decodeSceneSnapshot,
  encodeSceneSnapshot,
  maskHasCoverage,
  placementIdentity,
  pointInImageLocal,
  sceneAssetKeys,
  strokeBounds,
  CANVAS_SCENE_VERSION,
  type SceneObject,
} from '../canvasScene'

function rotation(deg: number): Mat6 {
  const r = (deg * Math.PI) / 180
  return [Math.cos(r), Math.sin(r), -Math.sin(r), Math.cos(r), 0, 0]
}

describe('canvasGeometry', () => {
  it('multiplies and inverts matrices correctly', () => {
    const m: Mat6 = [2, 0, 0, 4, 10, 20]
    const inv = invertMatrix(m)
    expect(applyMatrix(multiplyMatrices(m, inv), { x: 7, y: -3 })).toEqual({ x: 7, y: -3 })
    expect(applyMatrix(inv, applyMatrix(m, { x: 1, y: 1 }))).toEqual({ x: 1, y: 1 })
  })

  it('mask viewport maps natural pixels onto the anchored image', () => {
    // Image 100x50 at (200, 100) scaled x2, rotated 90°.
    const camera: Mat6 = [1, 0, 0, 1, 30, -10]
    const img = multiplyMatrices(
      [2, 0, 0, 2, 200, 100],
      rotation(90),
    )
    const vpt = maskViewportMatrix(camera, img, 100, 50)
    // Natural (0,0) lands where the transform chain puts it: translate gives
    // (-50,-25), rot90 gives (25,-50), scale+move gives (250,0), camera
    // gives (280,-10).
    const topLeft = applyMatrix(vpt, { x: 0, y: 0 })
    expect(topLeft.x).toBeCloseTo(280)
    expect(topLeft.y).toBeCloseTo(-10)
    // The same point maps back through the inverse for the anchor hit test.
    const back = applyMatrix(invertMatrix(vpt), topLeft)
    expect(back.x).toBeCloseTo(0)
    expect(back.y).toBeCloseTo(0)
  })

  it('mask strokes survive image transform changes (same natural point)', () => {
    const camera: Mat6 = [0.5, 0, 0, 0.5, 0, 0]
    const before = maskViewportMatrix(camera, [1, 0, 0, 1, 10, 10], 100, 100)
    const after = maskViewportMatrix(camera, [1, 0, 0, 1, 50, 80], 100, 100)
    // Natural coordinate is invariant; only its screen position moves.
    expect(applyMatrix(before, { x: 25, y: 25 })).toEqual({ x: -7.5, y: -7.5 })
    expect(applyMatrix(after, { x: 25, y: 25 })).toEqual({ x: 12.5, y: 27.5 })
  })

  it('zoomAtClient clamps zoom and keeps the anchor point fixed', () => {
    const camera: Mat6 = [1, 0, 0, 1, 20, 40]
    const zoomed = zoomAtClient(camera, { x: 100, y: 50 }, 2)
    expect(zoomed[0]).toBe(2)
    // Point under the cursor stays put.
    expect(
      sceneToClient(zoomed, clientToScene(camera, { x: 100, y: 50 })),
    ).toEqual({ x: 100, y: 50 })
    const huge = zoomAtClient(camera, { x: 0, y: 0 }, 100)
    expect(huge[0]).toBe(CANVAS_ZOOM_MAX)
    const tiny = zoomAtClient(camera, { x: 0, y: 0 }, 0.0001)
    expect(tiny[0]).toBe(CANVAS_ZOOM_MIN)
  })

  it('fitViewport centers content and clamps to zoom bounds', () => {
    const vpt = fitViewport({ minX: 0, minY: 0, maxX: 400, maxY: 200 }, 1000, 800, 100)
    // scale = min(800/400, 600/200) = 2 → within clamp
    expect(vpt[0]).toBe(2)
    expect(applyMatrix(vpt, { x: 200, y: 100 })).toEqual({ x: 500, y: 400 })
    const big = fitViewport({ minX: 0, minY: 0, maxX: 10, maxY: 10 }, 1000, 800, 100)
    expect(big[0]).toBe(CANVAS_ZOOM_MAX)
  })

  it('pathDataFromPoints emits M then L commands', () => {
    expect(pathDataFromPoints([])).toEqual([])
    expect(
      pathDataFromPoints([
        { x: 1, y: 2 },
        { x: 3, y: 4 },
      ]),
    ).toEqual([
      ['M', 1, 2],
      ['L', 3, 4],
    ])
  })

  it('guards editable targets for global shortcuts/paste', () => {
    const input = document.createElement('input')
    expect(isEditableTarget(input)).toBe(true)
    const div = document.createElement('div')
    div.setAttribute('contenteditable', 'true')
    const span = document.createElement('span')
    div.appendChild(span)
    expect(isEditableTarget(span)).toBe(true)
    expect(isEditableTarget(document.createElement('canvas'))).toBe(false)
    expect(isEditableTarget(null)).toBe(false)
    const dialog = document.createElement('div')
    dialog.setAttribute('data-canvas-dialog', '')
    const inside = document.createElement('button')
    dialog.appendChild(inside)
    expect(isEditableTarget(inside)).toBe(true)
    // role="dialog" content teleported to body is also shielded.
    const roleDialog = document.createElement('div')
    roleDialog.setAttribute('role', 'dialog')
    const inner = document.createElement('div')
    roleDialog.appendChild(inner)
    expect(isEditableTarget(inner)).toBe(true)
  })
})

describe('canvasScene', () => {
  function makeImage(overrides: Partial<SceneObject> = {}): SceneObject {
    return {
      role: 'image',
      objectId: 'o1',
      assetKey: 'source:o1',
      left: 0,
      top: 0,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      width: 100,
      height: 100,
      ...overrides,
    }
  }

  it('round-trips a scene snapshot and rejects malformed input', () => {
    const snapshot = {
      version: CANVAS_SCENE_VERSION,
      viewport: [1, 0, 0, 1, 5, 5] as Mat6,
      objects: [makeImage()],
    }
    const decoded = decodeSceneSnapshot(encodeSceneSnapshot(snapshot))
    expect(decoded?.objects).toHaveLength(1)
    expect(decoded?.objects[0].assetKey).toBe('source:o1')
    expect(decodeSceneSnapshot('not json')).toBeNull()
    expect(decodeSceneSnapshot('{"version":1,"objects":[]}')).toBeNull()
    expect(decodeSceneSnapshot(null)).toBeNull()
  })

  it('lists asset references and placement identities', () => {
    const snapshot = {
      version: CANVAS_SCENE_VERSION,
      objects: [
        makeImage({ assetKey: 'a' }),
        makeImage({ objectId: 'o2', assetKey: 'b', runId: 'r1', outputIndex: 2 }),
        { ...makeImage({ objectId: 't', assetKey: undefined }), role: 'text' as const, text: 'x' },
      ],
    }
    expect(sceneAssetKeys(snapshot).sort()).toEqual(['a', 'b'])
    expect(placementIdentity({ runId: 'r1', outputIndex: 2 })).toBe('r1:2')
    expect(placementIdentity({})).toBeNull()
  })

  it('detects mask coverage only when add strokes exist', () => {
    const add = { mode: 'add' as const, shape: 'round' as const, width: 20, points: [{ x: 1, y: 1 }] }
    const erase = { mode: 'erase' as const, shape: 'round' as const, width: 20, points: [{ x: 1, y: 1 }] }
    expect(maskHasCoverage([add, erase])).toBe(true)
    expect(maskHasCoverage([erase])).toBe(false)
    expect(maskHasCoverage([])).toBe(false)
    expect(maskHasCoverage(undefined)).toBe(false)
  })

  it('computes stroke bounds and local containment for the anchor pick', () => {
    const bounds = strokeBounds([
      { x: 10, y: 20 },
      { x: 30, y: 5 },
      { x: 15, y: 40 },
    ])
    expect(bounds).toEqual({ minX: 10, minY: 5, maxX: 30, maxY: 40 })
    expect(strokeBounds([])).toBeNull()
    expect(pointInImageLocal({ x: 0, y: 0 }, 100, 50)).toBe(true)
    expect(pointInImageLocal({ x: 101, y: 0 }, 100, 50)).toBe(false)
  })
})
