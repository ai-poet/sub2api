<template>
  <div v-if="hasQRCode" class="relative" ref="containerRef">
    <!-- 交流群按钮 -->
    <button
      @click="toggleTooltip"
      class="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-600 transition-all hover:bg-gray-100 hover:scale-105 dark:text-gray-400 dark:hover:bg-dark-800"
      :aria-label="t('nav.communityGroup')"
    >
      <Icon name="users" size="md" />
    </button>

    <!-- 二维码 Tooltip -->
    <Transition name="tooltip-slide">
      <div
        v-if="showTooltip"
        class="absolute right-0 top-full mt-2 z-50 w-64"
      >
        <!-- 小三角箭头 -->
        <div class="absolute -top-1.5 right-4 h-3 w-3 rotate-45 bg-white dark:bg-dark-800"></div>
        <!-- Tooltip 内容 -->
        <div class="relative overflow-hidden rounded-xl bg-white p-4 shadow-lg ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700">
          <p class="mb-3 text-center text-sm font-medium text-gray-900 dark:text-white">
            {{ t('nav.communityGroupTooltip') }}
          </p>
          <img
            :src="qrCodeUrl"
            alt="Community QR Code"
            class="mx-auto block h-48 w-48 rounded-lg object-contain"
          />
          <p class="mt-2 text-center text-xs text-gray-500 dark:text-gray-400">
            {{ t('nav.communityGroupScanHint') }}
          </p>
          <div v-if="communityGroupURL" class="mt-3 flex justify-center">
            <a
              :href="communityGroupURL"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-1 rounded-lg bg-primary-50 px-3 py-1.5 text-xs font-medium text-primary-700 transition-colors hover:bg-primary-100 dark:bg-primary-900/20 dark:text-primary-300"
            >
              <Icon name="externalLink" size="xs" />
              {{ t('nav.communityGroupJoin') }}
            </a>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const showTooltip = ref(false)
const containerRef = ref<HTMLElement | null>(null)

const qrCodeUrl = computed(() => appStore.cachedPublicSettings?.community_qr_code || '')
const communityGroupURL = computed(() => appStore.cachedPublicSettings?.community_group_url || '')
const hasQRCode = computed(() => !!qrCodeUrl.value)

function toggleTooltip() {
  showTooltip.value = !showTooltip.value
}

function closeTooltip() {
  showTooltip.value = false
}

function handleClickOutside(event: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    closeTooltip()
  }
}

function handleEscape(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    closeTooltip()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleEscape)
})
</script>
