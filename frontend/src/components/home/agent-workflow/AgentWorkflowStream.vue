<template>
  <section class="workspace-pane-content">
    <div v-show="frame.stage === 'draft'" class="draft-stage">
      <div class="draft-hero-section">
        <h3>{{ t('home.clientWorkflow.emptyTitle', { project: projectName }) }}</h3>
        <div class="hero-composer">
          <AgentWorkflowComposer variant="draft" :frame="frame" />
        </div>
      </div>
    </div>

    <div v-show="frame.stage === 'stream'" class="stream-stage">
      <div class="stream-content-wrapper">
        <article v-show="frame.showUser" class="user-message">
          <div class="user-bubble">{{ t('home.clientWorkflow.prompt') }}</div>
        </article>

        <article v-show="frame.introVisible" class="assistant-message">
          <p>{{ slice(introText, frame.introRatio) }}</p>
        </article>

        <div class="tool-sequence">
          <template v-for="(tool, index) in toolCalls" :key="tool.id">
            <div v-show="frame.steps[index] !== 'pending'" class="tool-call-row" :class="frame.steps[index]">
              <span class="tool-status-dot" :class="frame.steps[index]" aria-hidden="true"></span>
              <Icon :name="tool.icon" size="sm" />
              <span class="tool-label">{{ tool.label }}</span>
              <strong>{{ tool.detail }}</strong>
              <Icon v-if="frame.steps[index] === 'done'" name="check" size="xs" class="tool-done-check" />
            </div>
          </template>
        </div>

        <div v-show="frame.agentRunning" class="agent-status-row">
          <span class="working-dots" aria-hidden="true">
            <span class="working-dot"></span>
            <span class="working-dot"></span>
            <span class="working-dot"></span>
          </span>
          <span class="working-text">{{ runningLabel }}</span>
        </div>

        <article v-show="frame.finalVisible" class="assistant-message final">
          <p>{{ slice(finalLine1, frame.finalRatio1) }}</p>
          <p>{{ slice(finalLine2, frame.finalRatio2) }}</p>
          <div v-show="frame.complete" class="completion-row">
            <Icon name="checkCircle" size="sm" />
            <span>{{ t('home.clientWorkflow.streamComplete') }}</span>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AgentWorkflowComposer from './AgentWorkflowComposer.vue'
import type { StreamFrame } from './timeline'

const { t } = useI18n()

const props = defineProps<{
  projectName: string
  frame: StreamFrame
}>()

type IconName = InstanceType<typeof Icon>['$props']['name']

const toolCalls: Array<{ id: string; icon: IconName; label: string; detail: string }> = [
  { id: 'read-tree', icon: 'search', label: 'Read', detail: 'frontend/src/components/home' },
  { id: 'read-store', icon: 'document', label: 'Read', detail: 'stores/billing.ts' },
  { id: 'edit-view', icon: 'edit', label: 'Edit', detail: 'BillingDashboard.vue' },
  { id: 'edit-store', icon: 'edit', label: 'Edit', detail: 'stores/billing.ts' },
  { id: 'shell-typecheck', icon: 'terminal', label: 'Shell', detail: 'pnpm typecheck' },
  { id: 'shell-test', icon: 'terminal', label: 'Shell', detail: 'pnpm test:run billing' },
  { id: 'shell-build', icon: 'terminal', label: 'Shell', detail: 'pnpm build' },
]

const introText = computed(() => t('home.clientWorkflow.streamThinking'))
const finalLine1 = computed(() => t('home.clientWorkflow.streamStepOne'))
const finalLine2 = computed(() => t('home.clientWorkflow.streamStepTwo'))

// working 文案随当前步骤切换，强化"持续在干活"的感觉
const runningLabels = computed(() => [
  t('home.clientWorkflow.workingRead'),
  t('home.clientWorkflow.workingEdit'),
  t('home.clientWorkflow.workingVerify'),
])

const runningLabel = computed(() => {
  const step = props.frame.runningStep
  if (step < 0) return runningLabels.value[0]
  if (step <= 1) return runningLabels.value[0]
  if (step <= 3) return runningLabels.value[1]
  return runningLabels.value[2]
})

function slice(text: string, ratio: number): string {
  if (ratio >= 1) return text
  if (ratio <= 0) return ''
  const count = Math.max(0, Math.round(text.length * ratio))
  return text.slice(0, count)
}

export type { IconName }
</script>
