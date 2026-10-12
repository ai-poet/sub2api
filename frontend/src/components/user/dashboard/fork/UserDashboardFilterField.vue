<template>
  <div ref="rootRef" class="min-w-0" :class="full ? 'col-span-full' : ''">
    <label v-if="label" class="filter-field-label">{{ label }}</label>
    <slot />
  </div>
</template>

<script setup lang="ts">
// fork：筛选面板里的单个筛选条件（移植自 TokenRouter components/common/FilterField.vue），full 让字段占满整行。
// TokenRouter 的 Select 会自动向字段上报当前选项；本项目的 Select 没有这层接入，
// 所以已选条件标签统一走 TokenRouter 留给其他控件的 valueText / clear 接口。
import { computed, inject, onBeforeUnmount, ref } from 'vue'
import { FILTER_PANEL_KEY, type FilterChip } from './filterPanel'

const props = withDefaults(defineProps<{
  label?: string
  full?: boolean
  // valueText 非空表示条件已生效，内容显示在面板顶部的标签里。
  valueText?: string
}>(), {
  label: '',
  full: false,
  valueText: '',
})
const emit = defineEmits<{ (event: 'clear'): void }>()

const rootRef = ref<HTMLElement | null>(null)

const chip = computed<FilterChip | null>(() => {
  if (!props.valueText) return null
  return { label: props.label, text: props.valueText, clear: () => emit('clear'), el: rootRef.value }
})

const unregister = inject(FILTER_PANEL_KEY, null)?.register(chip)
onBeforeUnmount(() => unregister?.())
</script>

<style scoped>
/* 筛选面板字段标签比表单标签更轻，用中性灰小字，把视觉重心留给控件本身。 */
.filter-field-label {
  @apply mb-1.5 block text-xs font-medium text-gray-500 dark:text-dark-300;
}
</style>
