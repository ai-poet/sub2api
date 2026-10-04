<template>
  <!-- 右侧「计划」面板，同客户端 plan_review.rs：状态、完整计划、引用和修改意见、两个回答 -->
  <div class="flex min-h-0 flex-1 flex-col" data-test="preview-plan-panel" :data-status="pending ? 'pending' : 'approved'">
    <div class="flex flex-none items-center gap-2 border-b border-[color:var(--cw-border)] px-4 py-2.5">
      <span
        class="flex items-center gap-1.5 text-[12.5px] font-medium"
        :class="pending ? 'text-[color:var(--cw-accent)]' : 'text-[color:var(--cw-success)]'"
      >
        <ClientIcon :name="pending ? 'list' : 'check'" class="h-[13px] w-[13px]" />
        {{ pending ? t('home.clientWorkflow.plan.statusPending') : t('home.clientWorkflow.plan.statusApproved') }}
      </span>
    </div>

    <div class="min-h-0 flex-1 overflow-hidden px-4 py-3.5">
      <h4 class="text-[14px] font-semibold text-[color:var(--cw-text)]">{{ t('home.clientWorkflow.plan.title') }}</h4>
      <ol class="mt-2.5 flex flex-col gap-2 text-[12.5px] leading-[18px] text-[color:var(--cw-text-secondary)]">
        <li v-for="(step, index) in steps" :key="step" class="flex gap-2">
          <span class="w-3 shrink-0 text-right tabular-nums text-[color:var(--cw-text-tertiary)]">{{ index + 1 }}.</span>
          <span>{{ step }}</span>
        </li>
      </ol>
    </div>

    <div v-if="pending" class="cw-plan-footer flex flex-none flex-col gap-2 px-4 py-3">
      <div class="flex items-center gap-2">
        <span class="cw-secondary-button opacity-50">{{ t('home.clientWorkflow.plan.quote') }}</span>
        <span class="min-w-0 flex-1 truncate text-[12px] text-[color:var(--cw-text-tertiary)]">
          {{ t('home.clientWorkflow.plan.quoteHint') }}
        </span>
      </div>
      <div class="cw-notes min-h-[52px] rounded-[7px] px-2 py-1.5 text-[12.5px] text-[color:var(--cw-text-ghost)]">
        {{ t('home.clientWorkflow.plan.notesPlaceholder') }}
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <span class="cw-secondary-button">{{ t('home.clientWorkflow.plan.keepPlanning') }}</span>
        <span class="flex-1"></span>
        <span
          class="cw-primary-button no-key"
          :class="pressed ? 'is-pressed' : ''"
          data-test="preview-plan-approve"
        >{{ t('home.clientWorkflow.plan.approve') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'

defineProps<{
  pending: boolean
  pressed: boolean
}>()

const { t } = useI18n()

const steps = computed(() => [
  t('home.clientWorkflow.plan.step1'),
  t('home.clientWorkflow.plan.step2'),
  t('home.clientWorkflow.plan.step3'),
  t('home.clientWorkflow.plan.step4'),
])
</script>

<style scoped>
.cw-plan-footer {
  border-top: 1px solid var(--cw-border);
  background: var(--cw-raised);
}

.cw-notes {
  border: 1px solid var(--cw-border);
  background: var(--cw-inset);
}
</style>
