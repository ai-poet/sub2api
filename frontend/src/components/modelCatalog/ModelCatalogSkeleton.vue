<template>
  <!-- 首次加载的骨架：和卡片网格同样的断点与结构，数据到达时不跳版。 -->
  <section class="grid gap-4 xl:grid-cols-2 2xl:grid-cols-3" aria-busy="true" data-testid="model-catalog-skeleton">
    <div
      v-for="index in count"
      :key="index"
      class="rounded-3xl border border-gray-200/80 bg-white/90 p-5 dark:border-dark-700 dark:bg-dark-900/80"
    >
      <div class="flex items-center gap-2">
        <span class="skeleton-block h-6 w-20 rounded-full"></span>
        <span class="skeleton-block h-6 w-16 rounded-full"></span>
        <span class="ml-auto skeleton-block h-5 w-16 rounded-md"></span>
      </div>
      <div class="mt-5 flex items-center gap-3">
        <span class="skeleton-block h-10 w-10 rounded-xl"></span>
        <div class="flex-1 space-y-2">
          <span class="skeleton-block block h-5 w-1/2 rounded-md"></span>
          <span class="skeleton-block block h-3 w-1/3 rounded-md"></span>
        </div>
      </div>
      <div class="mt-5 space-y-3">
        <span v-for="row in 2" :key="row" class="skeleton-block block h-16 rounded-2xl"></span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 6 })
</script>

<style scoped>
.skeleton-block {
  @apply animate-pulse bg-gray-200/80 motion-reduce:animate-none dark:bg-dark-700;
}
</style>
