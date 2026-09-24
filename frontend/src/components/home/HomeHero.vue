<template>
  <section id="download" class="relative overflow-hidden">
    <!-- 装饰图形：只在宽屏出现，不参与布局 -->
    <div aria-hidden="true" class="pointer-events-none absolute inset-x-0 top-0 hidden h-[640px] lg:block">
      <span class="animate-home-float absolute left-[7%] top-[22%]">
        <span
          class="flex h-28 w-28 -rotate-12 items-center justify-center rounded-[30px] bg-primary-400 font-mono text-3xl font-black text-gray-900 shadow-[0_18px_40px_rgba(13,148,136,0.28)]"
        >&gt;_</span>
      </span>
      <span class="animate-home-float absolute right-[7%] top-[14%] [animation-delay:-3s]">
        <span
          class="flex h-32 w-32 rotate-[9deg] items-center justify-center rounded-[28px] bg-amber-300 text-5xl font-black text-gray-900 shadow-[0_18px_40px_rgba(217,119,6,0.22)]"
        >✦</span>
      </span>
      <span class="absolute left-[24%] top-[9%] h-5 w-5 rotate-12 bg-sky-300"></span>
      <span class="absolute right-[27%] top-[6%] h-6 w-6 -rotate-12 bg-orange-500"></span>
      <span class="absolute left-[15%] top-[62%] h-6 w-6 rotate-45 bg-white shadow-md dark:bg-white/80"></span>
      <span class="absolute right-[15%] top-[58%] h-5 w-5 rotate-12 bg-primary-600"></span>
    </div>

    <div class="relative mx-auto flex max-w-[1200px] flex-col items-center px-4 pt-12 text-center md:px-6 md:pt-20">
      <router-link
        v-if="hasClientDownloads && latestVersion"
        to="/changelog"
        data-test="hero-release-badge"
        class="inline-flex max-w-full items-center gap-2 rounded-full border border-black/10 bg-white/75 py-1 pl-1 pr-3 text-[13px] font-medium text-gray-700 backdrop-blur transition hover:border-black/20 hover:bg-white dark:border-white/10 dark:bg-white/5 dark:text-white/70 dark:hover:bg-white/10"
      >
        <span class="shrink-0 rounded-full bg-gray-900 px-2 py-0.5 text-[11px] font-bold text-white dark:bg-white dark:text-gray-900">
          {{ t('home.landing.hero.releaseBadge', { version: latestVersion }) }}
        </span>
        <span class="truncate">{{ t('home.landing.hero.releaseCta') }}</span>
        <Icon name="arrowRight" size="xs" class="shrink-0" />
      </router-link>

      <h1
        class="mt-6 max-w-full text-[clamp(2.5rem,9vw,5.75rem)] font-black leading-[1.02] tracking-[-0.045em] [overflow-wrap:anywhere] [text-wrap:balance]"
      >
        <span class="block text-gray-900 dark:text-white">{{ copy.titleLead }}</span>
        <span class="block text-primary-600 dark:text-primary-400">{{ copy.titleAccent }}</span>
      </h1>
      <p class="mt-6 max-w-[42rem] text-base leading-7 text-gray-600 md:text-lg md:leading-8 dark:text-white/65">
        {{ copy.subtitle }}
      </p>

      <ul data-test="hero-providers" class="mt-7 flex items-center justify-center gap-3">
        <li
          v-for="provider in providers"
          :key="provider.platform"
          :title="provider.label"
          class="flex h-12 w-12 items-center justify-center rounded-full border border-black/10 bg-white text-gray-900 shadow-sm dark:border-white/10 dark:bg-white/5 dark:text-white"
        >
          <PlatformIcon :platform="provider.platform" size="md" />
          <span class="sr-only">{{ provider.label }}</span>
        </li>
      </ul>

      <!-- 有客户端：下载为主，API 为次 -->
      <div v-if="hasClientDownloads" class="mt-9 flex w-full flex-col items-center gap-4">
        <HomeDownloadButton :options="clientDownloadOptions" />
        <router-link
          :to="dashboardPath"
          data-test="hero-connect-api"
          class="inline-flex items-center gap-1.5 text-sm font-semibold text-gray-700 underline-offset-4 transition hover:text-gray-900 hover:underline dark:text-white/70 dark:hover:text-white"
        >
          {{ t('home.landing.hero.useApi') }}
          <Icon name="arrowRight" size="xs" />
        </router-link>
      </div>

      <!-- 没有客户端：接入 API 为主 -->
      <div v-else class="mt-9 flex w-full flex-col justify-center gap-3 sm:w-auto sm:flex-row">
        <router-link
          :to="primaryTo"
          data-test="hero-primary-fallback"
          class="inline-flex h-14 items-center justify-center gap-2 rounded-xl bg-gray-900 px-7 text-[15px] font-bold text-white shadow-[0_14px_36px_rgba(15,17,20,0.22)] transition hover:bg-black dark:bg-white dark:text-gray-900 dark:hover:bg-gray-100"
        >
          {{ t('home.landing.hero.startApi') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          data-test="hero-secondary-docs"
          class="inline-flex h-14 items-center justify-center gap-2 rounded-xl border border-black/15 bg-white/70 px-7 text-[15px] font-semibold text-gray-900 transition hover:bg-white dark:border-white/15 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
        >
          {{ t('home.landing.hero.viewDocs') }}
          <Icon name="externalLink" size="sm" />
        </a>
        <router-link
          v-else-if="!isAuthenticated"
          to="/login"
          data-test="hero-secondary-login"
          class="inline-flex h-14 items-center justify-center gap-2 rounded-xl border border-black/15 bg-white/70 px-7 text-[15px] font-semibold text-gray-900 transition hover:bg-white dark:border-white/15 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
        >
          {{ t('home.login') }}
        </router-link>
      </div>
    </div>

    <!-- 主视觉：有客户端时是客户端界面，否则是接入示例 -->
    <div class="relative mx-auto mt-14 max-w-[1200px] px-4 md:mt-16 md:px-6">
      <div v-if="hasClientDownloads" data-test="client-showcase">
        <HomeAgentWorkflowPreview :site-name="siteName" />
      </div>
      <HomeApiTerminal v-else :base-url="baseUrl" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeAgentWorkflowPreview from '@/components/home/HomeAgentWorkflowPreview.vue'
import HomeApiTerminal from '@/components/home/HomeApiTerminal.vue'
import HomeDownloadButton from '@/components/home/HomeDownloadButton.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'
import {
  detectPreferredClientPlatform,
  getClientDownloadOptions,
} from '@/utils/clientDownloads'

const props = withDefaults(defineProps<{
  siteName: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  windowsUrl?: string
  macosUrl?: string
  /** 接入示例里展示的网关地址；留空用当前站点地址 */
  apiBaseUrl?: string
  /** 客户端最新版本号（来自更新日志），留空不显示发布徽标 */
  latestVersion?: string
}>(), {
  siteName: 'CheapRouter',
  windowsUrl: '',
  macosUrl: '',
  apiBaseUrl: '',
  latestVersion: '',
})

const { t } = useI18n()

const clientDownloadOptions = computed(() =>
  getClientDownloadOptions(
    { windowsUrl: props.windowsUrl, macosUrl: props.macosUrl },
    detectPreferredClientPlatform(),
  ),
)
const hasClientDownloads = computed(() => clientDownloadOptions.value.length > 0)

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/login'))
const baseUrl = computed(() => {
  const configured = props.apiBaseUrl.trim()
  if (configured) return configured
  return typeof window === 'undefined' ? '' : window.location.origin
})

const copy = computed(() => {
  const mode = hasClientDownloads.value ? 'client' : 'api'
  return {
    titleLead: t(`home.landing.hero.${mode}.titleLead`),
    titleAccent: t(`home.landing.hero.${mode}.titleAccent`),
    subtitle: t(`home.landing.hero.${mode}.subtitle`),
  }
})

// 公开定价目前只有这三家；新增平台时在这里补上图标。
const providers: Array<{ platform: GroupPlatform; label: string }> = [
  { platform: 'anthropic', label: 'Claude' },
  { platform: 'openai', label: 'GPT' },
  { platform: 'grok', label: 'Grok' },
]
</script>
