<script setup lang="ts">
/**
 * CanvasPromptPanel.vue 是画布的提示词与参数面板。
 *
 * 负责：选 API Key、选模型、写提示词、调尺寸/质量/张数，然后提交生成。
 * 提交前需要拿到画布的合成结果，所以由父组件通过 expose 提供。
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { ApiKey } from '@/types'

const { t } = useI18n()

/** 可选的图片尺寸。 */
const SIZES = ['1024x1024', '1024x1536', '1536x1024'] as const

/** 可选的质量档位。空串表示不传（用上游默认）。 */
const QUALITIES = ['', 'low', 'medium', 'high'] as const

const props = defineProps<{
  /** 当前用户的 API Key 列表。 */
  apiKeys: ApiKey[]
  /** 当前选中的 API Key ID。 */
  apiKeyId: number | null
  /** 可选的图片模型名。 */
  models: string[]
  /** 当前选中的模型。 */
  model: string
  /** 提示词。 */
  prompt: string
  /** 当前尺寸。 */
  size: string
  /** 当前质量档位。 */
  quality: string
  /** 当前生成张数。 */
  count: number
  /** 是否正在提交。 */
  submitting: boolean
  /** 提交失败时的错误文案。 */
  errorMessage: string
}>()

const emit = defineEmits<{
  (event: 'update:apiKeyId', value: number | null): void
  (event: 'update:model', value: string): void
  (event: 'update:prompt', value: string): void
  (event: 'update:size', value: string): void
  (event: 'update:quality', value: string): void
  (event: 'update:count', value: number): void
  (event: 'submit'): void
}>()

/** 提示词输入框的引用，用于快捷键聚焦。 */
const promptInput = ref<HTMLTextAreaElement | null>(null)

/** 是否可以提交：必须有 Key、模型和提示词。 */
const canSubmit = computed(() => {
  return (
    !props.submitting &&
    props.apiKeyId !== null &&
    props.model.trim().length > 0 &&
    props.prompt.trim().length > 0
  )
})

/** 尺寸选项。 */
const sizeOptions = computed(() => SIZES.map((value) => ({ value, label: value })))

/** 质量选项。 */
const qualityOptions = computed(() =>
  QUALITIES.map((value) => ({
    value,
    label: t(value ? `canvas.quality.${value}` : 'canvas.quality.default'),
  })),
)

function updateApiKey(value: string | number | boolean | null): void {
  emit('update:apiKeyId', typeof value === 'number' ? value : null)
}

function updateString(
  field: 'model' | 'size' | 'quality',
  value: string | number | boolean | null,
): void {
  const text = typeof value === 'string' ? value : String(value ?? '')
  if (field === 'model') emit('update:model', text)
  if (field === 'size') emit('update:size', text)
  if (field === 'quality') emit('update:quality', text)
}

/**
 * focusPrompt 让外部可以聚焦到提示词输入框。
 */
function focusPrompt(): void {
  promptInput.value?.focus()
}

defineExpose({ focusPrompt })
</script>

<template>
  <div class="flex h-full flex-col gap-4 overflow-y-auto p-4">
    <!-- API Key -->
    <div>
      <label class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-400">
        {{ $t('canvas.panel.apiKey') }}
      </label>
      <Select
        :model-value="props.apiKeyId"
        :options="props.apiKeys.map((key) => ({ value: key.id, label: key.name }))"
        :disabled="props.submitting"
        :placeholder="$t('canvas.panel.apiKeyPlaceholder')"
        @update:model-value="updateApiKey"
      />
    </div>

    <!-- 模型 -->
    <div>
      <label class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-400">
        {{ $t('canvas.panel.model') }}
      </label>
      <Select
        :model-value="props.model"
        :options="props.models.map((name) => ({ value: name, label: name }))"
        :disabled="props.submitting"
        :placeholder="$t('canvas.panel.modelPlaceholder')"
        @update:model-value="(value) => updateString('model', value)"
      />
    </div>

    <!-- 提示词 -->
    <div>
      <label class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-400">
        {{ $t('canvas.panel.prompt') }}
      </label>
      <textarea
        ref="promptInput"
        :value="props.prompt"
        :disabled="props.submitting"
        rows="5"
        :placeholder="$t('canvas.panel.promptPlaceholder')"
        class="w-full resize-y rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition-colors placeholder:text-gray-400 focus:border-primary-500 disabled:opacity-60 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-100 dark:placeholder:text-gray-500"
        @input="(event: Event) => emit('update:prompt', (event.target as HTMLTextAreaElement).value)"
      />
    </div>

    <!-- 尺寸 -->
    <div>
      <label class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-400">
        {{ $t('canvas.panel.size') }}
      </label>
      <Select
        :model-value="props.size"
        :options="sizeOptions"
        :disabled="props.submitting"
        @update:model-value="(value) => updateString('size', value)"
      />
    </div>

    <!-- 质量 -->
    <div>
      <label class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-400">
        {{ $t('canvas.panel.quality') }}
      </label>
      <Select
        :model-value="props.quality"
        :options="qualityOptions"
        :disabled="props.submitting"
        @update:model-value="(value) => updateString('quality', value)"
      />
    </div>

    <!-- 张数 -->
    <div>
      <label class="mb-1.5 block text-xs font-medium text-gray-600 dark:text-gray-400">
        {{ $t('canvas.panel.count') }}
      </label>
      <div class="flex items-center gap-2">
        <button
          type="button"
          :disabled="props.submitting || props.count <= 1"
          class="rounded-md border border-gray-200 px-2.5 py-1 text-sm text-gray-600 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-dark-700 dark:text-gray-400 dark:hover:bg-dark-700"
          @click="emit('update:count', props.count - 1)"
        >
          −
        </button>
        <span class="min-w-8 text-center text-sm text-gray-900 dark:text-gray-100">{{ props.count }}</span>
        <button
          type="button"
          :disabled="props.submitting || props.count >= 4"
          class="rounded-md border border-gray-200 px-2.5 py-1 text-sm text-gray-600 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-dark-700 dark:text-gray-400 dark:hover:bg-dark-700"
          @click="emit('update:count', props.count + 1)"
        >
          +
        </button>
      </div>
    </div>

    <!-- 错误 -->
    <p
      v-if="props.errorMessage"
      class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300"
    >
      {{ props.errorMessage }}
    </p>

    <!-- 提交 -->
    <button
      type="button"
      :disabled="!canSubmit"
      class="btn btn-primary mt-auto w-full justify-center disabled:cursor-not-allowed disabled:opacity-50"
      @click="emit('submit')"
    >
      <LoadingSpinner v-if="props.submitting" size="sm" class="mr-2" />
      {{ props.submitting ? $t('canvas.panel.submitting') : $t('canvas.panel.submit') }}
    </button>
  </div>
</template>
