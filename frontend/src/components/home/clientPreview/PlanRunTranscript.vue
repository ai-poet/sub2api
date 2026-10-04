<template>
  <!-- 第一幕的对话：计划行 → 子智能体行 → 执行行，状态行跟在最后；这一轮做完后收成「已工作 N 秒」 -->
  <div class="flex flex-col" data-test="preview-plan-transcript">
    <div
      v-if="frame.sent"
      class="cw-bubble cw-row-in ml-auto w-fit max-w-[88%] rounded-xl px-3 py-2 text-[14px] leading-5 sm:max-w-[540px]"
    >
      {{ t('home.clientWorkflow.transcript.prompt') }}
    </div>

    <div v-if="frame.sent && !frame.runDone" class="mt-4" data-test="preview-tool-rows">
      <div v-for="row in shownPlanRows" :key="row.key" class="cw-row-in flex h-[26px] items-center gap-2 text-[12.5px]">
        <ClientIcon :name="row.icon" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-tertiary)]" />
        <span class="shrink-0 font-medium text-[color:var(--cw-text-secondary)]">{{ row.verb }}</span>
      </div>

      <!-- 子智能体：一行一个，同客户端 subagent_row.rs；运行中标签带流光 -->
      <div
        v-for="row in shownSubagents"
        :key="row.key"
        class="cw-row-in flex h-[26px] min-w-0 items-center gap-2 text-[12.5px]"
        data-test="preview-subagent-row"
      >
        <ClientIcon name="bot" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-tertiary)]" />
        <span
          class="shrink-0 font-medium"
          :class="frame.subagentsRunning ? 'cw-shimmer' : 'text-[color:var(--cw-text-secondary)]'"
        >{{ t('home.clientWorkflow.transcript.subagent') }}</span>
        <span class="shrink-0 font-mono text-[color:var(--cw-kind-explore)]">explore</span>
        <span class="shrink-0 text-[color:var(--cw-text-ghost)]">·</span>
        <span class="min-w-0 truncate text-[color:var(--cw-text-tertiary)]">{{ row.task }}</span>
      </div>

      <div
        v-for="(row, index) in shownExecRows"
        :key="row.key"
        class="cw-row-in flex h-[26px] items-center gap-2 text-[12.5px]"
      >
        <ClientIcon :name="row.icon" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-tertiary)]" />
        <span
          class="shrink-0 font-medium"
          :class="row.live && index === shownExecRows.length - 1 ? 'cw-shimmer' : 'text-[color:var(--cw-text-secondary)]'"
        >{{ row.verb }}</span>
        <span class="min-w-0 truncate text-[color:var(--cw-text-tertiary)]">{{ row.target }}</span>
        <template v-if="row.stats">
          <span class="shrink-0 text-[color:var(--cw-success)]">+{{ row.stats.add }}</span>
          <span class="shrink-0 text-[color:var(--cw-danger)]">-{{ row.stats.del }}</span>
        </template>
      </div>

      <!-- 状态行：工作中 · 时长 · 本轮 tokens · 此刻在做什么（0.2.10） -->
      <div v-if="statusLine" class="mt-2 flex min-w-0 items-center gap-2 text-[12.5px]" data-test="preview-status-line">
        <span class="cw-wave shrink-0" aria-hidden="true"><i></i><i></i><i></i></span>
        <span class="cw-shimmer min-w-0 truncate">{{ statusLine }}</span>
      </div>
    </div>

    <!-- 这一轮完成：工具行收成一条，下面是回复和改动文件卡 -->
    <div v-else-if="frame.runDone" class="cw-settle mt-4" data-test="preview-turn-done">
      <div class="flex items-center gap-3 text-[13.5px] font-medium text-[color:var(--cw-text-tertiary)]">
        <span class="h-px flex-1 bg-[color:var(--cw-border-strong)]"></span>
        <span class="flex items-center gap-1">
          {{ t('home.clientWorkflow.transcript.workedFor', { seconds: frame.workSeconds }) }}
          <ClientIcon name="chevronRight" class="h-3 w-3" />
        </span>
        <span class="h-px flex-1 bg-[color:var(--cw-border-strong)]"></span>
      </div>
      <p class="mt-3 text-[14px] leading-[21px] text-[color:var(--cw-text)]">
        {{ t('home.clientWorkflow.transcript.reply') }}
      </p>
      <div class="cw-changes mt-3 flex flex-wrap items-center gap-2 rounded-xl px-3 py-2 text-[13px]">
        <ClientIcon name="fileDiff" class="h-3.5 w-3.5 text-[color:var(--cw-text-secondary)]" />
        <span class="text-[color:var(--cw-text)]">{{ t('home.clientWorkflow.transcript.changedFiles', { count: 2 }) }}</span>
        <span class="text-[color:var(--cw-success)]">+18</span>
        <span class="text-[color:var(--cw-danger)]">-5</span>
        <span class="ml-auto flex items-center gap-1.5">
          <span class="cw-outline-button">
            <ClientIcon name="fileDiff" class="h-3 w-3" />
            {{ t('home.clientWorkflow.transcript.review') }}
          </span>
          <span class="cw-outline-button">
            <ClientIcon name="rewind" class="h-3 w-3" />
            {{ t('home.clientWorkflow.transcript.undo') }}
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import type { ClientIconName } from './icons'
import { formatTokens, type PreviewFrame, type TurnStatus } from './timeline'

const props = defineProps<{
  frame: PreviewFrame
}>()

const { t } = useI18n()

interface ToolRow {
  key: string
  icon: ClientIconName
  verb: string
  target?: string
  stats?: { add: number; del: number }
  /** 正在执行的命令，带流光 */
  live?: boolean
}

// 行数与 timeline.ts 的 PLAN_ROW_COUNT / SUBAGENT_COUNT / EXEC_ROW_COUNT 对应；图标同客户端 activity_icon
const planRows = computed<ToolRow[]>(() => [
  { key: 'explore', icon: 'search', verb: t('home.clientWorkflow.transcript.explored') },
  { key: 'think', icon: 'sparkle', verb: t('home.clientWorkflow.transcript.thought') },
  { key: 'plan', icon: 'list', verb: t('home.clientWorkflow.transcript.handedOver') },
])

const subagents = computed(() => [
  { key: 'find', task: t('home.clientWorkflow.transcript.subagentFind') },
  { key: 'guard', task: t('home.clientWorkflow.transcript.subagentGuard') },
])

const execRows = computed<ToolRow[]>(() => [
  {
    key: 'edit-login',
    icon: 'pencil',
    verb: t('home.clientWorkflow.transcript.edited'),
    target: 'src/views/auth/LoginView.vue',
    stats: { add: 12, del: 3 },
  },
  {
    key: 'edit-router',
    icon: 'pencil',
    verb: t('home.clientWorkflow.transcript.edited'),
    target: 'src/router/index.ts',
    stats: { add: 6, del: 2 },
  },
  { key: 'test', icon: 'terminal', verb: t('home.clientWorkflow.transcript.running'), target: 'pnpm test', live: true },
])

const shownPlanRows = computed(() => planRows.value.slice(0, props.frame.planRows))
const shownSubagents = computed(() => subagents.value.slice(0, props.frame.subagentRows))
const shownExecRows = computed(() => execRows.value.slice(0, props.frame.execRows))

function statusLabel(status: TurnStatus, tools: number): string {
  if (status === 'runningTools') return t('home.clientWorkflow.status.runningTools', { count: tools })
  return t(`home.clientWorkflow.status.${status}`)
}

// 同客户端 turn_status.rs：「工作中 · 31 秒 · 2.3k tokens · 思考中…」，没有 token 时不写这一段
const statusLine = computed(() => {
  const { status, workSeconds, turnTokens, statusTools } = props.frame
  if (!status) return ''
  const parts = [t('home.clientWorkflow.transcript.working', { seconds: workSeconds })]
  if (turnTokens > 0) parts.push(t('home.clientWorkflow.status.tokens', { count: formatTokens(turnTokens) }))
  parts.push(statusLabel(status, statusTools))
  return parts.join(' · ')
})
</script>

<style scoped>
.cw-changes {
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-overlay);
}

.cw-wave {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--cw-text-tertiary);
}

.cw-wave i {
  height: 4px;
  width: 4px;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0.35;
}

@media (prefers-reduced-motion: no-preference) {
  .cw-wave i {
    animation: wave 1.2s ease-in-out infinite;
  }

  .cw-wave i:nth-child(2) {
    animation-delay: 0.15s;
  }

  .cw-wave i:nth-child(3) {
    animation-delay: 0.3s;
  }
}

@keyframes wave {
  0%,
  60%,
  100% {
    transform: translateY(0);
    opacity: 0.35;
  }
  30% {
    transform: translateY(-3px);
    opacity: 0.9;
  }
}
</style>
