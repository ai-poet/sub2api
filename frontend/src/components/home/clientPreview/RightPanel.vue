<template>
  <!-- 右侧面板：主栏够宽时是一列（客户端的右侧面板），窄时盖在对话上 -->
  <aside
    class="cw-panel"
    :class="{ 'is-column-open': frame.panel !== 'none', 'is-sheet-open': frame.sheet !== 'none' }"
    data-test="preview-right-panel"
    :data-panel="frame.panel"
  >
    <div class="flex h-[34px] flex-none items-center gap-1 border-b border-[color:var(--cw-border)] px-2">
      <span class="flex h-[26px] items-center gap-1.5 rounded-md bg-[color:var(--cw-overlay)] px-2 text-[12.5px] text-[color:var(--cw-text)]">
        <ClientIcon :name="frame.panel === 'team' ? 'users' : 'list'" class="h-3.5 w-3.5 text-[color:var(--cw-text-secondary)]" />
        {{ frame.panel === 'team' ? t('home.clientWorkflow.team.surface') : t('home.clientWorkflow.plan.surface') }}
      </span>
      <span class="flex-1"></span>
      <span class="cw-icon-button"><ClientIcon name="x" class="h-3 w-3" /></span>
    </div>
    <div class="flex min-h-0 flex-1 flex-col">
      <PlanPanel v-if="frame.panel === 'plan'" :pending="frame.plan === 'pending'" :pressed="frame.planApprovePressed" />
      <TeamPanel v-else-if="frame.panel === 'team'" :team="frame.team" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import PlanPanel from './PlanPanel.vue'
import TeamPanel from './TeamPanel.vue'
import type { PreviewFrame } from './timeline'

defineProps<{
  frame: PreviewFrame
}>()

const { t } = useI18n()
</script>

<style scoped>
/* 默认（主栏窄）：浮在对话上的面板，打开时从下方升起 */
.cw-panel {
  position: absolute;
  inset: 8px;
  z-index: 30;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-surface);
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.18);
  opacity: 0;
  visibility: hidden;
  transform: translateY(12px);
  transition:
    opacity 0.3s ease,
    transform 0.3s ease,
    visibility 0s linear 0.3s;
}

.cw-panel.is-sheet-open {
  opacity: 1;
  visibility: visible;
  transform: none;
  transition:
    opacity 0.3s ease,
    transform 0.3s ease,
    visibility 0s;
}

/* 主栏够宽：和客户端一样是右侧的一列，打开时从右边滑进来 */
@container cw-main (min-width: 700px) {
  .cw-panel {
    position: relative;
    inset: auto;
    z-index: auto;
    display: none;
    width: 340px;
    flex: none;
    border: 0;
    border-left: 1px solid var(--cw-sidebar-border);
    border-radius: 0;
    box-shadow: none;
    opacity: 1;
    visibility: visible;
    transform: none;
    transition: none;
  }

  .cw-panel.is-column-open {
    display: flex;
  }
}

@media (prefers-reduced-motion: no-preference) {
  @container cw-main (min-width: 700px) {
    .cw-panel.is-column-open {
      animation: cw-panel-in 0.3s ease-out;
    }
  }
}

@media (prefers-reduced-motion: reduce) {
  .cw-panel,
  .cw-panel.is-sheet-open {
    transition: none;
  }
}

@keyframes cw-panel-in {
  from {
    opacity: 0;
    transform: translateX(16px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
</style>
