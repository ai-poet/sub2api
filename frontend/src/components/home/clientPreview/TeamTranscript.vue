<template>
  <!-- 第二幕的对话：/agent-teams 命令、队长拟好团队计划的回复；团队跑完后再回一句 -->
  <div class="flex flex-col" data-test="preview-team-transcript">
    <div class="cw-bubble ml-auto w-fit max-w-[88%] rounded-xl px-3 py-2 text-[14px] leading-5 sm:max-w-[540px]">
      <span class="font-mono text-[color:var(--cw-accent)]">/agent-teams</span>
      {{ t('home.clientWorkflow.team.command') }}
    </div>

    <div class="mt-4 flex h-[26px] items-center gap-2 text-[12.5px]">
      <ClientIcon name="sparkle" class="h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-tertiary)]" />
      <span class="shrink-0 font-medium text-[color:var(--cw-text-secondary)]">{{ t('home.clientWorkflow.transcript.thought') }}</span>
    </div>
    <p class="mt-2 text-[14px] leading-[21px] text-[color:var(--cw-text)]">
      {{ t('home.clientWorkflow.team.draftReply') }}
    </p>

    <p
      v-if="frame.teamReply === 'done'"
      class="cw-settle mt-3 text-[14px] leading-[21px] text-[color:var(--cw-text)]"
      data-test="preview-team-done-reply"
    >
      {{ t('home.clientWorkflow.team.doneReply') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import type { PreviewFrame } from './timeline'

defineProps<{
  frame: PreviewFrame
}>()

const { t } = useI18n()
</script>
