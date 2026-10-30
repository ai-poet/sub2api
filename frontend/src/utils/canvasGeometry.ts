/**
 * canvasGeometry.ts holds the pure viewport and mask math for the canvas
 * editor. Keeping it Fabric-free lets the anchor invariants be unit-tested
 * without a real canvas 2d context.
 *
 * Matrices follow the Fabric/DOM 2x3 layout [a, b, c, d, e, f] mapping a
 * point {x, y} to {a*x + c*y + e, b*x + d*y + f}. The camera VPT has no
 * rotation or skew, so it is always [z, 0, 0, z, tx, ty].
 */

export interface Vec2 {
  x: number
  y: number
}

export type Mat6 = [number, number, number, number, number, number]

export const IDENTITY: Mat6 = [1, 0, 0, 1, 0, 0]

export const CANVAS_ZOOM_MIN = 0.2
export const CANVAS_ZOOM_MAX = 3

/** multiplyMatrices returns a * b — b applied to the point first. */
export function multiplyMatrices(a: Mat6, b: Mat6): Mat6 {
  return [
    a[0] * b[0] + a[2] * b[1],
    a[1] * b[0] + a[3] * b[1],
    a[0] * b[2] + a[2] * b[3],
    a[1] * b[2] + a[3] * b[3],
    a[0] * b[4] + a[2] * b[5] + a[4],
    a[1] * b[4] + a[3] * b[5] + a[5],
  ]
}

export function invertMatrix(m: Mat6): Mat6 {
  const det = m[0] * m[3] - m[1] * m[2]
  if (!det) return [...IDENTITY]
  const inv = 1 / det
  return [
    m[3] * inv,
    -m[1] * inv,
    -m[2] * inv,
    m[0] * inv,
    (m[2] * m[5] - m[3] * m[4]) * inv,
    (m[1] * m[4] - m[0] * m[5]) * inv,
  ]
}

export function applyMatrix(m: Mat6, p: Vec2): Vec2 {
  return { x: m[0] * p.x + m[2] * p.y + m[4], y: m[1] * p.x + m[3] * p.y + m[5] }
}

export function translation(x: number, y: number): Mat6 {
  return [1, 0, 0, 1, x, y]
}

/**
 * maskViewportMatrix maps natural image pixel coordinates to screen pixels:
 * camera * imageTransform * translate(-width/2, -height/2). A point drawn at
 * natural coordinate (px, py) therefore lands on the anchored image's
 * top-left corner when px/py are zero and follows every later move, scale,
 * and rotation of that object.
 */
export function maskViewportMatrix(camera: Mat6, imageMatrix: Mat6, width: number, height: number): Mat6 {
  return multiplyMatrices(multiplyMatrices(camera, imageMatrix), translation(-width / 2, -height / 2))
}

/**
 * clientToScene and sceneToClient convert between canvas element (client)
 * coordinates and scene coordinates through the camera VPT.
 */
export function clientToScene(camera: Mat6, client: Vec2): Vec2 {
  return applyMatrix(invertMatrix(camera), client)
}

export function sceneToClient(camera: Mat6, scene: Vec2): Vec2 {
  return applyMatrix(camera, scene)
}

/**
 * zoomAtClient returns a new camera that multiplies zoom by factor, clamped
 * to the allowed range, while keeping the given client point stationary —
 * the cursor stays over the same scene point.
 */
export function zoomAtClient(camera: Mat6, client: Vec2, factor: number): Mat6 {
  const zoom = clampZoom(camera[0] * factor)
  const ratio = zoom / camera[0]
  return [
    zoom,
    0,
    0,
    zoom,
    client.x - (client.x - camera[4]) * ratio,
    client.y - (client.y - camera[5]) * ratio,
  ]
}

export function clampZoom(value: number): number {
  return Math.min(CANVAS_ZOOM_MAX, Math.max(CANVAS_ZOOM_MIN, value))
}

export interface Bounds {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

export function emptyBounds(): Bounds {
  return { minX: Infinity, minY: Infinity, maxX: -Infinity, maxY: -Infinity }
}

export function extendBounds(bounds: Bounds, other: Bounds): Bounds {
  return {
    minX: Math.min(bounds.minX, other.minX),
    minY: Math.min(bounds.minY, other.minY),
    maxX: Math.max(bounds.maxX, other.maxX),
    maxY: Math.max(bounds.maxY, other.maxY),
  }
}

export function boundsValid(bounds: Bounds): boolean {
  return (
    Number.isFinite(bounds.minX) &&
    bounds.maxX >= bounds.minX &&
    bounds.maxY >= bounds.minY
  )
}

/**
 * fitViewport returns a camera that centers the scene bounds in the viewport
 * at the largest zoom that fits (clamped), with padding on each side.
 */
export function fitViewport(bounds: Bounds, viewW: number, viewH: number, padding = 48): Mat6 {
  const width = Math.max(1, bounds.maxX - bounds.minX)
  const height = Math.max(1, bounds.maxY - bounds.minY)
  const usableW = Math.max(1, viewW - padding * 2)
  const usableH = Math.max(1, viewH - padding * 2)
  const zoom = clampZoom(Math.min(usableW / width, usableH / height))
  const cx = (bounds.minX + bounds.maxX) / 2
  const cy = (bounds.minY + bounds.maxY) / 2
  return [zoom, 0, 0, zoom, viewW / 2 - cx * zoom, viewH / 2 - cy * zoom]
}

/**
 * pathDataFromPoints converts sampled pointer points into Fabric simple path
 * data (M followed by L commands), the representation PencilBrush exports.
 */
export function pathDataFromPoints(points: Vec2[]): Array<[string, ...number[]]> {
  if (points.length === 0) return []
  const data: Array<[string, ...number[]]> = [['M', points[0].x, points[0].y]]
  for (let i = 1; i < points.length; i += 1) {
    data.push(['L', points[i].x, points[i].y])
  }
  return data
}

/**
 * clipRectInImage computes the source-pixel rectangle a mask stroke can
 * affect: mask points live in natural image coordinates, so only the
 * [0, width] x [0, height] box participates in export.
 */
export function clipRectInImage(width: number, height: number): Bounds {
  return { minX: 0, minY: 0, maxX: width, maxY: height }
}

/**
 * isEditableTarget reports whether a DOM event target is a text-editing
 * surface. Global canvas shortcuts, paste, and drop ignore these so typing
 * in inputs, textareas, contenteditable nodes, dialogs, or IText editing is
 * never hijacked.
 */
export function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  if (target.isContentEditable) return true
  const tag = target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  return !!target.closest(
    '[data-canvas-dialog], [role="dialog"], input, textarea, [contenteditable="true"]',
  )
}
