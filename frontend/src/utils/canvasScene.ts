/**
 * canvasScene.ts holds the pure scene-snapshot format for the canvas editor:
 * what a scene record looks like on disk, how image src is swapped for
 * durable asset references, and the per-image mask stroke model. Keeping it
 * free of Fabric lets snapshot integrity and asset bookkeeping be tested
 * without a canvas 2d context.
 */

import type { Vec2 } from './canvasGeometry'

/** Format marker for stored scene JSON. */
export const CANVAS_SCENE_VERSION = 2

/** Object roles the editor distinguishes. */
export type CanvasObjectRole = 'image' | 'text'

/**
 * MaskStroke is one brush/eraser stroke captured in natural image pixel
 * coordinates (0..width, 0..height of the anchored source). Points are the
 * sampled polyline vertices; rendering derives round-cap or square-stamp
 * coverage from them, so the stored form survives brush-shape changes.
 */
export interface MaskStroke {
  mode: 'add' | 'erase'
  shape: 'round' | 'square'
  /** Stroke width in natural image pixels. */
  width: number
  /** Sampled polyline vertices in natural pixel coordinates. */
  points: Array<{ x: number; y: number }>
}

/**
 * SceneObject is the serializable form of one canvas object. `src` is
 * replaced by `asset://<key>` on save so snapshots never embed blob: URLs
 * or base64 payloads.
 */
export interface SceneObject {
  role: CanvasObjectRole
  /** Stable per-scene object id (uuid). */
  objectId: string
  /** Local-store asset key for image objects. */
  assetKey?: string
  /** Durable 'asset://<key>' reference; retention protects these. */
  src?: string
  /** Run/output identity for placed generation results (scene-level dedup). */
  runId?: string
  outputIndex?: number
  /**
   * World transform matrix [a,b,c,d,e,f] from calcTransformMatrix, captured
   * so multi-selection (group) transforms persist losslessly. Legacy
   * left/top/scale/angle fields are kept optional for older records.
   */
  transform?: [number, number, number, number, number, number]
  left?: number
  top?: number
  scaleX?: number
  scaleY?: number
  angle?: number
  /** Natural pixel dimensions of the source image. */
  width?: number
  height?: number
  /** IText payload. */
  text?: string
  fontSize?: number
  fill?: string
  /** Per-image mask strokes in natural pixel coordinates. */
  maskStrokes?: MaskStroke[]
}

/** A persisted scene snapshot. */
export interface SceneSnapshot {
  version: number
  /** Camera VPT saved with the scene. */
  viewport?: [number, number, number, number, number, number]
  objects: SceneObject[]
}

/**
 * maskHasCoverage reports whether strokes contain at least one 'add' stroke;
 * pure erase strokes produce an empty mask and are treated as no mask.
 */
export function maskHasCoverage(strokes: MaskStroke[] | undefined): boolean {
  return !!strokes && strokes.some((stroke) => stroke.mode === 'add' && stroke.points.length > 0)
}

/**
 * encodeSceneSnapshot serializes a snapshot deterministically for storage.
 */
export function encodeSceneSnapshot(snapshot: SceneSnapshot): string {
  return JSON.stringify(snapshot)
}

/**
 * decodeSceneSnapshot parses a stored scene JSON string. Returns null on
 * malformed input or an unsupported version so a corrupt record can never
 * overwrite the in-memory scene.
 */
export function decodeSceneSnapshot(raw: string | null | undefined): SceneSnapshot | null {
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as SceneSnapshot
    if (parsed?.version !== CANVAS_SCENE_VERSION || !Array.isArray(parsed.objects)) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

/**
 * sceneAssetKeys lists every asset key a snapshot references — used to warn
 * (not silently drop) when persisted bytes are missing at restore.
 */
export function sceneAssetKeys(snapshot: SceneSnapshot): string[] {
  return snapshot.objects
    .map((object) => object.assetKey)
    .filter((key): key is string => typeof key === 'string' && key.length > 0)
}

/**
 * placementIdentity returns the stable runId:index key used to dedup placed
 * generation outputs in the scene, or null for objects without identity.
 */
export function placementIdentity(object: Pick<SceneObject, 'runId' | 'outputIndex'>): string | null {
  if (object.runId === undefined || object.outputIndex === undefined) return null
  return `${object.runId}:${object.outputIndex}`
}

/**
 * strokeBounds computes the bounding box of a stroke's points in natural
 * coordinates — used to reject fully out-of-source strokes at export.
 */
export function strokeBounds(points: Array<{ x: number; y: number }>): {
  minX: number
  minY: number
  maxX: number
  maxY: number
} | null {
  if (points.length === 0) return null
  let minX = Infinity
  let minY = Infinity
  let maxX = -Infinity
  let maxY = -Infinity
  for (const p of points) {
    minX = Math.min(minX, p.x)
    minY = Math.min(minY, p.y)
    maxX = Math.max(maxX, p.x)
    maxY = Math.max(maxY, p.y)
  }
  return { minX, minY, maxX, maxY }
}

/**
 * scenePointInImage reports whether a scene-space point falls inside an
 * image object's natural bounds when mapped through its transform — the
 * anchor pick uses the inverse image matrix for this containment test.
 */
export function pointInImageLocal(local: Vec2, width: number, height: number): boolean {
  return local.x >= 0 && local.x <= width && local.y >= 0 && local.y <= height
}
