/**
 * canvasMask.ts rasterizes per-image mask strokes in natural image pixel
 * coordinates. It operates on an injected CanvasRenderingContext2D so the
 * exact pixel semantics can be verified with a software 2d context in tests.
 *
 * Two render modes share one stroke coverage definition:
 *  - preview: 'add' strokes painted in MASK_COLOR at full alpha, 'erase'
 *    strokes cut back out; callers composite the buffer once at a uniform
 *    opacity so overlapping strokes never darken.
 *  - export: untouched area is fully opaque white and painted (redraw) area
 *    is transparent — the polarity the image-edit backends expect. 'add'
 *    strokes punch alpha out, 'erase' strokes restore alpha.
 */

import type { MaskStroke } from './canvasScene'
import type { Vec2 } from './canvasGeometry'

export const MASK_PREVIEW_COLOR = 'rgb(244, 63, 94)'
export const MASK_EXPORT_COLOR = 'rgb(255, 255, 255)'

/**
 * squareStampCenters interpolates stamp centers along a polyline at half the
 * stroke width so consecutive square stamps overlap into a solid stroke.
 */
export function squareStampCenters(points: Vec2[], width: number): Vec2[] {
  const step = Math.max(1, width / 2)
  const centers: Vec2[] = []
  const push = (x: number, y: number) => {
    const last = centers[centers.length - 1]
    if (!last || Math.hypot(x - last.x, y - last.y) >= step * 0.9) {
      centers.push({ x, y })
    }
  }
  if (points.length === 0) return centers
  push(points[0].x, points[0].y)
  for (let i = 1; i < points.length; i += 1) {
    const from = points[i - 1]
    const to = points[i]
    const dist = Math.hypot(to.x - from.x, to.y - from.y)
    const segments = Math.max(1, Math.ceil(dist / step))
    for (let s = 1; s <= segments; s += 1) {
      push(from.x + ((to.x - from.x) * s) / segments, from.y + ((to.y - from.y) * s) / segments)
    }
  }
  return centers
}

/**
 * drawStrokeCoverage fills one stroke into ctx with the given color. Round
 * strokes render as round-cap/join polylines; square strokes as axis-aligned
 * stamp squares along the interpolated centers.
 */
export function drawStrokeCoverage(
  ctx: CanvasRenderingContext2D,
  stroke: MaskStroke,
  color: string,
): void {
  const { points, width } = stroke
  if (points.length === 0 || width <= 0) return
  ctx.fillStyle = color
  if (stroke.shape === 'square') {
    const half = width / 2
    for (const c of squareStampCenters(points, width)) {
      ctx.fillRect(c.x - half, c.y - half, width, width)
    }
    return
  }
  ctx.strokeStyle = color
  ctx.lineWidth = width
  ctx.lineJoin = 'round'
  ctx.lineCap = 'round'
  if (points.length === 1) {
    ctx.beginPath()
    ctx.arc(points[0].x, points[0].y, width / 2, 0, Math.PI * 2)
    ctx.fill()
    return
  }
  ctx.beginPath()
  ctx.moveTo(points[0].x, points[0].y)
  for (let i = 1; i < points.length; i += 1) {
    ctx.lineTo(points[i].x, points[i].y)
  }
  ctx.stroke()
}

/** renderMaskPreview draws coverage strokes in MASK_COLOR at full alpha. */
export function renderMaskPreview(
  ctx: CanvasRenderingContext2D,
  strokes: MaskStroke[],
  width: number,
  height: number,
  color: string = MASK_PREVIEW_COLOR,
): void {
  ctx.save()
  ctx.clearRect(0, 0, width, height)
  ctx.globalCompositeOperation = 'source-over'
  for (const stroke of strokes) {
    ctx.globalCompositeOperation =
      stroke.mode === 'erase' ? 'destination-out' : 'source-over'
    drawStrokeCoverage(ctx, stroke, color)
  }
  ctx.restore()
}

/**
 * renderMaskExport draws the backend mask: untouched pixels opaque white,
 * redraw (painted) pixels transparent. Erase strokes restore alpha.
 */
export function renderMaskExport(
  ctx: CanvasRenderingContext2D,
  strokes: MaskStroke[],
  width: number,
  height: number,
): void {
  ctx.save()
  ctx.clearRect(0, 0, width, height)
  ctx.globalCompositeOperation = 'source-over'
  ctx.fillStyle = MASK_EXPORT_COLOR
  ctx.fillRect(0, 0, width, height)
  for (const stroke of strokes) {
    ctx.globalCompositeOperation =
      stroke.mode === 'erase' ? 'source-over' : 'destination-out'
    drawStrokeCoverage(ctx, stroke, stroke.mode === 'erase' ? MASK_EXPORT_COLOR : '#000')
  }
  ctx.restore()
}

/**
 * hasRedrawArea reports whether the exported mask still contains at least
 * one fully transparent pixel — i.e. a painted region actually survives.
 * Pure-erase or fully erased strokes must reject before submit.
 */
export function hasRedrawArea(ctx: CanvasRenderingContext2D, width: number, height: number): boolean {
  const data = ctx.getImageData(0, 0, width, height).data
  for (let i = 3; i < data.length; i += 4) {
    if (data[i] === 0) return true
  }
  return false
}
