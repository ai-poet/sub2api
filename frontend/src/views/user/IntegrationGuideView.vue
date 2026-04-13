<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] space-y-6">
      <section class="rounded-3xl border border-gray-200/80 bg-white/90 p-5 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80">
        <div class="space-y-4">
          <div class="inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-950/30 dark:text-primary-300">
            <Icon name="book" size="sm" />
            {{ t('integrationGuide.caption') }}
          </div>

          <div class="max-w-4xl space-y-3">
            <p class="text-sm leading-6 text-gray-600 dark:text-gray-300">
              {{ t('integrationGuide.intro') }}
            </p>
            <div class="rounded-2xl border border-dashed border-amber-300 bg-amber-50/80 px-4 py-3 text-sm text-amber-800 dark:border-amber-700/50 dark:bg-amber-950/20 dark:text-amber-200">
              {{ t('integrationGuide.bindingNote') }}
            </div>
          </div>

          <EndpointPopover
            v-if="publicSettings?.api_base_url || (publicSettings?.custom_endpoints?.length ?? 0) > 0"
            :api-base-url="publicSettings?.api_base_url || ''"
            :custom-endpoints="publicSettings?.custom_endpoints || []"
          />
        </div>
      </section>

      <div v-if="loading" class="flex items-center justify-center py-16">
        <LoadingSpinner />
      </div>

      <section v-else class="grid gap-6 2xl:grid-cols-2">
        <IntegrationGuidePanel
          v-for="platform in platforms"
          :key="platform"
          :platform="platform"
          :api-keys="getKeysForPlatform(platform)"
          :base-url="baseUrl"
        />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { authAPI, keysAPI } from '@/api'
import type { ApiKey, GroupPlatform, PublicSettings } from '@/types'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EndpointPopover from '@/components/keys/EndpointPopover.vue'
import IntegrationGuidePanel from '@/components/keys/IntegrationGuidePanel.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const apiKeys = ref<ApiKey[]>([])
const publicSettings = ref<PublicSettings | null>(null)
const platforms: GroupPlatform[] = ['anthropic', 'openai']

const baseUrl = computed(() => publicSettings.value?.api_base_url || window.location.origin)

function isAvailableApiKey(apiKey: ApiKey, platform: GroupPlatform) {
  return apiKey.status === 'active' && apiKey.group?.platform === platform
}

function getKeysForPlatform(platform: GroupPlatform) {
  return apiKeys.value.filter((apiKey) => isAvailableApiKey(apiKey, platform))
}

async function loadAllApiKeys() {
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

  apiKeys.value = items
}

async function loadData() {
  loading.value = true

  const [keysResult, settingsResult] = await Promise.allSettled([
    loadAllApiKeys(),
    authAPI.getPublicSettings()
  ])

  if (keysResult.status === 'rejected') {
    console.error('Failed to load API keys for integration guide:', keysResult.reason)
    appStore.showError(t('integrationGuide.failedToLoadKeys'))
    apiKeys.value = []
  }

  if (settingsResult.status === 'fulfilled') {
    publicSettings.value = settingsResult.value
  } else {
    console.error('Failed to load public settings for integration guide:', settingsResult.reason)
    appStore.showError(t('integrationGuide.failedToLoadSettings'))
    publicSettings.value = null
  }

  loading.value = false
}

onMounted(() => {
  loadData()
})
</script>
