<template>
  <div ref="rootRef" class="relative shrink-0" @keydown.esc.stop.prevent="closeWithFocus">
    <button
      ref="triggerRef"
      type="button"
      class="btn btn-secondary btn-icon relative"
      :class="activeCount ? 'filter-trigger-active' : ''"
      :aria-label="t('common.filter')"
      :title="t('common.filter')"
      :aria-expanded="open"
      aria-haspopup="dialog"
      data-testid="usage-filter-trigger"
      @click="toggle"
    >
      <Icon name="filter" size="sm" />
      <span v-if="activeCount" class="filter-trigger-count" data-testid="usage-filter-count">{{ activeCount }}</span>
    </button>

    <MotionTransition name="dropdown-fade">
      <div
        v-if="open"
        class="filter-panel"
        :style="panelStyle"
        role="dialog"
        :aria-label="t('common.filter')"
        data-testid="usage-filter-panel"
        @click.stop
      >
        <div class="flex items-center justify-between gap-3 px-4 pb-2 pt-4">
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-gray-900 dark:text-dark-50">{{ t('common.filter') }}</span>
            <span v-if="activeCount" class="filter-panel-count">{{ activeCount }}</span>
          </div>
          <button
            type="button"
            class="filter-panel-reset"
            data-testid="usage-filter-reset"
            :disabled="!activeCount"
            @click="emit('reset')"
          >
            <Icon name="refresh" size="xs" />
            {{ t('common.reset') }}
          </button>
        </div>
        <div class="grid grid-cols-1 gap-3 overflow-y-auto px-4 pb-4 pt-1 sm:grid-cols-2">
          <slot />
        </div>
      </div>
    </MotionTransition>
  </div>
</template>

<script setup lang="ts">
// fork：仪表盘工具栏的筛选面板（参考 TokenRouter components/common/FilterDropdown.vue，只保留仪表盘用到的部分）。
// 本项目没有通用的筛选面板组件，所以放在仪表盘的 fork 目录里。
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import MotionTransition from '@/components/motion/MotionTransition.vue'

defineProps<{ activeCount: number }>()
const emit = defineEmits<{ (event: 'reset'): void }>()
const { t } = useI18n()

// 面板宽度（rem）与距视口边缘的最小距离（px）。
const PANEL_WIDTH_REM = 34
const VIEWPORT_GAP_PX = 16

const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const panelStyle = ref<Record<string, string>>({})

// 面板默认左缘对齐触发器，右侧放不下时向左平移，窄屏时收窄到视口宽度。
function updatePosition() {
  const rect = rootRef.value?.getBoundingClientRect()
  if (!rect) return
  const rem = Number.parseFloat(getComputedStyle(document.documentElement).fontSize) || 16
  const viewportWidth = document.documentElement.clientWidth || window.innerWidth
  const width = Math.min(PANEL_WIDTH_REM * rem, Math.max(0, viewportWidth - VIEWPORT_GAP_PX * 2))
  const left = Math.min(Math.max(rect.left, VIEWPORT_GAP_PX), viewportWidth - width - VIEWPORT_GAP_PX)
  panelStyle.value = { width: `${width}px`, left: `${left - rect.left}px` }
}

function toggle() {
  open.value = !open.value
  if (open.value) updatePosition()
}

// 键盘关闭后回到筛选按钮，便于继续操作工具栏。
function closeWithFocus() {
  open.value = false
  triggerRef.value?.focus()
}

// Select 的下拉挂在 body 上，点选候选项时不能误关外层筛选面板。
function closeOutside(event: MouseEvent) {
  const target = event.target
  if (!(target instanceof Node) || rootRef.value?.contains(target)) return
  if (target instanceof Element && target.closest('.select-dropdown-portal')) return
  open.value = false
}

function handleResize() {
  if (open.value) updatePosition()
}

onMounted(() => {
  document.addEventListener('click', closeOutside)
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', closeOutside)
  window.removeEventListener('resize', handleResize)
})

defineExpose({ close: () => { open.value = false } })
</script>

<style scoped>
/* 有生效条件时按钮描边和图标换成主色，右上角再挂数量角标。 */
.filter-trigger-active {
  @apply border-primary-300 text-primary-700 dark:border-primary-500/40 dark:text-primary-400;
}

.filter-trigger-count {
  @apply pointer-events-none absolute -right-1.5 -top-1.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1;
  @apply bg-primary-600 text-[10px] font-semibold leading-none text-white;
  @apply ring-2 ring-white dark:ring-dark-950;
}

/* 面板外观与日期范围选择器一致：大圆角、淡描边、柔和阴影。 */
.filter-panel {
  @apply absolute top-full z-20 mt-2 flex max-h-[min(70vh,42rem)] flex-col overflow-hidden;
  @apply rounded-xl border border-gray-200 bg-white shadow-lg shadow-black/10;
  @apply dark:border-dark-600 dark:bg-dark-800 dark:shadow-black/30;
}

.filter-panel-count {
  @apply inline-flex h-5 min-w-[1.25rem] items-center justify-center rounded-full px-1.5;
  @apply bg-primary-500/10 text-xs font-semibold text-primary-700 dark:text-primary-300;
}

.filter-panel-reset {
  @apply inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-gray-500;
  @apply transition-colors duration-fast hover:bg-gray-100 hover:text-gray-700;
  @apply disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent;
  @apply dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-dark-200;
}
</style>
