<template>
  <!-- 模型 ID 行：等宽 ID + 复制按钮；复制成功后图标短暂变成对勾。 -->
  <div class="flex min-w-0 items-center gap-1.5">
    <span class="shrink-0 rounded bg-gray-100 px-1 py-px font-mono text-[10px] font-semibold text-gray-500 dark:bg-dark-800 dark:text-gray-400">
      ID
    </span>
    <span class="min-w-0 truncate font-mono text-xs text-gray-500 dark:text-gray-400" :title="modelId">{{ modelId }}</span>
    <button
      type="button"
      class="copy-button"
      :class="{ 'copy-button-done': copied }"
      :aria-label="copied ? t('modelCatalog.copiedModelId') : t('modelCatalog.copyModelId')"
      :title="copied ? t('modelCatalog.copiedModelId') : t('modelCatalog.copyModelId')"
      data-testid="model-id-copy"
      @click="copy"
    >
      <Icon :name="copied ? 'check' : 'copy'" size="xs" :stroke-width="2" />
    </button>
  </div>
</template>

<script setup lang="ts">
// fork：模型 ID 复制（参考 TokenRouter components/common/ModelIdLabel.vue）。
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { COPY_FEEDBACK_MS } from '@/constants/motion'

const props = defineProps<{
  modelId: string
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const copied = ref(false)
let resetTimer: ReturnType<typeof setTimeout> | undefined

async function copy() {
  const ok = await copyToClipboard(props.modelId, t('modelCatalog.copiedModelId'))
  if (!ok) return
  copied.value = true
  if (resetTimer) clearTimeout(resetTimer)
  resetTimer = setTimeout(() => {
    copied.value = false
    resetTimer = undefined
  }, COPY_FEEDBACK_MS)
}

onBeforeUnmount(() => {
  if (resetTimer) clearTimeout(resetTimer)
})
</script>

<style scoped>
.copy-button {
  @apply shrink-0 rounded-md p-0.5 text-gray-400 transition-colors duration-fast ease-standard hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200;
}

.copy-button-done {
  @apply text-emerald-500 hover:text-emerald-500 dark:text-emerald-400;
}
</style>
