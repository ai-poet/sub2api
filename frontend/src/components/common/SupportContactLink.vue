<template>
  <a
    v-if="href"
    :href="href"
    :target="external ? '_blank' : undefined"
    :rel="external ? 'noopener noreferrer' : undefined"
    data-test="support-contact"
  >{{ tx(value) }}</a>
  <span v-else data-test="support-contact">{{ tx(value) }}</span>
</template>

<script setup lang="ts">
// 客服联系方式（fork 本地）：邮箱 / 网址渲染成链接，其余原样显示；文字走内容自动翻译。
import { computed } from 'vue'
import { useContentTranslation } from '@/composables/useContentTranslation'
import { supportContactHref } from '@/utils/supportContact'

const props = defineProps<{ value: string }>()

const { tx } = useContentTranslation()

const href = computed(() => supportContactHref(props.value))
const external = computed(() => href.value !== null && !href.value.startsWith('mailto:'))
</script>
