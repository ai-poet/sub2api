<template>
  <div
    ref="root"
    class="home-reveal"
    :class="{ 'home-reveal-visible': isVisible }"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const root = ref<HTMLElement | null>(null)
const isVisible = ref(false)

let observer: IntersectionObserver | null = null

onMounted(() => {
  if (typeof window === 'undefined') {
    isVisible.value = true
    return
  }

  if (!('IntersectionObserver' in window)) {
    isVisible.value = true
    return
  }

  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some(entry => entry.isIntersecting)) {
        isVisible.value = true
        observer?.disconnect()
        observer = null
      }
    },
    {
      rootMargin: '0px 0px -10% 0px',
      threshold: 0.16,
    },
  )

  if (root.value) {
    observer.observe(root.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
})
</script>

<style scoped>
.home-reveal {
  opacity: 0;
  transform: translateY(28px);
  transition:
    opacity 0.8s cubic-bezier(0.22, 1, 0.36, 1),
    transform 0.8s cubic-bezier(0.22, 1, 0.36, 1);
}

.home-reveal-visible {
  opacity: 1;
  transform: translateY(0);
}

@media (prefers-reduced-motion: reduce) {
  .home-reveal,
  .home-reveal-visible {
    opacity: 1;
    transform: none;
    transition: none;
  }
}
</style>
