<template>
  <div class="card h-full" data-testid="dashboard-quick-actions">
    <!-- 卡头与公告卡保持同一结构：图标 + 标题 -->
    <div class="flex min-h-[61px] items-center gap-2.5 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <Icon name="bolt" size="md" class="shrink-0 text-primary-600 dark:text-primary-400" />
      <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="space-y-2 p-4">
      <button
        v-for="action in actions"
        :key="action.key"
        type="button"
        :data-testid="`${action.key}-quick-action`"
        class="group flex w-full items-center gap-4 rounded-lg bg-gray-50 p-4 text-left transition duration-normal hover:bg-gray-100 dark:bg-dark-700/40 dark:hover:bg-dark-700"
        @click="router.push(action.to)"
      >
        <div
          class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl transition-transform duration-normal group-hover:scale-105"
          :class="action.tile"
        >
          <Icon :name="action.icon" size="lg" :class="action.iconClass" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">{{ action.title }}</p>
          <p class="truncate text-xs text-gray-500 dark:text-dark-400" :data-testid="`${action.key}-quick-action-desc`">{{ action.description }}</p>
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
// fork：快捷操作（移植自 TokenRouter UserDashboardQuickActions.vue）。
// 创建密钥一行顺带显示密钥数量（原统计卡里的「API 密钥」），购买入口跟随站点的购买开关和计费模式。
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import type { IconName } from './iconName'

const props = defineProps<{
  // 来自实时统计；还没取到时显示默认说明
  apiKeys?: { total: number; active: number } | null
}>()

const router = useRouter()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

// 购买入口与侧栏一致：公开设置开启购买且不是简易模式时才显示，加载失败时保持隐藏。
const purchaseEnabled = computed(() => (
  appStore.cachedPublicSettings?.purchase_subscription_enabled === true && !authStore.isSimpleMode
))

// 购买入口文案随站点计费模式切换，与侧栏一致。
const purchaseTitle = computed(() => {
  switch (resolveSiteBillingMode(appStore.cachedPublicSettings)) {
    case 'recharge_only':
      return t('nav.recharge')
    case 'subscription_only':
      return t('nav.subscribe')
    default:
      return t('nav.buySubscription')
  }
})

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

const actions = computed<QuickAction[]>(() => {
  const keys = props.apiKeys
  const list: QuickAction[] = [
    {
      key: 'api-key',
      to: '/keys',
      icon: 'key',
      title: t('dashboard.createApiKey'),
      description: keys && keys.total > 0
        ? t('dashboard.fork.quickActions.keyCount', { active: keys.active, total: keys.total })
        : t('dashboard.generateNewKey'),
      tile: 'bg-primary-100 dark:bg-primary-900/30',
      iconClass: 'text-primary-600 dark:text-primary-400',
      hoverClass: 'group-hover:text-primary-500',
    },
    {
      key: 'usage',
      to: '/usage',
      icon: 'chart',
      title: t('dashboard.viewUsage'),
      description: t('dashboard.checkDetailedLogs'),
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
      title: t('dashboard.batchImageAgent'),
      description: t('dashboard.batchImageAgentDesc'),
      tile: 'bg-sky-100 dark:bg-sky-900/30',
      iconClass: 'text-sky-600 dark:text-sky-400',
      hoverClass: 'group-hover:text-sky-500',
    })
  }
  if (purchaseEnabled.value) {
    list.push({
      key: 'purchase',
      to: '/purchase',
      icon: 'creditCard',
      title: purchaseTitle.value,
      description: t('dashboard.fork.quickActions.purchaseDesc'),
      tile: 'bg-rose-100 dark:bg-rose-900/30',
      iconClass: 'text-rose-600 dark:text-rose-400',
      hoverClass: 'group-hover:text-rose-500',
    })
  }
  list.push({
    key: 'redeem',
    to: '/redeem',
    icon: 'gift',
    title: t('dashboard.redeemCode'),
    description: t('dashboard.addBalanceWithCode'),
    tile: 'bg-amber-100 dark:bg-amber-900/30',
    iconClass: 'text-amber-600 dark:text-amber-400',
    hoverClass: 'group-hover:text-amber-500',
  })
  return list
})

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
