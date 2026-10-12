<template>
  <!-- fork：模型广场匿名视图的外壳，沿用首页的顶栏和页脚，访客从首页「定价」进来不换皮。 -->
  <div class="home-font-sans relative min-h-screen overflow-x-hidden bg-[#f6f4ef] text-gray-900 dark:bg-[#0b0c0e] dark:text-gray-100">
    <HomeHeader
      :site-name="siteName"
      :site-logo="siteLogo"
      :doc-url="docUrl"
      :is-dark="isDark"
      :is-authenticated="false"
      dashboard-path="/login"
      user-initial=""
      :show-changelog="hasClientDownloads"
      @toggle-theme="toggleTheme"
    />

    <main class="fork-content-reveal relative z-10 px-4 pb-20 pt-8 md:px-6 md:pt-12">
      <div class="mx-auto mb-8 max-w-[1600px]">
        <h1 class="text-3xl font-extrabold tracking-[-0.03em] text-gray-900 dark:text-white md:text-4xl">
          {{ t('modelCatalog.title') }}
        </h1>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {{ t('modelCatalog.public.subtitle', { siteName }) }}
        </p>
      </div>
      <slot />
    </main>

    <HomeFooter
      :site-name="siteName"
      :site-logo="siteLogo"
      :doc-url="docUrl"
      :current-year="currentYear"
      :has-client-downloads="hasClientDownloads"
      console-path="/login"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeFooter from '@/components/home/HomeFooter.vue'
import HomeHeader from '@/components/home/HomeHeader.vue'
import { useAppStore } from '@/stores'
import { applyDarkClass } from '@/utils/themeSwitch'
import { detectPreferredClientPlatform, getClientDownloadOptions } from '@/utils/clientDownloads'

const { t } = useI18n()
const appStore = useAppStore()

// 与 HomeView 相同的品牌回退规则：没配站点名时显示 CheapRouter。
const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name?.trim() || appStore.siteName?.trim() || ''
  return configuredName && configuredName !== 'Sub2API' ? configuredName : 'CheapRouter'
})
const siteLogo = computed(() => appStore.siteLogo || appStore.cachedPublicSettings?.site_logo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const hasClientDownloads = computed(
  () =>
    getClientDownloadOptions(
      {
        windowsUrl: appStore.cachedPublicSettings?.client_download_windows_url?.trim() || '',
        macosUrl: appStore.cachedPublicSettings?.client_download_macos_url?.trim() || '',
      },
      detectPreferredClientPlatform(),
    ).length > 0,
)
const currentYear = new Date().getFullYear()

const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  applyDarkClass(isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  isDark.value = document.documentElement.classList.contains('dark')
})
</script>
