<template>
  <!-- eslint-disable vue/no-mutating-props ——
       form 是 SettingsView 的共享 reactive 表单对象，保存仍由父组件统一提交。 -->
  <!-- fork：画布工作台。 -->
  <div class="card">
    <div class="flex items-center justify-between gap-4 p-6">
      <div class="min-w-0">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.settings.site.canvasStudioEnabled') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.canvasStudioEnabledDescription') }}
        </p>
      </div>
      <Toggle
        v-model="form.canvas_studio_enabled"
        data-testid="canvas-studio-toggle"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'

interface CanvasStudioSettingsForm {
  canvas_studio_enabled: boolean
}

defineProps<{ form: CanvasStudioSettingsForm }>()

const { t } = useI18n()
</script>
