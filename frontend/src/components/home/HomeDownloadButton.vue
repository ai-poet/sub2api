<template>
  <div v-if="selected" class="flex w-full flex-col items-center gap-3">
    <div
      class="inline-flex max-w-full items-stretch overflow-hidden rounded-xl bg-gray-900 text-white shadow-[0_14px_36px_rgba(15,17,20,0.22)] dark:bg-white dark:text-gray-900 dark:shadow-[0_14px_36px_rgba(0,0,0,0.5)]"
    >
      <button
        v-if="selected.type === 'command'"
        type="button"
        data-test="hero-primary-download"
        :data-platform="selected.id"
        class="inline-flex h-14 min-w-0 items-center gap-2.5 px-6 text-[15px] font-bold transition hover:bg-black dark:hover:bg-gray-100"
        @click="copyCommand"
      >
        <Icon name="clipboard" size="sm" class="shrink-0" />
        <span class="truncate">{{ t('home.landing.hero.copyInstall', { platform: selected.name }) }}</span>
      </button>
      <a
        v-else
        :href="selected.url"
        data-test="hero-primary-download"
        :data-platform="selected.id"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex h-14 min-w-0 items-center gap-2.5 px-6 text-[15px] font-bold transition hover:bg-black dark:hover:bg-gray-100"
      >
        <Icon name="download" size="sm" class="shrink-0" />
        <span class="truncate">{{ t('home.landing.hero.downloadFor', { platform: selected.name }) }}</span>
      </a>

      <!-- 另一平台：点一下切换；只有一个平台时这里只标示当前平台 -->
      <button
        v-for="option in otherOptions"
        :key="option.id"
        type="button"
        data-test="hero-platform-switch"
        :data-platform="option.id"
        :title="t('home.landing.hero.switchPlatform', { platform: option.name })"
        :aria-label="t('home.landing.hero.switchPlatform', { platform: option.name })"
        class="flex w-14 shrink-0 items-center justify-center border-l border-white/15 transition hover:bg-white/10 dark:border-black/10 dark:hover:bg-black/5"
        @click="selectedId = option.id"
      >
        <HomeOsIcon :platform="option.id" class="h-5 w-5" />
      </button>
      <span
        v-if="otherOptions.length === 0"
        class="flex w-14 shrink-0 items-center justify-center border-l border-white/15 dark:border-black/10"
      >
        <HomeOsIcon :platform="selected.id" class="h-5 w-5" />
      </span>
    </div>

    <div
      v-if="selected.type === 'command'"
      data-test="hero-install-command"
      class="flex w-full max-w-[34rem] items-center gap-2 rounded-xl border border-black/10 bg-white/85 p-1.5 pl-4 font-mono text-[13px] text-gray-700 dark:border-white/10 dark:bg-white/5 dark:text-white/80"
    >
      <code class="min-w-0 flex-1 truncate text-left">{{ selected.url }}</code>
      <button
        type="button"
        class="shrink-0 rounded-lg bg-gray-900 px-3 py-1.5 font-sans text-xs font-semibold text-white transition hover:bg-black dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
        @click="copyCommand"
      >
        {{ t('home.hero.installPrimary') }}
      </button>
    </div>
    <p v-if="selected.type === 'command'" class="text-xs text-gray-500 dark:text-white/45">
      {{ t('home.landing.hero.installHint') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import HomeOsIcon from '@/components/home/HomeOsIcon.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { ClientDownloadOption } from '@/utils/clientDownloads'

// 下载按钮：主段按访客系统给出下载链接（Windows）或复制安装命令（macOS），
// 右侧小段切换到另一个平台。options 为空时整个组件不渲染。
const props = defineProps<{
  options: ClientDownloadOption[]
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const selectedId = ref(props.options[0]?.id)
watch(
  () => props.options.map((option) => option.id).join(','),
  () => {
    if (!props.options.some((option) => option.id === selectedId.value)) {
      selectedId.value = props.options[0]?.id
    }
  },
)

const selected = computed(
  () => props.options.find((option) => option.id === selectedId.value) ?? props.options[0] ?? null,
)
const otherOptions = computed(() => props.options.filter((option) => option.id !== selected.value?.id))

function copyCommand() {
  if (selected.value?.type === 'command') {
    copyToClipboard(selected.value.url, t('home.download.commandCopied'))
  }
}
</script>
