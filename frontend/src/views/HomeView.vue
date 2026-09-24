<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="home-font-sans relative min-h-screen overflow-x-hidden bg-[#f6f4ef] text-gray-900 dark:bg-[#0b0c0e] dark:text-gray-100"
  >
    <HomeHeader
      :site-name="siteName"
      :site-logo="siteLogo"
      :doc-url="docUrl"
      :is-dark="isDark"
      :is-authenticated="isAuthenticated"
      :dashboard-path="dashboardPath"
      :user-initial="userInitial"
      :show-changelog="hasClientDownloads"
      @toggle-theme="toggleTheme"
    />

    <main class="relative z-10">
      <HomeHero
        :site-name="siteName"
        :doc-url="docUrl"
        :is-authenticated="isAuthenticated"
        :dashboard-path="dashboardPath"
        :windows-url="clientDownloadWindowsUrl"
        :macos-url="clientDownloadMacOSUrl"
        :api-base-url="apiBaseUrl"
        :latest-version="latestVersion"
      />

      <HomeReveal>
        <HomeFeatureConstellation :mode="homeMode" :site-name="siteName" :site-logo="siteLogo" />
      </HomeReveal>

      <HomeModelOrbit :mode="homeMode" :site-name="siteName" :site-logo="siteLogo" />

      <HomeReveal id="pricing" class="scroll-mt-20 px-4 pt-20 md:px-6 md:pt-28">
        <HomePricingSection :site-name="siteName" />
      </HomeReveal>

      <HomeReveal class="px-4 py-16 md:px-6 md:py-24">
        <HomeComparisonSection :site-name="siteName" />
      </HomeReveal>

      <HomeClosingCta
        :site-name="siteName"
        :doc-url="docUrl"
        :is-authenticated="isAuthenticated"
        :dashboard-path="dashboardPath"
        :client-download-options="clientDownloadOptions"
      />
    </main>

    <HomeFooter
      :site-name="siteName"
      :site-logo="siteLogo"
      :doc-url="docUrl"
      :current-year="currentYear"
      :has-client-downloads="hasClientDownloads"
      :console-path="isAuthenticated ? dashboardPath : '/login'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import HomeClosingCta from '@/components/home/HomeClosingCta.vue'
import HomeComparisonSection from '@/components/home/HomeComparisonSection.vue'
import HomeFeatureConstellation from '@/components/home/HomeFeatureConstellation.vue'
import HomeFooter from '@/components/home/HomeFooter.vue'
import HomeHeader from '@/components/home/HomeHeader.vue'
import HomeHero from '@/components/home/HomeHero.vue'
import HomeModelOrbit from '@/components/home/HomeModelOrbit.vue'
import HomePricingSection from '@/components/home/HomePricingSection.vue'
import HomeReveal from '@/components/home/HomeReveal.vue'
import { useClientChangelog } from '@/composables/useClientChangelog'
import { detectPreferredClientPlatform, getClientDownloadOptions } from '@/utils/clientDownloads'

const authStore = useAuthStore()
const appStore = useAppStore()
let titleSyncTimer: number | undefined

const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name?.trim() || appStore.siteName.trim()
  return configuredName && configuredName !== 'Sub2API' ? configuredName : 'CheapRouter'
})
const siteLogo = computed(() => appStore.siteLogo || appStore.cachedPublicSettings?.site_logo || '')
const apiBaseUrl = computed(() => appStore.cachedPublicSettings?.api_base_url?.trim() || appStore.apiBaseUrl || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const clientDownloadWindowsUrl = computed(
  () => appStore.cachedPublicSettings?.client_download_windows_url?.trim() || ''
)
const clientDownloadMacOSUrl = computed(
  () => appStore.cachedPublicSettings?.client_download_macos_url?.trim() || ''
)
const clientDownloadOptions = computed(() =>
  getClientDownloadOptions(
    { windowsUrl: clientDownloadWindowsUrl.value, macosUrl: clientDownloadMacOSUrl.value },
    detectPreferredClientPlatform(),
  ),
)
const hasClientDownloads = computed(() => clientDownloadOptions.value.length > 0)
// 有客户端时首页围绕客户端讲，没有时围绕 API 讲
const homeMode = computed<'client' | 'api'>(() => (hasClientDownloads.value ? 'client' : 'api'))

// 客户端最新版本号来自 GitHub Releases 同步的更新日志，只在有客户端时才去取
const { latestVersion: changelogLatestVersion, load: loadChangelog } = useClientChangelog()
const latestVersion = computed(() => (hasClientDownloads.value ? changelogLatestVersion.value : ''))
watch(
  hasClientDownloads,
  (enabled) => {
    if (enabled) void loadChangelog()
  },
  { immediate: true },
)
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))

const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.homePath)
const userInitial = computed(() => {
  const user = authStore.user
  if (!user?.email) return ''
  return user.email.charAt(0).toUpperCase()
})

function applyHomeDocumentTitle() {
  document.title = `${siteName.value} - Claude Code / Codex 一键接入 · 按量计费`
}

watch(
  [siteName, () => appStore.publicSettingsLoaded],
  () => {
    applyHomeDocumentTitle()
    window.setTimeout(applyHomeDocumentTitle)
  },
  { immediate: true, flush: 'post' },
)

const currentYear = computed(() => new Date().getFullYear())

function syncThemeState() {
  isDark.value = document.documentElement.classList.contains('dark')
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  syncThemeState()
  authStore.checkAuth()

  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }

  window.setTimeout(applyHomeDocumentTitle)
  titleSyncTimer = window.setInterval(applyHomeDocumentTitle, 250)
  window.setTimeout(() => {
    if (titleSyncTimer) {
      window.clearInterval(titleSyncTimer)
      titleSyncTimer = undefined
    }
  }, 3000)
})

onBeforeUnmount(() => {
  if (titleSyncTimer) {
    window.clearInterval(titleSyncTimer)
    titleSyncTimer = undefined
  }
})
</script>
