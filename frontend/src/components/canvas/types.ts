/**
 * Shared types for the Canvas Studio editor components. Kept in a plain
 * module because `<script setup>` cannot export symbols reused across
 * components.
 */

import type { CanvasScope } from '@/types/canvas'

export type { CanvasScope }

/** Pointer tools exposed by the toolbar and the V/B/E/T shortcuts. */
export type CanvasTool = 'select' | 'pan' | 'brush' | 'eraser' | 'text'

/** Mask brush stamp shape. */
export type CanvasBrushShape = 'round' | 'square'

/**
 * EditorStateSnapshot is emitted on every `state` event so the toolbar can
 * enable/disable affordances without reaching into Fabric internals.
 */
export interface EditorStateSnapshot {
  /** Scene restore finished; the editor accepts input. */
  ready: boolean
  /** Number of image objects currently on the canvas. */
  imageCount: number
  /** Number of selected objects (images + text). */
  selectedCount: number
  /** Number of selected image objects. */
  selectedImageCount: number
  /** Whether the current mask anchor has any strokes. */
  hasMask: boolean
  canUndo: boolean
  canRedo: boolean
  /** True while a corrupt stored scene blocks all persistence writes. */
  saveBlocked: boolean
  /** Current camera zoom, 0.2–3. */
  zoom: number
}

/** Payload of the `crop-request` event (2b wires the crop modal). */
export interface CropRequestPayload {
  blobs: Blob[]
  /** Scene-space drop point when the import came from a drag, else null. */
  at: { x: number; y: number } | null
}

/** placeOutput argument: identity fields drive scene-level dedup. */
export interface PlaceOutputInput {
  blob: Blob
  runId: string
  outputIndex: number
  /** Asset key when the blob is already persisted; reused, not duplicated. */
  assetKey?: string
  at?: { x: number; y: number } | null
}

/** exportInput result consumed by the submit pipeline. */
export interface ExportInputResult {
  images: Blob[]
  mask: Blob | null
}
