<template>
  <div class="mt-10 border-t border-black/8 pt-8 dark:border-white/10" data-test="legal-contact">
    <p class="text-sm leading-7 text-[#999] dark:text-white/38">{{ lead }}</p>
    <ul class="mt-3 space-y-1 text-sm leading-7 text-gray-600 dark:text-white/65">
      <li v-if="contactInfo">
        {{ t('home.legal.contactLabel') }}<SupportContactLink :value="contactInfo" :class="linkClass" />
      </li>
      <li>
        {{ t('home.legal.onlineSupport') }}
        <RouterLink to="/tickets" data-test="legal-contact-tickets" :class="linkClass">
          {{ t('home.legal.onlineSupportLink') }}
        </RouterLink>
      </li>
    </ul>
    <SupportCommunityQRCode class="mt-4 text-[#999] dark:text-white/38" size-class="h-32 w-32 ring-1 ring-black/8 dark:ring-white/10" />
  </div>
</template>

<script setup lang="ts">
// 服务条款 / 隐私政策 / 使用规范页底部的联系方式（fork 本地）：
// 管理后台配置的客服联系方式与交流群二维码（有才显示）+ 始终可用的在线工单入口。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SupportContactLink from '@/components/common/SupportContactLink.vue'
import SupportCommunityQRCode from '@/components/legal/SupportCommunityQRCode.vue'
import { useAppStore } from '@/stores/app'

defineProps<{ lead: string }>()

const { t } = useI18n()
const appStore = useAppStore()

const contactInfo = computed(() => appStore.contactInfo.trim())

const linkClass =
  'font-medium text-primary-600 underline-offset-4 hover:underline dark:text-primary-400'
</script>
