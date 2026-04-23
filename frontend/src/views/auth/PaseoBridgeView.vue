<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Connecting Paseo</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ statusMessage }}
        </p>
      </div>

      <transition name="fade">
        <div
          v-if="errorMessage"
          class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <p class="text-sm text-red-700 dark:text-red-400">
            {{ errorMessage }}
          </p>
        </div>
      </transition>

      <div
        v-if="callbackUrl"
        class="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-300"
      >
        <p>If Paseo did not open automatically, continue manually:</p>
        <a
          :href="callbackUrl"
          class="mt-3 inline-flex rounded-lg bg-gray-900 px-4 py-2 font-medium text-white dark:bg-white dark:text-gray-900"
        >
          Open Paseo
        </a>
      </div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AuthLayout } from '@/components/layout'
import { keysAPI } from '@/api'
import { getAuthToken, getRefreshToken, getTokenExpiresAt } from '@/api/auth'
import { rememberOAuthReturnPath } from '@/utils/auth-redirect'
import { buildPaseoCallbackUrl, normalizePaseoEndpoint } from './paseo-bridge'

const route = useRoute()
const router = useRouter()

const statusMessage = ref('Preparing your Sub2API session for Paseo...')
const errorMessage = ref('')
const callbackUrl = ref('')

const endpoint = computed(() => {
  const routeEndpoint = typeof route.query.endpoint === 'string' ? route.query.endpoint : ''
  const fallbackOrigin = typeof window !== 'undefined' ? window.location.origin : ''
  return normalizePaseoEndpoint(routeEndpoint || fallbackOrigin)
})

async function ensureApiKey(): Promise<string> {
  const existing = await keysAPI.list(1, 1)
  const currentKey = existing.items?.[0]?.key?.trim()
  if (currentKey) {
    return currentKey
  }

  const created = await keysAPI.create('Paseo Desktop')
  return created.key.trim()
}

onMounted(async () => {
  try {
    const accessToken = getAuthToken()
    if (!accessToken) {
      rememberOAuthReturnPath(route.fullPath)
      await router.replace({
        path: '/login',
        query: { redirect: route.fullPath },
      })
      return
    }

    statusMessage.value = 'Creating or loading your API key...'
    const apiKey = await ensureApiKey()

    const nextAccessToken = getAuthToken()
    const refreshToken = getRefreshToken()
    const expiresAt = getTokenExpiresAt()

    if (
      !nextAccessToken ||
      !refreshToken ||
      expiresAt === null ||
      !Number.isFinite(expiresAt) ||
      !apiKey
    ) {
      throw new Error('Missing token or API key state after browser login.')
    }

    callbackUrl.value = buildPaseoCallbackUrl({
      accessToken: nextAccessToken,
      refreshToken,
      expiresAt,
      apiKey,
      endpoint: endpoint.value,
    })

    statusMessage.value = 'Opening Paseo...'
    window.location.href = callbackUrl.value
  } catch (error: unknown) {
    errorMessage.value =
      error instanceof Error ? error.message : 'Unable to complete the Paseo login flow.'
    statusMessage.value = 'Unable to continue automatically.'
  }
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
