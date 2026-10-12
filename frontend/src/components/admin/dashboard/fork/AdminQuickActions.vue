<template>
  <!-- Quick Actions -->
  <div class="card p-4" data-testid="admin-quick-actions">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('admin.dashboard.quickActions') }}
      </h2>
    </div>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <button
        v-if="canUseBatchImage"
        type="button"
        class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition-colors hover:bg-sky-50 dark:bg-dark-800/50 dark:hover:bg-sky-900/20"
        data-testid="batch-image-quick-action"
        @click="router.push('/batch-image')"
      >
        <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400">
          <Icon name="sparkles" size="md" :stroke-width="2" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.dashboard.batchImage') }}
          </span>
          <span class="block text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.dashboard.batchImageDesc') }}
          </span>
        </span>
        <Icon name="chevronRight" size="sm" class="text-gray-400 group-hover:text-sky-500" />
      </button>
      <button
        type="button"
        class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition-colors hover:bg-emerald-50 dark:bg-dark-800/50 dark:hover:bg-emerald-900/20"
        data-testid="group-pricing-quick-action"
        @click="router.push('/admin/groups')"
      >
        <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400">
          <Icon name="grid" size="md" :stroke-width="2" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.dashboard.groupPricing') }}
          </span>
          <span class="block text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.dashboard.groupPricingDesc') }}
          </span>
        </span>
        <Icon name="chevronRight" size="sm" class="text-gray-400 group-hover:text-emerald-500" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的快捷操作（本项目原有：批量生图、分组定价）。TokenRouter 的管理员仪表盘没有这一块，
// 样式保持本项目原页面的写法不变，只是挪到页尾。
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'

const { t } = useI18n()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
