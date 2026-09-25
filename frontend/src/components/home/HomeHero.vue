<template>
  <section id="download" class="relative overflow-hidden">
    <!-- 小块装饰：只在宽屏出现，放在标题上方，不参与布局 -->
    <div aria-hidden="true" class="pointer-events-none absolute inset-x-0 top-0 hidden h-[640px] lg:block">
      <span class="absolute left-[24%] top-[9%] h-5 w-5 rotate-12 bg-sky-300"></span>
      <span class="absolute right-[27%] top-[6%] h-6 w-6 -rotate-12 bg-orange-500"></span>
      <span class="absolute left-[13%] top-[4%] h-6 w-6 rotate-45 bg-white shadow-md dark:bg-white/80"></span>
      <span class="absolute right-[14%] top-[8%] h-5 w-5 rotate-12 bg-primary-600"></span>
    </div>

    <div class="relative mx-auto flex max-w-[1200px] flex-col items-center px-4 pt-8 text-center md:px-6 md:pt-10">
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

      <!-- 标题只在逗号处断行，副标题讲质量，字号小一档 -->
      <h1
        data-test="hero-title"
        class="mt-4 max-w-full text-[clamp(2rem,5.6vw,4rem)] font-black leading-[1.1] tracking-[-0.04em] text-gray-900 [text-wrap:balance] dark:text-white"
      >
        <template v-for="(phrase, index) in titlePhrases" :key="index">
          <span class="inline-block">{{ phrase.text }}</span>{{ phrase.gap }}
        </template>
      </h1>
      <p
        data-test="hero-tagline"
        class="mt-3 text-[clamp(1.15rem,2.4vw,1.625rem)] font-extrabold tracking-[-0.02em] text-primary-600 dark:text-primary-400"
      >
        {{ t('home.landing.hero.tagline') }}
      </p>
      <!-- 有客户端时网关和客户端两段描述并列，没有时只讲网关 -->
      <div
        v-if="hasClientDownloads"
        data-test="hero-descriptions"
        class="mt-5 grid w-full max-w-[56rem] gap-5 md:grid-cols-2 md:gap-0 md:divide-x md:divide-black/10 md:text-left dark:md:divide-white/10"
      >
        <div
          v-for="item in descriptions"
          :key="item.key"
          :data-test="`hero-description-${item.key}`"
          class="md:px-8"
        >
          <span
            class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold"
            :class="item.key === 'client'
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-500/15 dark:text-primary-300'
              : 'bg-gray-900/[0.06] text-gray-800 dark:bg-white/10 dark:text-white/80'"
          >{{ item.label }}</span>
          <p class="mt-2 text-[15px] leading-6 text-gray-600 dark:text-white/65">{{ item.body }}</p>
        </div>
      </div>
      <p v-else class="mt-5 max-w-[42rem] text-base leading-7 text-gray-600 md:text-lg md:leading-8 dark:text-white/65">
        {{ copy.subtitle }}
      </p>

      <!-- 按钮和模型图标这一段内容很窄，两块大装饰挂在它两侧，任何宽度都压不到文字 -->
      <div class="relative mt-6 flex w-full flex-col items-center">
        <div aria-hidden="true" class="pointer-events-none absolute inset-0 hidden lg:block">
          <span class="animate-home-float absolute left-[3%] top-0">
            <span
              class="flex h-28 w-28 -rotate-12 items-center justify-center rounded-[30px] bg-primary-400 font-mono text-3xl font-black text-gray-900 shadow-[0_18px_40px_rgba(13,148,136,0.28)]"
            >&gt;_</span>
          </span>
          <span class="animate-home-float absolute right-[3%] top-[4%] [animation-delay:-3s]">
            <span
              class="flex h-32 w-32 rotate-[9deg] items-center justify-center rounded-[28px] bg-amber-300 text-5xl font-black text-gray-900 shadow-[0_18px_40px_rgba(217,119,6,0.22)]"
            >✦</span>
          </span>
        </div>

        <!-- 有客户端：下载为主 -->
        <div v-if="hasClientDownloads" class="relative flex w-full flex-col items-center">
          <HomeDownloadButton :options="clientDownloadOptions" />
        </div>

        <!-- 没有客户端：接入 API 为主 -->
        <div v-else class="relative flex w-full flex-col justify-center gap-3 sm:w-auto sm:flex-row">
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

        <!-- 已接入的模型和「直接使用 API」并成一行，放在按钮下面，省出一整行高度 -->
        <div class="relative mt-4 flex flex-wrap items-center justify-center gap-x-4 gap-y-2">
          <ul data-test="hero-providers" class="flex items-center justify-center gap-2">
            <li
              v-for="provider in providers"
              :key="provider.platform"
              :title="provider.label"
              class="flex h-9 w-9 items-center justify-center rounded-full border border-black/10 bg-white text-gray-900 shadow-sm dark:border-white/10 dark:bg-white/5 dark:text-white"
            >
              <PlatformIcon :platform="provider.platform" size="md" />
              <span class="sr-only">{{ provider.label }}</span>
            </li>
          </ul>
          <router-link
            v-if="hasClientDownloads"
            :to="dashboardPath"
            data-test="hero-connect-api"
            class="inline-flex items-center gap-1.5 text-sm font-semibold text-gray-700 underline-offset-4 transition hover:text-gray-900 hover:underline dark:text-white/70 dark:hover:text-white"
          >
            {{ t('home.landing.hero.useApi') }}
            <Icon name="arrowRight" size="xs" />
          </router-link>
        </div>
      </div>
    </div>

    <!-- 主视觉：有客户端时是客户端界面，否则是接入示例 -->
    <div class="relative mx-auto mt-10 max-w-[1200px] px-4 pb-10 md:mt-12 md:px-6 md:pb-16">
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
    title: t(`home.landing.hero.${mode}.title`),
    subtitle: t(`home.landing.hero.${mode}.subtitle`),
  }
})

// 标题按逗号切成几段，每段不拆开换行，避免「工作区」被折成「工 / 作区」。
// 英文段之间补回空格，中文段之间放零宽空格作为断行点。
const titlePhrases = computed(() => {
  const parts = copy.value.title.match(/[^，,]+[，,]?\s*/g) ?? [copy.value.title]
  return parts.map((part, index) => ({
    text: part.trimEnd(),
    gap: index === parts.length - 1 ? '' : /\s$/.test(part) ? ' ' : '​',
  }))
})

const descriptions = computed(() =>
  (['api', 'client'] as const).map((key) => ({
    key,
    label: t(`home.landing.hero.${key}.label`),
    body: t(`home.landing.hero.${key}.subtitle`),
  })),
)

// 网关已接入的主要模型；新增平台时在这里补上图标。
const providers: Array<{ platform: GroupPlatform; label: string }> = [
  { platform: 'anthropic', label: 'Claude' },
  { platform: 'openai', label: 'GPT' },
  { platform: 'grok', label: 'Grok' },
  { platform: 'deepseek', label: 'DeepSeek' },
  { platform: 'zhipu', label: 'GLM' },
  { platform: 'kimi', label: 'Kimi' },
]
</script>
