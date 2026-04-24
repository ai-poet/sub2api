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
import { keysAPI, userGroupsAPI } from '@/api'
import { getAuthToken, getRefreshToken, getTokenExpiresAt } from '@/api/auth'
import { clearStoredOAuthReturnPath, rememberOAuthReturnPath } from '@/utils/auth-redirect'
import { buildPaseoCallbackUrl, normalizePaseoEndpoint } from './paseo-bridge'
import type { ApiKey, Group, GroupPlatform } from '@/types'

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

type PaseoScopedPlatform = Extract<GroupPlatform, 'anthropic' | 'openai'>

interface PaseoPreparedKeys {
  apiKey: string
  claudeApiKey: string | null
  codexApiKey: string | null
}

function isRouteReadyApiKey(apiKey: ApiKey, platform: PaseoScopedPlatform): boolean {
  return (
    apiKey.status === 'active' &&
    apiKey.group?.status === 'active' &&
    apiKey.group.platform === platform
  )
}

function pickExistingRouteKey(apiKeys: ApiKey[], platform: PaseoScopedPlatform): ApiKey | null {
  return apiKeys.find((apiKey) => isRouteReadyApiKey(apiKey, platform)) ?? null
}

function pickCreatableGroup(groups: Group[], platform: PaseoScopedPlatform): Group | null {
  return groups.find((group) => group.status === 'active' && group.platform === platform) ?? null
}

async function loadAllApiKeys(): Promise<ApiKey[]> {
  const pageSize = 100
  let page = 1
  let pages = 1
  const items: ApiKey[] = []

  while (page <= pages) {
    const response = await keysAPI.list(page, pageSize)
    items.push(...response.items)
    pages = response.pages || 1
    page += 1
  }

  return items
}

function pickPrimaryApiKey(
  apiKeys: ApiKey[],
  scoped: { claude: ApiKey | null; codex: ApiKey | null }
): string | null {
  const fallback =
    scoped.claude?.key?.trim() ||
    scoped.codex?.key?.trim() ||
    apiKeys.find((apiKey) => apiKey.status === 'active')?.key?.trim() ||
    apiKeys[0]?.key?.trim() ||
    null
  return fallback && fallback.length > 0 ? fallback : null
}

async function ensurePaseoKeys(): Promise<PaseoPreparedKeys> {
  const apiKeys = await loadAllApiKeys()
  let claudeKey = pickExistingRouteKey(apiKeys, 'anthropic')
  let codexKey = pickExistingRouteKey(apiKeys, 'openai')

  if (!claudeKey || !codexKey) {
    const availableGroups = await userGroupsAPI.getAvailable()

    if (!claudeKey) {
      const group = pickCreatableGroup(availableGroups, 'anthropic')
      if (group) {
        claudeKey = await keysAPI.create('Paseo Desktop (Claude Code)', group.id)
      }
    }

    if (!codexKey) {
      const group = pickCreatableGroup(availableGroups, 'openai')
      if (group) {
        codexKey = await keysAPI.create('Paseo Desktop (Codex)', group.id)
      }
    }
  }

  let primaryApiKey = pickPrimaryApiKey(apiKeys, {
    claude: claudeKey,
    codex: codexKey
  })
  if (!primaryApiKey) {
    const created = await keysAPI.create('Paseo Desktop')
    primaryApiKey = created.key.trim()
  }

  return {
    apiKey: primaryApiKey,
    claudeApiKey: claudeKey?.key?.trim() || null,
    codexApiKey: codexKey?.key?.trim() || null
  }
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

    statusMessage.value = 'Preparing Claude Code and Codex routes...'
    const preparedKeys = await ensurePaseoKeys()

    const nextAccessToken = getAuthToken()
    const refreshToken = getRefreshToken()
    const expiresAt = getTokenExpiresAt()

    if (
      !nextAccessToken ||
      !refreshToken ||
      expiresAt === null ||
      !Number.isFinite(expiresAt) ||
      !preparedKeys.apiKey
    ) {
      throw new Error('Missing token or API key state after browser login.')
    }

    callbackUrl.value = buildPaseoCallbackUrl({
      accessToken: nextAccessToken,
      refreshToken,
      expiresAt,
      apiKey: preparedKeys.apiKey,
      claudeApiKey: preparedKeys.claudeApiKey,
      codexApiKey: preparedKeys.codexApiKey,
      endpoint: endpoint.value,
    })

    statusMessage.value = 'Opening Paseo...'
    clearStoredOAuthReturnPath()
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
