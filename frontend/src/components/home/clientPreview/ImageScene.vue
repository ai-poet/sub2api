<template>
  <div class="flex h-full flex-col" data-test="preview-image-studio">
    <!-- 工具栏：张数、余额、打开文件夹 -->
    <div class="flex-none px-4 pb-2 pt-1 sm:px-5">
      <div class="mx-auto flex max-w-[984px] items-center gap-2">
        <span class="text-[12px] text-[color:var(--cw-text-tertiary)]">
          {{ t('home.clientWorkflow.image.pictureCount', { count: pictureCount }) }}
        </span>
        <span class="flex-1"></span>
        <span class="cw-chip tabular-nums">
          <ClientIcon name="wallet" class="h-3 w-3" />
          <span>{{ balance }}</span>
        </span>
        <span class="cw-outline-button">{{ t('home.clientWorkflow.image.openFolder') }}</span>
      </div>
    </div>

    <!-- 图库 -->
    <div class="min-h-0 flex-1 overflow-hidden px-4 sm:px-5">
      <div class="mx-auto flex max-w-[984px] flex-wrap gap-3.5 pt-1">
        <div v-if="frame.imageJob !== 'idle'" class="cw-card" data-test="preview-image-job">
          <div
            v-if="frame.imageJob === 'drawing'"
            class="cw-square cw-running relative flex flex-col items-center justify-center gap-2 px-3.5 text-center"
          >
            <ClientIcon name="loaderCircle" class="cw-spin h-[18px] w-[18px] text-[color:var(--cw-text-secondary)]" />
            <span class="text-[13px] font-medium text-[color:var(--cw-text)]">
              {{ t('home.clientWorkflow.image.drawing', { time: drawTime }) }}
            </span>
            <span class="text-[11.5px] leading-4 text-[color:var(--cw-text-tertiary)]">
              {{ t('home.clientWorkflow.image.runningTask') }}
            </span>
            <span class="cw-icon-button absolute right-1.5 top-1.5">
              <ClientIcon name="x" class="h-3.5 w-3.5" />
            </span>
          </div>
          <DemoPicture
            v-else
            class="cw-reveal"
            image-key="cat"
            :alt="t('home.clientWorkflow.image.prompt')"
            data-test="preview-image-done"
          />
          <Caption :prompt="t('home.clientWorkflow.image.prompt')" :meta="NEW_META" />
        </div>

        <div v-for="picture in GALLERY" :key="picture.key" class="cw-card" data-test="preview-gallery-card">
          <DemoPicture :image-key="picture.key" :alt="t(`home.clientWorkflow.image.gallery.${picture.key}`)" />
          <Caption :prompt="t(`home.clientWorkflow.image.gallery.${picture.key}`)" :meta="picture.meta" />
        </div>
      </div>
    </div>

    <!-- 画图输入框 -->
    <div class="flex-none px-4 pb-4 pt-2 sm:px-5">
      <div class="cw-composer mx-auto flex max-w-[720px] flex-col gap-2 rounded-[13px] py-2.5">
        <div class="min-h-[40px] px-3.5 text-[14px] leading-5">
          <span v-if="typedPrompt" class="text-[color:var(--cw-text)]">{{ typedPrompt }}<i class="cw-caret"></i></span>
          <span v-else class="text-[color:var(--cw-text-ghost)]">{{ t('home.clientWorkflow.image.placeholder') }}</span>
        </div>
        <div class="flex items-center gap-0.5 px-2">
          <span class="cw-icon-button" :title="t('home.clientWorkflow.image.addImages')">
            <ClientIcon name="image" class="h-3.5 w-3.5" />
          </span>
          <span class="cw-menu-chip">gpt-image-2<ClientIcon name="chevronDown" class="h-2.5 w-2.5" /></span>
          <span class="cw-menu-chip max-sm:hidden">{{ t('home.clientWorkflow.image.group') }}<ClientIcon name="chevronDown" class="h-2.5 w-2.5" /></span>
          <span class="cw-menu-chip max-md:hidden">1024×1024<ClientIcon name="chevronDown" class="h-2.5 w-2.5" /></span>
          <span class="cw-menu-chip max-md:hidden">{{ t('home.clientWorkflow.image.quality') }}<ClientIcon name="chevronDown" class="h-2.5 w-2.5" /></span>
          <span class="cw-menu-chip max-sm:hidden">{{ t('home.clientWorkflow.image.count') }}<ClientIcon name="chevronDown" class="h-2.5 w-2.5" /></span>
          <span class="flex-1"></span>
          <span class="px-1.5 text-[11.5px] text-[color:var(--cw-text-tertiary)] max-sm:hidden">{{ t('home.clientWorkflow.image.mode') }}</span>
          <span class="px-1 text-[11.5px] text-[color:var(--cw-text-secondary)]">{{ t('home.clientWorkflow.image.estimate') }}</span>
          <span class="cw-generate ml-1" :class="typedPrompt ? '' : 'opacity-[0.55]'">{{ t('home.clientWorkflow.image.generate') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import { DEMO_IMAGES, type DemoImageKey } from './demoImages'
import type { PreviewFrame } from './timeline'

const props = defineProps<{
  frame: PreviewFrame
  balance: string
}>()

const { t } = useI18n()

// 图库里已有的作品
const GALLERY: Array<{ key: Exclude<DemoImageKey, 'cat'>; meta: string }> = [
  { key: 'sunset', meta: 'gpt-image-2 · 1536×1024 · $0.06' },
  { key: 'mountain', meta: 'gpt-image-2 · 1024×1024 · $0.04' },
  { key: 'city', meta: 'grok-imagine-image · 1024×1024' },
]
const NEW_META = 'gpt-image-2 · 1024×1024 · $0.04'

const pictureCount = computed(() => GALLERY.length + (props.frame.imageJob === 'done' ? 1 : 0))

// 提示词逐字打出来，提交后输入框清空
const typedPrompt = computed(() => {
  if (props.frame.submitted || props.frame.typed <= 0) return ''
  const chars = Array.from(t('home.clientWorkflow.image.prompt'))
  return chars.slice(0, Math.ceil(chars.length * props.frame.typed)).join('')
})

// 客户端的计时格式是 m:ss
const drawTime = computed(() => `0:${String(props.frame.drawSeconds).padStart(2, '0')}`)

// 一张方图；加载失败时换成中性的底色块，不显示裂图
const DemoPicture = defineComponent({
  props: {
    imageKey: { type: String as PropType<DemoImageKey>, required: true },
    alt: { type: String, required: true },
  },
  setup(pictureProps) {
    const failed = ref(false)
    return () =>
      h('div', { class: 'cw-square cw-picture' }, [
        failed.value
          ? h('div', { class: 'cw-picture-fallback', 'data-test': 'preview-picture-fallback' })
          : h('img', {
              src: DEMO_IMAGES[pictureProps.imageKey],
              alt: pictureProps.alt,
              width: 400,
              height: 400,
              decoding: 'async',
              fetchpriority: 'low',
              referrerpolicy: 'no-referrer',
              draggable: false,
              class: 'h-full w-full object-cover',
              onError: () => {
                failed.value = true
              },
            }),
      ])
  },
})

const Caption = defineComponent({
  props: {
    prompt: { type: String, required: true },
    meta: { type: String, required: true },
  },
  setup(captionProps) {
    return () =>
      h('div', { class: 'mt-1.5 flex flex-col gap-0.5' }, [
        h('div', { class: 'flex items-center gap-1' }, [
          h('span', { class: 'min-w-0 flex-1 truncate text-[12.5px] text-[color:var(--cw-text)]' }, captionProps.prompt),
          h(ClientIcon, { name: 'ellipsis', class: 'h-3.5 w-3.5 shrink-0 text-[color:var(--cw-text-tertiary)]' }),
        ]),
        h('div', { class: 'truncate text-[11.5px] text-[color:var(--cw-text-tertiary)]' }, captionProps.meta),
      ])
  },
})
</script>

<style scoped>
.cw-card {
  width: 140px;
}

@media (min-width: 640px) {
  .cw-card {
    width: 168px;
  }
}

.cw-card :deep(.cw-square) {
  aspect-ratio: 1 / 1;
  width: 100%;
  overflow: hidden;
  border-radius: 10px;
}

.cw-running {
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-raised);
}

.cw-card :deep(.cw-picture) {
  position: relative;
  border: 1px solid var(--cw-border);
  background-color: var(--cw-inset);
}

/* 图片加载失败时的底色块 */
.cw-card :deep(.cw-picture-fallback) {
  height: 100%;
  width: 100%;
  background: linear-gradient(135deg, var(--cw-raised), var(--cw-inset));
}

.cw-generate {
  display: inline-flex;
  height: 26px;
  align-items: center;
  border-radius: 7px;
  padding: 0 10px;
  font-size: 11.5px;
  font-weight: 500;
  background: var(--cw-inverse);
  color: var(--cw-on-inverse);
  transition: opacity 0.2s ease;
}

/* 刚画好的图从模糊到清晰 */
@media (prefers-reduced-motion: no-preference) {
  .cw-reveal {
    animation: reveal 0.6s ease-out;
  }
}

@keyframes reveal {
  from {
    opacity: 0;
    filter: blur(8px);
  }
  to {
    opacity: 1;
    filter: blur(0);
  }
}
</style>
