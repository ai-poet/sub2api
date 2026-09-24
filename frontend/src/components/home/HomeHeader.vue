<template>
  <header
    class="sticky top-0 z-40 border-b transition-colors duration-300"
    :class="
      scrolled
        ? 'border-black/10 bg-[#f6f4ef]/85 backdrop-blur-xl dark:border-white/10 dark:bg-[#0b0c0e]/85'
        : 'border-transparent bg-transparent'
    "
  >
    <nav class="mx-auto flex h-16 max-w-[1200px] items-center justify-between gap-3 px-4 md:px-6">
      <!-- Logo -->
      <router-link to="/home" class="flex min-w-0 items-center gap-2.5">
        <HomeBrandMark :site-name="siteName" :site-logo="siteLogo" class="h-8 w-8 rounded-lg text-[15px]" />
        <span class="truncate text-[1.15rem] font-extrabold tracking-[-0.03em] text-gray-900 dark:text-white">
          {{ siteName }}
        </span>
      </router-link>

      <!-- Nav links (desktop) -->
      <div class="hidden items-center gap-1 lg:flex">
        <router-link to="/models" :class="navLinkClass">
          {{ t('home.navModels') }}
        </router-link>
        <a href="#pricing" :class="navLinkClass" @click.prevent="goToSection('pricing')">
          {{ t('home.navPricing') }}
        </a>
        <router-link v-if="showChangelog" to="/changelog" data-test="nav-changelog" :class="navLinkClass">
          {{ t('home.navChangelog') }}
        </router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" :class="navLinkClass">
          {{ t('home.docs') }}
        </a>
      </div>

      <!-- Right actions -->
      <div class="flex items-center gap-2">
        <LocaleSwitcher />

        <button
          type="button"
          class="inline-flex h-9 w-9 items-center justify-center rounded-full text-[#666] transition hover:bg-black/5 hover:text-gray-900 dark:text-white/50 dark:hover:bg-white/10 dark:hover:text-white"
          :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          @click="$emit('toggleTheme')"
        >
          <Icon v-if="isDark" name="sun" size="md" />
          <Icon v-else name="moon" size="md" />
        </button>

        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="hidden h-10 shrink-0 items-center justify-center gap-1.5 rounded-xl bg-gray-900 px-4 text-[13.5px] font-semibold text-white transition hover:bg-black dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200 sm:inline-flex"
        >
          <span
            v-if="isAuthenticated && userInitial"
            class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-white/20 text-[10px] dark:bg-black/10"
          >
            {{ userInitial }}
          </span>
          <span>{{ isAuthenticated ? t('home.dashboard') : t('home.loginConsole') }}</span>
          <Icon name="arrowRight" size="sm" class="hidden sm:block" />
        </router-link>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import HomeBrandMark from '@/components/home/HomeBrandMark.vue'
import { useHomeSectionNav } from '@/components/home/useHomeSectionNav'
import Icon from '@/components/icons/Icon.vue'

withDefaults(
  defineProps<{
    siteName: string
    siteLogo?: string
    docUrl: string
    isDark: boolean
    isAuthenticated: boolean
    dashboardPath: string
    userInitial: string
    showChangelog?: boolean
  }>(),
  {
    siteLogo: '',
    showChangelog: true,
  },
)

defineEmits<{
  toggleTheme: []
}>()

const { t } = useI18n()
const { goToSection } = useHomeSectionNav()

const navLinkClass =
  'rounded-lg px-3.5 py-2 text-[14px] font-semibold text-gray-700 transition hover:bg-black/5 hover:text-gray-900 dark:text-white/65 dark:hover:bg-white/10 dark:hover:text-white'

// 页面顶部时页头透明，和首屏融为一体；滚动后加底色和分隔线。
const scrolled = ref(false)
function syncScrolled() {
  scrolled.value = window.scrollY > 8
}
onMounted(() => {
  syncScrolled()
  window.addEventListener('scroll', syncScrolled, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('scroll', syncScrolled)
})
</script>
