<template>
  <Teleport to="body">
    <div v-if="current" class="crop-backdrop" data-canvas-dialog role="dialog" aria-modal="true">
      <div ref="modalEl" class="crop-modal" tabindex="-1" @keydown.esc.stop="cancelRest">
        <header class="crop-head">
          <h3>{{ t('canvas.crop.title') }}</h3>
          <span class="counter">{{ t('canvas.crop.counter', { current: index + 1, total: queue.length }) }}</span>
        </header>
        <p class="hint">{{ t('canvas.crop.hint') }}</p>
        <canvas
          ref="canvasEl"
          class="crop-canvas"
          tabindex="0"
          @pointerdown="onDown"
          @pointermove="onMove"
          @pointerup="onUp"
          @pointercancel="onUp"
        />
        <div class="crop-actions">
          <button class="btn secondary" @click="skip">{{ t('canvas.crop.skip') }}</button>
          <button class="btn secondary" @click="cancelRest">{{ t('canvas.crop.cancelRest') }}</button>
          <button class="btn primary" :disabled="applying" @click="apply">{{ t('canvas.crop.confirm') }}</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
/**
 * Native rectangle crop modal. Takes a queue of blobs from the editor's
 * crop-request seam; emits the resulting blobs in order so the editor can
 * importBlobs them. No external dependency.
 */
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ queue: Blob[] }>()
const emit = defineEmits<{
  done: [blobs: Blob[]]
  cancel: []
}>()

const { t } = useI18n()
const canvasEl = ref<HTMLCanvasElement | null>(null)
const modalEl = ref<HTMLDivElement | null>(null)
let prevFocus: Element | null = null
const index = ref(0)
const current = ref<Blob | null>(null)
const results = ref<Blob[]>([])
const applying = ref(false)

let img: HTMLImageElement | null = null
let imgUrl = ''
// Display rect of the image inside the canvas, and crop rect in image px.
let disp = { x: 0, y: 0, w: 0, h: 0 }
let rect = { x: 0, y: 0, w: 0, h: 0 }
let drag: { mode: 'move' | 'nw' | 'ne' | 'sw' | 'se'; startX: number; startY: number; orig: typeof rect } | null = null
const HANDLE = 10

watch(
  () => props.queue,
  async (q) => {
    if (q.length === 0) return
    index.value = 0
    results.value = []
    await loadCurrent()
  },
  { immediate: true },
)

async function loadCurrent(): Promise<void> {
  const firstImage = index.value === 0 && !prevFocus
  if (firstImage) prevFocus = document.activeElement
  const blob = props.queue[index.value]
  if (!blob) {
    current.value = null
    ;(prevFocus as HTMLElement | null)?.focus?.()
    prevFocus = null
    emit('done', results.value)
    return
  }
  current.value = blob
  await nextTick()
  // The dialog only renders once `current` is set, so focus must move after
  // that render; otherwise Escape and the editor's modal guard never see it.
  if (firstImage) modalEl.value?.focus()
  imgUrl = URL.createObjectURL(blob)
  img = new Image()
  img.onload = () => {
    const cv = canvasEl.value!
    const cw = cv.clientWidth || 480
    const scale = Math.min(cw / img!.width, 320 / img!.height, 1)
    cv.width = img!.width * scale
    cv.height = img!.height * scale
    disp = { x: 0, y: 0, w: cv.width, h: cv.height }
    rect = { x: 0, y: 0, w: img!.width, h: img!.height }
    draw()
  }
  img.src = imgUrl
}

function draw(): void {
  const cv = canvasEl.value
  const ctx = cv?.getContext('2d')
  if (!cv || !ctx || !img) return
  ctx.clearRect(0, 0, cv.width, cv.height)
  ctx.drawImage(img, disp.x, disp.y, disp.w, disp.h)
  const sx = disp.w / img.width
  const sy = disp.h / img.height
  const rx = disp.x + rect.x * sx
  const ry = disp.y + rect.y * sy
  const rw = rect.w * sx
  const rh = rect.h * sy
  ctx.fillStyle = 'rgba(0,0,0,0.4)'
  ctx.fillRect(0, 0, cv.width, ry)
  ctx.fillRect(0, ry + rh, cv.width, cv.height - ry - rh)
  ctx.fillRect(0, ry, rx, rh)
  ctx.fillRect(rx + rw, ry, cv.width - rx - rw, rh)
  ctx.strokeStyle = '#14b8a6'
  ctx.lineWidth = 2
  ctx.strokeRect(rx, ry, rw, rh)
  ctx.fillStyle = '#14b8a6'
  for (const [hx, hy] of [[rx, ry], [rx + rw, ry], [rx, ry + rh], [rx + rw, ry + rh]]) {
    ctx.fillRect(hx - HANDLE / 2, hy - HANDLE / 2, HANDLE, HANDLE)
  }
}

function toImage(e: PointerEvent): { x: number; y: number } {
  const cv = canvasEl.value!
  const b = cv.getBoundingClientRect()
  const sx = img!.width / disp.w
  const sy = img!.height / disp.h
  return { x: (e.clientX - b.left - disp.x) * sx, y: (e.clientY - b.top - disp.y) * sy }
}

function onDown(e: PointerEvent): void {
  if (!img) return
  const p = toImage(e)
  const sx = img.width / disp.w
  const hx = HANDLE * sx
  const near = (a: number, b: number) => Math.abs(a - b) <= hx
  let mode: typeof drag extends null ? never : 'move' | 'nw' | 'ne' | 'sw' | 'se'
  if (near(p.x, rect.x) && near(p.y, rect.y)) mode = 'nw'
  else if (near(p.x, rect.x + rect.w) && near(p.y, rect.y)) mode = 'ne'
  else if (near(p.x, rect.x) && near(p.y, rect.y + rect.h)) mode = 'sw'
  else if (near(p.x, rect.x + rect.w) && near(p.y, rect.y + rect.h)) mode = 'se'
  else if (p.x > rect.x && p.x < rect.x + rect.w && p.y > rect.y && p.y < rect.y + rect.h) mode = 'move'
  else {
    // Click outside starts a new rect.
    rect = { x: p.x, y: p.y, w: 0, h: 0 }
    mode = 'se'
  }
  drag = { mode, startX: p.x, startY: p.y, orig: { ...rect } }
  canvasEl.value?.setPointerCapture?.(e.pointerId)
}

function onMove(e: PointerEvent): void {
  if (!drag || !img) return
  const p = toImage(e)
  const dx = p.x - drag.startX
  const dy = p.y - drag.startY
  const o = drag.orig
  const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v))
  if (drag.mode === 'move') {
    rect = { ...o, x: clamp(o.x + dx, 0, img.width - o.w), y: clamp(o.y + dy, 0, img.height - o.h) }
  } else {
    let { x, y, w, h } = o
    if (drag.mode.includes('e')) w = clamp(o.w + dx, 8, img.width - x)
    if (drag.mode.includes('s')) h = clamp(o.h + dy, 8, img.height - y)
    if (drag.mode.includes('w')) {
      const nx = clamp(o.x + dx, 0, o.x + o.w - 8)
      w = o.w + (o.x - nx)
      x = nx
    }
    if (drag.mode.includes('n')) {
      const ny = clamp(o.y + dy, 0, o.y + o.h - 8)
      h = o.h + (o.y - ny)
      y = ny
    }
    rect = { x, y, w, h }
  }
  draw()
}

function onUp(): void {
  drag = null
}

async function apply(): Promise<void> {
  // Ignore a double-click while the previous crop is still being encoded.
  if (applying.value || !img || !current.value) return
  applying.value = true
  try {
    await applyOnce()
  } finally {
    applying.value = false
  }
}

async function applyOnce(): Promise<void> {
  if (!img || !current.value) return
  const off = document.createElement('canvas')
  const w = Math.max(1, Math.round(rect.w))
  const h = Math.max(1, Math.round(rect.h))
  off.width = w
  off.height = h
  const ctx = off.getContext('2d')!
  ctx.drawImage(img, rect.x, rect.y, rect.w, rect.h, 0, 0, w, h)
  const type = current.value.type === 'image/png' ? 'image/png' : current.value.type
  const blob = await new Promise<Blob | null>((res) => off.toBlob(res, type))
  results.value.push(blob ?? current.value)
  await next()
}

async function skip(): Promise<void> {
  results.value.push(current.value!)
  await next()
}

async function next(): Promise<void> {
  if (imgUrl) URL.revokeObjectURL(imgUrl)
  imgUrl = ''
  img = null
  index.value += 1
  await loadCurrent()
}

/**
 * cancelRest stops the queue but keeps every image the user already
 * confirmed (cropped or skipped). It emits 'done' with the confirmed set —
 * 'cancel' fires only when nothing was confirmed yet.
 */
function cancelRest(): void {
  if (imgUrl) URL.revokeObjectURL(imgUrl)
  imgUrl = ''
  img = null
  ;(prevFocus as HTMLElement | null)?.focus?.()
  prevFocus = null
  const confirmed = results.value
  results.value = []
  index.value = 0
  current.value = null
  if (confirmed.length > 0) emit('done', confirmed)
  else emit('cancel')
}

onBeforeUnmount(() => {
  if (imgUrl) URL.revokeObjectURL(imgUrl)
  ;(prevFocus as HTMLElement | null)?.focus?.()
})

void props
</script>

<style scoped>
.crop-backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,0.55); z-index: 60;
  display: flex; align-items: center; justify-content: center;
}
.crop-modal {
  background: #fff; border-radius: 16px; padding: 18px; max-width: 92vw;
  box-shadow: 0 16px 48px rgba(0,0,0,0.3);
}
.dark .crop-modal { background: #111827; }
.crop-head { display: flex; justify-content: space-between; align-items: baseline; }
.crop-head h3 { font-size: 15px; font-weight: 600; }
.counter { font-size: 12px; color: #9ca3af; }
.hint { font-size: 12px; color: #6b7280; margin: 4px 0 10px; }
.crop-canvas { display: block; max-width: 80vw; max-height: 50vh; touch-action: none; }
.crop-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 12px; }
.btn { padding: 7px 16px; border-radius: 10px; font-size: 13px; }
.btn.primary { background: #14b8a6; color: #fff; }
.btn.secondary { border: 1px solid rgba(0,0,0,0.15); color: #4b5563; }
.dark .btn.secondary { border-color: rgba(255,255,255,0.15); color: #9ca3af; }
</style>
