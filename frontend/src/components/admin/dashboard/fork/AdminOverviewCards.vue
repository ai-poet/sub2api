<template>
  <!-- 管理员特有的全站概览：账号、用户、API 密钥、累计用量；卡片样式与用量区的指标卡一致 -->
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4" data-testid="admin-overview">
    <div v-for="card in cards" :key="card.key" class="card flex min-w-0 flex-col gap-3 p-4" :data-testid="`overview-${card.key}`">
      <div class="flex min-w-0 items-center gap-2">
        <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md" :class="card.tone">
          <Icon :name="card.icon" size="sm" />
        </span>
        <span class="truncate text-xs font-medium text-gray-500 dark:text-dark-400">{{ card.label }}</span>
      </div>
      <SkeletonBlock v-if="!stats" :width="96" :height="32" />
      <span v-else class="min-w-0 truncate text-2xl font-semibold tracking-tight tabular-nums text-gray-900 dark:text-dark-50">
        <AnimatedNumber :value="card.value" :from="0" :format="card.format" />
      </span>
      <SkeletonBlock v-if="!stats" :width="120" :height="20" />
      <div v-else class="min-w-0 space-y-1 text-xs text-gray-500 dark:text-dark-400">
        <!-- 账号：正常 / 异常 / 限流 -->
        <p v-if="card.key === 'accounts'" class="flex flex-wrap gap-x-2">
          <span class="text-green-600 dark:text-green-400">{{ stats.normal_accounts }} {{ t('common.active') }}</span>
          <span v-if="stats.error_accounts > 0" class="text-red-500">{{ stats.error_accounts }} {{ t('common.error') }}</span>
          <span v-if="stats.ratelimit_accounts > 0" class="text-amber-600 dark:text-amber-400">{{ stats.ratelimit_accounts }} {{ t('admin.dashboard.fork.rateLimited') }}</span>
        </p>
        <!-- 用户：今日新增、活跃 -->
        <p v-else-if="card.key === 'users'" class="flex flex-wrap gap-x-2">
          <span class="text-emerald-600 dark:text-emerald-400">+{{ stats.today_new_users }} {{ t('admin.dashboard.fork.newToday') }}</span>
          <span>{{ stats.active_users }} {{ t('admin.dashboard.activeUsers') }}</span>
        </p>
        <!-- API 密钥：活跃数 -->
        <p v-else-if="card.key === 'keys'" class="text-green-600 dark:text-green-400">
          {{ stats.active_api_keys }} {{ t('common.active') }}
        </p>
        <!-- 累计用量：累计与今日的实付 / 成本 / 标准 -->
        <template v-else>
          <p
            v-for="row in costRows"
            :key="row.key"
            class="flex flex-wrap items-center gap-x-1"
            :data-testid="`overview-costs-${row.key}`"
          >
            <span class="text-gray-400 dark:text-dark-500">{{ row.label }}</span>
            <span class="whitespace-nowrap text-green-600 dark:text-green-400" :title="t('admin.dashboard.fork.actualDescription')">{{ formatCost(row.actual) }}</span>
            <span class="text-gray-400 dark:text-gray-500"> / </span>
            <span class="whitespace-nowrap text-orange-500 dark:text-orange-400" :title="t('admin.dashboard.fork.accountCostDescription')">{{ formatCost(row.account) }}</span>
            <span class="text-gray-400 dark:text-gray-500"> / </span>
            <span class="whitespace-nowrap text-gray-400 dark:text-gray-500" :title="t('admin.dashboard.fork.standardDescription')">{{ formatCost(row.standard) }}</span>
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：管理员仪表盘的全站概览卡（本项目特有，TokenRouter 的用户仪表盘没有这一块）。
// 卡片外观沿用用量区指标卡（图标块 + 小标题 + 大数字 + 说明行）；数据来自标题下实时状态条的那次统计，不另外请求。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import AnimatedNumber from '@/components/motion/AnimatedNumber.vue'
import type { IconName } from '@/components/user/dashboard/fork/iconName'
import SkeletonBlock from '@/components/user/dashboard/fork/SkeletonBlock.vue'
import type { DashboardStats } from '@/types'
import { compactFormatterFor } from '@/utils/compactNumber'
import { formatCost } from './format'

const props = defineProps<{
  stats: DashboardStats | null
}>()

const { t } = useI18n()

interface OverviewCard {
  key: 'accounts' | 'users' | 'keys' | 'usage'
  icon: IconName
  tone: string
  label: string
  value: number
  format?: (value: number) => string
}

const cards = computed<OverviewCard[]>(() => {
  const stats = props.stats
  return [
    {
      key: 'accounts',
      icon: 'server',
      tone: 'bg-purple-500/10 text-purple-500 dark:text-purple-400',
      label: t('admin.dashboard.accounts'),
      value: stats?.total_accounts ?? 0,
    },
    {
      key: 'users',
      icon: 'users',
      tone: 'bg-emerald-500/10 text-emerald-500 dark:text-emerald-400',
      label: t('admin.dashboard.users'),
      value: stats?.total_users ?? 0,
    },
    {
      key: 'keys',
      icon: 'key',
      tone: 'bg-blue-500/10 text-blue-500 dark:text-blue-400',
      label: t('admin.dashboard.apiKeys'),
      value: stats?.total_api_keys ?? 0,
    },
    {
      key: 'usage',
      icon: 'database',
      tone: 'bg-indigo-500/10 text-indigo-500 dark:text-indigo-400',
      label: t('admin.dashboard.totalTokens'),
      value: stats?.total_tokens ?? 0,
      // 单位锁定为终值的单位，1.2B 从 0.00B 滚到 1.20B
      format: compactFormatterFor(stats?.total_tokens ?? 0, 2),
    },
  ]
})

// 累计用量卡的两行费用：累计与今日，各自的实付 / 成本（账号成本）/ 标准
const costRows = computed(() => {
  const stats = props.stats
  if (!stats) return []
  return [
    { key: 'total', label: t('admin.dashboard.fork.total'), actual: stats.total_actual_cost, account: stats.total_account_cost, standard: stats.total_cost },
    { key: 'today', label: t('admin.dashboard.fork.today'), actual: stats.today_actual_cost, account: stats.today_account_cost, standard: stats.today_cost },
  ]
})
</script>
