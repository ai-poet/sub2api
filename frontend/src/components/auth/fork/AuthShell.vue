<template>
  <div class="fork-auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- Background：页面底色 + 点阵背景动画 -->
    <div class="fork-auth-backdrop pointer-events-none fixed inset-0"></div>
    <AuthBackground />

    <!-- Content Container -->
    <div class="fork-content-reveal relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-xl shadow-lg shadow-primary-500/30 dark:shadow-none"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <!-- 品牌标题在浅色和深色主题下保持清晰对比。 -->
          <h1 class="mb-2 text-2xl font-bold text-gray-900 dark:text-white">
            {{ siteName }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ tx(siteSubtitle) }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div class="fork-auth-card rounded-xl p-8">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. {{ t('common.rightsReserved') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：登录 / 注册等认证页的外壳（移植自 TokenRouter components/layout/AuthLayout.vue 及其样式）。
// 上游的 components/layout/AuthLayout.vue 只负责挂载本组件。
// TokenRouter 的认证页是中性色调：近白页面底色、灰色点阵背景动画、半透明白卡片 + 细边框且没有阴影、
// 标题不用渐变；卡片里的按钮和输入框是平面风格（36px 高、8px 圆角、纯色主按钮）。
// 这些控件样式只在本外壳内生效（下方 .fork-auth-shell 前缀），不改全站的 style.css。
// 深色值取 TokenRouter 深色色阶的原值：认证页底色与本项目相同（#0A0A0B），卡片和控件按原样还原。
// 本项目保留：副标题走内容自动翻译（tx）。
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AuthBackground from './AuthBackground.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import { useContentTranslation } from '@/composables/useContentTranslation'

const { t } = useI18n()
const appStore = useAppStore()
const { tx } = useContentTranslation()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style>
/* 以下样式都挂在 .fork-auth-shell 下，只影响认证页。取值对应 TokenRouter style.css 的
   --page-bg、.card-glass、.btn / .btn-primary / .btn-secondary、.input、.input-label、.input-hint。 */

.fork-auth-shell,
.fork-auth-backdrop {
  background: #fcfcfe;
}

.dark .fork-auth-shell,
.dark .fork-auth-backdrop {
  background: #0a0a0b;
}

/* 卡片：半透明白 + 背景模糊 + 细边框，没有阴影 */
.fork-auth-card {
  @apply border border-primary-900/10 bg-white/75 backdrop-blur-xl;
  box-shadow: none;
}

.dark .fork-auth-card {
  background-color: #0f0f10;
  border-color: #3d3d42;
}

/* 按钮：36px 高、8px 圆角、无阴影 */
.fork-auth-shell .btn {
  min-height: 2.25rem;
  padding-top: 0.375rem;
  padding-bottom: 0.375rem;
  border-radius: 0.5rem;
  box-shadow: none;
  transition-duration: var(--motion-fast, 150ms);
}

.dark .fork-auth-shell .btn:focus {
  --tw-ring-offset-color: #0f0f10;
}

/* 主按钮：纯色，不用渐变和彩色阴影 */
.fork-auth-shell .btn-primary {
  @apply bg-primary-600 text-white hover:bg-primary-700;
  background-image: none;
  box-shadow: none;
}

.fork-auth-shell .btn-secondary {
  @apply border border-primary-900/10 bg-white text-primary-900 hover:border-black/20 hover:bg-gray-50 focus:border-primary-900/10 focus:ring-black/10;
  box-shadow: none;
}

.dark .fork-auth-shell .btn-secondary {
  background-color: #141416;
  border-color: #3d3d42;
  color: #fafafa;
}

.dark .fork-auth-shell .btn-secondary:hover {
  background-color: #27272a;
  border-color: #5f5f67;
}

/* 输入框：36px 高、8px 圆角、细边框，聚焦时用中性描边；只改上下内边距，左右留给带图标的输入框 */
.fork-auth-shell .input {
  @apply border-primary-900/10 bg-white placeholder:text-primary-900/45 focus:border-primary-900/10 focus:ring-black/10;
  min-height: 2.25rem;
  padding-top: 0.375rem;
  padding-bottom: 0.375rem;
  border-radius: 0.5rem;
}

.dark .fork-auth-shell .input {
  background-color: #141416;
  border-color: #3d3d42;
  color: #fafafa;
}

.dark .fork-auth-shell .input::placeholder {
  color: #8b8b94;
}

.dark .fork-auth-shell .input:focus {
  border-color: #8b8b94;
  --tw-ring-color: rgb(255 255 255 / 0.06);
}

.fork-auth-shell .input-label {
  @apply text-primary-900;
}

.dark .fork-auth-shell .input-label {
  color: #fafafa;
}

.fork-auth-shell .input-hint {
  @apply text-primary-900/80;
}

.dark .fork-auth-shell .input-hint {
  color: #a1a1aa;
}
</style>
