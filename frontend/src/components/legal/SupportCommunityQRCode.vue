<template>
  <figure v-if="qrCode" class="inline-flex flex-col items-center gap-2" data-test="support-community">
    <img
      :src="qrCode"
      :alt="t('home.legal.communityHint')"
      class="rounded-lg bg-white object-contain p-1.5"
      :class="sizeClass"
    />
    <figcaption class="text-center text-xs leading-5">
      {{ t('home.legal.communityHint') }}
      <a
        v-if="groupURL"
        :href="groupURL"
        target="_blank"
        rel="noopener noreferrer"
        data-test="support-community-join"
        class="ml-1 font-medium underline underline-offset-4"
      >{{ t('nav.communityGroupJoin') }}</a>
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
// 客服交流群二维码（fork 本地）：页脚和法律页底部复用后台「交流群二维码 / 交流群链接」作为公示的客服联系方式。
// 没配置二维码时不渲染。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { supportContactHref } from '@/utils/supportContact'

withDefaults(defineProps<{ sizeClass?: string }>(), { sizeClass: 'h-28 w-28' })

const { t } = useI18n()
const appStore = useAppStore()

const qrCode = computed(() => appStore.cachedPublicSettings?.community_qr_code?.trim() || '')
// 只接受 http(s) 链接，与后台设置的校验一致
const groupURL = computed(() => {
  const href = supportContactHref(appStore.cachedPublicSettings?.community_group_url)
  return href && !href.startsWith('mailto:') ? href : ''
})
</script>
