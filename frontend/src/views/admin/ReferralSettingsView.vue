<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <form v-else @submit.prevent="saveSettings" class="space-y-6">
        <!-- Referral Settings Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.referral.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.referral.description') }}
            </p>
          </div>
          <div class="space-y-4 p-6">
            <!-- Enable Toggle -->
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.referral.enabled') }}
                </label>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.referral.enabledDesc') }}
                </p>
              </div>
              <Toggle v-model="form.enabled" />
            </div>

            <!-- Max Per User -->
            <div>
              <label class="input-label">{{ t('admin.referral.maxPerUser') }}</label>
              <input v-model.number="form.max_per_user" type="number" min="0" class="input mt-1 w-40" />
              <p class="input-hint">{{ t('admin.referral.maxPerUserHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Referrer Rewards Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.referral.referrerRewards') }}
            </h2>
          </div>
          <div class="space-y-4 p-6">
            <div>
              <label class="input-label">{{ t('admin.referral.balanceReward') }}</label>
              <input v-model.number="form.referrer_balance_reward" type="number" min="0" step="0.01" class="input mt-1 w-40" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.referral.groupId') }}</label>
              <select v-model.number="form.referrer_group_id" class="input mt-1 w-64">
                <option :value="0">{{ t('admin.referral.noGroup') }}</option>
                <option v-for="group in groups" :key="group.id" :value="group.id">
                  {{ group.name }} ({{ group.platform }})
                </option>
              </select>
              <p class="input-hint">{{ t('admin.referral.groupIdHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.referral.subscriptionDays') }}</label>
              <input v-model.number="form.referrer_subscription_days" type="number" min="0" class="input mt-1 w-40" />
            </div>
          </div>
        </div>

        <!-- Referee Rewards Card -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.referral.refereeRewards') }}
            </h2>
          </div>
          <div class="space-y-4 p-6">
            <div>
              <label class="input-label">{{ t('admin.referral.balanceReward') }}</label>
              <input v-model.number="form.referee_balance_reward" type="number" min="0" step="0.01" class="input mt-1 w-40" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.referral.groupId') }}</label>
              <select v-model.number="form.referee_group_id" class="input mt-1 w-64">
                <option :value="0">{{ t('admin.referral.noGroup') }}</option>
                <option v-for="group in groups" :key="group.id" :value="group.id">
                  {{ group.name }} ({{ group.platform }})
                </option>
              </select>
              <p class="input-hint">{{ t('admin.referral.groupIdHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.referral.subscriptionDays') }}</label>
              <input v-model.number="form.referee_subscription_days" type="number" min="0" class="input mt-1 w-40" />
            </div>
          </div>
        </div>

        <!-- Save Button -->
        <div class="flex justify-end">
          <button type="submit" :disabled="saving" class="btn btn-primary">
            <svg v-if="saving" class="mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getReferralSettings, updateReferralSettings } from '@/api/admin/referral'
import { getAll as getAllGroups } from '@/api/admin/groups'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { ReferralSettings, AdminGroup } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const saving = ref(false)
const groups = ref<AdminGroup[]>([])
const form = reactive<ReferralSettings>({
  enabled: false,
  referrer_balance_reward: 0,
  referrer_group_id: 0,
  referrer_subscription_days: 0,
  referee_balance_reward: 0,
  referee_group_id: 0,
  referee_subscription_days: 0,
  max_per_user: 0
})

onMounted(async () => {
  try {
    const [settings, groupList] = await Promise.all([
      getReferralSettings(),
      getAllGroups()
    ])
    Object.assign(form, settings)
    groups.value = groupList
  } catch (error: any) {
    appStore.showError(t('admin.referral.loadFailed'))
  } finally {
    loading.value = false
  }
})

async function saveSettings() {
  saving.value = true
  try {
    await updateReferralSettings({ ...form })
    appStore.showSuccess(t('admin.referral.saved'))
  } catch (error: any) {
    appStore.showError(t('admin.referral.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>