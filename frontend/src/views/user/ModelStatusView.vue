<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelStatus.autoRefresh') }}</div>
          <div class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ t('modelStatus.lastUpdated') }}:
            <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">
              {{ lastUpdatedLabel }}
            </span>
          </div>
        </div>

        <button type="button" class="btn btn-secondary" :disabled="refreshing" @click="handleRefresh">
          <span
            v-if="refreshing"
            class="mr-2 inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
          ></span>
          <Icon name="refresh" size="sm" class="mr-2" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div
        v-if="initialLoading"
        class="card flex items-center justify-center py-16"
      >
        <div class="h-10 w-10 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div
        v-else-if="!featureEnabled"
        class="card p-10 text-center"
      >
        <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="server" size="lg" class="text-gray-400" />
        </div>
        <h3 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('modelStatus.featureDisabledTitle') }}
        </h3>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('modelStatus.featureDisabledDescription') }}
        </p>
      </div>

      <template v-else>
        <div class="grid gap-4 md:grid-cols-4">
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelStatus.totalGroups') }}</div>
            <div class="mt-3 text-3xl font-semibold text-gray-900 dark:text-white">{{ items.length }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelStatus.healthyGroups') }}</div>
            <div class="mt-3 text-3xl font-semibold text-emerald-600 dark:text-emerald-400">{{ healthyCount }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelStatus.degradedGroups') }}</div>
            <div class="mt-3 text-3xl font-semibold text-amber-600 dark:text-amber-400">{{ degradedCount }}</div>
          </div>
          <div class="card p-5">
            <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelStatus.downGroups') }}</div>
            <div class="mt-3 text-3xl font-semibold text-rose-600 dark:text-rose-400">{{ downCount }}</div>
          </div>
        </div>

        <div
          v-if="loadError"
          class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-300"
        >
          {{ loadError }}
        </div>

        <div
          v-if="items.length === 0"
          class="card p-10 text-center"
        >
          <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
            <Icon name="chartBar" size="lg" class="text-gray-400" />
          </div>
          <h3 class="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('modelStatus.emptyTitle') }}
          </h3>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {{ t('modelStatus.emptyDescription') }}
          </p>
        </div>

        <div v-else class="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="item in items"
            :key="item.group.id"
            class="card p-5"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <PlatformIcon :platform="item.group.platform" size="sm" />
                  <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                    {{ item.group.name }}
                  </h3>
                </div>
                <p class="mt-2 line-clamp-2 text-sm text-gray-500 dark:text-gray-400">
                  {{ item.group.description || t('modelStatus.noDescription') }}
                </p>
              </div>
              <span :class="['badge', getGroupRuntimeStatusBadgeClass(getItemStatus(item))]">
                {{ getSummaryStatusText(item.summary) }}
              </span>
            </div>

            <div class="mt-4 grid grid-cols-2 gap-3">
              <div class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.latestProbe') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ item.summary.observed_at ? formatRelativeTime(item.summary.observed_at) : t('modelStatus.waitingForProbe') }}
                </div>
              </div>

              <div class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.latestLatency') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatGroupRuntimeLatency(item.summary.latency_ms) }}
                </div>
              </div>

              <div class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.availability24') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatGroupRuntimeAvailability(item.availability_24h) }}
                </div>
              </div>

              <div class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.availability7') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatGroupRuntimeAvailability(item.availability_7d) }}
                </div>
              </div>
            </div>

            <div
              class="mt-4 rounded-xl border px-3 py-3"
              :class="getGroupRuntimeStatusSurfaceClass(getItemStatus(item))"
            >
              <div class="text-xs font-medium opacity-80">{{ t('modelStatus.latestResult') }}</div>
              <div class="mt-2 text-sm leading-6">
                {{ getSummaryPreview(item.summary) }}
              </div>
            </div>

            <div class="mt-4 flex items-center justify-between">
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.summary.observed_at ? formatDateTime(item.summary.observed_at) : '—' }}
              </div>
              <button type="button" class="btn btn-secondary btn-sm" @click="openDetails(item)">
                {{ t('modelStatus.openDetails') }}
              </button>
            </div>
          </article>
        </div>
      </template>
    </div>

    <BaseDialog
      :show="showDetails"
      :title="selectedItem ? selectedItem.group.name : t('modelStatus.title')"
      width="extra-wide"
      @close="closeDetails"
    >
      <div v-if="selectedItem" class="space-y-6">
        <div class="flex flex-col gap-3 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/80 md:flex-row md:items-center md:justify-between">
          <div>
            <div class="flex items-center gap-2">
              <PlatformIcon :platform="selectedItem.group.platform" size="sm" />
              <div class="text-base font-semibold text-gray-900 dark:text-white">
                {{ selectedItem.group.name }}
              </div>
            </div>
            <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t(`admin.groups.platforms.${selectedItem.group.platform}`) }}
            </div>
          </div>
          <span :class="['badge', getGroupRuntimeStatusBadgeClass(getItemStatus(selectedItem))]">
            {{ getSummaryStatusText(selectedItem.summary) }}
          </span>
        </div>

        <div class="grid gap-4 md:grid-cols-4">
          <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.latestProbe') }}</div>
            <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              {{ selectedItem.summary.observed_at ? formatDateTime(selectedItem.summary.observed_at) : t('modelStatus.waitingForProbe') }}
            </div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.latestLatency') }}</div>
            <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              {{ formatGroupRuntimeLatency(selectedItem.summary.latency_ms) }}
            </div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.availability24') }}</div>
            <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              {{ formatGroupRuntimeAvailability(selectedItem.availability_24h) }}
            </div>
          </div>
          <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelStatus.availability7') }}</div>
            <div class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              {{ formatGroupRuntimeAvailability(selectedItem.availability_7d) }}
            </div>
          </div>
        </div>

        <div
          v-if="detailError"
          class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-300"
        >
          {{ detailError }}
        </div>

        <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
          <div class="rounded-2xl border border-gray-200 p-5 dark:border-dark-700">
            <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div>
                <div class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('modelStatus.historyTitle') }}
                </div>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('modelStatus.historyDescription') }}
                </p>
              </div>

              <div class="inline-flex rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
                <button
                  type="button"
                  class="rounded-lg px-3 py-1.5 text-sm transition-colors"
                  :class="detailPeriod === '24h' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-gray-400'"
                  @click="changeDetailPeriod('24h')"
                >
                  {{ t('modelStatus.period24h') }}
                </button>
                <button
                  type="button"
                  class="rounded-lg px-3 py-1.5 text-sm transition-colors"
                  :class="detailPeriod === '7d' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 dark:text-gray-400'"
                  @click="changeDetailPeriod('7d')"
                >
                  {{ t('modelStatus.period7d') }}
                </button>
              </div>
            </div>

            <div v-if="detailLoading" class="flex items-center justify-center py-16">
              <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
            </div>

            <div v-else-if="historyBuckets.length === 0" class="mt-6 rounded-xl border border-dashed border-gray-200 px-4 py-12 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
              {{ t('modelStatus.noHistory') }}
            </div>

            <div v-else class="mt-6 space-y-4">
              <div class="flex h-44 items-end gap-1 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
                <div
                  v-for="bucket in historyBuckets"
                  :key="`${bucket.bucket_start}-${bucket.bucket_end}`"
                  class="group flex h-full min-w-0 flex-1 items-end"
                  :title="buildHistoryTooltip(bucket)"
                >
                  <div
                    class="w-full rounded-t-sm transition-all duration-200 group-hover:opacity-90"
                    :class="getGroupRuntimeStatusBarClass(bucket.latest_status || (bucket.down_count > 0 ? 'down' : 'up'))"
                    :style="{ height: `${historyBarHeight(bucket)}%` }"
                  ></div>
                </div>
              </div>

              <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                <span>{{ historyBuckets[0] ? formatDateTime(historyBuckets[0].bucket_start) : '—' }}</span>
                <span>{{ historyBuckets[historyBuckets.length - 1] ? formatDateTime(historyBuckets[historyBuckets.length - 1].bucket_end) : '—' }}</span>
              </div>
            </div>
          </div>

          <div class="space-y-6">
            <div class="rounded-2xl border border-gray-200 p-5 dark:border-dark-700">
              <div class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('modelStatus.eventsTitle') }}
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('modelStatus.eventsDescription') }}
              </p>

              <div v-if="detailLoading" class="flex items-center justify-center py-12">
                <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
              </div>

              <div v-else-if="events.length === 0" class="mt-5 rounded-xl border border-dashed border-gray-200 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
                {{ t('modelStatus.noEvents') }}
              </div>

              <div v-else class="mt-5 space-y-3">
                <div
                  v-for="event in events"
                  :key="event.id"
                  class="rounded-xl border border-gray-200 px-4 py-3 dark:border-dark-700"
                >
                  <div class="flex items-start justify-between gap-3">
                    <div class="flex items-center gap-2">
                      <span :class="['badge', getGroupRuntimeStatusBadgeClass(event.event_type === 'down' ? 'down' : 'up')]">
                        {{ t(`modelStatus.eventTypes.${event.event_type}`) }}
                      </span>
                      <span class="text-xs text-gray-500 dark:text-gray-400">
                        {{ formatRelativeTime(event.observed_at) }}
                      </span>
                    </div>
                    <span class="text-xs text-gray-500 dark:text-gray-400">
                      {{ formatDateTime(event.observed_at) }}
                    </span>
                  </div>

                  <div class="mt-3 flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                    <span :class="['badge', getGroupRuntimeStatusBadgeClass(event.from_status || 'unknown')]">
                      {{ t(`modelStatus.statuses.${normalizeGroupRuntimeStatus(event.from_status)}`) }}
                    </span>
                    <Icon name="arrowRight" size="xs" />
                    <span :class="['badge', getGroupRuntimeStatusBadgeClass(event.to_status || 'unknown')]">
                      {{ t(`modelStatus.statuses.${normalizeGroupRuntimeStatus(event.to_status)}`) }}
                    </span>
                  </div>

                  <div class="mt-3 text-xs text-gray-500 dark:text-gray-400">
                    <span>{{ t('modelStatus.latestLatency') }}: {{ formatGroupRuntimeLatency(event.latency_ms) }}</span>
                    <span class="mx-2">·</span>
                    <span>{{ t('modelStatus.httpCode') }}: {{ event.http_code ?? '-' }}</span>
                    <span v-if="event.sub_status" class="mx-2">·</span>
                    <span v-if="event.sub_status">{{ t('modelStatus.subStatus') }}: {{ event.sub_status }}</span>
                  </div>

                  <div
                    v-if="event.error_detail"
                    class="mt-3 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-300"
                  >
                    {{ event.error_detail }}
                  </div>
                </div>
              </div>
            </div>

            <div
              class="rounded-2xl border p-5"
              :class="getGroupRuntimeStatusSurfaceClass(getItemStatus(selectedItem))"
            >
              <div class="text-base font-semibold">{{ t('modelStatus.latestResult') }}</div>
              <div class="mt-3 whitespace-pre-wrap break-words text-sm leading-6">
                {{ getSummaryPreview(selectedItem.summary) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { groupStatusAPI } from '@/api/groupStatus'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import type {
  GroupStatusEvent,
  GroupStatusHistoryBucket,
  GroupStatusListItem,
} from '@/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import {
  formatGroupRuntimeAvailability,
  formatGroupRuntimeLatency,
  getGroupRuntimeStatusBadgeClass,
  getGroupRuntimeStatusBarClass,
  getGroupRuntimeStatusSurfaceClass,
  normalizeGroupRuntimeStatus,
  shortenRuntimeExcerpt,
} from '@/utils/groupStatus'

const POLL_INTERVAL_MS = 30_000

const { t } = useI18n()
const appStore = useAppStore()

const initialLoading = ref(true)
const refreshing = ref(false)
const loadError = ref('')
const featureForcedOff = ref(false)
const lastUpdatedAt = ref<Date | null>(null)
const items = ref<GroupStatusListItem[]>([])

const showDetails = ref(false)
const selectedItem = ref<GroupStatusListItem | null>(null)
const detailLoading = ref(false)
const detailError = ref('')
const detailPeriod = ref<'24h' | '7d'>('24h')
const historyBuckets = ref<GroupStatusHistoryBucket[]>([])
const events = ref<GroupStatusEvent[]>([])

let pollTimer: number | null = null

const featureEnabled = computed(() => {
  return (appStore.cachedPublicSettings?.group_status_enabled ?? false) && !featureForcedOff.value
})

const lastUpdatedLabel = computed(() => {
  if (!lastUpdatedAt.value) {
    return t('modelStatus.notAvailable')
  }
  return `${formatRelativeTime(lastUpdatedAt.value)} (${formatDateTime(lastUpdatedAt.value)})`
})

const healthyCount = computed(() => items.value.filter((item) => getItemStatus(item) === 'up').length)
const degradedCount = computed(() => items.value.filter((item) => getItemStatus(item) === 'degraded').length)
const downCount = computed(() => items.value.filter((item) => getItemStatus(item) === 'down').length)

function getItemStatus(item: GroupStatusListItem): 'up' | 'degraded' | 'down' | 'unknown' {
  if (item.summary.stable_status) {
    return normalizeGroupRuntimeStatus(item.summary.stable_status)
  }
  if (item.summary.latest_status) {
    return normalizeGroupRuntimeStatus(item.summary.latest_status)
  }
  return 'unknown'
}

function getSummaryStatusText(summary: GroupStatusListItem['summary']): string {
  if (!summary.observed_at) {
    return t('modelStatus.waiting')
  }
  return t(`modelStatus.statuses.${normalizeGroupRuntimeStatus(summary.stable_status || summary.latest_status)}`)
}

function getSummaryPreview(summary: GroupStatusListItem['summary']): string {
  if (summary.error_detail) {
    return shortenRuntimeExcerpt(summary.error_detail, 180)
  }
  if (summary.response_excerpt) {
    return shortenRuntimeExcerpt(summary.response_excerpt, 180)
  }
  if (!summary.observed_at) {
    return t('modelStatus.waitingForProbe')
  }
  return t('modelStatus.notAvailable')
}

async function ensurePublicSettingsLoaded() {
  if (appStore.publicSettingsLoaded) {
    return
  }
  await appStore.fetchPublicSettings()
}

async function loadStatuses(manual: boolean = false) {
  if (!featureEnabled.value) {
    items.value = []
    initialLoading.value = false
    refreshing.value = false
    return
  }

  if (initialLoading.value && items.value.length === 0) {
    initialLoading.value = true
  } else {
    refreshing.value = true
  }

  try {
    const data = await groupStatusAPI.listStatuses()
    items.value = data
    lastUpdatedAt.value = new Date()
    loadError.value = ''

    if (selectedItem.value) {
      const matched = data.find((item) => item.group.id === selectedItem.value?.group.id)
      if (matched) {
        selectedItem.value = matched
      }
    }
  } catch (error: any) {
    if (error?.status === 404 || error?.code === 'GROUP_STATUS_FEATURE_DISABLED') {
      featureForcedOff.value = true
      items.value = []
    }

    loadError.value = error?.message || t('modelStatus.loadFailed')
    if (manual) {
      appStore.showError(loadError.value)
    }
  } finally {
    initialLoading.value = false
    refreshing.value = false
  }
}

async function loadDetails() {
  if (!selectedItem.value) {
    return
  }

  detailLoading.value = true
  detailError.value = ''
  try {
    const [history, recentEvents] = await Promise.all([
      groupStatusAPI.getHistory(selectedItem.value.group.id, detailPeriod.value),
      groupStatusAPI.getEvents(selectedItem.value.group.id, 20)
    ])
    historyBuckets.value = history
    events.value = recentEvents
  } catch (error: any) {
    detailError.value = error?.message || t('modelStatus.detailLoadFailed')
  } finally {
    detailLoading.value = false
  }
}

function handleRefresh() {
  void loadStatuses(true)
}

function openDetails(item: GroupStatusListItem) {
  selectedItem.value = item
  detailPeriod.value = '24h'
  showDetails.value = true
  void loadDetails()
}

function closeDetails() {
  showDetails.value = false
  selectedItem.value = null
  historyBuckets.value = []
  events.value = []
  detailError.value = ''
}

function changeDetailPeriod(period: '24h' | '7d') {
  if (detailPeriod.value === period) {
    return
  }
  detailPeriod.value = period
}

function historyBarHeight(bucket: GroupStatusHistoryBucket): number {
  if (bucket.total_count === 0) {
    return 8
  }
  return Math.max(10, Math.min(100, Math.round(bucket.availability)))
}

function buildHistoryTooltip(bucket: GroupStatusHistoryBucket): string {
  return [
    `${formatDateTime(bucket.bucket_start)} - ${formatDateTime(bucket.bucket_end)}`,
    `${t('modelStatus.bucketAvailability')}: ${formatGroupRuntimeAvailability(bucket.availability)}`,
    `${t('modelStatus.sampleCount')}: ${bucket.total_count}`,
    `${t('modelStatus.avgLatency')}: ${formatGroupRuntimeLatency(bucket.avg_latency_ms)}`
  ].join('\n')
}

function startPolling() {
  stopPolling()
  if (!featureEnabled.value) {
    return
  }
  pollTimer = window.setInterval(() => {
    void loadStatuses(false)
  }, POLL_INTERVAL_MS)
}

function stopPolling() {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
}

watch(detailPeriod, () => {
  if (showDetails.value && selectedItem.value) {
    void loadDetails()
  }
})

watch(featureEnabled, (enabled) => {
  if (!enabled) {
    stopPolling()
    return
  }
  startPolling()
})

onMounted(async () => {
  try {
    await ensurePublicSettingsLoaded()
    await loadStatuses(false)
    startPolling()
  } finally {
    initialLoading.value = false
  }
})

onBeforeUnmount(() => {
  stopPolling()
})
</script>
