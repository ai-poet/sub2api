<template>
  <div class="card" data-testid="admin-quick-actions">
    <!-- 卡头与用户仪表盘的快捷操作一致：图标 + 标题 -->
    <div class="flex min-h-[61px] items-center gap-2.5 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <Icon name="bolt" size="md" class="shrink-0 text-primary-600 dark:text-primary-400" />
      <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-dark-50">{{ t('admin.dashboard.quickActions') }}</h2>
    </div>
    <div class="space-y-2 p-4">
      <button
        v-for="action in actions"
        :key="action.key"
        type="button"
        :data-testid="`${action.key}-quick-action`"
        class="group flex w-full items-center gap-4 rounded-lg bg-gray-50 p-4 text-left transition duration-normal hover:bg-gray-100 dark:bg-dark-700/50 dark:hover:bg-dark-700"
        @click="router.push(action.to)"
      >
        <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl transition-transform group-hover:scale-105" :class="action.tile">
          <Icon :name="action.icon" size="lg" :class="action.iconClass" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-dark-50">{{ action.title }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ action.description }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition duration-fast group-hover:translate-x-0.5 dark:text-dark-500"
          :class="action.hoverClass"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的快捷操作，行样式与用户仪表盘的快捷操作（TokenRouter UserDashboardQuickActions）一致。
// 入口：用户管理、账号管理、分组定价，以及有权限时的批量生图。
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
  iconClass: string
  hoverClass: string
}

const { t } = useI18n()
const router = useRouter()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

// 类名写成完整字符串，保证 Tailwind 能扫描到
const actions = computed<QuickAction[]>(() => {
  const list: QuickAction[] = [
    {
      key: 'users',
      to: '/admin/users',
      icon: 'users',
      title: t('admin.dashboard.manageUsers'),
      description: t('admin.dashboard.viewUserAccounts'),
      tile: 'bg-primary-100 dark:bg-primary-900/30',
      iconClass: 'text-primary-600 dark:text-primary-400',
      hoverClass: 'group-hover:text-primary-500',
    },
    {
      key: 'accounts',
      to: '/admin/accounts',
      icon: 'server',
      title: t('admin.dashboard.manageAccounts'),
      description: t('admin.dashboard.configureAiAccounts'),
      tile: 'bg-purple-100 dark:bg-purple-900/30',
      iconClass: 'text-purple-600 dark:text-purple-400',
      hoverClass: 'group-hover:text-purple-500',
    },
    {
      key: 'group-pricing',
      to: '/admin/groups',
      icon: 'grid',
      title: t('admin.dashboard.groupPricing'),
      description: t('admin.dashboard.groupPricingDesc'),
      tile: 'bg-emerald-100 dark:bg-emerald-900/30',
      iconClass: 'text-emerald-600 dark:text-emerald-400',
      hoverClass: 'group-hover:text-emerald-500',
    },
  ]
  if (canUseBatchImage.value) {
    list.push({
      key: 'batch-image',
      to: '/batch-image',
      icon: 'sparkles',
      title: t('admin.dashboard.batchImage'),
      description: t('admin.dashboard.batchImageDesc'),
      tile: 'bg-sky-100 dark:bg-sky-900/30',
      iconClass: 'text-sky-600 dark:text-sky-400',
      hoverClass: 'group-hover:text-sky-500',
    })
  }
  return list
})

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
