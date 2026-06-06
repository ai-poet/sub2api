<template>
  <aside class="source-panel">
    <div class="source-panel-header">
      <div class="source-panel-tabs">
        <button class="active" type="button" aria-hidden="true">Changes</button>
        <button type="button" aria-hidden="true">Files</button>
      </div>
    </div>
    <div class="source-panel-toolbar">
      <span>Uncommitted</span>
      <div class="source-panel-actions" aria-hidden="true">
        <Icon name="menu" size="sm" />
        <Icon name="filter" size="sm" />
        <Icon name="more" size="sm" />
      </div>
    </div>
    <div class="source-list">
      <div v-for="(file, index) in changedFiles" :key="file.path" class="source-row" :style="{ '--row-index': index }">
        <Icon name="document" size="sm" />
        <div class="source-row-copy">
          <span class="source-row-title">{{ file.name }}</span>
          <span class="source-row-subtitle">{{ file.path }}</span>
        </div>
        <span class="diff-add">{{ file.plus }}</span>
        <span class="diff-del">{{ file.minus }}</span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type { StreamFrame } from './timeline'

defineProps<{
  frame: StreamFrame
}>()

const changedFiles = [
  { name: 'HomeAgentWorkflowPreview.vue', path: 'frontend/src/components/home', plus: '+188', minus: '-0' },
  { name: 'AgentWorkflowSidebar.vue', path: 'frontend/src/components/home/agent-workflow', plus: '+122', minus: '-0' },
  { name: 'AgentWorkflowStream.vue', path: 'frontend/src/components/home/agent-workflow', plus: '+96', minus: '-0' },
  { name: 'HomeHero.vue', path: 'frontend/src/components/home', plus: '+2', minus: '-4' },
  { name: 'zh.ts', path: 'frontend/src/i18n/locales', plus: '+9', minus: '-2' },
]
</script>
