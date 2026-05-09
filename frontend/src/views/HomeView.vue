<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else
    class="home-font-sans relative min-h-screen overflow-hidden bg-[#f8fafb] text-[#161616] dark:bg-[#0f1114] dark:text-[#f3f1ed]"
  >
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="home-grid-pattern absolute inset-0 opacity-[0.25] dark:opacity-[0.16]"></div>
    </div>

    <HomeHeader
      :site-name="siteName"
      :doc-url="docUrl"
      :is-dark="isDark"
      :is-authenticated="isAuthenticated"
      :dashboard-path="dashboardPath"
      :user-initial="userInitial"
      @toggle-theme="toggleTheme"
    />

    <main class="relative z-10">
      <HomeHero
        :site-subtitle="siteSubtitle"
        :doc-url="docUrl"
        :is-authenticated="isAuthenticated"
        :dashboard-path="dashboardPath"
        :windows-url="clientDownloadWindowsUrl"
        :macos-url="clientDownloadMacOSUrl"
      />

      <HomeReveal class="px-4 pb-8 md:px-6">
        <HomeProofStrip />
      </HomeReveal>

      <HomeReveal v-if="hasClientDownloads" class="px-4 py-8 md:px-6 md:py-12">
        <HomeDownloadSection
          :windows-url="clientDownloadWindowsUrl"
          :macos-url="clientDownloadMacOSUrl"
        />
      </HomeReveal>

      <HomeReveal class="px-4 py-12 md:px-6 md:py-16">
        <HomeValueSection />
      </HomeReveal>

      <HomeReveal id="pricing" class="px-4 py-8 md:px-6 md:py-12">
        <HomePricingSection />
      </HomeReveal>

      <HomeReveal class="px-4 pb-8 md:px-6">
        <HomeComparisonSection />
      </HomeReveal>

      <HomeReveal class="px-4 py-12 md:px-6 md:py-16">
        <HomeTrustSection />
      </HomeReveal>

      <HomeReveal class="px-4 py-8 md:px-6 md:py-12">
        <HomeFinalCta
          :doc-url="docUrl"
          :is-authenticated="isAuthenticated"
          :dashboard-path="dashboardPath"
        />
      </HomeReveal>
    </main>

    <HomeFooter
      :site-name="siteName"
      :doc-url="docUrl"
      :current-year="currentYear"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import HomeComparisonSection from '@/components/home/HomeComparisonSection.vue'
import HomeDownloadSection from '@/components/home/HomeDownloadSection.vue'
import HomePricingSection from '@/components/home/HomePricingSection.vue'
import HomeFinalCta from '@/components/home/HomeFinalCta.vue'
import HomeFooter from '@/components/home/HomeFooter.vue'
import HomeHeader from '@/components/home/HomeHeader.vue'
import HomeHero from '@/components/home/HomeHero.vue'
import HomeProofStrip from '@/components/home/HomeProofStrip.vue'
import HomeReveal from '@/components/home/HomeReveal.vue'
import HomeTrustSection from '@/components/home/HomeTrustSection.vue'
import HomeValueSection from '@/components/home/HomeValueSection.vue'

const authStore = useAuthStore()
const appStore = useAppStore()
const homeDocumentTitle = 'CheapRouter - Claude Code / Codex 一键接入 · 按量计费'
let titleSyncTimer: number | undefined

const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name?.trim() || appStore.siteName.trim()
  return configuredName && configuredName !== 'Sub2API' ? configuredName : 'CheapRouter'
})
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const clientDownloadWindowsUrl = computed(
  () => appStore.cachedPublicSettings?.client_download_windows_url?.trim() || ''
)
const clientDownloadMacOSUrl = computed(
  () => appStore.cachedPublicSettings?.client_download_macos_url?.trim() || ''
)
const hasClientDownloads = computed(
  () => Boolean(clientDownloadWindowsUrl.value) || Boolean(clientDownloadMacOSUrl.value)
)

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const userInitial = computed(() => {
  const user = authStore.user
  if (!user?.email) return ''
  return user.email.charAt(0).toUpperCase()
})

function applyHomeDocumentTitle() {
  document.title = homeDocumentTitle
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
