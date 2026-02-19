<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('shop.title') }}</h1>
      </div>

      <div v-if="loading" class="flex justify-center py-12">
        <Icon name="loader" size="xl" class="animate-spin text-primary-500" />
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
          <div class="space-y-2">
            <button
              v-for="method in paymentMethods"
              :key="method.value"
              class="btn btn-secondary w-full"
              @click="createOrder(method.value)"
            >
              {{ method.label }}
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
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/ui/Icon.vue'
import { shopAPI, type ShopProduct } from '@/api/shop'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const { showToast } = useToast()

const products = ref<ShopProduct[]>([])
const loading = ref(false)
const ordering = ref<number | null>(null)
const selectedProduct = ref<ShopProduct | null>(null)

const paymentMethods = [
  { value: 'alipay', label: t('shop.alipay') },
  { value: 'wxpay', label: t('shop.wxpay') },
  { value: 'qqpay', label: t('shop.qqpay') },
  { value: 'usdt', label: 'USDT' },
]

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
    window.location.href = result.pay_url
  } catch (e: any) {
    showToast(e?.message || t('common.unknownError'), 'error')
  } finally {
    ordering.value = null
  }
}

onMounted(loadProducts)
</script>
