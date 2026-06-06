<template>
  <div
    class="agent-workflow-preview"
    data-test="agent-workflow-preview"
    :aria-label="t('home.clientWorkflow.ariaLabel')"
  >
    <div class="paseo-client-shell">
      <AgentWorkflowSidebar :frame="frame" @select="onSelectWorktree" />

      <main class="workspace-column">
        <header class="screen-header">
          <div class="header-left">
            <button class="header-icon-slot" type="button" aria-hidden="true">
              <Icon name="menu" size="sm" />
            </button>
            <div class="header-title-container">
              <span class="header-title">{{ activeWorktreeName }}</span>
            </div>
          </div>
          <div class="header-right">
            <span class="header-balance">
              <Icon name="creditCard" size="sm" />
              {{ t('home.clientWorkflow.balance') }}
            </span>
            <button class="source-control-button" type="button" aria-hidden="true" :title="diffTooltip">
              <Icon name="clipboard" size="sm" />
              <span class="diff-add">{{ activeDiff.additions }}</span>
              <span class="diff-del">{{ activeDiff.deletions }}</span>
            </button>
          </div>
        </header>

        <div class="workspace-tabs-row">
          <div class="tab-chip active">
            <span class="tab-focus-indicator"></span>
            <span class="tab-handle">
              <span class="tab-icon">
                <span class="provider-dot">◎</span>
              </span>
              <span class="tab-label">{{ activeWorktreeName || t('home.clientWorkflow.tabAgent') }}</span>
            </span>
            <button class="tab-close-button" type="button" aria-hidden="true" tabindex="-1">
              <Icon name="x" size="xs" />
            </button>
          </div>
          <button class="new-tab-action-button" type="button" aria-hidden="true">
            <Icon name="plus" size="sm" />
          </button>
          <button class="new-tab-action-button" type="button" aria-hidden="true">
            <Icon name="terminal" size="sm" />
          </button>
        </div>

        <AgentWorkflowStream :project-name="previewProjectName" :frame="frame" @replay="restart" />

        <footer v-show="frame.liveComposerVisible" class="composer-section">
          <div class="composer-content">
            <AgentWorkflowComposer variant="live" :frame="frame" />
          </div>
        </footer>
      </main>

      <AgentWorkflowChangesPanel :frame="frame" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AgentWorkflowChangesPanel from './agent-workflow/AgentWorkflowChangesPanel.vue'
import AgentWorkflowComposer from './agent-workflow/AgentWorkflowComposer.vue'
import AgentWorkflowSidebar from './agent-workflow/AgentWorkflowSidebar.vue'
import AgentWorkflowStream from './agent-workflow/AgentWorkflowStream.vue'
import { useTimeline } from './agent-workflow/timeline'
import { selectableWorkspaces } from './agent-workflow/workspaces'

const { t } = useI18n()

const projectNameOptions = [
  'northstar-dashboard',
  'atlas-billing',
  'pulse-console',
  'forge-workbench',
  'orbit-admin',
  'harbor-api',
]

const previewProjectName = computed(() => {
  const index = Math.floor(Math.random() * projectNameOptions.length)
  return projectNameOptions[index]
})

// 7 个步骤、5 个工作区（其中最后一个是 creating，selectable 只 4 个）
const { frame, selectWorktree, restart } = useTimeline(7, selectableWorkspaces.length)

const activeWorkspace = computed(() => selectableWorkspaces[frame.value.activeWorktree] ?? selectableWorkspaces[0])
const activeWorktreeName = computed(() => activeWorkspace.value?.name ?? '')
const activeDiff = computed(() => activeWorkspace.value?.diff ?? { additions: '+0', deletions: '-0' })
const diffTooltip = computed(() => `${activeDiff.value.additions} ${activeDiff.value.deletions}`)

function onSelectWorktree(index: number) {
  selectWorktree(index)
}
</script>

<style>
.agent-workflow-preview {
  --surface-0: #181b1a;
  --surface-1: #1e2120;
  --surface-2: #272a29;
  --surface-3: #434645;
  --surface-4: #595b5b;
  --surface-sidebar: #141716;
  --surface-sidebar-hover: #1c1f1e;
  --foreground: #fafafa;
  --foreground-muted: #a1a5a4;
  --border: #252b2a;
  --border-accent: #2f3534;
  --accent: #20744a;
  --accent-bright: #7ccba0;
  --destructive: #ef4444;
  --radius-sm: 2px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-2xl: 16px;
  --content-max: 820px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-2xl);
  background: var(--surface-0);
  color: var(--foreground);
  box-shadow: 0 24px 72px rgba(0, 0, 0, 0.34);
  font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.agent-workflow-preview .paseo-client-shell {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr) minmax(280px, 400px);
  height: clamp(560px, 54vw, 720px);
  min-height: 0;
  background: var(--surface-0);
}

.agent-workflow-preview .paseo-sidebar,
.agent-workflow-preview .workspace-column,
.agent-workflow-preview .source-panel {
  min-width: 0;
  min-height: 0;
}

.agent-workflow-preview .paseo-sidebar {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
  background: var(--surface-sidebar);
}

.agent-workflow-preview .desktop-traffic-area {
  display: flex;
  width: 78px;
  height: 45px;
  align-items: center;
  gap: 8px;
  padding-left: 16px;
}

.agent-workflow-preview .traffic {
  width: 11px;
  height: 11px;
  border-radius: 9999px;
}

.agent-workflow-preview .traffic-red {
  background: #ee806a;
}

.agent-workflow-preview .traffic-yellow {
  background: #f2c85f;
}

.agent-workflow-preview .traffic-green {
  background: #80c76b;
}

.agent-workflow-preview .sidebar-primary-actions {
  display: grid;
  gap: 2px;
  padding: 8px 8px 4px;
}

.agent-workflow-preview .sidebar-primary-action,
.agent-workflow-preview .project-row,
.agent-workflow-preview .workspace-row,
.agent-workflow-preview .host-trigger {
  min-width: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
}

.agent-workflow-preview .workspace-row {
  width: 100%;
  cursor: pointer;
}

.agent-workflow-preview .workspace-row[disabled] {
  cursor: default;
}

.agent-workflow-preview .workspace-row:focus-visible {
  outline: 2px solid var(--accent-bright);
  outline-offset: -2px;
}

.agent-workflow-preview .sidebar-primary-action {
  display: flex;
  min-height: 28px;
  align-items: center;
  gap: 8px;
  border-radius: var(--radius-md);
  background: transparent;
  padding: 6px 8px;
  color: var(--foreground);
  font-size: 14px;
  font-weight: 500;
}

.agent-workflow-preview .sidebar-section {
  min-height: 0;
  flex: 1;
}

.agent-workflow-preview .sidebar-header-row,
.agent-workflow-preview .sidebar-header-title-row,
.agent-workflow-preview .sidebar-header-icons,
.agent-workflow-preview .project-row,
.agent-workflow-preview .project-row-left,
.agent-workflow-preview .project-trailing-actions,
.agent-workflow-preview .workspace-row-main,
.agent-workflow-preview .workspace-row-left,
.agent-workflow-preview .workspace-row-right,
.agent-workflow-preview .sidebar-footer,
.agent-workflow-preview .footer-icon-row,
.agent-workflow-preview .host-trigger,
.agent-workflow-preview .header-left,
.agent-workflow-preview .header-right,
.agent-workflow-preview .header-title-container,
.agent-workflow-preview .workspace-tabs-row,
.agent-workflow-preview .tab-chip,
.agent-workflow-preview .message-input-button-row,
.agent-workflow-preview .status-controls,
.agent-workflow-preview .mode-badge,
.agent-workflow-preview .right-button-group,
.agent-workflow-preview .completion-row,
.agent-workflow-preview .source-panel-tabs,
.agent-workflow-preview .source-panel-toolbar,
.agent-workflow-preview .source-panel-actions,
.agent-workflow-preview .source-row {
  display: flex;
  align-items: center;
}

.agent-workflow-preview .sidebar-header-row {
  justify-content: space-between;
  padding: 12px 12px 4px 16px;
  user-select: none;
}

.agent-workflow-preview .sidebar-header-title-row,
.agent-workflow-preview .project-row-left,
.agent-workflow-preview .workspace-row-left {
  min-width: 0;
  flex: 1;
  gap: 8px;
}

.agent-workflow-preview .sidebar-header-title {
  color: var(--foreground-muted);
  font-size: 14px;
  font-weight: 600;
}

.agent-workflow-preview .sidebar-header-icons,
.agent-workflow-preview .footer-icon-row,
.agent-workflow-preview .project-trailing-actions {
  flex-shrink: 0;
  gap: 4px;
}

.agent-workflow-preview .sidebar-header-icons button,
.agent-workflow-preview .footer-icon-row button,
.agent-workflow-preview .project-new-worktree-button {
  display: grid;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  place-items: center;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--foreground-muted);
}

.agent-workflow-preview .sidebar-header-icons button {
  width: 28px;
  height: 28px;
}

.agent-workflow-preview .sidebar-header-icons button:hover,
.agent-workflow-preview .footer-icon-row button:hover,
.agent-workflow-preview .project-new-worktree-button:hover,
.agent-workflow-preview .sidebar-primary-action:hover {
  background: var(--surface-sidebar-hover);
}

.agent-workflow-preview .sidebar-list {
  padding: 4px 12px 12px;
}

.agent-workflow-preview .project-row {
  min-height: 28px;
  justify-content: space-between;
  gap: 4px;
  border-radius: var(--radius-md);
  padding: 4px;
  color: var(--foreground-muted);
  font-size: 14px;
}

.agent-workflow-preview .project-icon {
  display: grid;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--foreground-muted);
  font-size: 9px;
}

.agent-workflow-preview .project-title,
.agent-workflow-preview .workspace-branch-text,
.agent-workflow-preview .host-trigger-text,
.agent-workflow-preview .header-title,
.agent-workflow-preview .tab-label,
.agent-workflow-preview .mode-badge span,
.agent-workflow-preview .source-row-title,
.agent-workflow-preview .source-row-subtitle,
.agent-workflow-preview .persona-badge,
.agent-workflow-preview .workspace-meta-row {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-workflow-preview .workspace-row {
  display: block;
  min-height: 28px;
  margin-bottom: 2px;
  border-radius: var(--radius-md);
  padding: 4px 4px 4px 24px;
  user-select: none;
  transition: background-color 320ms ease;
}

.agent-workflow-preview .sidebar-row-selected,
.agent-workflow-preview .workspace-row:hover {
  background: var(--surface-sidebar-hover);
}

.agent-workflow-preview .workspace-row-main {
  width: 100%;
  justify-content: space-between;
  gap: 4px;
}

.agent-workflow-preview .workspace-row-right {
  flex-shrink: 0;
  gap: 4px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.agent-workflow-preview .workspace-status-dot {
  position: relative;
  display: grid;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  place-items: center;
  border-radius: 9999px;
}

.agent-workflow-preview .workspace-status-dot::before {
  width: 7px;
  height: 7px;
  border-radius: 9999px;
  background: var(--accent-bright);
  content: '';
}

.agent-workflow-preview .workspace-status-dot.muted::before {
  background: var(--surface-4);
}

.agent-workflow-preview .workspace-status-dot.active::before {
  width: 9px;
  height: 9px;
  background: var(--accent-bright);
  box-shadow: 0 0 0 0 rgba(124, 203, 160, 0.5);
  animation: agentWorkflowDotPulse 1.6s ease-in-out infinite;
}

.agent-workflow-preview .workspace-status-dot.creating::before {
  background: var(--foreground-muted);
  animation: agentWorkflowDotPulse 1.1s ease-in-out infinite;
}

.agent-workflow-preview .status-dot-overlay {
  position: absolute;
  right: 2px;
  bottom: 2px;
  width: 5px;
  height: 5px;
  border: 1px solid var(--surface-sidebar);
  border-radius: 9999px;
  background: var(--destructive);
}

.agent-workflow-preview .workspace-branch-text {
  flex: 1;
  color: var(--foreground);
  font-size: 14px;
  line-height: 18px;
  opacity: 0.76;
}

.agent-workflow-preview .workspace-row:hover .workspace-branch-text,
.agent-workflow-preview .sidebar-row-selected .workspace-branch-text {
  opacity: 1;
}

.agent-workflow-preview .persona-badge {
  max-width: 92px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  padding: 1px 5px;
  color: var(--foreground-muted);
  font-size: 10px;
  line-height: 14px;
  font-weight: 500;
}

.agent-workflow-preview .workspace-meta-row {
  padding-left: 22px;
  color: var(--foreground-muted);
  font-size: 11px;
  line-height: 16px;
}

.agent-workflow-preview .workspace-creating-text {
  color: var(--foreground-muted);
  font-size: 11px;
  font-weight: 600;
}

.agent-workflow-preview .workspace-script-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: #3b82f6;
}

.agent-workflow-preview .workspace-active-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--accent-bright);
  animation: agentWorkflowDotPulse 1.6s ease-in-out infinite;
}

.agent-workflow-preview .diff-add {
  color: #4ade80;
}

.agent-workflow-preview .diff-del {
  color: #ef4444;
}

.agent-workflow-preview .sidebar-footer {
  justify-content: space-between;
  border-top: 1px solid var(--border);
  padding: 12px 16px;
}

.agent-workflow-preview .host-trigger {
  flex: 1;
  justify-content: flex-start;
  gap: 8px;
  border-radius: var(--radius-lg);
  padding: 4px 8px;
}

.agent-workflow-preview .host-status-dot {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 9999px;
  background: #4ade80;
}

.agent-workflow-preview .host-trigger-text {
  color: var(--foreground-muted);
  font-size: 14px;
}

.agent-workflow-preview .workspace-column {
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--surface-0);
}

.agent-workflow-preview .screen-header {
  display: flex;
  height: 48px;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border);
  background: var(--surface-0);
  padding: 0 8px;
  user-select: none;
}

.agent-workflow-preview .header-left {
  min-width: 0;
  flex: 1;
  gap: 8px;
}

.agent-workflow-preview .header-right {
  flex-shrink: 0;
  gap: 8px;
}

.agent-workflow-preview .header-icon-slot {
  display: grid;
  padding: 8px;
  place-items: center;
  border: 0;
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--foreground-muted);
}

.agent-workflow-preview .header-icon-slot:hover,
.agent-workflow-preview .header-action-button:hover,
.agent-workflow-preview .source-control-button:hover {
  background: var(--surface-2);
}

.agent-workflow-preview .header-title-container {
  min-width: 0;
  flex: 1;
  gap: 8px;
  overflow: hidden;
}

.agent-workflow-preview .header-title {
  flex-shrink: 1;
  color: var(--foreground);
  font-size: 15px;
  font-weight: 500;
  letter-spacing: -0.01em;
}

.agent-workflow-preview .header-balance,
.agent-workflow-preview .header-action-button,
.agent-workflow-preview .source-control-button {
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--foreground);
  font-size: 14px;
  white-space: nowrap;
}

.agent-workflow-preview .header-action-button,
.agent-workflow-preview .source-control-button {
  padding: 8px;
}

.agent-workflow-preview .source-control-button {
  padding-inline: 12px;
}

.agent-workflow-preview .workspace-tabs-row {
  height: 36px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border);
  background: var(--surface-0);
  overflow: visible;
}

.agent-workflow-preview .tab-chip {
  position: relative;
  height: 36px;
  max-width: 260px;
  min-width: 160px;
  gap: 4px;
  border-right: 1px solid var(--border);
  padding: 8px 12px;
  color: var(--foreground);
  user-select: none;
}

.agent-workflow-preview .tab-handle {
  display: flex;
  flex: 1;
  min-width: 0;
  align-items: center;
  gap: 4px;
}

.agent-workflow-preview .tab-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
}

.agent-workflow-preview .tab-focus-indicator {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 2px;
  background: var(--accent);
}

.agent-workflow-preview .provider-dot {
  display: inline-grid;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  place-items: center;
  font-size: 15px;
  line-height: 1;
}

.agent-workflow-preview .tab-label {
  flex: 1;
  font-size: 14px;
  color: var(--foreground-muted);
  font-weight: 400;
}

.agent-workflow-preview .tab-chip.active .tab-label,
.agent-workflow-preview .tab-chip:hover .tab-label {
  color: var(--foreground);
}

.agent-workflow-preview .tab-close-button {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 18px;
  height: 18px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--foreground-muted);
  opacity: 0;
  transition: opacity 160ms ease, background-color 160ms ease, color 160ms ease;
}

.agent-workflow-preview .tab-chip:hover .tab-close-button,
.agent-workflow-preview .tab-chip.active .tab-close-button {
  opacity: 1;
}

.agent-workflow-preview .tab-close-button:hover {
  background: var(--surface-3);
  color: var(--foreground);
}

.agent-workflow-preview .new-tab-action-button {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 22px;
  height: 22px;
  margin-left: 8px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--foreground-muted);
}

.agent-workflow-preview .new-tab-action-button:hover {
  background: var(--surface-2);
  color: var(--foreground);
}

.agent-workflow-preview .workspace-pane-content {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.agent-workflow-preview .draft-stage,
.agent-workflow-preview .stream-stage {
  position: absolute;
  inset: 0;
}

.agent-workflow-preview .draft-stage {
  display: grid;
  place-items: center;
  background: var(--surface-0);
  padding: 32px 24px;
  z-index: 2;
}

.agent-workflow-preview .draft-hero-section {
  display: grid;
  width: 100%;
  max-width: 980px;
  gap: 24px;
}

.agent-workflow-preview .draft-hero-section h3 {
  margin: 0;
  color: var(--foreground);
  font-size: clamp(24px, 2.4vw, 34px);
  font-weight: 500;
  line-height: 42px;
  text-align: center;
}

.agent-workflow-preview .hero-composer {
  position: relative;
  z-index: 10;
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface-0);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.agent-workflow-preview .message-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 12px;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-2xl);
  background: var(--surface-1);
  padding: 16px;
  transition: border-color 200ms ease-in-out;
}

.agent-workflow-preview .hero-message-input {
  border-width: 0;
  background: var(--surface-0);
}

.agent-workflow-preview .text-input-scroll-wrapper {
  position: relative;
  min-height: 46px;
  color: var(--foreground);
  font-size: 16px;
  line-height: 22.4px;
}

.agent-workflow-preview .typed-prompt {
  display: inline-block;
  min-height: 22px;
  vertical-align: top;
  overflow-wrap: anywhere;
}

.agent-workflow-preview .typing-caret {
  display: inline-block;
  width: 2px;
  height: 20px;
  margin-left: 1px;
  background: var(--foreground);
  vertical-align: -4px;
}

.agent-workflow-preview .message-input-button-row {
  justify-content: space-between;
  gap: 12px;
  margin-inline: -6px;
}

.agent-workflow-preview .status-controls {
  min-width: 0;
  flex: 1;
  gap: 4px;
  overflow: hidden;
}

.agent-workflow-preview .attach-button,
.agent-workflow-preview .voice-button,
.agent-workflow-preview .send-button,
.agent-workflow-preview .mode-icon-badge {
  display: inline-grid;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  place-items: center;
  border: 0;
  border-radius: 9999px;
  background: transparent;
  color: var(--foreground-muted);
}

.agent-workflow-preview .send-button {
  margin-left: 4px;
  background: var(--accent);
  color: #fff;
}

.agent-workflow-preview .mode-badge {
  min-width: 0;
  max-width: 220px;
  height: 28px;
  flex-shrink: 1;
  gap: 4px;
  border-radius: var(--radius-2xl);
  padding-inline: 8px;
  color: var(--foreground-muted);
  font-size: 14px;
}

.agent-workflow-preview .mode-badge.cloud-group {
  max-width: 260px;
  flex-shrink: 2;
}

.agent-workflow-preview button.mode-badge {
  cursor: pointer;
  border: 0;
  font: inherit;
}

.agent-workflow-preview .mode-badge-active,
.agent-workflow-preview .mode-badge:focus-visible {
  background: var(--surface-2);
  outline: none;
}

.agent-workflow-preview .mode-badge.tone-danger svg {
  color: var(--destructive);
}

.agent-workflow-preview .mode-badge.tone-info svg {
  color: #60a5fa;
}

.agent-workflow-preview .mode-badge.tone-success svg {
  color: #4ade80;
}

.agent-workflow-preview .mode-badge.tone-muted svg {
  color: var(--foreground-muted);
}

.agent-workflow-preview .select-menu-anchor {
  position: relative;
  display: inline-flex;
  flex-shrink: 1;
  min-width: 0;
}

.agent-workflow-preview .select-menu {
  position: absolute;
  left: 0;
  z-index: 30;
  display: grid;
  width: max-content;
  min-width: 180px;
  max-width: 280px;
  gap: 2px;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-lg);
  background: var(--surface-1);
  padding: 4px;
  box-shadow: 0 12px 28px rgba(0, 0, 0, 0.42);
  animation: agentWorkflowMenuIn 140ms ease-out;
}

.agent-workflow-preview .select-menu-option {
  display: grid;
  gap: 2px;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--foreground);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.agent-workflow-preview .select-menu-option:hover,
.agent-workflow-preview .select-menu-option:focus-visible {
  background: var(--surface-2);
  outline: none;
}

.agent-workflow-preview .select-menu-option-active {
  background: var(--surface-2);
}

.agent-workflow-preview .select-menu-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
}

.agent-workflow-preview .select-menu-dot {
  width: 7px;
  height: 7px;
  border-radius: 9999px;
  flex-shrink: 0;
}

.agent-workflow-preview .select-menu-dot.dot-amber { background: #fbbf24; }
.agent-workflow-preview .select-menu-dot.dot-blue { background: #60a5fa; }
.agent-workflow-preview .select-menu-dot.dot-violet { background: #a78bfa; }
.agent-workflow-preview .select-menu-dot.dot-red { background: var(--destructive); }

.agent-workflow-preview .select-menu-description {
  color: var(--foreground-muted);
  font-size: 12px;
  line-height: 16px;
}

@keyframes agentWorkflowMenuIn {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}

.agent-workflow-preview .mode-badge:hover,
.agent-workflow-preview .mode-icon-badge:hover,
.agent-workflow-preview .attach-button:hover,
.agent-workflow-preview .voice-button:hover {
  background: var(--surface-2);
}

.agent-workflow-preview .mode-icon-badge svg {
  color: #fbbf24;
}

.agent-workflow-preview .right-button-group {
  flex-shrink: 0;
  gap: 4px;
}

.agent-workflow-preview .stream-stage {
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px;
}

.agent-workflow-preview .stream-content-wrapper {
  display: grid;
  width: 100%;
  max-width: var(--content-max);
  margin: 0 auto;
  gap: 8px;
  padding-bottom: 172px;
}

.agent-workflow-preview .user-message {
  display: flex;
  justify-content: flex-end;
  padding-inline: 8px;
  animation: agentWorkflowRowIn 280ms ease-out;
}

.agent-workflow-preview .user-bubble {
  max-width: 100%;
  min-width: 0;
  border-radius: var(--radius-2xl);
  border-top-right-radius: var(--radius-sm);
  background: var(--surface-2);
  padding: 16px;
  color: var(--foreground);
  font-size: 16px;
  line-height: 22px;
  overflow-wrap: anywhere;
}

.agent-workflow-preview .assistant-message {
  display: grid;
  gap: 10px;
  padding: 12px 8px;
  color: var(--foreground);
  font-size: 16px;
  line-height: 22px;
  min-height: 22px;
}

.agent-workflow-preview .assistant-message.final {
  animation: agentWorkflowRowIn 280ms ease-out;
}

.agent-workflow-preview .assistant-message p {
  margin: 0;
}

.agent-workflow-preview .stream-line {
  display: inline;
  overflow-wrap: anywhere;
}

.agent-workflow-preview .stream-caret {
  display: inline-block;
  width: 2px;
  height: 18px;
  margin-left: 3px;
  background: var(--foreground);
  vertical-align: -3px;
  animation: agentWorkflowCaretBlink 1s steps(1) infinite;
}

.agent-workflow-preview .agent-status-row {
  display: flex;
  gap: 8px;
  padding: 6px 8px 2px;
  color: var(--foreground-muted);
  font-size: 14px;
  animation: agentWorkflowRowIn 240ms ease-out;
}

.agent-workflow-preview .working-dots {
  display: inline-flex;
  height: 16px;
  flex-shrink: 0;
  align-items: center;
  gap: 4px;
}

.agent-workflow-preview .working-dot {
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: var(--foreground-muted);
  opacity: 0.3;
  animation: agentWorkflowWorkingDot 1.2s ease-in-out infinite;
}

.agent-workflow-preview .working-dot:nth-child(2) {
  animation-delay: 0.16s;
}

.agent-workflow-preview .working-dot:nth-child(3) {
  animation-delay: 0.32s;
}

.agent-workflow-preview .working-text {
  background: linear-gradient(
    100deg,
    var(--foreground-muted) 0%,
    var(--foreground-muted) 35%,
    var(--foreground) 50%,
    var(--foreground-muted) 65%,
    var(--foreground-muted) 100%
  );
  background-size: 220% 100%;
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  animation: agentWorkflowShimmer 2s linear infinite;
}

.agent-workflow-preview .tool-sequence {
  display: grid;
  gap: 4px;
}

.agent-workflow-preview .tool-call-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  border-radius: var(--radius-md);
  padding: 4px 6px;
  color: var(--foreground-muted);
  font-size: 14px;
  animation: agentWorkflowRowIn 260ms ease-out;
}

.agent-workflow-preview .tool-call-row.running {
  background: var(--surface-1);
}

.agent-workflow-preview .tool-status-dot {
  position: relative;
  width: 7px;
  height: 7px;
  flex-shrink: 0;
  border-radius: 9999px;
  background: var(--surface-4);
}

.agent-workflow-preview .tool-status-dot.running {
  background: #fbbf24;
  animation: agentWorkflowToolPing 1.1s ease-out infinite;
}

.agent-workflow-preview .tool-status-dot.done {
  background: #4ade80;
}

.agent-workflow-preview .tool-label {
  flex-shrink: 0;
  font-weight: 600;
  color: var(--foreground);
}

.agent-workflow-preview .tool-call-row strong {
  min-width: 0;
  overflow: hidden;
  color: var(--foreground-muted);
  font: inherit;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-workflow-preview .tool-done-check {
  margin-left: auto;
  flex-shrink: 0;
  color: #4ade80;
}

.agent-workflow-preview .completion-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--accent-bright);
  font-size: 14px;
}

.agent-workflow-preview .replay-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  margin: 4px 8px;
  border: 1px solid var(--border-accent);
  border-radius: 999px;
  background: var(--surface-1);
  padding: 6px 12px;
  color: var(--foreground-muted);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  transition: background-color 160ms ease, color 160ms ease, border-color 160ms ease;
  animation: agentWorkflowRowIn 280ms ease-out;
}

.agent-workflow-preview .replay-pill:hover {
  background: var(--surface-2);
  color: var(--foreground);
  border-color: var(--surface-3);
}

.agent-workflow-preview .replay-pill:focus-visible {
  outline: 2px solid var(--accent-bright);
  outline-offset: 2px;
}

.agent-workflow-preview .live-message-input {
  animation: agentWorkflowRowIn 280ms ease-out;
}

.agent-workflow-preview .composer-section {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 5;
  background: var(--surface-0);
}

.agent-workflow-preview .composer-content {
  width: 100%;
  max-width: var(--content-max);
  margin: 0 auto;
  padding: 16px;
}

.agent-workflow-preview .live-message-input {
  animation: agentWorkflowRowIn 280ms ease-out;
}

.agent-workflow-preview .composer-placeholder {
  display: block;
  overflow: hidden;
  max-width: calc(100% - 112px);
  color: var(--surface-4);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-workflow-preview .focus-hint {
  position: absolute;
  top: 0;
  right: 0;
  color: var(--foreground-muted);
  font-size: 12px;
  opacity: 0.5;
}

.agent-workflow-preview .mic-icon {
  position: relative;
  width: 10px;
  height: 16px;
  border: 1.5px solid currentColor;
  border-radius: 999px;
}

.agent-workflow-preview .mic-icon::after {
  position: absolute;
  right: -4px;
  bottom: -5px;
  left: -4px;
  height: 7px;
  border-right: 1.5px solid currentColor;
  border-bottom: 1.5px solid currentColor;
  border-left: 1.5px solid currentColor;
  border-radius: 0 0 8px 8px;
  content: '';
}

.agent-workflow-preview .source-panel {
  display: flex;
  flex-direction: column;
  border-left: 1px solid var(--border);
  background: var(--surface-0);
}

.agent-workflow-preview .source-panel-header {
  height: 48px;
  border-bottom: 1px solid var(--border);
  padding: 6px 8px;
}

.agent-workflow-preview .source-panel-tabs {
  gap: 4px;
}

.agent-workflow-preview .source-panel-tabs button {
  height: 36px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  padding-inline: 12px;
  color: var(--foreground-muted);
  font-size: 14px;
}

.agent-workflow-preview .source-panel-tabs button.active {
  background: var(--surface-2);
  color: var(--foreground);
}

.agent-workflow-preview .source-panel-toolbar {
  height: 36px;
  justify-content: space-between;
  border-bottom: 1px solid var(--border);
  padding-inline: 12px;
  color: var(--foreground-muted);
  font-size: 14px;
}

.agent-workflow-preview .source-panel-actions {
  gap: 8px;
}

.agent-workflow-preview .source-list {
  min-height: 0;
  overflow: hidden;
  padding: 8px;
}

.agent-workflow-preview .source-row {
  min-width: 0;
  gap: 8px;
  border-radius: var(--radius-md);
  padding: 8px;
  color: var(--foreground-muted);
  opacity: 1;
}

.agent-workflow-preview .source-row:hover {
  background: var(--surface-1);
}

.agent-workflow-preview .source-row-copy {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 2px;
}

.agent-workflow-preview .source-row-title {
  color: var(--foreground);
  font-size: 14px;
}

.agent-workflow-preview .source-row-subtitle {
  color: var(--foreground-muted);
  font-size: 12px;
}

@keyframes agentWorkflowRowIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes agentWorkflowWorkingDot {
  0%, 100% { opacity: 0.3; transform: translateY(0); }
  50% { opacity: 1; transform: translateY(-6px); }
}

@keyframes agentWorkflowShimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

@keyframes agentWorkflowToolPing {
  0% { box-shadow: 0 0 0 0 rgba(251, 191, 36, 0.5); }
  70%, 100% { box-shadow: 0 0 0 5px rgba(251, 191, 36, 0); }
}

@keyframes agentWorkflowDotPulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 1; }
}

@media (max-width: 1120px) {
  .agent-workflow-preview .paseo-client-shell {
    grid-template-columns: 280px minmax(0, 1fr);
  }

  .agent-workflow-preview .source-panel {
    display: none;
  }

  .agent-workflow-preview .header-balance,
  .agent-workflow-preview .header-action-button {
    display: none;
  }
}

@media (max-width: 720px) {
  .agent-workflow-preview {
    border-radius: var(--radius-lg);
  }

  .agent-workflow-preview .paseo-client-shell {
    display: grid;
    grid-template-columns: 104px minmax(0, 1fr);
    height: 560px;
  }

  .agent-workflow-preview .paseo-sidebar {
    display: flex;
  }

  .agent-workflow-preview .desktop-traffic-area,
  .agent-workflow-preview .sidebar-primary-actions,
  .agent-workflow-preview .sidebar-header-icons,
  .agent-workflow-preview .sidebar-footer,
  .agent-workflow-preview .project-trailing-actions,
  .agent-workflow-preview .workspace-meta-row,
  .agent-workflow-preview .persona-badge,
  .agent-workflow-preview .workspace-row-right {
    display: none;
  }

  .agent-workflow-preview .sidebar-header-row {
    padding: 10px 8px 4px;
  }

  .agent-workflow-preview .sidebar-header-title {
    font-size: 12px;
  }

  .agent-workflow-preview .sidebar-list {
    padding: 4px;
  }

  .agent-workflow-preview .project-row {
    padding: 4px 2px;
  }

  .agent-workflow-preview .project-title,
  .agent-workflow-preview .workspace-branch-text {
    font-size: 11px;
  }

  .agent-workflow-preview .workspace-row {
    padding-left: 4px;
    padding-right: 2px;
  }

  .agent-workflow-preview .workspace-status-dot {
    width: 12px;
  }

  .agent-workflow-preview .workspace-status-dot::before {
    width: 6px;
    height: 6px;
  }

  .agent-workflow-preview .source-panel {
    display: none;
  }

  .agent-workflow-preview .workspace-column {
    height: 100%;
  }

  .agent-workflow-preview .screen-header {
    height: 56px;
    padding: 8px;
  }

  .agent-workflow-preview .header-title-container {
    flex-direction: column;
    align-items: flex-start;
    gap: 0;
  }

  .agent-workflow-preview .header-title {
    max-width: 160px;
    font-size: 14px;
  }

  .agent-workflow-preview .header-right {
    display: none;
  }

  .agent-workflow-preview .tab-chip {
    max-width: calc(100vw - 166px);
    min-width: 0;
  }

  .agent-workflow-preview .draft-stage {
    padding: 24px 12px;
  }

  .agent-workflow-preview .draft-hero-section h3 {
    font-size: 24px;
    line-height: 30px;
  }

  .agent-workflow-preview .stream-stage {
    padding: 10px 8px;
  }

  .agent-workflow-preview .stream-content-wrapper {
    gap: 4px;
    padding-bottom: 20px;
  }

  .agent-workflow-preview .user-bubble,
  .agent-workflow-preview .assistant-message {
    font-size: 14px;
    line-height: 20px;
  }

  .agent-workflow-preview .user-bubble {
    max-height: 58px;
    padding: 10px 12px;
  }

  .agent-workflow-preview .assistant-message {
    gap: 6px;
    padding: 6px 8px;
  }

  .agent-workflow-preview .agent-status-row {
    padding-block: 4px 0;
    font-size: 13px;
  }

  .agent-workflow-preview .tool-call-row {
    font-size: 13px;
  }

  .agent-workflow-preview .composer-content {
    padding: 12px;
  }

  .agent-workflow-preview .composer-section {
    position: relative;
    flex-shrink: 0;
  }

  .agent-workflow-preview .message-input-wrapper {
    padding: 12px;
  }

  .agent-workflow-preview .message-input-button-row {
    align-items: flex-end;
    gap: 8px;
  }

  .agent-workflow-preview .status-controls {
    flex-wrap: wrap;
    row-gap: 2px;
  }

  .agent-workflow-preview .text-input-scroll-wrapper {
    min-height: 42px;
    font-size: 14px;
    line-height: 20px;
  }

  .agent-workflow-preview .attach-button {
    width: 24px;
    height: 24px;
  }

  .agent-workflow-preview .mode-badge {
    height: 24px;
    padding-inline: 5px;
    font-size: 11px;
  }

  .agent-workflow-preview .mode-badge.cloud-group {
    max-width: 66px;
  }

  .agent-workflow-preview .mode-badge:nth-of-type(2) {
    max-width: 112px;
  }

  .agent-workflow-preview .mode-badge:nth-of-type(3) {
    max-width: 78px;
  }

  .agent-workflow-preview .mode-badge:nth-of-type(4) {
    max-width: 74px;
  }

  .agent-workflow-preview .mode-icon-badge,
  .agent-workflow-preview .voice-button {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .agent-workflow-preview .working-dot,
  .agent-workflow-preview .working-text,
  .agent-workflow-preview .stream-caret,
  .agent-workflow-preview .tool-status-dot,
  .agent-workflow-preview .workspace-active-dot,
  .agent-workflow-preview .workspace-status-dot.active::before,
  .agent-workflow-preview .workspace-status-dot.creating::before,
  .agent-workflow-preview .user-message,
  .agent-workflow-preview .assistant-message.final,
  .agent-workflow-preview .agent-status-row,
  .agent-workflow-preview .tool-call-row,
  .agent-workflow-preview .live-message-input {
    animation: none;
  }

  .agent-workflow-preview .working-dot {
    opacity: 0.6;
  }

  .agent-workflow-preview .working-text {
    -webkit-text-fill-color: var(--foreground-muted);
  }

  .agent-workflow-preview .tool-status-dot.running {
    background: #fbbf24;
  }

  .agent-workflow-preview .workspace-row {
    transition: none;
  }
}
</style>
