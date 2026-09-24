<template>
  <svg v-if="platform === 'windows'" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path d="M3 5.1 10.4 4v7.2H3V5.1Zm8.3-1.2L21 2.5v8.7h-9.7V3.9ZM3 12.8h7.4V20L3 18.9v-6.1Zm8.3 0H21v8.7l-9.7-1.4v-7.3Z" />
  </svg>
  <svg v-else viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path
      d="M16.4 12.6c0-2.6 2.1-3.8 2.2-3.9-1.2-1.8-3.1-2-3.8-2-1.6-.2-3.1.9-3.9.9-.8 0-2-.9-3.4-.9-1.7 0-3.3 1-4.2 2.6-1.8 3.1-.5 7.7 1.3 10.2.9 1.2 1.9 2.6 3.2 2.6 1.3-.1 1.8-.8 3.3-.8 1.6 0 2 .8 3.4.8 1.4 0 2.3-1.3 3.1-2.5 1-1.4 1.4-2.8 1.4-2.9 0 0-2.6-1-2.6-4.1ZM13.9 5c.7-.9 1.2-2 1.1-3.2-1 0-2.3.7-3 1.6-.7.8-1.2 2-1.1 3.1 1.1.1 2.3-.6 3-1.5Z"
    />
  </svg>
</template>

<script setup lang="ts">
import type { ClientDownloadPlatform } from '@/utils/clientDownloads'

defineProps<{
  platform: ClientDownloadPlatform
}>()
</script>
