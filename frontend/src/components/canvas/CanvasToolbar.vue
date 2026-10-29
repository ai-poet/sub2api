<script setup lang="ts">
/**
 * CanvasToolbar.vue 是画布工作台的工具栏。
 *
 * 只做画布内的操作：切工具、调笔刷、撤销重做、删图层、清空。
 * 提交相关的操作在参数面板里，两块职责分开。
 */
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { CanvasTool } from './types'

const props = defineProps<{
  /** 当前工具。 */
  tool: CanvasTool
  /** 笔刷粗细。 */
  brushSize: number
  /** 是否可以撤销。 */
  canUndo: boolean
  /** 是否可以重做。 */
  canRedo: boolean
  /** 是否有可删除的选中对象。 */
  canDelete: boolean
  /** 是否已禁用（提交中）。 */
  disabled: boolean
}>()

const emit = defineEmits<{
  (event: 'update:tool', value: CanvasTool): void
  (event: 'update:brushSize', value: number): void
  (event: 'undo'): void
  (event: 'redo'): void
  (event: 'delete'): void
  (event: 'clear'): void
}>()

/** 工具按钮定义。图标名取自 Icon.vue 已有的图标集合。 */
const tools = computed(() => [
  { id: 'select' as CanvasTool, icon: 'grid', label: 'canvas.tools.select' },
  { id: 'brush' as CanvasTool, icon: 'sparkles', label: 'canvas.tools.brush' },
  { id: 'eraser' as CanvasTool, icon: 'ban', label: 'canvas.tools.eraser' },
  { id: 'text' as CanvasTool, icon: 'edit', label: 'canvas.tools.text' },
] as const)

/** 笔刷粗细档位。 */
const brushSizes = [8, 16, 32, 64]

/**
 * selectTool 切换工具。
 */
function selectTool(tool: CanvasTool): void {
  emit('update:tool', tool)
}
</script>

<template>
  <div
    class="flex flex-wrap items-center gap-1 border-b border-gray-200 bg-white px-2 py-1.5 dark:border-dark-700 dark:bg-dark-800"
  >
    <!-- 工具切换 -->
    <div class="flex items-center gap-0.5">
      <button
        v-for="item in tools"
        :key="item.id"
        type="button"
        :disabled="disabled"
        :title="$t(item.label)"
        :aria-label="$t(item.label)"
        class="rounded-md p-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-40"
        :class="
          props.tool === item.id
            ? 'bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400'
            : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'
        "
        @click="selectTool(item.id)"
      >
        <Icon :name="item.icon" size="sm" />
      </button>
    </div>

    <div class="mx-1 h-5 w-px bg-gray-200 dark:bg-dark-700" />

    <!-- 笔刷粗细 -->
    <div v-if="props.tool === 'brush' || props.tool === 'eraser'" class="flex items-center gap-1">
      <button
        v-for="size in brushSizes"
        :key="size"
        type="button"
        :disabled="disabled"
        class="rounded-md px-2 py-1 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-40"
        :class="
          props.brushSize === size
            ? 'bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400'
            : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'
        "
        @click="emit('update:brushSize', size)"
      >
        {{ size }}
      </button>
    </div>

    <div class="mx-1 h-5 w-px bg-gray-200 dark:bg-dark-700" />

    <!-- 历史 -->
    <button
      type="button"
      :disabled="disabled || !props.canUndo"
      :title="$t('canvas.actions.undo')"
      :aria-label="$t('canvas.actions.undo')"
      class="rounded-md p-1.5 text-gray-500 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-400 dark:hover:bg-dark-700"
      @click="emit('undo')"
    >
      <Icon name="arrowLeft" size="sm" />
    </button>
    <button
      type="button"
      :disabled="disabled || !props.canRedo"
      :title="$t('canvas.actions.redo')"
      :aria-label="$t('canvas.actions.redo')"
      class="rounded-md p-1.5 text-gray-500 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-400 dark:hover:bg-dark-700"
      @click="emit('redo')"
    >
      <Icon name="arrowRight" size="sm" />
    </button>

    <div class="mx-1 h-5 w-px bg-gray-200 dark:bg-dark-700" />

    <!-- 删除与清空 -->
    <button
      type="button"
      :disabled="disabled || !props.canDelete"
      :title="$t('canvas.actions.delete')"
      :aria-label="$t('canvas.actions.delete')"
      class="rounded-md p-1.5 text-gray-500 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-400 dark:hover:bg-dark-700"
      @click="emit('delete')"
    >
      <Icon name="trash" size="sm" />
    </button>
    <button
      type="button"
      :disabled="disabled"
      :title="$t('canvas.actions.clear')"
      :aria-label="$t('canvas.actions.clear')"
      class="rounded-md p-1.5 text-gray-500 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-400 dark:hover:bg-dark-700"
      @click="emit('clear')"
    >
      <Icon name="xCircle" size="sm" />
    </button>
  </div>
</template>
