<template>
  <AppLayout>
    <div class="payment-page-layout">
      <div class="card flex-1 min-h-0 overflow-hidden">
        <div v-if="!paymentAdminUrl" class="flex h-full items-center justify-center p-10 text-center">
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="creditCard" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('purchase.notConfiguredTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('purchase.notConfiguredDesc') }}
            </p>
          </div>
        </div>

        <div v-else class="payment-embed-shell">
          <a
            :href="paymentAdminUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm payment-open-fab"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" :stroke-width="2" />
            {{ t('purchase.openInNewTab') }}
          </a>
          <iframe
            :src="paymentAdminUrl"
            class="payment-embed-frame"
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
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'

const { t, locale } = useI18n()
const authStore = useAuthStore()

const paymentTheme = ref<'light' | 'dark'>('light')
let themeObserver: MutationObserver | null = null

const paymentAdminUrl = computed(() => {
  if (!authStore.token) return ''
  return buildEmbeddedUrl('/pay/admin', undefined, authStore.token, paymentTheme.value, locale.value)
})

onMounted(() => {
  paymentTheme.value = detectTheme()

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      paymentTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }
})

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.payment-page-layout {
  @apply flex flex-col;
  height: calc(100vh - 64px - 4rem);
}

.payment-embed-shell {
  @apply relative;
  @apply h-full w-full overflow-hidden rounded-2xl;
  @apply bg-gradient-to-b from-gray-50 to-white dark:from-dark-900 dark:to-dark-950;
  @apply p-0;
}

.payment-open-fab {
  @apply absolute right-3 top-3 z-10;
  @apply shadow-sm backdrop-blur supports-[backdrop-filter]:bg-white/80 dark:supports-[backdrop-filter]:bg-dark-800/80;
}

.payment-embed-frame {
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
