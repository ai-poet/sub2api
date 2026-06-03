<template>
  <div class="home-font-sans relative min-h-screen overflow-hidden bg-[#f8fafb] text-[#161616] dark:bg-[#0f1114] dark:text-[#f3f1ed]">
    <!-- Background grid pattern -->
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
      <!-- Page Header -->
      <section class="px-4 pb-8 pt-12 md:px-6 md:pb-12 md:pt-16">
        <div class="mx-auto max-w-[900px] text-center">
          <h1 class="mb-3 text-3xl font-bold tracking-tight text-[#111] dark:text-white md:text-4xl">
            {{ t('changelog.title') }}
          </h1>
          <p class="mx-auto max-w-[520px] text-base text-[#645d54] dark:text-white/60">
            {{ t('changelog.subtitle') }}
          </p>
        </div>
      </section>

      <!-- Timeline -->
      <section class="px-4 pb-16 md:px-6 md:pb-24">
        <div class="mx-auto max-w-[720px]">
          <!-- Empty state -->
          <div v-if="entries.length === 0" class="py-20 text-center">
            <div class="mx-auto mb-5 flex h-20 w-20 items-center justify-center rounded-2xl bg-black/[0.04] dark:bg-white/[0.06]">
              <svg class="h-9 w-9 text-[#999] dark:text-white/40" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
              </svg>
            </div>
            <h3 class="mb-2 text-lg font-semibold text-[#333] dark:text-white/80">
              {{ t('changelog.emptyTitle') }}
            </h3>
            <p class="mx-auto max-w-[360px] text-sm text-[#888] dark:text-white/50">
              {{ t('changelog.emptyDesc') }}
            </p>
          </div>

          <!-- Timeline entries -->
          <div v-else class="relative">
            <!-- Vertical line -->
            <div class="absolute left-[15px] top-0 bottom-0 w-px bg-black/10 dark:bg-white/10 md:left-[19px]"></div>

            <div class="space-y-8">
              <div
                v-for="(entry, index) in entries"
                :key="index"
                class="relative pl-10 md:pl-12"
              >
                <!-- Dot -->
                <div
                  class="absolute left-[7px] top-2 h-4 w-4 rounded-full border-2 border-white bg-primary-500 shadow-sm dark:border-[#0f1114] md:left-[11px]"
                ></div>

                <!-- Card -->
                <div
                  class="rounded-2xl border border-black/8 bg-white/80 p-5 shadow-sm backdrop-blur-sm transition hover:shadow-md dark:border-white/8 dark:bg-[rgba(24,26,32,0.7)] md:p-6"
                >
                  <!-- Header: version + date -->
                  <div class="mb-3 flex flex-wrap items-center gap-2">
                    <span class="inline-flex items-center rounded-lg bg-primary-50 px-2.5 py-1 text-xs font-bold text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                      v{{ entry.version }}
                    </span>
                    <span v-if="entry.published_at" class="text-xs text-[#888] dark:text-white/45">
                      {{ formatDate(entry.published_at) }}
                    </span>
                  </div>

                  <!-- Title -->
                  <h3
                    v-if="entry.title"
                    class="mb-3 text-base font-semibold text-[#222] dark:text-white/90"
                  >
                    {{ entry.title }}
                  </h3>

                  <!-- Items -->
                  <ul v-if="entry.renderedItems.length > 0" class="space-y-2">
                    <li
                      v-for="(item, itemIdx) in entry.renderedItems"
                      :key="itemIdx"
                      class="flex items-start gap-2"
                    >
                      <span class="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-primary-400 dark:bg-primary-500"></span>
                      <div
                        class="markdown-body prose prose-sm max-w-none text-sm text-[#555] dark:prose-invert dark:text-white/70"
                        v-html="item.html"
                      ></div>
                    </li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
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
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import HomeHeader from '@/components/home/HomeHeader.vue'
import HomeFooter from '@/components/home/HomeFooter.vue'
import { renderSafeMarkdown } from '@/utils/markdown'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const currentYear = computed(() => new Date().getFullYear())

const entries = computed(() =>
  (appStore.cachedPublicSettings?.client_changelog_entries || []).map((entry) => ({
    ...entry,
    renderedItems: entry.items.map((item) => ({
      source: item,
      html: renderSafeMarkdown(item),
    })),
  }))
)

const isDark = ref(document.documentElement.classList.contains('dark'))

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const userInitial = computed(() => {
  const user = authStore.user
  if (!user?.email) return ''
  return user.email.charAt(0).toUpperCase()
})

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function syncThemeState() {
  isDark.value = document.documentElement.classList.contains('dark')
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateStr)) return dateStr
  try {
    const d = new Date(dateStr + 'T00:00:00')
    if (isNaN(d.getTime())) return dateStr
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
  } catch {
    return dateStr
  }
}

onMounted(() => {
  syncThemeState()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
