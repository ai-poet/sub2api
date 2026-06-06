<template>
  <div v-if="open" ref="menuRef" class="select-menu" role="listbox" :style="positionStyle">
    <button
      v-for="option in options"
      :key="option.id"
      type="button"
      class="select-menu-option"
      :class="{ 'select-menu-option-active': option.id === modelValue }"
      :data-test="`select-menu-option-${option.id}`"
      role="option"
      :aria-selected="option.id === modelValue"
      @click="onPick(option.id)"
    >
      <span class="select-menu-label">
        <span v-if="option.indicatorClass" class="select-menu-dot" :class="option.indicatorClass"></span>
        <span class="select-menu-title">{{ option.label }}</span>
      </span>
      <span v-if="option.description" class="select-menu-description">{{ option.description }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { SelectMenuOption } from './select-menu-types'

export type { SelectMenuOption }

const props = defineProps<{
  open: boolean
  modelValue: string
  options: SelectMenuOption[]
  // 'top-start' 把菜单抬升到锚点上方（composer 在底部时更合适）
  placement?: 'top-start' | 'bottom-start'
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'close'): void
}>()

const menuRef = ref<HTMLElement | null>(null)

const positionStyle = computed(() => {
  if (props.placement === 'bottom-start') return { top: '100%', marginTop: '6px' }
  return { bottom: '100%', marginBottom: '6px' }
})

function onPick(id: string) {
  emit('update:modelValue', id)
  emit('close')
}

function onDocClick(event: MouseEvent) {
  if (!props.open) return
  const root = menuRef.value
  if (!root) return
  // 锚点是父级 mode-badge，由父级自身的 toggle 处理；这里只处理点空白的关闭
  if (root.contains(event.target as Node)) return
  // 父级 badge 与菜单可能在同一个 inline 容器里，通过最近的 .select-menu-anchor 跳过
  const target = event.target as HTMLElement | null
  if (target && target.closest('.select-menu-anchor')) return
  emit('close')
}

function onKey(event: KeyboardEvent) {
  if (!props.open) return
  if (event.key === 'Escape') {
    event.preventDefault()
    emit('close')
  }
}

watch(
  () => props.open,
  (isOpen) => {
    if (typeof document === 'undefined') return
    if (isOpen) {
      document.addEventListener('mousedown', onDocClick, true)
      document.addEventListener('keydown', onKey)
    } else {
      document.removeEventListener('mousedown', onDocClick, true)
      document.removeEventListener('keydown', onKey)
    }
  },
  { immediate: false },
)

onMounted(() => {
  if (props.open && typeof document !== 'undefined') {
    document.addEventListener('mousedown', onDocClick, true)
    document.addEventListener('keydown', onKey)
  }
})

onBeforeUnmount(() => {
  if (typeof document === 'undefined') return
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('keydown', onKey)
})
</script>
