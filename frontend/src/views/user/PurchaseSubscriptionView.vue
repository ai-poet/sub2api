<template>
  <AppLayout>
    <div class="purchase-page-layout">
      <div class="card flex-1 min-h-0 overflow-hidden">
        <div v-if="loading" class="flex h-full items-center justify-center py-12">
          <div
            class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else-if="!purchaseEnabled"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="creditCard" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('purchase.notEnabledTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('purchase.notEnabledDesc') }}
            </p>
          </div>
        </div>

        <div
          v-else-if="!isValidUrl"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('purchase.notConfiguredTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('purchase.notConfiguredDesc') }}
            </p>
          </div>
        </div>

        <div v-else class="purchase-embed-shell">
          <a
            :href="purchaseUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm purchase-open-fab"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" :stroke-width="2" />
            {{ t('purchase.openInNewTab') }}
          </a>
          <iframe
            ref="purchaseFrame"
            :src="purchaseUrl"
            class="purchase-embed-frame"
            allowfullscreen
          ></iframe>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(false)
const purchaseTheme = ref<'light' | 'dark'>('light')
const purchaseFrame = ref<HTMLIFrameElement | null>(null)
let themeObserver: MutationObserver | null = null

// 支付页在 iframe 内完成到账，顶栏余额来自 authStore.user，不会自己更新。
// 支付页订单完成时会 postMessage 通知，这里收到后重新拉一次用户信息。
const PAYMENT_SUCCESS_MESSAGE_TYPE = 'SUB2API_PAYMENT_SUCCESS'

function onPurchaseMessage(event: MessageEvent) {
  // 只认当前 iframe 发来的消息：支付页可能部署在任意配置的域名下，
  // 用 source 比对比维护 origin 白名单更可靠，也挡掉了其他窗口的伪造消息。
  if (!purchaseFrame.value || event.source !== purchaseFrame.value.contentWindow) return
  const data = event.data as { type?: unknown } | null
  if (!data || typeof data !== 'object' || data.type !== PAYMENT_SUCCESS_MESSAGE_TYPE) return

  authStore.refreshUser().catch(() => {
    // 刷新失败不打扰用户：余额会在下一次常规刷新时对齐。
  })
}

// 「新窗口打开」的支付页没有宿主可通知，用户付完切回本页时补一次刷新。
function onVisibilityChange() {
  if (document.visibilityState !== 'visible') return
  authStore.refreshUser().catch(() => {})
}

const purchaseEnabled = computed(() => {
  return appStore.cachedPublicSettings?.purchase_subscription_enabled ?? false
})

const purchaseUrl = computed(() => {
  if (!purchaseEnabled.value) return ''
  const configuredUrl = (appStore.cachedPublicSettings?.purchase_subscription_url || '').trim()
  const baseUrl = configuredUrl || '/pay'
  return buildEmbeddedUrl(baseUrl, authStore.user?.id, authStore.token, purchaseTheme.value, locale.value)
})

const isValidUrl = computed(() => {
  return purchaseUrl.value.length > 0
})

onMounted(async () => {
  purchaseTheme.value = detectTheme()

  if (typeof window !== 'undefined') {
    window.addEventListener('message', onPurchaseMessage)
  }

  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibilityChange)

    themeObserver = new MutationObserver(() => {
      purchaseTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }

  if (appStore.publicSettingsLoaded) return
  loading.value = true
  try {
    await appStore.fetchPublicSettings()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('message', onPurchaseMessage)
  }
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.purchase-page-layout {
  @apply flex flex-col;
  height: calc(100vh - 64px - 4rem);
}

.purchase-embed-shell {
  @apply relative;
  @apply h-full w-full overflow-hidden rounded-2xl;
  @apply bg-gradient-to-b from-gray-50 to-white dark:from-dark-900 dark:to-dark-950;
  @apply p-0;
}

.purchase-open-fab {
  @apply absolute right-3 top-3 z-10;
  @apply shadow-sm backdrop-blur supports-[backdrop-filter]:bg-white/80 dark:supports-[backdrop-filter]:bg-dark-800/80;
}

.purchase-embed-frame {
  display: block;
  margin: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}
</style>
