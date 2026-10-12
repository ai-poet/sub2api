<template>
  <!-- 模型能力标识：输入模态 → 输出模态，每个图标悬停与读屏给出模态名称。 -->
  <span class="inline-flex min-w-0 flex-wrap items-center justify-end gap-1" data-testid="model-modality-icons">
    <span
      v-for="modality in capabilities.input"
      :key="`input-${modality}`"
      class="modality-chip"
      :class="TONES[modality]"
      :data-modality="`input-${modality}`"
      :title="t('modelCatalog.modalities.input', { modality: label(modality) })"
      role="img"
      :aria-label="t('modelCatalog.modalities.input', { modality: label(modality) })"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-3 w-3" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" :d="PATHS[modality]" />
      </svg>
    </span>
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-3 w-3 text-gray-400 dark:text-dark-500" aria-hidden="true">
      <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5 21 12m0 0-7.5 7.5M21 12H3" />
    </svg>
    <span
      v-for="modality in capabilities.output"
      :key="`output-${modality}`"
      class="modality-chip"
      :class="TONES[modality]"
      :data-modality="`output-${modality}`"
      :title="t('modelCatalog.modalities.output', { modality: label(modality) })"
      role="img"
      :aria-label="t('modelCatalog.modalities.output', { modality: label(modality) })"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-3 w-3" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" :d="PATHS[modality]" />
      </svg>
    </span>
  </span>
</template>

<script setup lang="ts">
// fork：模型广场的模态图标（参考 TokenRouter ModelCapabilityTags / ModalityIcon）。
// 配色参考 OpenRouter：文字=蓝、图片=绿、音频=琥珀、视频=玫红；图形取 heroicons outline。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { resolveModelCapabilities, type ModelModality } from '@/utils/modelCapabilities'

const props = defineProps<{
  model: string
  billingMode?: string
}>()

const { t } = useI18n()

const capabilities = computed(() => resolveModelCapabilities(props.model, props.billingMode))

const PATHS: Record<ModelModality, string> = {
  text: 'M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12',
  image:
    'm2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 19.5h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Z',
  audio:
    'M19.114 5.636a9 9 0 0 1 0 12.728M16.463 8.288a5.25 5.25 0 0 1 0 7.424M6.75 8.25l4.72-4.72a.75.75 0 0 1 1.28.53v15.88a.75.75 0 0 1-1.28.53l-4.72-4.72H4.51c-.88 0-1.704-.507-1.938-1.354A9.009 9.009 0 0 1 2.25 12c0-.83.112-1.633.322-2.396C2.806 8.756 3.63 8.25 4.51 8.25H6.75Z',
  video:
    'm15.75 10.5 4.72-4.72a.75.75 0 0 1 1.28.53v11.38a.75.75 0 0 1-1.28.53l-4.72-4.72M4.5 18.75h9a2.25 2.25 0 0 0 2.25-2.25v-9a2.25 2.25 0 0 0-2.25-2.25h-9A2.25 2.25 0 0 0 2.25 7.5v9a2.25 2.25 0 0 0 2.25 2.25Z',
}

const TONES: Record<ModelModality, string> = {
  text: 'bg-blue-500/10 text-blue-600 dark:bg-blue-400/10 dark:text-blue-300',
  image: 'bg-green-500/10 text-green-600 dark:bg-green-400/10 dark:text-green-300',
  audio: 'bg-amber-500/10 text-amber-600 dark:bg-amber-400/10 dark:text-amber-300',
  video: 'bg-rose-500/10 text-rose-600 dark:bg-rose-400/10 dark:text-rose-300',
}

function label(modality: ModelModality): string {
  return t(`modelCatalog.modalities.${modality}`)
}
</script>

<style scoped>
.modality-chip {
  @apply inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-md;
}
</style>
