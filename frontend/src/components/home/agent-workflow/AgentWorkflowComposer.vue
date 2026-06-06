<template>
  <div class="message-input-wrapper" :class="{ 'hero-message-input': variant === 'draft', 'live-message-input': variant === 'live' }">
    <div class="text-input-scroll-wrapper">
      <template v-if="variant === 'draft'">
        <span class="typed-prompt">{{ typedPrompt }}</span>
        <span v-if="frame && frame.caretOn" class="typing-caret"></span>
      </template>
      <template v-else>
        <span class="composer-placeholder">Message the agent, tag @files, or use /commands and /skills</span>
        <span class="focus-hint">⌘L to focus</span>
      </template>
    </div>

    <div class="message-input-button-row">
      <div class="status-controls">
        <button class="attach-button" type="button" aria-hidden="true">
          <Icon name="plus" size="sm" />
        </button>
        <span class="mode-badge cloud-group">
          <Icon name="cloud" size="sm" />
          <span>{{ t('home.clientWorkflow.group') }}</span>
        </span>
        <span class="mode-badge">
          <span class="provider-dot">◎</span>
          <span>{{ t('home.clientWorkflow.model') }}</span>
          <Icon name="chevronDown" size="xs" />
        </span>
        <span class="mode-badge">
          <Icon name="brain" size="sm" />
          <span>{{ t('home.clientWorkflow.thinking') }}</span>
          <Icon name="chevronDown" size="xs" />
        </span>
        <span class="mode-badge access bypass">
          <Icon name="exclamationTriangle" size="sm" />
          <span>{{ t('home.clientWorkflow.mode') }}</span>
          <Icon name="chevronDown" size="xs" />
        </span>
        <span class="mode-icon-badge">
          <Icon name="bolt" size="sm" />
        </span>
      </div>
      <div class="right-button-group">
        <button v-if="variant === 'live'" class="voice-button" type="button" aria-hidden="true">
          <span class="mic-icon"></span>
        </button>
        <button
          class="send-button"
          :class="{ 'draft-send-button': variant === 'draft' }"
          :style="sendStyle"
          type="button"
          aria-hidden="true"
        >
          <Icon name="arrowUp" size="sm" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { StreamFrame } from './timeline'

const props = defineProps<{
  variant: 'draft' | 'live'
  frame?: StreamFrame
}>()

const { t } = useI18n()

const fullPrompt = computed(() => t('home.clientWorkflow.prompt'))

const typedPrompt = computed(() => {
  if (props.variant !== 'draft' || !props.frame) return fullPrompt.value
  const ratio = props.frame.promptRatio
  if (ratio >= 1) return fullPrompt.value
  const count = Math.max(0, Math.round(fullPrompt.value.length * ratio))
  return fullPrompt.value.slice(0, count)
})

const sendStyle = computed(() => {
  if (props.variant !== 'draft' || !props.frame) return {}
  const pulse = props.frame.sendPulse
  if (pulse <= 0) return {}
  return {
    transform: `scale(${1 + pulse * 0.1})`,
    boxShadow: `0 0 0 ${pulse * 9}px rgba(32, 116, 74, ${0.22 * (1 - pulse)})`,
  }
})
</script>

