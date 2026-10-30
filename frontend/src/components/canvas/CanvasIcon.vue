<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    class="canvas-icon"
    aria-hidden="true"
  >
    <path v-for="(d, i) in paths" :key="i" :d="d" />
  </svg>
</template>

<script setup lang="ts">
/**
 * CanvasIcon provides the small stroke-icon set the canvas toolbar needs and
 * the shared icon registry does not have (brush, eraser, text, pan hand,
 * undo/redo, merge, fit, crop, history).
 */
import { computed } from 'vue'

const ICONS: Record<string, string[]> = {
  select: ['M4 3l7 17 2.2-7.2L20 10.5 4 3z'],
  pan: [
    'M8 12V5.5a1.5 1.5 0 0 1 3 0V11',
    'M11 11V4.5a1.5 1.5 0 0 1 3 0V11',
    'M14 11V6a1.5 1.5 0 0 1 3 0v6.5',
    'M8 10.5 6.2 8.7a1.6 1.6 0 0 0-2.3 2.3L8 15',
    'M8 15v3a4 4 0 0 0 4 4h1a4 4 0 0 0 4-4v-6',
  ],
  brush: ['M15 4l5 5-8.5 8.5a4 4 0 0 1-2.8 1.2c-2.6 0-4.2-1.8-4.2-4 0-1 .4-2 1.2-2.8L15 4z', 'M13 6l5 5'],
  eraser: ['M5 15l8-9 6 6-5.5 5H9l-4-2z', 'M4 21h16'],
  text: ['M5 5h14', 'M12 5v14', 'M9 19h6'],
  undo: ['M9 14 4 9l5-5', 'M4 9h10a6 6 0 0 1 0 12h-2'],
  redo: ['M15 14l5-5-5-5', 'M20 9H10a6 6 0 0 0 0 12h2'],
  merge: ['M4 10l4 4-4 4', 'M20 10l-4 4 4 4', 'M12 3v18'],
  fit: ['M4 9V4h5', 'M15 4h5v5', 'M20 15v5h-5', 'M9 20H4v-5'],
  zoomIn: ['M11 8v6', 'M8 11h6', 'M21 21l-4.8-4.8', 'M17 11a6 6 0 1 1-12 0 6 6 0 0 1 12 0z'],
  zoomOut: ['M8 11h6', 'M21 21l-4.8-4.8', 'M17 11a6 6 0 1 1-12 0 6 6 0 0 1 12 0z'],
  crop: ['M7 2v15a2 2 0 0 0 2 2h13', 'M2 7h15a2 2 0 0 1 2 2v13'],
  history: ['M12 8v4l3 3', 'M21 12a9 9 0 1 1-9-9c2.5 0 4.8 1 6.4 2.6L21 8', 'M21 3v5h-5'],
  square: ['M5 5h14v14H5z'],
  circle: ['M12 5a7 7 0 1 1 0 14 7 7 0 0 1 0-14z'],
  layers: ['M12 3l9 5-9 5-9-5 9-5z', 'M3 13l9 5 9-5'],
}

const props = defineProps<{ name: string }>()
const paths = computed(() => ICONS[props.name] ?? ICONS.select)
</script>

<style scoped>
.canvas-icon {
  width: 1.1rem;
  height: 1.1rem;
  display: block;
}
</style>
