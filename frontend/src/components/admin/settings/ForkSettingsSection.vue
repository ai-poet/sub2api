<template>
  <!-- eslint-disable vue/no-mutating-props ——
       form 是父组件 SettingsView 的共享 reactive 表单对象：本区块按约定直接 v-model
       其字段（与 SettingsView 其余内联设置区块一致），保存动作仍由父组件统一提交。 -->
  <div class="space-y-6">
    <!-- 分组运行状态 -->
    <div
      class="flex items-center justify-between rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-900/40 dark:bg-blue-950/20"
    >
      <div>
        <h3 class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.settings.site.groupStatusEnabled') }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.site.groupStatusEnabledDescription') }}
        </p>
      </div>
      <Toggle v-model="form.group_status_enabled" />
    </div>

    <!-- 购买订阅（sub2apipay 集成） -->
    <div class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-600">
      <div class="flex items-center justify-between">
        <div>
          <label class="font-medium text-gray-900 dark:text-white">
            {{ t('admin.settings.purchase.enabled') }}
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.purchase.enabledHint') }}
          </p>
        </div>
        <Toggle v-model="form.purchase_subscription_enabled" />
      </div>

      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.purchase.url') }}
        </label>
        <input
          v-model="form.purchase_subscription_url"
          type="url"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.purchase.urlPlaceholder')"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.purchase.urlHint') }}
        </p>
      </div>

      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.purchase.openMode') }}
        </label>
        <div class="flex gap-4">
          <label class="flex cursor-pointer items-center gap-2">
            <input
              v-model="form.purchase_subscription_open_mode"
              type="radio"
              value="iframe"
              class="h-4 w-4 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.purchase.openModeIframe') }}
            </span>
          </label>
          <label class="flex cursor-pointer items-center gap-2">
            <input
              v-model="form.purchase_subscription_open_mode"
              type="radio"
              value="new_window"
              class="h-4 w-4 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ t('admin.settings.purchase.openModeNewWindow') }}
            </span>
          </label>
        </div>
      </div>
    </div>

    <!-- 客户端下载链接 -->
    <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.clientDownloads.windowsUrl') }}
        </label>
        <input
          v-model="form.client_download_windows_url"
          type="url"
          class="input font-mono text-sm"
          placeholder="https://..."
        />
      </div>
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.clientDownloads.macosUrl') }}
        </label>
        <input
          v-model="form.client_download_macos_url"
          type="text"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.clientDownloads.macosUrlPlaceholder')"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.clientDownloads.macosUrlHint') }}
        </p>
      </div>
    </div>

    <!-- 交流群 -->
    <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.site.communityGroupURL') }}
        </label>
        <input
          v-model="form.community_group_url"
          type="text"
          class="input font-mono text-sm"
          :placeholder="t('admin.settings.site.communityGroupURLPlaceholder')"
        />
      </div>
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.settings.site.communityQRCode') }}
        </label>
        <ImageUpload
          v-model="form.community_qr_code"
          mode="image"
          :upload-label="t('admin.settings.site.uploadQRCode')"
          :remove-label="t('admin.settings.site.remove')"
          :hint="t('admin.settings.site.qrCodeHint')"
          :max-size="500 * 1024"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ImageUpload from '@/components/common/ImageUpload.vue'
import Toggle from '@/components/common/Toggle.vue'

// fork 自有的站点设置区块。抽成独立组件是为了让上游 SettingsView.vue 保持原样，
// 后续同步上游时这些定制不再产生冲突。
interface ForkSettingsForm {
  group_status_enabled: boolean
  purchase_subscription_enabled: boolean
  purchase_subscription_url: string
  purchase_subscription_open_mode: string
  client_download_windows_url: string
  client_download_macos_url: string
  community_group_url: string
  community_qr_code: string
}

defineProps<{ form: ForkSettingsForm }>()

const { t } = useI18n()
</script>
