<template>
  <div class="flex h-full flex-col" data-test="preview-agent-scene">
    <!-- 对话记录：内容最宽 720，和客户端一样居中 -->
    <div class="min-h-0 flex-1 overflow-hidden px-4 pt-3 sm:px-5">
      <div class="mx-auto flex max-w-[720px] flex-col">
        <div class="user-bubble ml-auto w-fit max-w-[88%] rounded-xl px-3 py-2 text-[14px] leading-5 sm:max-w-[540px]">
          {{ t('home.clientWorkflow.transcript.prompt') }}
        </div>

        <!-- 进行中：工具行逐条出现 -->
        <div v-if="!frame.runDone" class="mt-4" data-test="preview-tool-rows">
          <div
            v-for="(row, index) in rows"
            :key="row.key"
            class="flex h-[26px] items-center gap-2 text-[12.5px] transition-all duration-300 ease-out"
            :class="index < frame.visibleRows ? 'translate-y-0 opacity-100' : 'translate-y-1.5 opacity-0'"
          >
            <ClientIcon :name="row.icon" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-tertiary)]" />
            <span
              class="shrink-0 font-medium"
              :class="row.live && index === frame.visibleRows - 1 ? 'cw-shimmer' : 'text-[color:var(--cw-text-secondary)]'"
            >{{ row.verb }}</span>
            <span v-if="row.target" class="min-w-0 truncate text-[color:var(--cw-text-tertiary)]">{{ row.target }}</span>
            <template v-if="row.stats">
              <span class="shrink-0 text-[color:var(--cw-success)]">+{{ row.stats.add }}</span>
              <span class="shrink-0 text-[color:var(--cw-danger)]">-{{ row.stats.del }}</span>
            </template>
          </div>

          <div class="mt-2 flex items-center gap-2 text-[12.5px]">
            <span class="wave" aria-hidden="true"><i></i><i></i><i></i></span>
            <span class="cw-shimmer">{{ t('home.clientWorkflow.transcript.working', { seconds: frame.workSeconds }) }}</span>
          </div>
        </div>

        <!-- 这一轮完成：工具行收成一条，下面是回复和改动文件卡 -->
        <div v-else class="settle mt-4" data-test="preview-turn-done">
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
          <div class="changes mt-3 flex flex-wrap items-center gap-2 rounded-xl px-3 py-2 text-[13px]">
            <ClientIcon name="fileDiff" class="h-3.5 w-3.5 text-[color:var(--cw-text-secondary)]" />
            <span class="text-[color:var(--cw-text)]">{{ t('home.clientWorkflow.transcript.changedFiles', { count: 2 }) }}</span>
            <span class="text-[color:var(--cw-success)]">+18</span>
            <span class="text-[color:var(--cw-danger)]">-5</span>
            <span class="ml-auto flex items-center gap-1.5">
              <span class="outline-button">
                <ClientIcon name="fileDiff" class="h-3 w-3" />
                {{ t('home.clientWorkflow.transcript.review') }}
              </span>
              <span class="outline-button">
                <ClientIcon name="rewind" class="h-3 w-3" />
                {{ t('home.clientWorkflow.transcript.undo') }}
              </span>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 输入框、模型选择器和底栏 -->
    <div class="px-4 pb-1.5 pt-2 sm:px-5">
      <div class="relative mx-auto max-w-[720px]">
        <ModelPicker :open="frame.pickerOpen" :active-index="frame.pickerIndex" :site-name="siteName" />

        <div class="composer rounded-[13px] py-2.5">
          <div class="min-h-[22px] px-3.5 text-[14px] text-[color:var(--cw-text-ghost)]">
            {{ t('home.clientWorkflow.composer.placeholder') }}
          </div>
          <div class="mt-2 flex items-center gap-1 px-2.5">
            <span
              class="cw-chip transition-colors"
              :class="frame.pickerOpen ? 'bg-[color:var(--cw-overlay-strong)]' : ''"
              data-test="preview-model-chip"
            >
              <ClientIcon name="providerWaku" class="h-3 w-3 text-[color:var(--cw-accent)]" />
              <span>gpt-5.6-sol</span>
            </span>
            <span class="cw-chip">{{ t('home.clientWorkflow.composer.effort') }}</span>
            <span class="cw-chip">
              <ClientIcon name="lockOpen" class="h-3 w-3" />
              <span>{{ t('home.clientWorkflow.composer.access') }}</span>
            </span>
            <span class="cw-chip max-sm:hidden">
              <ClientIcon name="wrench" class="h-3 w-3" />
              <span>{{ t('home.clientWorkflow.composer.build') }}</span>
            </span>
            <span
              class="ml-auto flex h-[26px] w-[26px] shrink-0 items-center justify-center rounded-full bg-[color:var(--cw-overlay-strong)]"
              :class="frame.runDone ? 'text-[color:var(--cw-text-ghost)]' : 'text-[color:var(--cw-text)]'"
            >
              <ClientIcon v-if="frame.runDone" name="arrowUp" class="h-3.5 w-3.5" />
              <ClientIcon v-else name="stop" class="h-3.5 w-3.5" />
            </span>
          </div>
        </div>

        <!-- 工作区底栏 -->
        <div class="mt-1 flex h-7 items-center gap-0.5">
          <span class="cw-chip">
            <ClientIcon name="folder" class="h-3 w-3" />
            <span>{{ t('home.clientWorkflow.footer.project') }}</span>
          </span>
          <span class="cw-chip">
            <ClientIcon name="laptop" class="h-3 w-3" />
            <span>{{ t('home.clientWorkflow.footer.local') }}</span>
          </span>
          <span class="cw-chip max-sm:hidden">
            <ClientIcon name="gitBranch" class="h-3 w-3" />
            <span>main</span>
          </span>
          <span class="ml-auto"></span>
          <span class="cw-chip tabular-nums" data-test="preview-balance">
            <ClientIcon name="wallet" class="h-3 w-3" />
            <span>{{ balance }}</span>
          </span>
          <svg class="mx-1.5 h-3.5 w-3.5 -rotate-90 text-[color:var(--cw-gauge)]" viewBox="0 0 16 16" aria-hidden="true">
            <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-opacity="0.18" stroke-width="2.5" />
            <circle
              cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-width="2.5"
              stroke-linecap="round" stroke-dasharray="37.7" :stroke-dashoffset="frame.runDone ? 22 : 27"
            />
          </svg>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import ModelPicker from './ModelPicker.vue'
import type { ClientIconName } from './icons'
import type { PreviewFrame } from './timeline'

defineProps<{
  frame: PreviewFrame
  balance: string
  siteName: string
}>()

const { t } = useI18n()

interface ToolRow {
  key: string
  icon: ClientIconName
  verb: string
  target?: string
  stats?: { add: number; del: number }
  /** 最后一行是正在执行的命令，带流光 */
  live?: boolean
}

// 行数与 timeline.ts 的 TOOL_ROW_COUNT 对应；图标同客户端 activity_icon
const rows = computed<ToolRow[]>(() => [
  { key: 'explore', icon: 'search', verb: t('home.clientWorkflow.transcript.explored') },
  { key: 'think', icon: 'sparkle', verb: t('home.clientWorkflow.transcript.thought') },
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
</script>

<style scoped>
.user-bubble {
  background: var(--cw-raised);
  color: var(--cw-text);
}

.composer {
  border: 1px solid var(--cw-border);
  background: var(--cw-composer);
}

.changes {
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-overlay);
}

.outline-button {
  display: inline-flex;
  height: 26px;
  align-items: center;
  gap: 5px;
  border-radius: 7px;
  border: 1px solid var(--cw-border-strong);
  padding: 0 9px;
  font-size: 12px;
  color: var(--cw-text-secondary);
}

.wave {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--cw-text-tertiary);
}

.wave i {
  height: 4px;
  width: 4px;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0.35;
}

.settle {
  animation: settle-in 0.35s ease-out;
}

@media (prefers-reduced-motion: no-preference) {
  .wave i {
    animation: wave 1.2s ease-in-out infinite;
  }

  .wave i:nth-child(2) {
    animation-delay: 0.15s;
  }

  .wave i:nth-child(3) {
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

@keyframes settle-in {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
