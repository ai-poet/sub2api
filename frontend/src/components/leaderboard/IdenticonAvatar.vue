<template>
  <img :src="src" alt="" aria-hidden="true" class="select-none rounded-full ring-2 ring-white dark:ring-dark-800" :class="sizeClass" />
</template>

<script setup lang="ts">
// fork：排行榜头像。种子是后端给的化名（avatar_seed），同一用户各榜单图案一致，但反推不出是谁。
import { computed } from 'vue'
import { identiconDataUri } from '@/utils/identicon'

const props = withDefaults(defineProps<{ seed: string; sizeClass?: string }>(), { sizeClass: 'h-10 w-10' })

const src = computed(() => identiconDataUri(props.seed))
</script>
