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

        <div
          v-for="(workspace, index) in worktrees"
          :key="workspace.name"
          class="workspace-row"
          :class="{
            'sidebar-row-selected': index === activeIndex,
            'workspace-row-creating': workspace.creating,
          }"
        >
          <div class="workspace-row-main">
            <div class="workspace-row-left">
              <span
                class="workspace-status-dot"
                :class="[workspace.statusClass, { creating: workspace.creating, active: index === activeIndex }]"
              >
                <span v-if="workspace.attention" class="status-dot-overlay"></span>
              </span>
              <span class="workspace-branch-text">{{ workspace.name }}</span>
              <span v-if="workspace.persona" class="persona-badge">{{ workspace.persona }}</span>
            </div>
            <div class="workspace-row-right">
              <span v-if="index === activeIndex && !workspace.creating" class="workspace-active-dot" aria-hidden="true"></span>
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
        </div>
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

const { t } = useI18n()

const props = defineProps<{
  frame: StreamFrame
}>()

const worktrees = [
  {
    name: 'homepage-billing',
    statusClass: 'active',
    attention: true,
    script: true,
    diff: { additions: '+42', deletions: '-8' },
  },
  {
    name: 'pricing-copy',
    statusClass: 'muted',
    persona: 'Reviewer',
    summary: '3 skills',
    diff: { additions: '+12', deletions: '-2' },
  },
  {
    name: 'api-metering',
    statusClass: 'ready',
    persona: 'Builder',
    summary: '2 skills',
    diff: { additions: '+28', deletions: '-4' },
  },
  {
    name: 'docs-worktree',
    statusClass: 'muted',
    persona: 'Writer',
    summary: '1 skill',
  },
  {
    name: 'new-agent-flow',
    statusClass: 'creating',
    creating: true,
  },
]

// 只在非 creating 的工作区之间轮转高亮
const selectableCount = worktrees.filter((w) => !w.creating).length
const activeIndex = computed(() => props.frame.activeWorktree % selectableCount)
</script>
