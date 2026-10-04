<template>
  <!-- 演示窗口下方的章节导航：点一章跳过去，进度条显示这一章走到哪；暂停键让触屏用户也能停下来 -->
  <nav class="mx-auto mt-4 flex max-w-[680px] items-center gap-3 px-1" :aria-label="label" data-test="preview-chapter-nav">
    <ol class="flex min-w-0 flex-1 items-stretch gap-2 sm:gap-3">
      <li v-for="(chapter, index) in chapters" :key="chapter.id" class="min-w-0 flex-1">
        <button
          type="button"
          class="group flex w-full flex-col gap-1.5 rounded-md py-1 text-left outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60"
          :aria-current="index === current ? 'true' : undefined"
          :data-test="`preview-chapter-${chapter.id}`"
          @click="emit('select', index)"
        >
          <span
            class="truncate text-[12px] font-semibold transition-colors"
            :class="index === current
              ? 'text-gray-900 dark:text-white'
              : 'text-gray-500 group-hover:text-gray-800 dark:text-white/45 dark:group-hover:text-white/80'"
          >{{ chapter.label }}</span>
          <span v-if="showProgress" class="relative h-[2px] w-full overflow-hidden rounded-full bg-black/10 dark:bg-white/15" aria-hidden="true">
            <span
              class="absolute inset-0 origin-left bg-primary-500 transition-transform duration-100 ease-linear motion-reduce:transition-none"
              :style="{ transform: `scaleX(${fill(index)})` }"
              data-test="preview-chapter-fill"
            ></span>
          </span>
          <span
            v-else
            class="h-[2px] w-full rounded-full"
            :class="index === current ? 'bg-primary-500' : 'bg-black/10 dark:bg-white/15'"
            aria-hidden="true"
          ></span>
        </button>
      </li>
    </ol>

    <button
      v-if="showProgress"
      type="button"
      class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-black/10 text-gray-600 outline-none transition-colors hover:bg-black/5 hover:text-gray-900 focus-visible:ring-2 focus-visible:ring-primary-500/60 dark:border-white/15 dark:text-white/60 dark:hover:bg-white/10 dark:hover:text-white"
      :aria-label="paused ? playLabel : pauseLabel"
      :title="paused ? playLabel : pauseLabel"
      data-test="preview-chapter-toggle"
      @click="emit('toggle')"
    >
      <svg v-if="paused" class="h-3 w-3" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M5 3.2v9.6a.6.6 0 0 0 .9.52l7.6-4.8a.6.6 0 0 0 0-1.04L5.9 2.68A.6.6 0 0 0 5 3.2Z" />
      </svg>
      <svg v-else class="h-3 w-3" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <rect x="3.5" y="2.5" width="3" height="11" rx="1" />
        <rect x="9.5" y="2.5" width="3" height="11" rx="1" />
      </svg>
    </button>
  </nav>
</template>

<script setup lang="ts">
const props = defineProps<{
  chapters: ReadonlyArray<{ id: string; label: string }>
  current: number
  /** 当前章走过的比例，0~1 */
  progress: number
  paused: boolean
  /** 减少动态效果时不放进度条和暂停键，章节按钮只用来切换静态画面 */
  showProgress: boolean
  label: string
  pauseLabel: string
  playLabel: string
}>()

const emit = defineEmits<{
  select: [index: number]
  toggle: []
}>()

function fill(index: number): number {
  if (index < props.current) return 1
  if (index > props.current) return 0
  return props.progress
}
</script>
