<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('shop.title') }}</h1>
      </div>

      <div v-if="loading" class="flex justify-center py-12">
        <Icon name="refresh" size="xl" class="animate-spin text-primary-500" />
      </div>

      <div v-else-if="products.length === 0" class="card p-12 text-center text-gray-500">
        {{ t('shop.noProducts') }}
      </div>

      <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div
          v-for="product in products"
          :key="product.id"
          class="card flex flex-col p-6"
        >
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ product.name }}</h3>
          <p v-if="product.description" class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ product.description }}
          </p>
          <div class="mt-4 flex items-end justify-between">
            <div>
              <span class="text-2xl font-bold text-primary-600">{{ product.price.toFixed(2) }}</span>
              <span class="ml-1 text-sm text-gray-500">{{ product.currency }}</span>
            </div>
            <span
              :class="product.stock_count > 0 ? 'text-green-600' : 'text-red-500'"
              class="text-sm"
            >
              {{ product.stock_count > 0 ? t('shop.inStock', { n: product.stock_count }) : t('shop.outOfStock') }}
            </span>
          </div>
          <button
            class="btn btn-primary mt-4"
            :disabled="product.stock_count <= 0 || ordering === product.id"
            @click="openPayDialog(product)"
          >
            {{ ordering === product.id ? t('common.loading') : t('shop.buy') }}
          </button>
        </div>
      </div>

      <!-- Payment method dialog -->
      <div v-if="selectedProduct" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="card w-full max-w-sm p-6">
          <h3 class="mb-4 text-lg font-semibold">{{ t('shop.selectPayment') }}</h3>
          <div v-if="paymentChannels.length === 0" class="text-center text-gray-500 py-4">
            {{ t('shop.noPaymentMethod') }}
          </div>
          <div v-else class="space-y-2">
            <button
              v-for="channel in paymentChannels"
              :key="channel.id"
              class="btn btn-secondary w-full"
              @click="createOrder(channel.id)"
            >
              {{ channel.name }}
            </button>
          </div>
          <button class="btn btn-ghost mt-3 w-full" @click="selectedProduct = null">
            {{ t('common.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { shopAPI, type ShopProduct, type PaymentChannel } from '@/api/shop'
import { useToast } from '@/composables/useToast'

const PENDING_ORDER_KEY = 'shop_pending_order'
const PENDING_ORDER_POLL_INTERVAL = 5000

const { t } = useI18n()
const { showToast } = useToast()

const products = ref<ShopProduct[]>([])
const paymentChannels = ref<PaymentChannel[]>([])
const loading = ref(false)
const ordering = ref<number | null>(null)
const selectedProduct = ref<ShopProduct | null>(null)
const checkingOrder = ref(false)
let pendingOrderTimer: number | null = null

interface PendingOrder {
  orderNo: string
  productId: number
  productName: string
  createdAt: number
}

function savePendingOrder(orderNo: string, productId: number, productName: string) {
  const pending: PendingOrder = {
    orderNo,
    productId,
    productName,
    createdAt: Date.now(),
  }
  localStorage.setItem(PENDING_ORDER_KEY, JSON.stringify(pending))
}

function getPendingOrder(): PendingOrder | null {
  try {
    const data = localStorage.getItem(PENDING_ORDER_KEY)
    if (!data) return null
    const pending = JSON.parse(data) as PendingOrder
    // 如果订单超过1小时，清除
    if (Date.now() - pending.createdAt > 60 * 60 * 1000) {
      localStorage.removeItem(PENDING_ORDER_KEY)
      return null
    }
    return pending
  } catch {
    return null
  }
}

function clearPendingOrder() {
  localStorage.removeItem(PENDING_ORDER_KEY)
}

function stopPendingOrderPolling() {
  if (pendingOrderTimer !== null) {
    window.clearInterval(pendingOrderTimer)
    pendingOrderTimer = null
  }
}

async function checkPendingOrder(showPendingToast = true): Promise<boolean> {
  const pending = getPendingOrder()
  if (!pending) return false

  checkingOrder.value = true
  try {
    const order = await shopAPI.queryOrder(pending.orderNo)
    if (order.status === 'paid') {
      showToast(t('shop.paymentSuccess', { product: pending.productName }), 'success')
      clearPendingOrder()
      return false
    } else if (order.status === 'expired' || order.status === 'cancelled') {
      showToast(t('shop.paymentExpired'), 'warning')
      clearPendingOrder()
      return false
    } else {
      if (showPendingToast) {
        showToast(t('shop.paymentPending'), 'info')
      }
      return true
    }
  } catch {
    // 查询失败，保留订单状态并继续轮询
    return true
  } finally {
    checkingOrder.value = false
  }
}

function startPendingOrderPolling() {
  stopPendingOrderPolling()
  if (!getPendingOrder()) return

  checkPendingOrder(true).then((stillPending) => {
    if (!stillPending) return
    pendingOrderTimer = window.setInterval(async () => {
      const activePending = getPendingOrder()
      if (!activePending) {
        stopPendingOrderPolling()
        return
      }
      const keepPolling = await checkPendingOrder(false)
      if (!keepPolling) {
        stopPendingOrderPolling()
      }
    }, PENDING_ORDER_POLL_INTERVAL)
  })
}

async function loadProducts() {
  loading.value = true
  try {
    products.value = await shopAPI.getProducts()
  } catch {
    showToast(t('common.unknownError'), 'error')
  } finally {
    loading.value = false
  }
}

async function loadPaymentChannels() {
  try {
    paymentChannels.value = await shopAPI.getChannels()
  } catch {
    // 如果加载失败，使用默认支付方式
    paymentChannels.value = [
      { id: 'alipay', name: t('shop.alipay'), icon: 'wallet', provider: 'epay', fee: 0 },
      { id: 'wxpay', name: t('shop.wxpay'), icon: 'credit-card', provider: 'epay', fee: 0 },
    ]
  }
}

function openPayDialog(product: ShopProduct) {
  selectedProduct.value = product
}

async function createOrder(paymentMethod: string) {
  if (!selectedProduct.value) return
  const product = selectedProduct.value
  selectedProduct.value = null
  ordering.value = product.id
  try {
    const result = await shopAPI.createOrder(product.id, paymentMethod)
    // 保存订单状态以便支付后恢复
    savePendingOrder(result.order.order_no, product.id, product.name)
    window.location.href = result.pay_url
  } catch (e: any) {
    showToast(e?.message || t('common.unknownError'), 'error')
  } finally {
    ordering.value = null
  }
}

onMounted(() => {
  loadProducts()
  loadPaymentChannels()
  startPendingOrderPolling()
})

onUnmounted(() => {
  stopPendingOrderPolling()
})
</script>
