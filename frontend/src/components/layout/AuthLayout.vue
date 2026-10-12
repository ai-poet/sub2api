<template>
  <!-- fork：认证页外壳换成 TokenRouter 的样式（components/auth/fork/AuthShell.vue：中性底色、点阵背景动画、平面控件） -->
  <AuthShell>
    <slot />
    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </AuthShell>
</template>

<script setup lang="ts">
import AuthShell from '@/components/auth/fork/AuthShell.vue' // fork：TokenRouter 风格认证页
</script>
