<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
        {{ t('orderHistory.title') }}
      </h1>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('orderHistory.subtitle') }}
      </p>
    </div>

    <!-- 筛选器 -->
    <div class="mb-4 flex items-center gap-4">
      <select
        v-model="statusFilter"
        class="rounded-md border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white"
        @change="loadOrders"
      >
        <option value="">{{ t('orderHistory.allStatus') }}</option>
        <option value="pending">{{ t('orderHistory.statusPending') }}</option>
        <option value="paid">{{ t('orderHistory.statusPaid') }}</option>
        <option value="expired">{{ t('orderHistory.statusExpired') }}</option>
        <option value="cancelled">{{ t('orderHistory.statusCancelled') }}</option>
      </select>

      <button
        class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        @click="loadOrders"
      >
        {{ t('common.refresh') }}
      </button>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-blue-600 border-t-transparent"></div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="orders.length === 0" class="rounded-lg bg-gray-50 py-12 text-center dark:bg-gray-800">
      <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('orderHistory.noOrders') }}</p>
    </div>

    <!-- 订单列表 -->
    <div v-else class="space-y-4">
      <div
        v-for="order in orders"
        :key="order.id"
        class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
      >
        <div class="flex items-start justify-between">
          <div class="flex-1">
            <div class="flex items-center gap-2">
              <h3 class="font-medium text-gray-900 dark:text-white">{{ order.product_name }}</h3>
              <span
                :class="[
                  'rounded-full px-2 py-0.5 text-xs font-medium',
                  getStatusClass(order.status)
                ]"
              >
                {{ getStatusText(order.status) }}
              </span>
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('orderHistory.orderNo') }}: {{ order.order_no }}
            </p>
            <div class="mt-2 flex items-center gap-4 text-sm">
              <span class="font-medium text-gray-900 dark:text-white">
                {{ order.amount }} {{ order.currency }}
              </span>
              <span v-if="order.payment_method" class="text-gray-500 dark:text-gray-400">
                {{ order.payment_method }}
              </span>
            </div>
            <div class="mt-2 text-xs text-gray-400 dark:text-gray-500">
              {{ t('orderHistory.createdAt') }}: {{ formatDate(order.created_at) }}
              <span v-if="order.paid_at" class="ml-2">
                | {{ t('orderHistory.paidAt') }}: {{ formatDate(order.paid_at) }}
              </span>
              <span v-if="order.expires_at && order.status === 'pending'" class="ml-2">
                | {{ t('orderHistory.expiresAt') }}: {{ formatDate(order.expires_at) }}
              </span>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button
              v-if="order.status === 'pending'"
              class="rounded-md bg-red-100 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-400 dark:hover:bg-red-900/50"
              @click="handleCancelOrder(order.order_no)"
            >
              {{ t('orderHistory.cancel') }}
            </button>
            <router-link
              v-if="order.status === 'pending'"
              :to="`/shop?order=${order.order_no}`"
              class="rounded-md bg-blue-100 px-3 py-1.5 text-sm font-medium text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:hover:bg-blue-900/50"
            >
              {{ t('orderHistory.continuePay') }}
            </router-link>
          </div>
        </div>
      </div>
    </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { shopAPI, type ShopOrder } from '@/api/shop'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const { showToast } = useToast()

const loading = ref(false)
const orders = ref<ShopOrder[]>([])
const statusFilter = ref('')

const loadOrders = async () => {
  loading.value = true
  try {
    orders.value = await shopAPI.getMyOrders(statusFilter.value || undefined)
  } catch {
    showToast(t('common.unknownError'), 'error')
  } finally {
    loading.value = false
  }
}

const handleCancelOrder = async (orderNo: string) => {
  if (!confirm(t('orderHistory.cancelConfirm'))) {
    return
  }
  try {
    await shopAPI.cancelOrder(orderNo)
    await loadOrders()
    showToast(t('common.success'), 'success')
  } catch {
    showToast(t('orderHistory.cancelFailed'), 'error')
  }
}

const getStatusClass = (status: string) => {
  switch (status) {
    case 'pending':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'paid':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'expired':
      return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
    case 'cancelled':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'pending':
      return t('orderHistory.statusPending')
    case 'paid':
      return t('orderHistory.statusPaid')
    case 'expired':
      return t('orderHistory.statusExpired')
    case 'cancelled':
      return t('orderHistory.statusCancelled')
    default:
      return status
  }
}

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  return date.toLocaleString()
}

onMounted(() => {
  loadOrders()
})
</script>
