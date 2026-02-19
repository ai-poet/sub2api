<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white flex-1">{{ t('admin.shop.title') }}</h2>
          <div class="flex gap-2">
            <button @click="loadProducts" :disabled="loading" class="btn btn-secondary">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="showCreateDialog = true" class="btn btn-primary">
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('admin.shop.createProduct') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="products" :loading="loading">
          <template #cell-is_active="{ value }">
            <span :class="value ? 'badge-success' : 'badge-gray'" class="badge">
              {{ value ? t('common.active') : t('common.inactive') }}
            </span>
          </template>
          <template #cell-stock_count="{ row }">
            <div class="flex items-center gap-2">
              <span>{{ row.stock_count }}</span>
              <button @click="openStockDialog(row)" class="btn btn-xs btn-secondary">
                {{ t('admin.shop.manageStock') }}
              </button>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex gap-2">
              <button @click="openEditDialog(row)" class="btn btn-xs btn-secondary">
                {{ t('common.edit') }}
              </button>
              <button @click="deleteProduct(row.id)" class="btn btn-xs btn-danger">
                {{ t('common.delete') }}
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <!-- Create/Edit Product Dialog -->
    <div v-if="showCreateDialog || editingProduct" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="card w-full max-w-md p-6">
        <h3 class="mb-4 text-lg font-semibold">
          {{ editingProduct ? t('admin.shop.editProduct') : t('admin.shop.createProduct') }}
        </h3>
        <form @submit.prevent="saveProduct" class="space-y-4">
          <div>
            <label class="input-label">{{ t('admin.shop.productName') }}</label>
            <input v-model="form.name" type="text" required class="input mt-1" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.shop.description') }}</label>
            <textarea v-model="form.description" class="input mt-1" rows="2" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="input-label">{{ t('admin.shop.price') }}</label>
              <input v-model.number="form.price" type="number" step="0.01" min="0" required class="input mt-1" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.shop.currency') }}</label>
              <input v-model="form.currency" type="text" class="input mt-1" placeholder="CNY" />
            </div>
          </div>
          <div>
            <label class="input-label">{{ t('admin.shop.redeemType') }}</label>
            <select v-model="form.redeem_type" required class="input mt-1">
              <option value="balance">{{ t('admin.shop.typeBalance') }}</option>
              <option value="subscription">{{ t('admin.shop.typeSubscription') }}</option>
            </select>
          </div>
          <div v-if="form.redeem_type === 'balance'">
            <label class="input-label">{{ t('admin.shop.redeemValue') }}</label>
            <input v-model.number="form.redeem_value" type="number" step="0.01" min="0" class="input mt-1" />
          </div>
          <div v-if="form.redeem_type === 'subscription'" class="grid grid-cols-2 gap-3">
            <div>
              <label class="input-label">{{ t('admin.shop.groupId') }}</label>
              <input v-model.number="form.group_id" type="number" class="input mt-1" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.shop.validityDays') }}</label>
              <input v-model.number="form.validity_days" type="number" min="1" class="input mt-1" />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="input-label">{{ t('admin.shop.sortOrder') }}</label>
              <input v-model.number="form.sort_order" type="number" class="input mt-1" />
            </div>
            <div class="flex items-center gap-2 pt-6">
              <input v-model="form.is_active" type="checkbox" id="is_active" class="h-4 w-4" />
              <label for="is_active" class="text-sm">{{ t('common.active') }}</label>
            </div>
          </div>
          <div>
            <label class="input-label">{{ t('admin.shop.creemProductId') }}</label>
            <input v-model="form.creem_product_id" type="text" class="input mt-1 font-mono text-sm" placeholder="prod_xxx" />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.shop.creemProductIdHint') }}</p>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <button type="button" class="btn btn-secondary" @click="closeProductDialog">{{ t('common.cancel') }}</button>
            <button type="submit" class="btn btn-primary" :disabled="saving">{{ t('common.save') }}</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Stock Management Dialog -->
    <div v-if="stockProduct" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="card w-full max-w-lg p-6">
        <h3 class="mb-4 text-lg font-semibold">
          {{ t('admin.shop.stockFor', { name: stockProduct.name }) }}
        </h3>
        <div class="mb-4 flex gap-2">
          <input v-model.number="addStockCount" type="number" min="1" class="input flex-1" :placeholder="t('admin.shop.stockCount')" />
          <button class="btn btn-primary" :disabled="addingStock" @click="addStock">
            {{ t('admin.shop.addStock') }}
          </button>
        </div>
        <div class="max-h-64 overflow-y-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b text-left text-gray-500">
                <th class="pb-2">ID</th>
                <th class="pb-2">{{ t('admin.shop.stockStatus') }}</th>
                <th class="pb-2">{{ t('common.createdAt') }}</th>
                <th class="pb-2"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in stockList" :key="s.id" class="border-b">
                <td class="py-1">{{ s.id }}</td>
                <td class="py-1">
                  <span :class="s.status === 'available' ? 'text-green-600' : 'text-gray-400'">{{ s.status }}</span>
                </td>
                <td class="py-1 text-gray-500">{{ formatDate(s.created_at) }}</td>
                <td class="py-1">
                  <button v-if="s.status === 'available'" @click="deleteStock(s.id)" class="text-red-500 hover:text-red-700">
                    <Icon name="trash" size="sm" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="mt-4 flex justify-end">
          <button class="btn btn-secondary" @click="stockProduct = null">{{ t('common.close') }}</button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminShopAPI, type AdminShopProduct, type ShopProductStock } from '@/api/admin/shop'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const { showToast } = useToast()

const products = ref<AdminShopProduct[]>([])
const loading = ref(false)
const saving = ref(false)
const showCreateDialog = ref(false)
const editingProduct = ref<AdminShopProduct | null>(null)
const stockProduct = ref<AdminShopProduct | null>(null)
const stockList = ref<ShopProductStock[]>([])
const addStockCount = ref(10)
const addingStock = ref(false)

const columns = [
  { key: 'id', label: 'ID' },
  { key: 'name', label: t('admin.shop.productName') },
  { key: 'price', label: t('admin.shop.price') },
  { key: 'redeem_type', label: t('admin.shop.redeemType') },
  { key: 'stock_count', label: t('admin.shop.stock') },
  { key: 'is_active', label: t('common.status') },
  { key: 'actions', label: t('common.actions') },
]

const defaultForm = () => ({
  name: '',
  description: '',
  price: 0,
  currency: 'CNY',
  redeem_type: 'balance',
  redeem_value: 0,
  group_id: undefined as number | undefined,
  validity_days: 30,
  is_active: true,
  sort_order: 0,
  creem_product_id: '',
})

const form = ref(defaultForm())

async function loadProducts() {
  loading.value = true
  try {
    products.value = await adminShopAPI.listProducts()
  } catch {
    showToast(t('common.unknownError'), 'error')
  } finally {
    loading.value = false
  }
}

function openEditDialog(product: AdminShopProduct) {
  editingProduct.value = product
  form.value = {
    name: product.name,
    description: product.description || '',
    price: product.price,
    currency: product.currency,
    redeem_type: product.redeem_type,
    redeem_value: product.redeem_value,
    group_id: product.group_id,
    validity_days: product.validity_days,
    is_active: product.is_active,
    sort_order: product.sort_order,
    creem_product_id: product.creem_product_id || '',
  }
}

function closeProductDialog() {
  showCreateDialog.value = false
  editingProduct.value = null
  form.value = defaultForm()
}

async function saveProduct() {
  saving.value = true
  try {
    if (editingProduct.value) {
      await adminShopAPI.updateProduct(editingProduct.value.id, form.value)
    } else {
      await adminShopAPI.createProduct(form.value)
    }
    showToast(t('common.success'), 'success')
    closeProductDialog()
    await loadProducts()
  } catch (e: any) {
    showToast(e?.message || t('common.unknownError'), 'error')
  } finally {
    saving.value = false
  }
}

async function deleteProduct(id: number) {
  if (!confirm(t('common.confirm'))) return
  try {
    await adminShopAPI.deleteProduct(id)
    showToast(t('common.success'), 'success')
    await loadProducts()
  } catch (e: any) {
    showToast(e?.message || t('common.unknownError'), 'error')
  }
}

async function openStockDialog(product: AdminShopProduct) {
  stockProduct.value = product
  await loadStockList(product.id)
}

async function loadStockList(productId: number) {
  stockList.value = await adminShopAPI.getStockList(productId)
}

async function addStock() {
  if (!stockProduct.value) return
  addingStock.value = true
  try {
    const result = await adminShopAPI.addStock(stockProduct.value.id, addStockCount.value)
    showToast(t('admin.shop.stockAdded', { n: result.added }), 'success')
    await loadStockList(stockProduct.value.id)
    await loadProducts()
  } catch (e: any) {
    showToast(e?.message || t('common.unknownError'), 'error')
  } finally {
    addingStock.value = false
  }
}

async function deleteStock(stockId: number) {
  try {
    await adminShopAPI.deleteStock(stockId)
    if (stockProduct.value) await loadStockList(stockProduct.value.id)
    await loadProducts()
  } catch (e: any) {
    showToast(e?.message || t('common.unknownError'), 'error')
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString()
}

onMounted(loadProducts)
</script>
