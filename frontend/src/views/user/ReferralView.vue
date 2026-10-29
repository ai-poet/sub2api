<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('referral.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('referral.description') }}</p>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-16" data-testid="referral-loading">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>
      <div v-else-if="loadError" class="card p-12 text-center">
        <Icon name="exclamationCircle" size="xl" class="mx-auto mb-4 text-red-500" />
        <p class="mb-4 text-gray-500 dark:text-dark-400">{{ loadError }}</p>
        <button class="btn btn-primary" @click="loadData">{{ t('common.retry') }}</button>
      </div>

      <template v-else-if="info">
        <div class="card border-primary-200 bg-primary-50 p-5 dark:border-primary-800/50 dark:bg-primary-900/20">
          <div class="flex items-start gap-3">
            <Icon name="infoCircle" class="mt-0.5 text-primary-600 dark:text-primary-400" />
            <div class="text-sm text-primary-800 dark:text-primary-300">
              <p class="font-semibold">{{ t('referral.rulesTitle') }}</p>
              <p class="mt-1">{{ t('referral.ruleRate', { rate: info.effective_rebate_rate_percent }) }}</p>
              <p v-if="info.rewards?.rebate_freeze_hours" class="mt-1">{{ t('referral.ruleFreeze', { hours: info.rewards.rebate_freeze_hours }) }}</p>
              <p class="mt-1">{{ t('referral.ruleTransfer') }}</p>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.rebateRate') }}</p>
            <p class="mt-2 text-2xl font-bold text-primary-600 dark:text-primary-400">{{ info.effective_rebate_rate_percent }}%</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.totalInvited') }}</p>
            <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ info.stats.total_count }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-bold text-green-600 dark:text-green-400" data-testid="affiliate-available">{{ money(info.aff_quota) }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('referral.totalEarned') }}</p>
            <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ money(info.aff_history_quota) }}</p>
            <p class="mt-1 text-xs text-amber-600 dark:text-amber-400">{{ t('referral.frozenQuota') }} {{ money(info.aff_frozen_quota) }}</p>
          </div>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <section class="card p-6">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('referral.yourLink') }}</h2>
            <label class="input-label mt-5">{{ t('referral.code') }}</label>
            <div class="mt-1 flex gap-2">
              <input readonly :value="info.referral_code" class="input flex-1 font-mono" data-testid="affiliate-code" />
              <button class="btn btn-secondary" @click="copy(info.referral_code, 'code')">{{ copiedCode ? t('referral.copied') : t('common.copy') }}</button>
            </div>
            <label class="input-label mt-4">{{ t('referral.inviteLink') }}</label>
            <div class="mt-1 flex gap-2">
              <input readonly :value="referralLink" class="input flex-1" data-testid="affiliate-link" />
              <button class="btn btn-primary" @click="copy(referralLink, 'link')">{{ copiedLink ? t('referral.copied') : t('referral.copyLink') }}</button>
            </div>
          </section>

          <section class="card flex flex-col justify-between p-6">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('referral.transferTitle') }}</h2>
              <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('referral.transferDescription') }}</p>
              <p class="mt-5 text-3xl font-bold text-gray-900 dark:text-white">{{ money(info.aff_quota) }}</p>
            </div>
            <button
              class="btn btn-primary mt-6 w-full"
              data-testid="affiliate-transfer"
              :disabled="transferring || info.aff_quota <= 0"
              @click="transfer"
            >
              {{ transferring ? t('referral.transferring') : t('referral.transferAction') }}
            </button>
          </section>
        </div>

        <section class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('referral.invitees') }}</h2>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead><tr class="border-b border-gray-100 text-left text-xs uppercase text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th class="px-6 py-3">{{ t('referral.referee') }}</th>
                <th class="px-6 py-3">{{ t('referral.totalRebate') }}</th>
                <th class="px-6 py-3">{{ t('referral.time') }}</th>
              </tr></thead>
              <tbody>
                <tr v-if="!info.invitees?.length"><td colspan="3" class="px-6 py-8 text-center text-sm text-gray-400">{{ t('referral.noInvitees') }}</td></tr>
                <tr v-for="item in info.invitees" :key="item.user_id" class="border-b border-gray-50 text-sm dark:border-dark-700/50">
                  <td class="px-6 py-4 text-gray-900 dark:text-white">{{ item.email }}</td>
                  <td class="px-6 py-4 text-green-600 dark:text-green-400">{{ money(item.total_rebate) }}</td>
                  <td class="px-6 py-4 text-gray-500">{{ formatDate(item.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('referral.history') }}</h2>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead><tr class="border-b border-gray-100 text-left text-xs uppercase text-gray-500 dark:border-dark-700 dark:text-dark-400">
                <th class="px-6 py-3">{{ t('referral.type') }}</th>
                <th class="px-6 py-3">{{ t('referral.referee') }}</th>
                <th class="px-6 py-3">{{ t('referral.amount') }}</th>
                <th class="px-6 py-3">{{ t('referral.status') }}</th>
                <th class="px-6 py-3">{{ t('referral.time') }}</th>
              </tr></thead>
              <tbody>
                <tr v-if="!history.length"><td colspan="5" class="px-6 py-8 text-center text-sm text-gray-400">{{ t('referral.noHistory') }}</td></tr>
                <tr v-for="item in history" :key="item.id" class="border-b border-gray-50 text-sm dark:border-dark-700/50">
                  <td class="px-6 py-4"><span :class="item.action === 'accrue' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'" class="rounded-full px-2.5 py-1 text-xs">{{ t(`referral.action.${item.action}`) }}</span></td>
                  <td class="px-6 py-4 text-gray-900 dark:text-white">{{ item.source_email || '—' }}</td>
                  <td class="px-6 py-4 font-medium text-gray-900 dark:text-white">{{ money(item.amount) }}</td>
                  <td class="px-6 py-4 text-gray-500">{{ item.frozen_until ? t('referral.frozenUntil', { time: formatDate(item.frozen_until) }) : t('referral.completed') }}</td>
                  <td class="px-6 py-4 text-gray-500">{{ formatDate(item.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <Pagination v-if="pagination.total > pagination.page_size" :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" :show-page-size-selector="false" @update:page="handlePageChange" />
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getReferralInfo, getReferralHistory, transferAffiliateQuota } from '@/api/referral'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { ReferralInfo, UserReferral } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const transferring = ref(false)
const loadError = ref('')
const info = ref<ReferralInfo | null>(null)
const history = ref<UserReferral[]>([])
const copiedCode = ref(false)
const copiedLink = ref(false)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const referralLink = computed(() => {
  if (!info.value) return ''
  if (info.value.referral_link) return info.value.referral_link
  const origin = typeof window === 'undefined' ? '' : window.location.origin
  return `${origin}/register?ref=${info.value.referral_code}`
})

const money = (value: number) => `$${Number(value || 0).toFixed(2)}`
const formatDate = (value: string) => new Date(value).toLocaleString()

async function loadHistory() {
  const data = await getReferralHistory({ page: pagination.page, page_size: pagination.page_size })
  history.value = data.items || []
  pagination.total = data.total || 0
  pagination.page = data.page || 1
}

async function loadData() {
  loading.value = true
  loadError.value = ''
  try {
    const [detail] = await Promise.all([getReferralInfo(), loadHistory()])
    info.value = detail
  } catch (error: any) {
    loadError.value = error.response?.data?.detail || error.message || t('referral.loadFailed')
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadHistory().catch(() => appStore.showError(t('referral.loadFailed')))
}

async function copy(value: string, kind: 'code' | 'link') {
  try {
    await navigator.clipboard.writeText(value)
    const target = kind === 'code' ? copiedCode : copiedLink
    target.value = true
    setTimeout(() => { target.value = false }, 2000)
  } catch {
    appStore.showError(t('referral.copyFailed'))
  }
}

async function transfer() {
  if (!info.value || transferring.value || info.value.aff_quota <= 0) return
  transferring.value = true
  try {
    const entry = await transferAffiliateQuota()
    appStore.showSuccess(t('referral.transferSucceeded', { amount: money(entry.amount) }))
    await Promise.all([getReferralInfo().then(detail => { info.value = detail }), loadHistory()])
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('referral.transferFailed'))
  } finally {
    transferring.value = false
  }
}

onMounted(loadData)
</script>
