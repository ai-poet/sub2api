<script setup lang="ts">
/**
 * CanvasEditor.vue 是画布工作台的核心编辑器。
 *
 * 主画布保存底图与文本标注；另一张透明画布只保存遮罩笔迹。
 * 两张画布使用相同的逻辑尺寸与 CSS 尺寸，因此指针坐标和导出像素始终对齐。
 * 快照同时包含两张画布，撤销/重做不会只恢复一半现场。
 */
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { Canvas, FabricImage, FabricObject, FabricText, PencilBrush } from 'fabric'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { buildComposite, type CompositeResult } from '@/utils/canvasComposite'
import type { CanvasObjectRole, CanvasTool } from './types'

const HISTORY_LIMIT = 50

const props = withDefaults(
  defineProps<{
    tool: CanvasTool
    brushSize: number
    maskColor: string
    textColor: string
    background: string
    /** 输出逻辑尺寸，如 1024x1024。 */
    size?: string
  }>(),
  { size: '1024x1024' },
)

const emit = defineEmits<{
  (event: 'change'): void
  (event: 'select', payload: { count: number }): void
  (event: 'history', payload: { canUndo: boolean; canRedo: boolean }): void
  (event: 'composite', payload: CompositeResult): void
}>()

const canvasElement = ref<HTMLCanvasElement | null>(null)
const maskElement = ref<HTMLCanvasElement | null>(null)
const fabricCanvas = shallowRef<Canvas | null>(null)
const maskCanvas = shallowRef<Canvas | null>(null)
const sourceLayer = shallowRef<FabricImage | null>(null)

const canvasWidth = ref(1024)
const canvasHeight = ref(1024)
const objectCount = ref(0)
const compositing = ref(false)
const ready = ref(false)
const historyStack = ref<string[]>([])
const historyIndex = ref(-1)

const canUndo = computed(() => historyIndex.value > 0)
const canRedo = computed(
  () => historyIndex.value >= 0 && historyIndex.value < historyStack.value.length - 1,
)
const isEmpty = computed(() => objectCount.value === 0)
const displayRatio = computed(() => `${canvasWidth.value} / ${canvasHeight.value}`)

function parseSize(value: string): { width: number; height: number } {
  const match = /^(\d+)x(\d+)$/.exec(value)
  if (!match) return { width: 1024, height: 1024 }
  return { width: Number(match[1]), height: Number(match[2]) }
}

function readRole(object: FabricObject): CanvasObjectRole | '' {
  const data = (object as unknown as { data?: Record<string, unknown> }).data
  const role = data?.canvasRole
  return role === 'source' || role === 'annotation' ? role : ''
}

function writeRole(object: FabricObject, role: CanvasObjectRole): void {
  const target = object as unknown as { data?: Record<string, unknown> }
  target.data = { ...(target.data ?? {}), canvasRole: role }
}

/** toObject(['data']) 是关键：默认序列化不会带自定义 data，恢复后底图角色会丢失。 */
function captureSnapshot(): string {
  const canvas = fabricCanvas.value
  const mask = maskCanvas.value
  if (!canvas || !mask) return ''
  return JSON.stringify({
    version: 1,
    width: canvasWidth.value,
    height: canvasHeight.value,
    scene: canvas.toObject(['data']),
    mask: mask.toObject(),
  })
}

function emitHistory(): void {
  emit('history', { canUndo: canUndo.value, canRedo: canRedo.value })
}

/** 历史保存「每次操作完成后」的状态，避免在 mouse:down 时猜测操作是否会发生。 */
function commitHistory(): void {
  const snapshot = captureSnapshot()
  if (!snapshot || historyStack.value[historyIndex.value] === snapshot) {
    emitHistory()
    return
  }
  const current = historyStack.value.slice(0, historyIndex.value + 1)
  const next = [...current, snapshot].slice(-HISTORY_LIMIT)
  historyStack.value = next
  historyIndex.value = next.length - 1
  emitHistory()
}

function resetHistory(): void {
  const snapshot = captureSnapshot()
  historyStack.value = snapshot ? [snapshot] : []
  historyIndex.value = snapshot ? 0 : -1
  emitHistory()
}

function syncSceneState(): void {
  const canvas = fabricCanvas.value
  if (!canvas) return
  objectCount.value = canvas.getObjects().length
  sourceLayer.value =
    (canvas.getObjects().find((object) => readRole(object) === 'source') as FabricImage | undefined) ??
    null
}

function fitSource(): void {
  const image = sourceLayer.value
  if (!image || !image.width || !image.height) return
  const scale = Math.min(canvasWidth.value / image.width, canvasHeight.value / image.height)
  image.set({
    scaleX: scale,
    scaleY: scale,
    left: (canvasWidth.value - image.width * scale) / 2,
    top: (canvasHeight.value - image.height * scale) / 2,
  })
  image.setCoords()
}

function applyTool(): void {
  const canvas = fabricCanvas.value
  const mask = maskCanvas.value
  if (!canvas || !mask) return

  // 主画布永远由 Fabric 负责对象选择；自由绘制只发生在遮罩画布。
  canvas.isDrawingMode = false
  canvas.selection = props.tool === 'select'
  for (const object of canvas.getObjects()) {
    const selectable = props.tool === 'select' && readRole(object) !== 'source'
    object.selectable = selectable
    object.evented = selectable
  }

  const drawing = props.tool === 'brush' || props.tool === 'eraser'
  mask.isDrawingMode = drawing
  mask.selection = false
  const brush = mask.freeDrawingBrush as PencilBrush | undefined
  if (brush) {
    brush.width = props.brushSize
    brush.color = drawing && props.tool === 'eraser' ? 'rgba(255,255,255,0.8)' : props.maskColor
  }
  canvas.requestRenderAll()
}

function setCssSize(canvas: Canvas): void {
  // Fabric 的 getScenePoint 会把 CSS 缩放换算回逻辑像素，不能用 CSS transform 缩放容器。
  canvas.setDimensions({ width: '100%', height: '100%' }, { cssOnly: true })
}

async function applySnapshot(serialized: string): Promise<void> {
  const canvas = fabricCanvas.value
  const mask = maskCanvas.value
  if (!canvas || !mask || !serialized) return

  const parsed = JSON.parse(serialized) as {
    width?: number
    height?: number
    scene: Record<string, unknown>
    mask?: Record<string, unknown>
  }
  canvasWidth.value = parsed.width || 1024
  canvasHeight.value = parsed.height || 1024
  canvas.setDimensions({ width: canvasWidth.value, height: canvasHeight.value })
  mask.setDimensions({ width: canvasWidth.value, height: canvasHeight.value })

  await canvas.loadFromJSON(parsed.scene)
  await mask.loadFromJSON(parsed.mask ?? { objects: [] })
  for (const path of mask.getObjects()) {
    path.selectable = false
    path.evented = false
  }
  syncSceneState()
  applyTool()
  setCssSize(canvas)
  setCssSize(mask)
  canvas.discardActiveObject()
  canvas.requestRenderAll()
  mask.requestRenderAll()
  emit('select', { count: 0 })
}

async function undo(): Promise<void> {
  if (!canUndo.value) return
  historyIndex.value -= 1
  await applySnapshot(historyStack.value[historyIndex.value])
  emitHistory()
  emit('change')
}

async function redo(): Promise<void> {
  if (!canRedo.value) return
  historyIndex.value += 1
  await applySnapshot(historyStack.value[historyIndex.value])
  emitHistory()
  emit('change')
}

function scaleObjects(canvas: Canvas, scaleX: number, scaleY: number, skipSource: boolean): void {
  for (const object of canvas.getObjects()) {
    if (skipSource && readRole(object) === 'source') continue
    object.set({
      left: object.left * scaleX,
      top: object.top * scaleY,
      scaleX: object.scaleX * scaleX,
      scaleY: object.scaleY * scaleY,
    })
    object.setCoords()
  }
}

function resize(size: string, record = true): void {
  const canvas = fabricCanvas.value
  const mask = maskCanvas.value
  if (!canvas || !mask) return
  const next = parseSize(size)
  if (next.width === canvasWidth.value && next.height === canvasHeight.value) return
  const scaleX = next.width / canvasWidth.value
  const scaleY = next.height / canvasHeight.value
  scaleObjects(canvas, scaleX, scaleY, true)
  scaleObjects(mask, scaleX, scaleY, false)
  canvasWidth.value = next.width
  canvasHeight.value = next.height
  canvas.setDimensions(next)
  mask.setDimensions(next)
  fitSource()
  setCssSize(canvas)
  setCssSize(mask)
  applyTool()
  if (record) {
    commitHistory()
    emit('change')
  }
}

async function addImage(file: Blob): Promise<void> {
  const canvas = fabricCanvas.value
  if (!canvas) return
  const url = URL.createObjectURL(file)
  try {
    const image = await FabricImage.fromURL(url)
    if (!image.width || !image.height) throw new Error('Invalid image')
    image.set({ selectable: false, evented: false })
    writeRole(image, 'source')
    if (sourceLayer.value) canvas.remove(sourceLayer.value)
    sourceLayer.value = image
    fitSource()
    canvas.insertAt(0, image)
    syncSceneState()
    applyTool()
    canvas.requestRenderAll()
    commitHistory()
    emit('change')
  } finally {
    URL.revokeObjectURL(url)
  }
}

function addText(text: string): void {
  const canvas = fabricCanvas.value
  if (!canvas || !text.trim()) return
  const label = new FabricText(text.trim(), {
    left: canvasWidth.value * 0.08,
    top: canvasHeight.value * 0.08,
    fill: props.textColor,
    fontSize: Math.max(20, Math.round(canvasWidth.value / 32)),
    fontFamily: 'Inter, system-ui, sans-serif',
  })
  writeRole(label, 'annotation')
  canvas.add(label)
  syncSceneState()
  applyTool()
  canvas.setActiveObject(label)
  canvas.requestRenderAll()
  emit('select', { count: 1 })
  commitHistory()
  emit('change')
}

function removeSelected(): void {
  const canvas = fabricCanvas.value
  if (!canvas) return
  const removable = canvas.getActiveObjects().filter((object) => readRole(object) !== 'source')
  if (removable.length === 0) return
  canvas.discardActiveObject()
  canvas.remove(...removable)
  syncSceneState()
  canvas.requestRenderAll()
  emit('select', { count: 0 })
  commitHistory()
  emit('change')
}

function clearCanvas(): void {
  const canvas = fabricCanvas.value
  const mask = maskCanvas.value
  if (!canvas || !mask) return
  canvas.discardActiveObject()
  canvas.clear()
  canvas.backgroundColor = props.background
  mask.clear()
  mask.backgroundColor = ''
  sourceLayer.value = null
  syncSceneState()
  applyTool()
  canvas.requestRenderAll()
  mask.requestRenderAll()
  emit('select', { count: 0 })
  commitHistory()
  emit('change')
}

function hasContent(): boolean {
  return (fabricCanvas.value?.getObjects().length ?? 0) > 0
}

async function composite(): Promise<CompositeResult | null> {
  const canvas = fabricCanvas.value
  const mask = maskCanvas.value
  if (!canvas || !mask || !hasContent()) return null
  compositing.value = true
  try {
    // viewportTransform 始终为单位矩阵；导出的即是用户选择的逻辑尺寸。
    const result = await buildComposite(canvas.toCanvasElement(1), mask.toCanvasElement(1), {
      width: canvasWidth.value,
      height: canvasHeight.value,
      maskWidth: canvasWidth.value,
      maskHeight: canvasHeight.value,
      background: props.background,
    })
    if (result) emit('composite', result)
    return result
  } finally {
    compositing.value = false
  }
}

function serialize(): string {
  return captureSnapshot()
}

async function restore(serialized: string): Promise<void> {
  if (!serialized) return
  await applySnapshot(serialized)
  resetHistory()
  emit('change')
}

async function loadResultImage(blob: Blob): Promise<void> {
  await addImage(blob)
}

function setBrushSize(size: number): void {
  const brush = maskCanvas.value?.freeDrawingBrush as PencilBrush | undefined
  if (brush) brush.width = size
}

defineExpose({
  addImage,
  addText,
  removeSelected,
  clearCanvas,
  hasContent,
  composite,
  serialize,
  restore,
  loadResultImage,
  undo,
  redo,
  setBrushSize,
})

onMounted(() => {
  const element = canvasElement.value
  const maskEl = maskElement.value
  if (!element || !maskEl) return
  const initial = parseSize(props.size)
  canvasWidth.value = initial.width
  canvasHeight.value = initial.height

  const canvas = new Canvas(element, {
    ...initial,
    backgroundColor: props.background,
    preserveObjectStacking: true,
    selection: props.tool === 'select',
  })
  const mask = new Canvas(maskEl, {
    ...initial,
    backgroundColor: '',
    selection: false,
    skipTargetFind: true,
    isDrawingMode: true,
  })
  mask.freeDrawingBrush = new PencilBrush(mask)
  fabricCanvas.value = canvas
  maskCanvas.value = mask

  canvas.on('selection:created', () => emit('select', { count: canvas.getActiveObjects().length }))
  canvas.on('selection:updated', () => emit('select', { count: canvas.getActiveObjects().length }))
  canvas.on('selection:cleared', () => emit('select', { count: 0 }))
  canvas.on('object:modified', () => {
    commitHistory()
    emit('change')
  })
  canvas.on('text:changed', () => {
    commitHistory()
    emit('change')
  })
  mask.on('path:created', ({ path }) => {
    // 白色只是绘制中的预览；最终 path 用 destination-out 真正擦除透明通道。
    if (props.tool === 'eraser') path.globalCompositeOperation = 'destination-out'
    path.selectable = false
    path.evented = false
    mask.requestRenderAll()
    commitHistory()
    emit('change')
  })

  setCssSize(canvas)
  setCssSize(mask)
  applyTool()
  ready.value = true
  resetHistory()
})

onBeforeUnmount(() => {
  ready.value = false
  void fabricCanvas.value?.dispose()
  void maskCanvas.value?.dispose()
  fabricCanvas.value = null
  maskCanvas.value = null
})

watch(() => props.tool, applyTool)
watch(() => props.brushSize, setBrushSize)
watch(() => props.maskColor, applyTool)
watch(() => props.size, (value) => {
  if (ready.value) resize(value)
})
watch(() => props.background, (value) => {
  if (!fabricCanvas.value) return
  fabricCanvas.value.backgroundColor = value
  fabricCanvas.value.requestRenderAll()
})
</script>

<template>
  <div class="flex h-full w-full items-start justify-center overflow-auto bg-gray-100 p-4 dark:bg-dark-900">
    <div class="relative w-full max-w-3xl shrink-0" :style="{ aspectRatio: displayRatio }">
      <div class="absolute inset-0">
        <canvas ref="canvasElement" />
      </div>

      <!-- 遮罩层始终可见，但只有笔刷工具接收指针事件。 -->
      <div
        class="absolute inset-0 z-10"
        :class="tool === 'brush' || tool === 'eraser' ? 'cursor-crosshair' : 'pointer-events-none'"
      >
        <canvas ref="maskElement" />
      </div>

      <div
        v-if="isEmpty"
        class="pointer-events-none absolute inset-0 z-20 flex flex-col items-center justify-center gap-2 text-center"
      >
        <Icon name="cube" size="xl" class="text-gray-400" />
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('canvas.empty.title') }}</p>
        <p class="text-xs text-gray-400 dark:text-gray-500">{{ $t('canvas.empty.hint') }}</p>
      </div>

      <div
        v-if="compositing"
        class="absolute inset-0 z-30 flex items-center justify-center bg-white/60 dark:bg-black/50"
      >
        <LoadingSpinner size="lg" />
      </div>
    </div>
  </div>
</template>
