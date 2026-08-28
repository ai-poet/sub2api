<template>
  <section class="relative w-full overflow-hidden bg-white px-4 pt-12 md:px-6 md:pt-16 dark:bg-[#0f1114]">

    <div class="mx-auto w-full max-w-[1380px] min-w-0">

      <!-- Category tag pills -->
      <div class="flex flex-wrap gap-2">
        <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.coding') }}</span>
        <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.agent') }}</span>
        <span class="rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-500 dark:border-white/10 dark:bg-white/5 dark:text-white/50">{{ t('home.hero.tags.tools') }}</span>
      </div>

      <!-- Headline -->
      <h1 class="mt-6 max-w-full text-[clamp(1.95rem,7.8vw,5rem)] font-black leading-[1.06] tracking-[-0.04em] [overflow-wrap:anywhere] [text-wrap:balance] sm:text-[clamp(2.6rem,6vw,5rem)]">
        <span class="block text-[#111] dark:text-white">
          <span class="block sm:inline">{{ t('home.hero.titleLeadPrimary') }}</span>
          <span v-if="hasClientDownloads" class="block sm:ml-[0.18em] sm:inline">{{ t('home.hero.titleLeadSecondary') }}</span>
        </span>
        <span v-if="titleAccent" class="block text-primary-600 dark:text-primary-400">{{ titleAccent }}</span>
        <span v-if="titleTail" class="block text-[#111] dark:text-white">{{ titleTail }}</span>
      </h1>

      <!-- CLI icons strip -->
      <div class="mt-5 flex flex-wrap items-center gap-2.5">
        <span class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 dark:border-white/10 dark:bg-white/5 dark:text-white/70">
          <PlatformIcon platform="anthropic" size="md" />
          Claude Code
        </span>
        <span class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 dark:border-white/10 dark:bg-white/5 dark:text-white/70">
          <PlatformIcon platform="openai" size="md" />
          Codex
        </span>
        <span class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 dark:border-white/10 dark:bg-white/5 dark:text-white/70">
          <PlatformIcon platform="grok" size="md" />
          Grok
        </span>
        <span class="inline-flex items-center gap-1.5 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-3 py-1.5 text-sm font-medium text-gray-400 dark:border-white/15 dark:bg-white/[0.03] dark:text-white/40">
          + {{ t('home.hero.tags.more') }}
        </span>
      </div>

      <!-- CTAs -->
      <div class="mt-6 flex flex-col gap-3 sm:flex-row">
        <!-- 终端命令类型主按钮 -->
        <button
          v-if="primaryClientDownloadOption && primaryClientDownloadOption.type === 'command'"
          type="button"
          :data-platform="primaryClientDownloadOption.id"
          data-test="hero-primary-download"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full bg-[#111] px-8 text-[15px] font-bold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5] sm:w-auto"
          @click="handlePrimaryClick"
        >
          <span>{{ t('home.hero.installPrimary') }}</span>
          <Icon name="clipboard" size="sm" />
        </button>
        <!-- 下载链接类型主按钮 -->
        <a
          v-else-if="primaryClientDownloadOption"
          :href="primaryClientDownloadOption.url"
          :data-platform="primaryClientDownloadOption.id"
          data-test="hero-primary-download"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full bg-[#111] px-8 text-[15px] font-bold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5] sm:w-auto"
        >
          <span>{{ t('home.hero.downloadPrimary') }}</span>
          <Icon name="download" size="sm" />
        </a>
        <router-link
          v-if="primaryClientDownloadOption"
          :to="dashboardPath"
          data-test="hero-connect-api"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 active:translate-y-0 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
        >
          <span>{{ t('home.hero.connectApi') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>
        <router-link
          v-if="!primaryClientDownloadOption"
          :to="primaryTo"
          data-test="hero-primary-fallback"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full bg-[#111] px-8 text-[15px] font-bold text-white transition hover:-translate-y-[1px] hover:bg-black active:translate-y-0 dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5] sm:w-auto"
        >
          <span>{{ t('home.hero.startApi') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>

        <a
          v-if="!primaryClientDownloadOption && docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
        >
          <span>{{ t('home.viewDocs') }}</span>
          <Icon name="externalLink" size="sm" />
        </a>
        <router-link
          v-else-if="!primaryClientDownloadOption"
          to="/login"
          class="inline-flex h-14 w-full items-center justify-center gap-2 rounded-full border border-gray-200 bg-white px-8 text-[15px] font-semibold text-[#111] transition hover:-translate-y-[1px] hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10 sm:w-auto"
        >
          <span>{{ t('home.login') }}</span>
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>

      <!-- macOS install command preview -->
      <div
        v-if="primaryClientDownloadOption && primaryClientDownloadOption.type === 'command'"
        class="mt-4 flex max-w-2xl items-center gap-2 rounded-2xl border border-black/8 bg-[#16181d] p-2 pl-4 font-mono text-sm text-white shadow-sm dark:border-white/10 dark:bg-[#0f1114]"
        data-test="hero-install-command"
      >
        <code class="min-w-0 flex-1 truncate text-white/90">{{ primaryClientDownloadOption.url }}</code>
        <button
          type="button"
          class="shrink-0 rounded-xl bg-white/10 px-3 py-2 text-xs font-semibold text-white transition hover:bg-white/20"
          @click="handlePrimaryClick"
        >
          {{ t('home.hero.installPrimary') }}
        </button>
      </div>

      <!-- Note -->
      <p
        v-if="hasClientDownloads"
        class="mt-2 max-w-full text-sm leading-6 text-gray-400 [overflow-wrap:anywhere] dark:text-white/35"
      >
        {{ t('home.hero.primaryNote') }}
      </p>
    </div>

    <!-- ===== Agent Workflow Preview: 直接嵌入首屏（参考 Cursor 布局） ===== -->
    <div v-if="hasClientDownloads" data-test="client-showcase" class="mx-auto mt-10 max-w-[1380px] pb-12 md:pb-16">
      <!-- Feature pills -->
      <div class="mb-5 flex flex-wrap gap-2">
        <span v-for="pill in pills" :key="pill" class="rounded-full border border-gray-200 bg-white px-3.5 py-1.5 text-sm text-gray-600 dark:border-white/10 dark:bg-white/5 dark:text-white/70">
          {{ pill }}
        </span>
      </div>

      <HomeAgentWorkflowPreview :site-name="siteName" />

      <!-- Client advantages strip -->
      <div data-test="client-advantages" class="mt-6 grid gap-3 sm:grid-cols-3">
        <div
          v-for="card in advantageCards"
          :key="card.title"
          class="rounded-xl border border-gray-200 bg-white p-5 dark:border-white/10 dark:bg-white/5"
        >
          <h3 class="text-[15px] font-semibold text-[#111] dark:text-white">{{ card.title }}</h3>
          <p class="mt-1.5 text-sm leading-6 text-gray-500 dark:text-white/55">{{ card.body }}</p>
        </div>
      </div>

      <!-- API-only card -->
      <div
        data-test="api-only-card"
        class="mt-6 flex flex-col gap-4 rounded-2xl border border-gray-200 bg-white p-5 sm:flex-row sm:items-center sm:justify-between dark:border-white/10 dark:bg-white/5"
      >
        <div>
          <h3 class="text-[15px] font-semibold text-[#111] dark:text-white">
            {{ t('home.clientShowcase.apiOnly.title') }}
          </h3>
          <p class="mt-1.5 max-w-[38rem] text-sm leading-6 text-gray-500 dark:text-white/55">
            {{ t('home.clientShowcase.apiOnly.body') }}
          </p>
        </div>
        <div class="flex shrink-0 flex-wrap items-center gap-3">
          <router-link
            :to="dashboardPath"
            data-test="api-only-dashboard"
            class="inline-flex h-10 items-center gap-1.5 rounded-full bg-[#111] px-5 text-sm font-semibold text-white transition hover:bg-black dark:bg-white dark:text-[#111] dark:hover:bg-[#ece9e5]"
          >
            <span>{{ t('home.clientShowcase.apiOnly.dashboardCta') }}</span>
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            data-test="api-only-docs"
            class="inline-flex h-10 items-center gap-1.5 rounded-full border border-gray-200 bg-white px-5 text-sm font-semibold text-[#111] transition hover:bg-gray-50 dark:border-white/12 dark:bg-white/5 dark:text-white dark:hover:bg-white/10"
          >
            <span>{{ t('home.clientShowcase.apiOnly.docsCta') }}</span>
            <Icon name="externalLink" size="sm" />
          </a>
        </div>
      </div>
    </div>

  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import HomeAgentWorkflowPreview from '@/components/home/HomeAgentWorkflowPreview.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useClipboard } from '@/composables/useClipboard'
import {
  detectPreferredClientPlatform,
  getClientDownloadOptions,
} from '@/utils/clientDownloads'

const props = withDefaults(defineProps<{
  siteName: string
  siteSubtitle: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  windowsUrl?: string
  macosUrl?: string
}>(), {
  siteName: 'CheapRouter',
})

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const primaryTo = computed(() => (props.isAuthenticated ? props.dashboardPath : '/login'))
const titleAccent = computed(() => t('home.hero.titleAccent').trim())
const titleTail = computed(() => t('home.hero.titleTail').trim())
const preferredClientPlatform = computed(() => detectPreferredClientPlatform())
const clientDownloadOptions = computed(() =>
  getClientDownloadOptions(
    {
      windowsUrl: props.windowsUrl,
      macosUrl: props.macosUrl,
    },
    preferredClientPlatform.value,
  ),
)
const primaryClientDownloadOption = computed(() => clientDownloadOptions.value[0] ?? null)
const hasClientDownloads = computed(() => clientDownloadOptions.value.length > 0)

function handlePrimaryClick() {
  if (primaryClientDownloadOption.value?.type === 'command') {
    copyToClipboard(primaryClientDownloadOption.value.url, t('home.download.commandCopied'))
  }
}

const pills = computed(() => [
  t('home.clientShowcase.pills.autoRoute'),
  t('home.clientShowcase.pills.groupSwitch'),
  t('home.clientShowcase.pills.liveBalance'),
  t('home.clientShowcase.pills.cliInstall'),
  t('home.clientShowcase.pills.aggregate'),
])

const advantageCards = computed(() => [
  {
    title: t('home.clientShowcase.advantages.tiny.title'),
    body: t('home.clientShowcase.advantages.tiny.body'),
  },
  {
    title: t('home.clientShowcase.advantages.native.title'),
    body: t('home.clientShowcase.advantages.native.body'),
  },
  {
    title: t('home.clientShowcase.advantages.ready.title'),
    body: t('home.clientShowcase.advantages.ready.body'),
  },
])
</script>
