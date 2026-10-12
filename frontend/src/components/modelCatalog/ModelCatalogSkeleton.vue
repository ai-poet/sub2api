<template>
  <!-- 首次加载的骨架：与分组分区、模型卡片网格同样的结构和断点，数据到达时不跳版 -->
  <div class="space-y-4" aria-busy="true" data-testid="model-catalog-skeleton">
    <section v-for="group in 2" :key="group" class="card overflow-hidden" aria-hidden="true">
      <div class="space-y-3 border-b border-gray-100 px-4 py-4 dark:border-dark-700 md:px-5">
        <div class="flex gap-2">
          <span class="skeleton-block h-6 w-24 rounded-full"></span>
          <span class="skeleton-block h-6 w-20 rounded-full"></span>
        </div>
        <div class="flex items-center gap-3">
          <span class="skeleton-block h-12 w-12 rounded-xl"></span>
          <div class="space-y-2">
            <span class="skeleton-block block h-5 w-40 rounded-md"></span>
            <span class="skeleton-block block h-3 w-56 rounded-md"></span>
          </div>
        </div>
      </div>
      <div class="grid grid-cols-1 gap-4 p-4 md:grid-cols-2 md:p-5 lg:grid-cols-3">
        <div v-for="card in 3" :key="card" class="rounded-xl border border-gray-100 p-4 dark:border-dark-700">
          <span class="skeleton-block block h-5 w-2/3 rounded-md"></span>
          <span class="skeleton-block mt-2 block h-3 w-2/5 rounded-md"></span>
          <span class="skeleton-block mt-4 block h-9 rounded-md"></span>
          <span class="skeleton-block mt-4 block h-3 w-1/2 rounded-md"></span>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.skeleton-block {
  @apply animate-pulse bg-gray-200/80 motion-reduce:animate-none dark:bg-dark-700;
}
</style>
