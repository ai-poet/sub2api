<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('admin.referral.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('admin.referral.description') }}</p>
      </div>

      <section class="card">
        <button class="flex w-full items-center justify-between px-6 py-4 text-left" @click="settingsOpen = !settingsOpen">
          <span class="font-semibold text-gray-900 dark:text-white">{{ t('admin.referral.policy') }}</span>
          <span class="text-sm text-primary-600">{{ settingsOpen ? t('common.collapse') : t('common.expand') }}</span>
        </button>
        <form v-if="settingsOpen" class="grid gap-5 border-t border-gray-100 p-6 dark:border-dark-700 md:grid-cols-2" @submit.prevent="saveSettings">
          <div class="flex items-center justify-between md:col-span-2">
            <div><p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.referral.enabled') }}</p><p class="text-xs text-gray-500">{{ t('admin.referral.enabledDesc') }}</p></div>
            <Toggle v-model="settings.enabled" />
          </div>
          <div><label class="input-label">{{ t('admin.referral.rebateRate') }}</label><div class="mt-1 flex items-center gap-2"><input v-model.number="settings.rebate_rate_percent" type="number" min="0" max="100" step="0.01" class="input" /><span>%</span></div></div>
          <div><label class="input-label">{{ t('admin.referral.freezeHours') }}</label><input v-model.number="settings.rebate_freeze_hours" type="number" min="0" max="720" class="input mt-1" /><p class="input-hint">{{ t('admin.referral.zeroUnlimited') }}</p></div>
          <div><label class="input-label">{{ t('admin.referral.durationDays') }}</label><input v-model.number="settings.rebate_duration_days" type="number" min="0" max="3650" class="input mt-1" /><p class="input-hint">{{ t('admin.referral.zeroUnlimited') }}</p></div>
          <div><label class="input-label">{{ t('admin.referral.perInviteeCap') }}</label><input v-model.number="settings.rebate_per_invitee_cap" type="number" min="0" step="0.01" class="input mt-1" /><p class="input-hint">{{ t('admin.referral.zeroUnlimited') }}</p></div>
          <div class="flex items-center justify-between md:col-span-2"><span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.referral.adminRecharge') }}</span><Toggle v-model="settings.admin_recharge_enabled" /></div>
          <div class="md:col-span-2"><button type="submit" class="btn btn-primary" :disabled="saving">{{ saving ? t('common.saving') : t('common.save') }}</button></div>
        </form>
      </section>

      <section class="card overflow-hidden">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <div class="flex gap-2">
            <button v-for="tab in tabs" :key="tab" :class="activeTab === tab ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400' : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700'" class="rounded-lg px-4 py-2 text-sm font-medium" @click="changeTab(tab)">{{ t(`admin.referral.tabs.${tab}`) }}</button>
          </div>
          <form class="flex gap-2" @submit.prevent="reload"><input v-model="search" class="input w-64" :placeholder="t('admin.referral.searchPlaceholder')" /><button class="btn btn-secondary" type="submit">{{ t('common.search') }}</button></form>
        </div>

        <div v-if="activeTab === 'invites' && selected.size" class="flex flex-wrap items-center gap-3 border-b border-gray-100 bg-gray-50 px-6 py-3 dark:border-dark-700 dark:bg-dark-900/30">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.referral.selected', { count: selected.size }) }}</span>
          <input v-model.number="batchRate" type="number" min="0" max="100" step="0.01" class="input w-32" placeholder="0-100" />
          <button class="btn btn-primary btn-sm" :disabled="batchRate === null" @click="applyBatchRate(false)">{{ t('admin.referral.batchRate') }}</button>
          <button class="btn btn-secondary btn-sm" @click="applyBatchRate(true)">{{ t('admin.referral.followGlobal') }}</button>
        </div>

        <div v-if="loading" class="flex justify-center py-16"><div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div></div>
        <div v-else class="overflow-x-auto">
          <table v-if="activeTab === 'invites'" class="w-full">
            <thead><tr class="border-b border-gray-100 text-left text-xs uppercase text-gray-500 dark:border-dark-700"><th class="px-4 py-3"><input type="checkbox" :checked="allSelected" @change="toggleAll" /></th><th class="px-4 py-3">{{ t('admin.referral.invitee') }}</th><th class="px-4 py-3">{{ t('admin.referral.inviter') }}</th><th class="px-4 py-3">{{ t('referral.code') }}</th><th class="px-4 py-3">{{ t('admin.referral.customRate') }}</th><th class="px-4 py-3">{{ t('referral.totalEarned') }}</th><th class="px-4 py-3">{{ t('common.actions') }}</th></tr></thead>
            <tbody>
              <tr v-if="!relations.length"><td colspan="7" class="px-6 py-10 text-center text-sm text-gray-400">{{ t('admin.referral.empty') }}</td></tr>
              <tr v-for="item in relations" :key="item.user_id" class="border-b border-gray-50 text-sm dark:border-dark-700/50">
                <td class="px-4 py-4"><input type="checkbox" :checked="selected.has(item.user_id)" @change="toggleSelected(item.user_id)" /></td>
                <td class="px-4 py-4"><button class="text-left text-primary-600 hover:underline" @click="showOverview(item.user_id)">{{ item.email || `#${item.user_id}` }}</button><p class="text-xs text-gray-400">#{{ item.user_id }}</p></td>
                <td class="px-4 py-4 text-gray-600 dark:text-gray-300">{{ item.inviter_email || `#${item.inviter_id}` }}</td>
                <td class="px-4 py-4 font-mono">{{ item.aff_code }}</td>
                <td class="px-4 py-4">{{ item.aff_rebate_rate_percent == null ? t('admin.referral.global') : `${item.aff_rebate_rate_percent}%` }}</td>
                <td class="px-4 py-4">{{ money(item.aff_history_quota) }}</td>
                <td class="px-4 py-4"><button class="text-primary-600 hover:underline" @click="editUser(item)">{{ t('common.edit') }}</button></td>
              </tr>
            </tbody>
          </table>

          <table v-else class="w-full">
            <thead><tr class="border-b border-gray-100 text-left text-xs uppercase text-gray-500 dark:border-dark-700"><th class="px-6 py-3">{{ t('admin.referral.inviter') }}</th><th v-if="activeTab === 'rebates'" class="px-6 py-3">{{ t('admin.referral.invitee') }}</th><th class="px-6 py-3">{{ t('referral.amount') }}</th><th v-if="activeTab === 'rebates'" class="px-6 py-3">{{ t('admin.referral.orderRef') }}</th><th class="px-6 py-3">{{ activeTab === 'rebates' ? t('referral.status') : t('admin.referral.balanceAfter') }}</th><th class="px-6 py-3">{{ t('referral.time') }}</th></tr></thead>
            <tbody>
              <tr v-if="!ledger.length"><td colspan="6" class="px-6 py-10 text-center text-sm text-gray-400">{{ t('admin.referral.empty') }}</td></tr>
              <tr v-for="item in ledger" :key="item.id" class="border-b border-gray-50 text-sm dark:border-dark-700/50">
                <td class="px-6 py-4">{{ item.email || `#${item.user_id}` }}</td>
                <td v-if="activeTab === 'rebates'" class="px-6 py-4">{{ item.source_email || `#${item.source_user_id}` }}</td>
                <td class="px-6 py-4 font-medium text-green-600">{{ money(item.amount) }}</td>
                <td v-if="activeTab === 'rebates'" class="max-w-56 truncate px-6 py-4 font-mono text-xs" :title="item.source_order_ref || ''">{{ item.source_order_ref || '—' }}</td>
                <td class="px-6 py-4 text-gray-500">{{ activeTab === 'rebates' ? (item.frozen_until ? t('referral.frozenUntil', { time: formatDate(item.frozen_until) }) : t('referral.completed')) : money(item.balance_after || 0) }}</td>
                <td class="px-6 py-4 text-gray-500">{{ formatDate(item.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="pagination.total > pagination.page_size" :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" :show-page-size-selector="false" @update:page="page => { pagination.page = page; reload() }" />
      </section>

      <section v-if="editing" class="card p-6" data-testid="affiliate-editor">
        <div class="flex items-start justify-between"><div><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.referral.editUser', { user: editing.email || `#${editing.user_id}` }) }}</h2><p class="mt-1 text-xs text-gray-500">{{ t('admin.referral.editHint') }}</p></div><button class="text-gray-400" @click="editing = null">×</button></div>
        <div class="mt-5 grid gap-4 md:grid-cols-2">
          <div><label class="input-label">{{ t('referral.code') }}</label><input v-model="editCode" class="input mt-1 font-mono" maxlength="32" /></div>
          <div><label class="input-label">{{ t('admin.referral.customRate') }}</label><div class="mt-1 flex gap-2"><input v-model.number="editRate" type="number" min="0" max="100" step="0.01" class="input" :placeholder="t('admin.referral.global')" /><button class="btn btn-secondary" @click="saveRate(true)">{{ t('admin.referral.followGlobal') }}</button></div></div>
        </div>
        <div class="mt-5 flex gap-2"><button class="btn btn-primary" @click="saveUser">{{ t('common.save') }}</button><button class="btn btn-secondary" @click="resetCode">{{ t('admin.referral.resetCode') }}</button></div>
      </section>

      <section v-if="overview" class="card p-6"><div class="flex justify-between"><h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.referral.overview', { user: overview.email || `#${overview.user_id}` }) }}</h2><button class="text-gray-400" @click="overview = null">×</button></div><div class="mt-4 grid grid-cols-2 gap-4 md:grid-cols-5"><div v-for="stat in overviewStats" :key="stat.label" class="rounded-lg bg-gray-50 p-4 dark:bg-dark-900/40"><p class="text-xs text-gray-500">{{ stat.label }}</p><p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ stat.value }}</p></div></div></section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useAppStore } from '@/stores'
import {
  getReferralSettings, updateReferralSettings, listReferrals, listAffiliateRebates,
  listAffiliateTransfers, updateAffiliateUser, resetAffiliateUserCode,
  batchSetAffiliateRate, getAffiliateUserOverview
} from '@/api/admin/referral'
import type { AffiliateDetail, AffiliateLedgerEntry, AffiliateRelation, ReferralSettings } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const tabs = ['invites', 'rebates', 'transfers'] as const
type Tab = typeof tabs[number]
const activeTab = ref<Tab>('invites')
const loading = ref(false)
const saving = ref(false)
const settingsOpen = ref(false)
const search = ref('')
const relations = ref<AffiliateRelation[]>([])
const ledger = ref<AffiliateLedgerEntry[]>([])
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const settings = reactive<ReferralSettings>({ enabled: false, rebate_rate_percent: 20, rebate_freeze_hours: 0, rebate_duration_days: 0, rebate_per_invitee_cap: 0, admin_recharge_enabled: false })
const selected = reactive(new Set<number>())
const batchRate = ref<number | null>(null)
const editing = ref<AffiliateRelation | null>(null)
const overview = ref<AffiliateDetail | null>(null)
const editCode = ref('')
const editRate = ref<number | null>(null)

const money = (value: number) => `$${Number(value || 0).toFixed(2)}`
const formatDate = (value: string) => new Date(value).toLocaleString()
const allSelected = computed(() => relations.value.length > 0 && relations.value.every(item => selected.has(item.user_id)))
const overviewStats = computed(() => overview.value ? [
  { label: t('admin.referral.customRate'), value: `${overview.value.effective_rebate_rate_percent}%` },
  { label: t('referral.totalInvited'), value: overview.value.aff_count },
  { label: t('referral.availableQuota'), value: money(overview.value.aff_quota) },
  { label: t('referral.frozenQuota'), value: money(overview.value.aff_frozen_quota) },
  { label: t('referral.totalEarned'), value: money(overview.value.aff_history_quota) }
] : [])

async function reload() {
  loading.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.page_size, search: search.value || undefined }
    if (activeTab.value === 'invites') {
      const data = await listReferrals(params)
      relations.value = data.items || []
      ledger.value = []
      pagination.total = data.total || 0
    } else {
      const data = await (activeTab.value === 'rebates' ? listAffiliateRebates(params) : listAffiliateTransfers(params))
      ledger.value = data.items || []
      relations.value = []
      pagination.total = data.total || 0
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.referral.loadFailed'))
  } finally { loading.value = false }
}

function changeTab(tab: Tab) { activeTab.value = tab; pagination.page = 1; selected.clear(); reload() }
function toggleSelected(id: number) { selected.has(id) ? selected.delete(id) : selected.add(id) }
function toggleAll() { allSelected.value ? selected.clear() : relations.value.forEach(item => selected.add(item.user_id)) }
function editUser(item: AffiliateRelation) { editing.value = item; editCode.value = item.aff_code; editRate.value = item.aff_rebate_rate_percent ?? null }

async function saveUser() {
  if (!editing.value) return
  try {
    const detail = await updateAffiliateUser(editing.value.user_id, { aff_code: editCode.value, ...(editRate.value == null ? { clear_rebate_rate: true } : { aff_rebate_rate_percent: editRate.value }) })
    editing.value = null; overview.value = detail
    appStore.showSuccess(t('admin.referral.saved')); await reload()
  } catch (error: any) { appStore.showError(error.response?.data?.detail || t('admin.referral.saveFailed')) }
}
async function saveRate(clear: boolean) {
  if (!editing.value) return
  try { await updateAffiliateUser(editing.value.user_id, clear ? { clear_rebate_rate: true } : { aff_rebate_rate_percent: editRate.value ?? 0 }); editing.value = null; await reload() } catch (error: any) { appStore.showError(error.response?.data?.detail || t('admin.referral.saveFailed')) }
}
async function resetCode() {
  if (!editing.value) return
  try { await resetAffiliateUserCode(editing.value.user_id); editing.value = null; await reload() } catch (error: any) { appStore.showError(error.response?.data?.detail || t('admin.referral.saveFailed')) }
}
async function applyBatchRate(clear: boolean) {
  try { await batchSetAffiliateRate({ user_ids: [...selected], ...(clear ? { clear: true } : { aff_rebate_rate_percent: batchRate.value ?? 0 }) }); selected.clear(); appStore.showSuccess(t('admin.referral.saved')); await reload() } catch (error: any) { appStore.showError(error.response?.data?.detail || t('admin.referral.saveFailed')) }
}
async function showOverview(id: number) { try { overview.value = await getAffiliateUserOverview(id) } catch (error: any) { appStore.showError(error.response?.data?.detail || t('admin.referral.loadFailed')) } }
async function saveSettings() { saving.value = true; try { await updateReferralSettings({ ...settings }); appStore.showSuccess(t('admin.referral.saved')) } catch (error: any) { appStore.showError(error.response?.data?.detail || t('admin.referral.saveFailed')) } finally { saving.value = false } }

onMounted(async () => { try { Object.assign(settings, await getReferralSettings()) } catch { appStore.showError(t('admin.referral.loadFailed')) }; await reload() })
</script>
