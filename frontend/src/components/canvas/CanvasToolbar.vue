<template>
  <div class="canvas-toolbar" role="toolbar" aria-label="canvas tools">
    <button
      v-for="t in tools"
      :key="t.name"
      class="tool-btn"
      :class="{ active: tool === t.name }"
      :title="t.title"
      :disabled="!state.ready"
      @click="$emit('update:tool', t.name)"
    >
      <CanvasIcon :name="t.icon" />
    </button>
    <span class="divider" />
    <template v-if="tool === 'brush' || tool === 'eraser'">
      <button
        class="tool-btn"
        :class="{ active: brushShape === 'round' }"
        :title="t('canvas.tools.brush') + ' ●'"
        @click="$emit('update:brushShape', 'round')"
      ><CanvasIcon name="circle" /></button>
      <button
        class="tool-btn"
        :class="{ active: brushShape === 'square' }"
        :title="t('canvas.tools.brush') + ' ■'"
        @click="$emit('update:brushShape', 'square')"
      ><CanvasIcon name="square" /></button>
      <input
        type="range"
        class="brush-slider"
        :min="4"
        :max="160"
        :value="brushSize"
        :title="String(brushSize)"
        @input="$emit('update:brushSize', Number(($event.target as HTMLInputElement).value))"
      />
      <span class="divider" />
    </template>
    <template v-if="tool === 'text'">
      <input
        type="color"
        class="color-input"
        :value="textColor"
        @input="$emit('update:textColor', ($event.target as HTMLInputElement).value)"
      />
      <input
        type="range"
        class="brush-slider"
        :min="10"
        :max="120"
        :value="textSize"
        :title="t('canvas.composer.prompt')"
        @input="$emit('update:textSize', Number(($event.target as HTMLInputElement).value))"
      />
      <span class="divider" />
    </template>
    <button class="tool-btn" :disabled="!state.canUndo" :title="t('canvas.actions.undo')" @click="$emit('undo')"><CanvasIcon name="undo" /></button>
    <button class="tool-btn" :disabled="!state.canRedo" :title="t('canvas.actions.redo')" @click="$emit('redo')"><CanvasIcon name="redo" /></button>
    <span class="divider" />
    <button class="tool-btn" :disabled="state.selectedCount === 0" :title="t('canvas.actions.delete')" @click="$emit('remove-selected')"><Icon name="trash" size="sm" /></button>
    <button class="tool-btn" :disabled="!state.hasMask" :title="t('canvas.actions.clearMask')" @click="$emit('clear-mask')"><CanvasIcon name="eraser" /></button>
    <button class="tool-btn" :disabled="state.selectedCount === 0" :title="t('canvas.actions.mergeToImage')" @click="$emit('flatten')"><CanvasIcon name="merge" /></button>
    <button class="tool-btn" :disabled="state.selectedImageCount === 0" :title="t('canvas.actions.downloadSelected')" @click="$emit('download')"><Icon name="download" size="sm" /></button>
    <span class="divider" />
    <button class="tool-btn" :disabled="state.imageCount === 0" :title="t('canvas.actions.fit')" @click="$emit('fit')"><CanvasIcon name="fit" /></button>
    <button class="tool-btn" :title="t('canvas.actions.zoomOut')" @click="$emit('zoom', 0.8)"><CanvasIcon name="zoomOut" /></button>
    <span class="zoom-label">{{ Math.round(state.zoom * 100) }}%</span>
    <button class="tool-btn" :title="t('canvas.actions.zoomIn')" @click="$emit('zoom', 1.25)"><CanvasIcon name="zoomIn" /></button>
    <span class="divider" />
    <button class="tool-btn" :title="t('canvas.actions.importImage')" @click="$emit('upload')"><Icon name="upload" size="sm" /></button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import CanvasIcon from './CanvasIcon.vue'
import type { CanvasTool, EditorStateSnapshot } from './types'

const props = defineProps<{
  tool: CanvasTool
  brushSize: number
  brushShape: 'round' | 'square'
  textColor: string
  textSize: number
  state: EditorStateSnapshot
}>()

defineEmits<{
  'update:tool': [t: CanvasTool]
  'update:brushSize': [v: number]
  'update:brushShape': [v: 'round' | 'square']
  'update:textColor': [v: string]
  'update:textSize': [v: number]
  undo: []
  redo: []
  'remove-selected': []
  'clear-mask': []
  flatten: []
  download: []
  fit: []
  zoom: [factor: number]
  upload: []
}>()

const { t } = useI18n()
void props

const tools = computed(() => [
  { name: 'select' as const, icon: 'select', title: t('canvas.tools.select') },
  { name: 'pan' as const, icon: 'pan', title: t('canvas.tools.pan') },
  { name: 'brush' as const, icon: 'brush', title: t('canvas.tools.brush') },
  { name: 'eraser' as const, icon: 'eraser', title: t('canvas.tools.eraser') },
  { name: 'text' as const, icon: 'text', title: t('canvas.tools.text') },
])
</script>

<style scoped>
.canvas-toolbar {
  position: absolute;
  top: 12px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 6px 10px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  backdrop-filter: blur(8px);
  z-index: 20;
  max-width: calc(100vw - 24px);
  overflow-x: auto;
  scrollbar-width: none;
}
.canvas-toolbar::-webkit-scrollbar { display: none; }
/* Corner buttons share the top row on phones — drop the pill under them. */
@media (max-width: 767px) {
  .canvas-toolbar { top: 56px; }
}
.dark .canvas-toolbar { background: rgba(17, 24, 39, 0.92); }
.tool-btn {
  display: flex; align-items: center; justify-content: center;
  width: 32px; height: 32px; border-radius: 8px;
  color: #4b5563; transition: background 0.15s, color 0.15s;
}
.dark .tool-btn { color: #9ca3af; }
.tool-btn:hover:not(:disabled) { background: rgba(20, 184, 166, 0.12); color: #14b8a6; }
.tool-btn.active { background: #14b8a6; color: #fff; }
.tool-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.divider { width: 1px; height: 20px; background: rgba(0, 0, 0, 0.1); margin: 0 4px; }
.dark .divider { background: rgba(255, 255, 255, 0.12); }
.brush-slider { width: 90px; accent-color: #14b8a6; }
.color-input { width: 28px; height: 28px; border: none; background: none; padding: 0; cursor: pointer; }
.zoom-label { font-size: 11px; color: #6b7280; min-width: 38px; text-align: center; }
</style>
