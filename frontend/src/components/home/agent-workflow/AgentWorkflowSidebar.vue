<template>
  <aside class="paseo-sidebar">
    <div class="desktop-traffic-area" aria-hidden="true">
      <span class="traffic traffic-red"></span>
      <span class="traffic traffic-yellow"></span>
      <span class="traffic traffic-green"></span>
    </div>

    <div class="sidebar-primary-actions">
      <button class="sidebar-primary-action" type="button" aria-hidden="true">
        <Icon name="plus" size="sm" />
        <span>{{ t('home.clientWorkflow.workspaceName') }}</span>
      </button>
    </div>

    <div class="sidebar-section">
      <div class="sidebar-header-row">
        <div class="sidebar-header-title-row">
          <Icon name="grid" size="sm" />
          <span class="sidebar-header-title">{{ t('home.clientWorkflow.workspaces') }}</span>
        </div>
        <div class="sidebar-header-icons" aria-hidden="true">
          <button type="button"><Icon name="refresh" size="sm" /></button>
          <button type="button"><Icon name="plus" size="sm" /></button>
        </div>
      </div>

      <div class="sidebar-list">
        <div class="project-row">
          <div class="project-row-left">
            <span class="project-icon">S</span>
            <span class="project-title">sub2api</span>
          </div>
          <div class="project-trailing-actions">
            <button class="project-new-worktree-button" type="button" aria-hidden="true" :title="t('home.clientWorkflow.newWorktree')">
              <Icon name="plus" size="sm" />
            </button>
            <Icon name="chevronDown" size="xs" />
          </div>
        </div>

        <button
          v-for="(workspace, index) in workspaces"
          :key="workspace.name"
          class="workspace-row"
          type="button"
          :data-test="`workspace-row-${workspace.name}`"
          :class="{
            'sidebar-row-selected': !workspace.creating && selectableIndexFor(index) === activeIndex,
            'workspace-row-creating': workspace.creating,
          }"
          :disabled="workspace.creating"
          :aria-pressed="!workspace.creating && selectableIndexFor(index) === activeIndex"
          @click="onSelect(index, workspace.creating)"
        >
          <div class="workspace-row-main">
            <div class="workspace-row-left">
              <span
                class="workspace-status-dot"
                :class="[
                  workspace.statusClass,
                  { creating: workspace.creating, active: !workspace.creating && selectableIndexFor(index) === activeIndex },
                ]"
              >
                <span v-if="workspace.attention" class="status-dot-overlay"></span>
              </span>
              <span class="workspace-branch-text">{{ workspace.name }}</span>
              <span v-if="workspace.persona" class="persona-badge">{{ workspace.persona }}</span>
            </div>
            <div class="workspace-row-right">
              <span
                v-if="!workspace.creating && selectableIndexFor(index) === activeIndex"
                class="workspace-active-dot"
                aria-hidden="true"
              ></span>
              <span v-else-if="workspace.script" class="workspace-script-dot" aria-hidden="true"></span>
              <span v-if="workspace.creating" class="workspace-creating-text">{{ t('home.clientWorkflow.creatingWorktree') }}</span>
              <template v-else-if="workspace.diff">
                <span class="diff-add">{{ workspace.diff.additions }}</span>
                <span class="diff-del">{{ workspace.diff.deletions }}</span>
              </template>
            </div>
          </div>
          <div v-if="workspace.summary" class="workspace-meta-row">
            <span>{{ workspace.summary }}</span>
          </div>
        </button>
      </div>
    </div>

    <div class="sidebar-footer">
      <div class="host-trigger">
        <span class="host-status-dot"></span>
        <span class="host-trigger-text">local daemon</span>
      </div>
      <div class="footer-icon-row" aria-hidden="true">
        <button type="button"><Icon name="cloud" size="sm" /></button>
        <button type="button"><Icon name="cog" size="sm" /></button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { StreamFrame } from './timeline'
import { workspaces, selectableWorkspaces } from './workspaces'

const { t } = useI18n()

const props = defineProps<{
  frame: StreamFrame
}>()

const emit = defineEmits<{
  (e: 'select', selectableIndex: number): void
}>()

// 把全量列表的下标映射到 selectable 列表的下标（creating 项返回 -1）
const selectableIndexFor = (fullIndex: number): number => {
  const item = workspaces[fullIndex]
  if (!item || item.creating) return -1
  return selectableWorkspaces.indexOf(item)
}

const selectableCount = selectableWorkspaces.length
const activeIndex = computed(() => {
  if (selectableCount === 0) return 0
  const i = props.frame.activeWorktree
  return ((i % selectableCount) + selectableCount) % selectableCount
})

function onSelect(fullIndex: number, creating: boolean | undefined) {
  if (creating) return
  const selIndex = selectableIndexFor(fullIndex)
  if (selIndex >= 0) emit('select', selIndex)
}
</script>
