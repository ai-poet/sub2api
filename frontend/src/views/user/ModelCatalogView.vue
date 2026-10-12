<template>
  <AppLayout v-if="isAuthenticated">
    <ModelCatalogBody
      :items="allItems"
      :loading="loading"
      :refreshing="refreshing"
      :load-error="loadError"
      :last-updated-at="lastUpdatedAt"
      :balance-credit-cny-per-usd="balanceCreditCnyPerUsd"
      :usd-exchange-rate="usdExchangeRate"
      :payment-config-notice="paymentConfigNotice"
      :availability="availability"
      :initial-search="initialSearch"
      @refresh="refreshCatalog"
    />
  </AppLayout>

  <PublicCatalogShell v-else>
    <ModelCatalogBody
      public-mode
      :items="allItems"
      :loading="loading"
      :refreshing="refreshing"
      :load-error="loadError"
      :last-updated-at="lastUpdatedAt"
      :balance-credit-cny-per-usd="balanceCreditCnyPerUsd"
      :usd-exchange-rate="usdExchangeRate"
      :initial-search="initialSearch"
      @refresh="refreshCatalog"
    />
  </PublicCatalogShell>
</template>

<script setup lang="ts">
// fork：模型广场。登录用户走 /models/catalog（含专属倍率、其他分组、可用率条）；
// 未登录访客在公开定价开启时走 /pricing/public 的匿名视图（见 router/publicCatalog.ts）。
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import ModelCatalogBody from '@/components/modelCatalog/ModelCatalogBody.vue'
import PublicCatalogShell from '@/components/modelCatalog/PublicCatalogShell.vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import {
  fetchBalanceCreditCnyPerUsd,
  mapPublicPricingToCatalogItems,
  modelCatalogAPI,
  type ModelCatalogItem,
} from '@/api/modelCatalog'
import { fetchPublicCurrencyRates, getPublicPricing } from '@/api/pricing'
import { getHistories, type GroupStatusHistoryOverview } from '@/api/groupStatus'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const route = useRoute()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const initialSearch = computed(() => (typeof route?.query?.q === 'string' ? route.query.q : ''))

const allItems = ref<ModelCatalogItem[]>([])
const loading = ref(true)
const refreshing = ref(false)
const loadError = ref('')
const balanceCreditCnyPerUsd = ref<number | null>(null)
const usdExchangeRate = ref<number | null>(null)
const paymentConfigError = ref<string | null>(null)
const lastUpdatedAt = ref<Date | null>(null)
const availability = ref<Record<number, GroupStatusHistoryOverview>>({})

const paymentConfigNotice = computed(() => {
  if (loading.value || refreshing.value) {
    return ''
  }
  if (!paymentConfigError.value) {
    return ''
  }
  return t('modelCatalog.paymentNoticeDescription')
})

onMounted(async () => {
  await loadCatalog()
})

async function loadCatalog(options: { refresh?: boolean } = {}) {
  if (options.refresh) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  loadError.value = ''

  try {
    if (!appStore.publicSettingsLoaded) {
      await appStore.fetchPublicSettings()
    }

    if (isAuthenticated.value) {
      const response = await modelCatalogAPI.getCatalog()
      allItems.value = response.items || []
      lastUpdatedAt.value = new Date()
      void loadAvailability()
      await loadPaymentConfig()
    } else {
      await loadPublicCatalog()
    }
  } catch (error) {
    console.error('Failed to load model catalog:', error)
    loadError.value = t('modelCatalog.loadFailedDescription')
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

async function loadPublicCatalog() {
  const response = await getPublicPricing()
  if (!response.enabled) {
    allItems.value = []
    loadError.value = t('modelCatalog.public.disabled')
    return
  }
  allItems.value = mapPublicPricingToCatalogItems(response.items || [])
  lastUpdatedAt.value = new Date()
  const rates = await fetchPublicCurrencyRates({
    purchaseSubscriptionUrl: appStore.cachedPublicSettings?.purchase_subscription_url,
  })
  balanceCreditCnyPerUsd.value = rates.balanceCreditCnyPerUsd
  usdExchangeRate.value = rates.usdExchangeRate
}

// 可用率条是锦上添花：只在开启运行状态时请求一次，失败就不显示，不影响列表。
async function loadAvailability() {
  if (appStore.cachedPublicSettings?.group_status_enabled !== true) {
    availability.value = {}
    return
  }
  try {
    const histories = await getHistories('24h')
    availability.value = Object.fromEntries(histories.map(history => [history.group_id, history]))
  } catch {
    availability.value = {}
  }
}

async function loadPaymentConfig() {
  const result = await fetchBalanceCreditCnyPerUsd({
    purchaseSubscriptionUrl: appStore.cachedPublicSettings?.purchase_subscription_url,
    userId: authStore.user?.id,
    token: authStore.token,
    locale: locale.value,
  })

  balanceCreditCnyPerUsd.value = result.balanceCreditCnyPerUsd
  usdExchangeRate.value = result.usdExchangeRate
  paymentConfigError.value = result.error
}

async function refreshCatalog() {
  await loadCatalog({ refresh: true })
}
</script>
