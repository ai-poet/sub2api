<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Page Header -->
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('referral.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ t('referral.description') }}
        </p>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- Error State -->
      <div v-else-if="loadError" class="card p-12 text-center">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
          <Icon name="exclamationCircle" size="xl" class="text-red-500 dark:text-red-400" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('referral.loadFailed') }}
        </h3>
        <p class="mb-4 text-gray-500 dark:text-dark-400">{{ loadError }}</p>
        <button @click="loadData" class="btn btn-primary">{{ t('common.retry') }}</button>
      </div>

      <template v-else-if="info">
        <!-- Reward Rules Card -->
        <div class="card border-primary-200 bg-primary-50 dark:border-primary-800/50 dark:bg-primary-900/20">
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-100 dark:bg-primary-900/30">
                <Icon name="infoCircle" size="md" class="text-primary-600 dark:text-primary-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-primary-800 dark:text-primary-300">
                  {{ t('referral.rulesTitle') }}
                </h3>
                <ul class="mt-2 list-inside list-disc space-y-1 text-sm text-primary-700 dark:text-primary-400">
                  <li>{{ t('referral.rule1') }}</li>
                  <li>{{ t('referral.rule2') }}</li>
                  <li>{{ t('referral.rule3') }}</li>
                </ul>
              </div>
            </div>
          </div>
        </div>

        <!-- Referral Link Card -->
        <div class="rounded-xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-800">
          <h2 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('referral.yourLink') }}
          </h2>
          <div class="flex items-center gap-3">
            <input type="text" readonly :value="referralLink" class="input flex-1" />
            <button @click="copyLink" class="btn btn-primary whitespace-nowrap">
              {{ copied ? t('referral.copied') : t('referral.copyLink') }}
            </button>
          </div>
          <p class="mt-2 text-xs text-gray-400 dark:text-dark-500">
            {{ t('referral.code') }}: {{ info.referral_code }}
          </p>
        </div>

        <!-- Stats Cards -->
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.totalInvited') }}</p>
            <p class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ info.stats.total_count }}</p>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.rewarded') }}</p>
            <p class="mt-1 text-2xl font-bold text-green-600 dark:text-green-400">{{ info.stats.rewarded_count }}</p>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.pending') }}</p>
            <p class="mt-1 text-2xl font-bold text-amber-600 dark:text-amber-400">{{ info.stats.pending_count }}</p>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.totalEarned') }}</p>
            <p class="mt-1 text-2xl font-bold text-primary-600 dark:text-primary-400">${{ info.stats.total_balance_earn.toFixed(2) }}</p>
          </div>
        </div>

        <!-- History Table -->
        <div class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.history') }}
            </h2>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead>
                <tr class="border-b border-gray-200 dark:border-dark-700">
                  <th class="px-6 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('referral.referee') }}</th>
                  <th class="px-6 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('referral.status') }}</th>
                  <th class="px-6 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('referral.reward') }}</th>
                  <th class="px-6 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('referral.time') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="history.length === 0">
                  <td colspan="4" class="px-6 py-8 text-center text-sm text-gray-400 dark:text-dark-500">
                    {{ t('referral.noHistory') }}
                  </td>
                </tr>
                <tr v-for="item in history" :key="item.id" class="border-b border-gray-100 dark:border-dark-700/50">
                  <td class="px-6 py-4 text-sm text-gray-900 dark:text-white">{{ item.referee_email || '-' }}</td>
                  <td class="px-6 py-4">
                    <span
                      :class="item.status === 'rewarded'
                        ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                        : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'"
                      class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium"
                    >
                      {{ item.status === 'rewarded' ? t('referral.statusRewarded') : t('referral.statusPending') }}
                    </span>
                  </td>
                  <td class="px-6 py-4 text-sm text-gray-900 dark:text-white">
                    <span v-if="item.status === 'rewarded'">
                      ${{ item.referrer_balance_reward.toFixed(2) }}
                      <span v-if="item.referrer_subscription_days > 0" class="text-gray-400">
                        + {{ item.referrer_subscription_days }}{{ t('referral.days') }}
                      </span>
                    </span>
                    <span v-else class="text-gray-400">-</span>
                  </td>
                  <td class="px-6 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDate(item.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <!-- Pagination -->
          <Pagination
            v-if="pagination.total > pagination.page_size"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            :show-page-size-selector="false"
            @update:page="handlePageChange"
          />
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getReferralInfo, getReferralHistory } from '@/api/referral'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { ReferralInfo, UserReferral } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const loadError = ref('')
const info = ref<ReferralInfo | null>(null)
const history = ref<UserReferral[]>([])
const copied = ref(false)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const referralLink = computed(() => {
  if (!info.value) return ''
  if (info.value.referral_link) return info.value.referral_link
  if (typeof window === 'undefined') return `/register?ref=${info.value.referral_code}`
  return `${window.location.origin}/register?ref=${info.value.referral_code}`
})

async function loadData() {
  loading.value = true
  loadError.value = ''
  try {
    const [infoData, historyData] = await Promise.all([
      getReferralInfo(),
      getReferralHistory({ page: pagination.page, page_size: pagination.page_size })
    ])
    info.value = infoData
    history.value = historyData.items || []
    pagination.total = historyData.total || 0
    pagination.page = historyData.page || 1
  } catch (error: any) {
    loadError.value = error.response?.data?.detail || error.message || t('common.unknownError')
  } finally {
    loading.value = false
  }
}

async function loadHistory() {
  try {
    const historyData = await getReferralHistory({ page: pagination.page, page_size: pagination.page_size })
    history.value = historyData.items || []
    pagination.total = historyData.total || 0
    pagination.page = historyData.page || 1
  } catch (error) {
    console.error('Failed to load referral history:', error)
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadHistory()
}

async function copyLink() {
  if (!info.value) return
  try {
    await navigator.clipboard.writeText(referralLink.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    appStore.showError(t('referral.copyFailed'))
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(loadData)
</script>
