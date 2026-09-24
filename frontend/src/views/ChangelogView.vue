<template>
  <div class="home-font-sans relative min-h-screen overflow-x-hidden bg-[#f6f4ef] text-gray-900 dark:bg-[#0b0c0e] dark:text-gray-100">
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
      <!-- 刊头 -->
      <section class="border-y border-black/10 bg-primary-100 dark:border-white/10 dark:bg-primary-950">
        <div class="mx-auto grid max-w-[1200px] gap-10 px-4 py-16 md:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)] md:items-end md:px-6 md:py-24">
          <div>
            <p class="text-[11px] font-bold uppercase tracking-[0.24em] text-primary-800 dark:text-primary-300">
              {{ t('changelog.overline') }}
            </p>
            <h1
              class="mt-6 text-[clamp(2.8rem,8vw,6rem)] font-black leading-[0.96] tracking-[-0.05em] text-gray-900 [overflow-wrap:anywhere] [text-wrap:balance] dark:text-white"
            >
              {{ t('changelog.headline', { siteName }) }}
            </h1>
          </div>
          <div class="border-t-2 border-gray-900 pt-5 dark:border-white">
            <p class="text-base leading-7 text-gray-700 dark:text-white/70">{{ t('changelog.subtitle') }}</p>
            <a
              v-if="hasClientDownloads"
              href="/home#download"
              data-test="changelog-download"
              class="mt-5 inline-flex items-center gap-1.5 text-sm font-bold text-gray-900 underline underline-offset-4 dark:text-white"
              @click.prevent="goToSection('download')"
            >
              {{ t('changelog.download') }}
              <Icon name="arrowRight" size="xs" />
            </a>
          </div>
        </div>
      </section>

      <div class="bg-[#111214] text-white">
        <div
          class="mx-auto flex max-w-[1200px] items-center justify-between gap-4 px-4 py-4 text-[11px] font-bold uppercase tracking-[0.2em] md:px-6"
        >
          <span>{{ t('changelog.versions', { count: entries.length }) }}</span>
          <span v-if="entries.length > 0" class="text-primary-300">
            {{ t('changelog.latest') }} · v{{ entries[0].version }}
          </span>
        </div>
      </div>

      <section class="mx-auto grid max-w-[1200px] gap-10 px-4 py-14 md:grid-cols-[200px_minmax(0,1fr)] md:gap-14 md:px-6 md:py-20">
        <aside class="md:sticky md:top-24 md:self-start">
          <router-link
            to="/home"
            class="text-sm font-semibold text-gray-700 underline underline-offset-4 hover:text-gray-900 dark:text-white/70 dark:hover:text-white"
          >
            ← {{ t('changelog.back') }}
          </router-link>
          <div class="mt-10 hidden md:block">
            <HomeBrandMark :site-name="siteName" :site-logo="siteLogo" class="h-14 w-14 rounded-xl text-2xl" />
            <p class="mt-4 text-[11px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-white/45">
              {{ siteName }} / {{ t('changelog.title') }}
            </p>
          </div>
        </aside>

        <div>
          <p v-if="loading && entries.length === 0" data-test="changelog-loading" class="py-20 text-center text-sm text-gray-500 dark:text-white/50">
            {{ t('changelog.loading') }}
          </p>

          <div v-else-if="entries.length === 0" data-test="changelog-empty" class="py-20 text-center">
            <h3 class="text-lg font-bold text-gray-800 dark:text-white/85">{{ t('changelog.emptyTitle') }}</h3>
            <p class="mx-auto mt-2 max-w-[360px] text-sm text-gray-500 dark:text-white/50">{{ t('changelog.emptyDesc') }}</p>
          </div>

          <template v-else>
            <article
              v-for="(entry, index) in entries"
              :key="entry.version"
              data-test="changelog-entry"
              class="grid gap-4 border-b border-black/10 py-12 first:pt-0 last:border-b-0 md:grid-cols-[56px_minmax(0,1fr)] dark:border-white/10"
            >
              <span class="font-mono text-xs font-bold text-primary-700 dark:text-primary-400">
                {{ String(index + 1).padStart(2, '0') }}
              </span>
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-3 text-sm text-gray-500 dark:text-white/50">
                  <time v-if="entry.published_at" :datetime="entry.published_at">{{ formatReleaseDate(entry.published_at) }}</time>
                  <span
                    v-if="index === 0"
                    class="rounded-full bg-primary-600 px-2 py-0.5 text-[11px] font-bold uppercase tracking-[0.08em] text-white"
                  >
                    {{ t('changelog.latest') }}
                  </span>
                </div>
                <h2 class="mt-3 text-[clamp(1.9rem,4.4vw,3rem)] font-black leading-[1.05] tracking-[-0.04em] text-gray-900 [overflow-wrap:anywhere] dark:text-white">
                  v{{ entry.version }}<span v-if="entry.title" class="text-gray-500 dark:text-white/50"> — {{ entry.title }}</span>
                </h2>

                <ul
                  v-if="entry.items.length > 0"
                  class="mt-7 grid border-l border-t border-black/15 dark:border-white/15"
                  :class="isCompact(entry) ? 'sm:grid-cols-2' : ''"
                >
                  <li
                    v-for="(item, itemIndex) in entry.items"
                    :key="itemIndex"
                    class="flex gap-3 border-b border-r border-black/15 bg-white/60 p-4 dark:border-white/15 dark:bg-white/[0.03] md:p-5"
                  >
                    <span class="mt-2 h-2 w-2 shrink-0 bg-primary-500"></span>
                    <MarkdownRenderer
                      :content="item"
                      class-name="changelog-item min-w-0 flex-1 text-sm leading-6 text-gray-700 dark:text-white/75"
                    />
                  </li>
                  <!-- 两栏排版时条目数为奇数，用深色块补齐网格 -->
                  <li
                    v-if="isCompact(entry) && entry.items.length % 2 === 1"
                    aria-hidden="true"
                    class="hidden border-b border-r border-black/15 bg-[#111214] sm:block dark:border-white/15 dark:bg-white/10"
                  ></li>
                </ul>
              </div>
            </article>
          </template>
        </div>
      </section>
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
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import type { ClientChangelogEntry } from '@/api/changelog'
import HomeBrandMark from '@/components/home/HomeBrandMark.vue'
import HomeFooter from '@/components/home/HomeFooter.vue'
import HomeHeader from '@/components/home/HomeHeader.vue'
import { useHomeSectionNav } from '@/components/home/useHomeSectionNav'
import Icon from '@/components/icons/Icon.vue'
import MarkdownRenderer from '@/components/common/MarkdownRenderer.vue'
import { useClientChangelog } from '@/composables/useClientChangelog'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { goToSection } = useHomeSectionNav()
const { entries, loading, load } = useClientChangelog()

const siteName = computed(() => {
  const configuredName = appStore.cachedPublicSettings?.site_name?.trim() || appStore.siteName.trim()
  return configuredName && configuredName !== 'Sub2API' ? configuredName : 'CheapRouter'
})
const siteLogo = computed(() => appStore.siteLogo || appStore.cachedPublicSettings?.site_logo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const hasClientDownloads = computed(() =>
  Boolean(appStore.cachedPublicSettings?.client_download_windows_url?.trim()) ||
  Boolean(appStore.cachedPublicSettings?.client_download_macos_url?.trim())
)
const currentYear = computed(() => new Date().getFullYear())

const isDark = ref(document.documentElement.classList.contains('dark'))

const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.homePath)
const userInitial = computed(() => {
  const user = authStore.user
  if (!user?.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// 条目都短时排成两栏网格，有长段落时单栏更好读。
const COMPACT_ITEM_LENGTH = 160
function isCompact(entry: ClientChangelogEntry): boolean {
  return entry.items.length > 1 && entry.items.every((item) => item.length <= COMPACT_ITEM_LENGTH)
}

function formatReleaseDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleDateString(locale.value, { year: 'numeric', month: 'long', day: 'numeric' })
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  isDark.value = document.documentElement.classList.contains('dark')
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  void load()
})
</script>

<style scoped>
.changelog-item :deep(p) {
  margin: 0;
}

.changelog-item :deep(code) {
  @apply rounded bg-black/[0.06] px-1 py-0.5 text-[0.85em] dark:bg-white/10;
}
</style>
