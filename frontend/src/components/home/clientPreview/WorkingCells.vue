<template>
  <!-- 固定 3×3 单元格：只改变亮度，不改变任何布局尺寸 -->
  <span class="cw-working-cells" aria-hidden="true">
    <i
      v-for="cell in cells"
      :key="cell.key"
      :style="{ '--cw-cell-phase': cell.phase }"
    ></i>
  </span>
</template>

<script setup lang="ts">
const cells = Array.from({ length: 9 }, (_, index) => {
  const row = Math.floor(index / 3)
  const col = index % 3
  return {
    key: index,
    phase: (2 - row + Math.abs(col - 1)) / 5,
  }
})
</script>
