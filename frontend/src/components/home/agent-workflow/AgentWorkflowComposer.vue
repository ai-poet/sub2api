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

        <span class="select-menu-anchor">
          <button
            type="button"
            class="mode-badge"
            :class="{ 'mode-badge-active': openSelector === 'model' }"
            data-test="composer-model-badge"
            :aria-expanded="openSelector === 'model'"
            @click.stop="toggle('model')"
          >
            <span class="provider-dot">◎</span>
            <span>{{ modelLabel }}</span>
            <Icon name="chevronDown" size="xs" />
          </button>
          <AgentWorkflowSelectMenu
            :open="openSelector === 'model'"
            :model-value="selectedModel"
            :options="modelOptions"
            @update:model-value="onSelectModel"
            @close="closeSelector"
          />
        </span>

        <span class="select-menu-anchor">
          <button
            type="button"
            class="mode-badge"
            :class="{ 'mode-badge-active': openSelector === 'thinking' }"
            data-test="composer-thinking-badge"
            :aria-expanded="openSelector === 'thinking'"
            @click.stop="toggle('thinking')"
          >
            <Icon name="brain" size="sm" />
            <span>{{ thinkingLabel }}</span>
            <Icon name="chevronDown" size="xs" />
          </button>
          <AgentWorkflowSelectMenu
            :open="openSelector === 'thinking'"
            :model-value="selectedThinking"
            :options="thinkingOptions"
            @update:model-value="onSelectThinking"
            @close="closeSelector"
          />
        </span>

        <span class="select-menu-anchor">
          <button
            type="button"
            class="mode-badge access"
            :class="[modeBadgeTone, { 'mode-badge-active': openSelector === 'mode' }]"
            data-test="composer-mode-badge"
            :aria-expanded="openSelector === 'mode'"
            @click.stop="toggle('mode')"
          >
            <Icon :name="modeIconName" size="sm" />
            <span>{{ modeLabel }}</span>
            <Icon name="chevronDown" size="xs" />
          </button>
          <AgentWorkflowSelectMenu
            :open="openSelector === 'mode'"
            :model-value="selectedMode"
            :options="modeOptions"
            @update:model-value="onSelectMode"
            @close="closeSelector"
          />
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
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { StreamFrame } from './timeline'
import AgentWorkflowSelectMenu from './AgentWorkflowSelectMenu.vue'
import type { SelectMenuOption } from './select-menu-types'
import {
  MODE_OPTIONS,
  THINKING_OPTIONS,
  MODEL_OPTIONS,
  DEFAULT_MODE_ID,
  DEFAULT_THINKING_ID,
  DEFAULT_MODEL_ID,
} from './composer-options'

const props = defineProps<{
  variant: 'draft' | 'live'
  frame?: StreamFrame
}>()

const { t } = useI18n()

type IconName = InstanceType<typeof Icon>['$props']['name']
type SelectorKey = 'model' | 'thinking' | 'mode'

const openSelector = ref<SelectorKey | null>(null)
const selectedModel = ref<string>(DEFAULT_MODEL_ID)
const selectedThinking = ref<string>(DEFAULT_THINKING_ID)
const selectedMode = ref<string>(DEFAULT_MODE_ID)

const modelOptions = MODEL_OPTIONS as SelectMenuOption[]
const thinkingOptions = THINKING_OPTIONS as SelectMenuOption[]
const modeOptions = MODE_OPTIONS as SelectMenuOption[]

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

function findOption(opts: SelectMenuOption[], id: string, fallback: SelectMenuOption): SelectMenuOption {
  return opts.find((opt) => opt.id === id) ?? fallback
}

const modelLabel = computed(() => findOption(modelOptions, selectedModel.value, modelOptions[0]).label)
const thinkingLabel = computed(() => findOption(thinkingOptions, selectedThinking.value, thinkingOptions[0]).label)
const modeLabel = computed(() => findOption(modeOptions, selectedMode.value, modeOptions[0]).label)

const modeIconName = computed<IconName>(() => {
  switch (selectedMode.value) {
    case 'plan':
      return 'eye'
    case 'acceptEdits':
      return 'check'
    case 'always-ask':
      return 'shield'
    default:
      return 'exclamationTriangle'
  }
})

const modeBadgeTone = computed(() => {
  switch (selectedMode.value) {
    case 'bypassPermissions':
      return 'tone-danger'
    case 'plan':
      return 'tone-info'
    case 'acceptEdits':
      return 'tone-success'
    default:
      return 'tone-muted'
  }
})

function toggle(key: SelectorKey) {
  openSelector.value = openSelector.value === key ? null : key
}

function closeSelector() {
  openSelector.value = null
}

function onSelectModel(id: string) {
  selectedModel.value = id
}
function onSelectThinking(id: string) {
  selectedThinking.value = id
}
function onSelectMode(id: string) {
  selectedMode.value = id
}
</script>
