<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Background Decoration -->
    <div class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content：合规声明未确认前不挂载页面内容，避免首屏一批管理接口同时 423 刷出成串错误提示 -->
      <main class="p-4 md:p-6 lg:p-8">
        <slot v-if="!complianceBlocking" />
        <div v-else class="flex min-h-[40vh] items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('legal.adminCompliance') }}
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminComplianceStore } from '@/stores/adminCompliance'
import { useI18n } from 'vue-i18n'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const complianceStore = useAdminComplianceStore()
const { t } = useI18n()
// 管理员首次进入需要确认部署合规声明；弹窗期间不渲染页面内容（见模板注释）。运维管理员不要求确认。
const complianceBlocking = computed(() => authStore.isAdmin && complianceStore.shouldShow)
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
