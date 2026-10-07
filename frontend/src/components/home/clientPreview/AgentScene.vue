<template>
  <div class="flex h-full flex-col" data-test="preview-agent-scene">
    <!-- 对话记录：内容最宽 720，和客户端一样居中；还没发出第一条消息时是新任务的空白页和活动概览 -->
    <div class="min-h-0 flex-1 overflow-hidden px-4 pt-3 sm:px-5">
      <WelcomeOverview v-if="frame.chapter === 'plan' && !frame.sent" :site-name="siteName" />
      <div v-else class="mx-auto max-w-[720px]">
        <TeamTranscript v-if="frame.chapter === 'team'" :frame="frame" />
        <PlanRunTranscript v-else :frame="frame" />
      </div>
    </div>

    <!-- 输入框、计划卡片、模型选择器和底栏 -->
    <div class="px-4 pb-1.5 pt-2 sm:px-5">
      <div class="relative mx-auto max-w-[720px]">
        <ModelPicker :open="frame.pickerOpen" :site-name="siteName" />

        <!-- 计划交上来后，输入框上方只留一行：标题、查看计划和两个回答（permission_card.rs） -->
        <div
          v-if="frame.planCard"
          class="cw-plan-card cw-row-in mb-2 flex items-center gap-2 rounded-xl px-3 py-2.5"
          data-test="preview-plan-card"
        >
          <ClientIcon name="list" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-accent)]" />
          <span class="shrink-0 text-[12.5px] font-medium text-[color:var(--cw-text)]">
            {{ t('home.clientWorkflow.plan.readyTitle') }}
          </span>
          <span class="min-w-0 flex-1 truncate text-[12px] text-[color:var(--cw-text-tertiary)] max-md:hidden">
            {{ t('home.clientWorkflow.plan.cardHint') }}
          </span>
          <span class="flex-1 md:hidden"></span>
          <span class="cw-secondary-button max-sm:hidden">
            <ClientIcon name="list" class="h-3 w-3" />
            {{ t('home.clientWorkflow.plan.viewInPanel') }}
          </span>
          <span class="cw-primary-button" :class="frame.planApprovePressed ? 'is-pressed' : ''">
            <span class="cw-key">1</span>
            {{ t('home.clientWorkflow.plan.approve') }}
          </span>
          <span class="cw-secondary-button max-sm:hidden">
            <span class="cw-key">2</span>
            {{ t('home.clientWorkflow.plan.keepPlanning') }}
          </span>
        </div>

        <div class="cw-composer rounded-[13px] py-2.5">
          <div class="min-h-[22px] px-3.5 text-[14px] leading-[22px]">
            <span v-if="frame.draft" class="line-clamp-2 text-[color:var(--cw-text)]" data-test="preview-composer-draft">
              {{ t('home.clientWorkflow.transcript.prompt') }}<i class="cw-caret"></i>
            </span>
            <span v-else class="block truncate text-[color:var(--cw-text-ghost)]">{{ t('home.clientWorkflow.composer.placeholder') }}</span>
          </div>
          <div class="mt-2 flex items-center gap-1 px-2.5">
            <span
              class="cw-chip transition-colors"
              :class="frame.pickerOpen ? 'bg-[color:var(--cw-overlay-strong)]' : ''"
              data-test="preview-model-chip"
            >
              <ClientIcon name="providerWaku" class="h-3 w-3 text-[color:var(--cw-accent)]" />
              <span>Claude Sonnet 5.5</span>
            </span>
            <span class="cw-chip">{{ t('home.clientWorkflow.composer.effort') }}</span>
            <!-- 计划 / 构建模式（composer.rs：计划用 list 图标、强调色） -->
            <span
              class="cw-chip"
              :class="frame.mode === 'plan' ? 'text-[color:var(--cw-accent)]' : ''"
              data-test="preview-mode-chip"
            >
              <ClientIcon
                :name="frame.mode === 'plan' ? 'list' : 'wrench'"
                class="h-3 w-3"
                :class="frame.mode === 'plan' ? '' : 'text-[color:var(--cw-text-tertiary)]'"
              />
              <span>{{ frame.mode === 'plan' ? t('home.clientWorkflow.composer.plan') : t('home.clientWorkflow.composer.build') }}</span>
            </span>
            <span class="cw-chip max-sm:hidden">
              <ClientIcon name="lockOpen" class="h-3 w-3" />
              <span>{{ t('home.clientWorkflow.composer.access') }}</span>
            </span>
            <span
              class="ml-auto flex h-[26px] w-[26px] shrink-0 items-center justify-center rounded-full bg-[color:var(--cw-overlay-strong)]"
              :class="running || frame.draft ? 'text-[color:var(--cw-text)]' : 'text-[color:var(--cw-text-ghost)]'"
            >
              <ClientIcon v-if="running" name="stop" class="h-3.5 w-3.5" />
              <ClientIcon v-else name="arrowUp" class="h-3.5 w-3.5" />
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
              stroke-linecap="round" stroke-dasharray="37.7" :stroke-dashoffset="gaugeOffset"
              class="transition-[stroke-dashoffset] duration-500 motion-reduce:transition-none"
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
import PlanRunTranscript from './PlanRunTranscript.vue'
import TeamTranscript from './TeamTranscript.vue'
import WelcomeOverview from './WelcomeOverview.vue'
import type { PreviewFrame } from './timeline'

const props = defineProps<{
  frame: PreviewFrame
  balance: string
  siteName: string
}>()

const { t } = useI18n()

// 第一幕的这一轮在跑时，发送键变成停止键
const running = computed(() => props.frame.chapter === 'plan' && props.frame.sent && !props.frame.runDone)

// 上下文用量环：这一轮越往后用得越多
const gaugeOffset = computed(() => {
  if (props.frame.chapter === 'team') return 24
  if (!props.frame.sent) return 30
  return props.frame.runDone ? 22 : 27
})
</script>

<style scoped>
.cw-plan-card {
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-raised);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.08);
}
</style>
