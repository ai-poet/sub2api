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
    class="home-font-sans relative min-h-screen overflow-hidden bg-[#f5efe6] text-[#161616] dark:bg-[#0f1114] dark:text-[#f3f1ed]"
  >
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="home-grid-pattern absolute inset-0 opacity-50 dark:opacity-[0.16]"></div>
      <div
        class="animate-home-glow absolute left-[-10%] top-[-13rem] h-[30rem] w-[30rem] rounded-full bg-[#d8c6ab]/45 blur-3xl dark:bg-[#132f35]/45"
      ></div>
      <div
        class="absolute right-[-6%] top-[10rem] h-[24rem] w-[24rem] rounded-full bg-primary-200/55 blur-3xl dark:bg-primary-900/20"
      ></div>
      <div
        class="absolute bottom-[-10rem] left-[16%] h-[24rem] w-[24rem] rounded-full bg-white/60 blur-3xl dark:bg-[#1f2630]/70"
      ></div>
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
      />

      <HomeReveal class="px-4 pb-8 md:px-6">
        <HomeProofStrip />
      </HomeReveal>

      <HomeReveal class="px-4 py-12 md:px-6 md:py-16">
        <HomeValueSection />
      </HomeReveal>

      <HomeReveal class="px-4 py-8 md:px-6 md:py-12">
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
import { computed, onMounted, ref } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import HomeComparisonSection from '@/components/home/HomeComparisonSection.vue'
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

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

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
})
</script>
