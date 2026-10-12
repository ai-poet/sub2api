<template>
  <div class="card p-4" data-testid="admin-quick-actions">
    <h2 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dashboard.quickActions') }}</h2>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <button
        v-for="action in actions"
        :key="action.key"
        type="button"
        :data-testid="`${action.key}-quick-action`"
        class="group flex items-center gap-3 rounded-lg bg-gray-50 p-3 text-left transition duration-normal hover:bg-gray-100 dark:bg-dark-700/40 dark:hover:bg-dark-700"
        @click="router.push(action.to)"
      >
        <span
          class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg transition-transform duration-normal group-hover:scale-105"
          :class="action.tile"
        >
          <Icon :name="action.icon" size="md" :stroke-width="2" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ action.title }}</span>
          <span class="block truncate text-xs text-gray-500 dark:text-gray-400">{{ action.description }}</span>
        </span>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition duration-fast group-hover:translate-x-0.5"
          :class="action.hoverClass"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的快捷操作（本项目特有：批量生图、分组定价），行样式与用户仪表盘的快捷操作一致。
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import type { IconName } from '@/components/user/dashboard/fork/iconName'

interface QuickAction {
  key: string
  to: string
  icon: IconName
  title: string
  description: string
  tile: string
  hoverClass: string
}

const { t } = useI18n()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

const actions = computed<QuickAction[]>(() => {
  const list: QuickAction[] = []
  if (canUseBatchImage.value) {
    list.push({
      key: 'batch-image',
      to: '/batch-image',
      icon: 'sparkles',
      title: t('admin.dashboard.batchImage'),
      description: t('admin.dashboard.batchImageDesc'),
      tile: 'bg-sky-100 text-sky-600 dark:bg-sky-900/30 dark:text-sky-400',
      hoverClass: 'group-hover:text-sky-500',
    })
  }
  list.push({
    key: 'group-pricing',
    to: '/admin/groups',
    icon: 'grid',
    title: t('admin.dashboard.groupPricing'),
    description: t('admin.dashboard.groupPricingDesc'),
    tile: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400',
    hoverClass: 'group-hover:text-emerald-500',
  })
  return list
})

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
