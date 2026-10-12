<template>
  <div v-if="!isSimple && platformCards.length > 0" class="card p-4" data-testid="platform-breakdown">
    <div class="mb-4 flex min-h-7 items-center justify-between gap-2">
      <div class="flex min-w-0 items-baseline gap-2">
        <h3 class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-dark-50">
          <Icon name="server" size="sm" class="text-primary-600 dark:text-primary-400" />
          {{ t('dashboard.platformBreakdown') }}
        </h3>
        <span class="truncate text-xs text-gray-500 dark:text-dark-400">{{ t('dashboard.fork.platforms.subtitle') }}</span>
      </div>
      <span class="shrink-0 text-xs text-gray-500 dark:text-dark-400">
        {{ t('dashboard.platformCount', { count: platformCount }) }}
      </span>
    </div>
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <div
        v-for="item in platformCards"
        :key="item.platform"
        data-testid="platform-card"
        :data-platform="item.platform"
        class="rounded-xl p-3"
        :class="item.isOther
          ? 'border border-dashed border-gray-200 dark:border-dark-600'
          : 'bg-gray-50 dark:bg-dark-700/40'"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="flex min-w-0 items-center gap-2 text-sm font-medium text-gray-900 dark:text-dark-50">
            <span
              v-if="!item.isOther"
              class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-white text-gray-700 shadow-sm dark:bg-dark-800 dark:text-dark-100"
            >
              <PlatformIcon :platform="item.platform as GroupPlatform" size="sm" />
            </span>
            <span class="truncate">{{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}</span>
          </span>
          <span class="shrink-0 text-sm font-semibold tabular-nums text-gray-900 dark:text-dark-50" :title="t('dashboard.actual')">
            {{ formatUsd(item.total_actual_cost, 4) }}
          </span>
        </div>
        <dl class="mt-2 space-y-1 text-xs">
          <div class="flex items-center justify-between gap-2">
            <dt class="text-gray-500 dark:text-dark-400">{{ t('dashboard.todayCost') }}</dt>
            <dd class="tabular-nums text-gray-900 dark:text-dark-100">{{ formatUsd(item.today_actual_cost, 4) }}</dd>
          </div>
          <div class="flex items-center justify-between gap-2">
            <dt class="text-gray-500 dark:text-dark-400">{{ t('dashboard.requests') }}</dt>
            <dd class="tabular-nums text-gray-700 dark:text-dark-200">
              {{ item.total_requests > 0 ? formatNumber(item.total_requests) : '-' }}
            </dd>
          </div>
          <div class="flex items-center justify-between gap-2">
            <dt class="text-gray-500 dark:text-dark-400">{{ t('dashboard.tokens') }}</dt>
            <dd class="tabular-nums text-gray-700 dark:text-dark-200">
              {{ item.total_tokens > 0 ? formatTokensK(item.total_tokens) : '-' }}
            </dd>
          </div>
        </dl>

        <!-- 配额区：仅当配额存在、不是「其他」且至少一个窗口配了限额时显示 -->
        <div
          v-if="hasAnyLimit(item.quota) && !item.isOther"
          class="mt-3 space-y-2 border-t border-gray-200 pt-2 dark:border-dark-600"
          data-testid="platform-quota"
        >
          <p class="text-[10px] uppercase tracking-wide text-gray-400 dark:text-dark-500">
            {{ t('dashboard.platformQuota.title') }}
          </p>
          <template v-for="w in QUOTA_WINDOWS" :key="w">
            <div v-if="quotaLimit(item.quota, w) !== null" class="space-y-1">
              <div class="flex items-center justify-between gap-2 text-xs">
                <span class="text-gray-600 dark:text-dark-300">{{ t(`dashboard.platformQuota.${w}`) }}</span>
                <!-- 限额为 0：完全禁用 -->
                <span v-if="quotaLimit(item.quota, w) === 0" class="text-red-500">{{ t('dashboard.platformQuota.disabled') }}</span>
                <span v-else class="tabular-nums text-gray-700 dark:text-dark-200">
                  {{ formatUsd(quotaUsage(item.quota, w), 2) }} / {{ formatUsd(quotaLimit(item.quota, w) as number, 2) }}
                </span>
              </div>
              <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  class="quota-bar h-full rounded-full"
                  :class="quotaLimit(item.quota, w) === 0 ? 'bg-red-500' : quotaBarClass(quotaPercent(item.quota, w))"
                  :style="{ '--bar-w': `${quotaLimit(item.quota, w) === 0 ? 100 : quotaPercent(item.quota, w)}%` }"
                ></div>
              </div>
              <p v-if="quotaLimit(item.quota, w) !== 0 && quotaResetsAt(item.quota, w)" class="text-[10px] text-gray-400 dark:text-dark-500">
                {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(quotaResetsAt(item.quota, w)) }) }}
              </p>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：按平台拆分的用量与平台配额。原先放在上游 UserDashboardStats.vue 的第三行，
// 仪表盘换成 TokenRouter 布局后单独成卡，计算规则与原组件一致（「其他」差额卡、限额卡片规则、固定排序）。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { PlatformDashboardStats, UserDashboardStats } from '@/api/usage'
import type { GroupPlatform, PlatformQuotaItem } from '@/types'
import { formatNumberLocaleString as formatNumber, formatTokensK } from '@/utils/format'
import { formatUsd } from './format'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

type QuotaWindow = 'daily' | 'weekly' | 'monthly'

const props = defineProps<{
  stats: UserDashboardStats | null
  platformQuotas?: PlatformQuotaItem[] | null
  isSimple?: boolean
}>()
const { t } = useI18n()

const QUOTA_WINDOWS: QuotaWindow[] = ['daily', 'weekly', 'monthly']

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  zhipu: 'Zhipu GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  typesafe: 'TypeSafe / Jev',
}

const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'typesafe']

// 各平台之和小于总值的差额作为「其他」卡片显示（后端按平台聚合时过滤了无法归属平台的行）。
const OTHER_THRESHOLD = 0.0001

const platformLabel = (platform: string) => PLATFORM_LABELS[platform] ?? platform

function hasAnyLimit(q: PlatformQuotaItem | undefined): boolean {
  if (!q) return false
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

const platformCards = computed<FusedPlatformCard[]>(() => {
  const byPlat = new Map<string, PlatformDashboardStats>()
  for (const item of props.stats?.by_platform ?? []) byPlat.set(item.platform, item)

  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const q of props.platformQuotas ?? []) byQuota.set(q.platform, q)

  // 卡片集合 = 有用量的平台 ∪ 至少配置了一档限额的平台；三档全空的限额记录等价于不限额。
  const platforms = new Set<string>(byPlat.keys())
  for (const [platform, q] of byQuota) {
    if (hasAnyLimit(q)) platforms.add(platform)
  }

  const cards: FusedPlatformCard[] = []
  for (const platform of platforms) {
    const stat = byPlat.get(platform)
    cards.push({
      platform,
      total_actual_cost: stat?.total_actual_cost ?? 0,
      today_actual_cost: stat?.today_actual_cost ?? 0,
      total_requests: stat?.total_requests ?? 0,
      total_tokens: stat?.total_tokens ?? 0,
      quota: byQuota.get(platform),
    })
  }

  // 按固定顺序排列，未知平台按名称排在后面
  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  const total = props.stats?.total_actual_cost ?? 0
  const today = props.stats?.today_actual_cost ?? 0
  const diffTotal = Math.max(0, total - cards.reduce((sum, card) => sum + card.total_actual_cost, 0))
  const diffToday = Math.max(0, today - cards.reduce((sum, card) => sum + card.today_actual_cost, 0))
  if (diffTotal > OTHER_THRESHOLD || diffToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: diffTotal,
      today_actual_cost: diffToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true,
    })
  }
  return cards
})

// 标题右侧的平台计数 = 实际渲染的平台卡片数，不含「其他」差额卡。
const platformCount = computed(() => platformCards.value.filter((card) => !card.isOther).length)

const quotaLimit = (q: PlatformQuotaItem | undefined, w: QuotaWindow): number | null => q?.[`${w}_limit_usd`] ?? null
const quotaUsage = (q: PlatformQuotaItem | undefined, w: QuotaWindow): number => q?.[`${w}_usage_usd`] ?? 0
const quotaResetsAt = (q: PlatformQuotaItem | undefined, w: QuotaWindow): string | null => q?.[`${w}_window_resets_at`] ?? null

function quotaPercent(q: PlatformQuotaItem | undefined, w: QuotaWindow): number {
  const limit = quotaLimit(q, w)
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((quotaUsage(q, w) / limit) * 100)))
}

function quotaBarClass(percent: number): string {
  if (percent >= 95) return 'bg-red-500'
  if (percent >= 75) return 'bg-amber-500'
  return 'bg-emerald-500'
}

function formatResetTime(iso: string | null): string {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}
</script>

<style scoped>
/* 配额条从 0 伸展到当前占比 */
.quota-bar {
  width: var(--bar-w);
  animation: quota-bar var(--dash-top-model-bar-ms, 600ms) var(--motion-ease) both;
}

@keyframes quota-bar {
  from {
    width: 0;
  }
  to {
    width: var(--bar-w);
  }
}

@media (prefers-reduced-motion: reduce) {
  .quota-bar {
    animation: none;
  }
}
</style>
