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
        <span class="outline-button">{{ t('home.clientWorkflow.image.openFolder') }}</span>
      </div>
    </div>

    <!-- 图库 -->
    <div class="min-h-0 flex-1 overflow-hidden px-4 sm:px-5">
      <div class="mx-auto flex max-w-[984px] flex-wrap gap-3.5 pt-1">
        <div v-if="frame.imageJob !== 'idle'" class="card" data-test="preview-image-job">
          <div
            v-if="frame.imageJob === 'drawing'"
            class="square running relative flex flex-col items-center justify-center gap-2 px-3.5 text-center"
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
          <div v-else class="square picture art-cat reveal" data-test="preview-image-done">
            <span class="cat-glow"></span>
            <span class="cat-head"></span>
            <span class="cat-laptop"></span>
          </div>
          <Caption :prompt="t('home.clientWorkflow.image.prompt')" :meta="NEW_META" />
        </div>

        <div v-for="picture in GALLERY" :key="picture.key" class="card">
          <div class="square picture" :class="picture.art"></div>
          <Caption :prompt="t(`home.clientWorkflow.image.gallery.${picture.key}`)" :meta="picture.meta" />
        </div>
      </div>
    </div>

    <!-- 画图输入框 -->
    <div class="flex-none px-4 pb-4 pt-2 sm:px-5">
      <div class="composer mx-auto flex max-w-[720px] flex-col gap-2 rounded-[13px] py-2.5">
        <div class="min-h-[40px] px-3.5 text-[14px] leading-5">
          <span v-if="typedPrompt" class="text-[color:var(--cw-text)]">{{ typedPrompt }}<i class="caret"></i></span>
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
          <span class="generate ml-1" :class="typedPrompt ? '' : 'opacity-[0.55]'">{{ t('home.clientWorkflow.image.generate') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import ClientIcon from './ClientIcon.vue'
import type { PreviewFrame } from './timeline'

const props = defineProps<{
  frame: PreviewFrame
  balance: string
}>()

const { t } = useI18n()

// 图库里已有的作品；图片用 CSS 画，不引入图片资源
const GALLERY = [
  { key: 'sunset', art: 'art-sunset', meta: 'gpt-image-2 · 1536×1024 · $0.06' },
  { key: 'mountain', art: 'art-mountain', meta: 'gpt-image-2 · 1024×1024 · $0.04' },
  { key: 'city', art: 'art-city', meta: 'grok-imagine-image · 1024×1024' },
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
.card {
  width: 140px;
}

@media (min-width: 640px) {
  .card {
    width: 168px;
  }
}

.square {
  aspect-ratio: 1 / 1;
  width: 100%;
  overflow: hidden;
  border-radius: 10px;
}

.running {
  border: 1px solid var(--cw-border-strong);
  background: var(--cw-raised);
}

.picture {
  position: relative;
  border: 1px solid var(--cw-border);
  background-color: var(--cw-inset);
}

.composer {
  border: 1px solid var(--cw-border);
  background: var(--cw-composer);
}

.outline-button {
  display: inline-flex;
  height: 26px;
  align-items: center;
  border-radius: 7px;
  border: 1px solid var(--cw-border-strong);
  padding: 0 10px;
  font-size: 11.5px;
  color: var(--cw-text-secondary);
}

.generate {
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

.caret {
  display: inline-block;
  width: 1px;
  height: 16px;
  margin-left: 1px;
  vertical-align: -3px;
  background: var(--cw-accent);
}

/* ===== CSS 画的作品 ===== */

.art-sunset {
  background:
    radial-gradient(circle at 50% 58%, #fff3c4 0 11%, rgba(255, 214, 140, 0.8) 12%, transparent 26%),
    repeating-linear-gradient(0deg, rgba(255, 255, 255, 0.12) 0 2px, transparent 2px 9px) bottom / 100% 40% no-repeat,
    linear-gradient(180deg, #ffb36b 0%, #ff7a7a 45%, #7a4fd6 62%, #2c2a6e 100%);
}

.art-mountain {
  background:
    radial-gradient(circle at 20% 18%, #fff 0 1px, transparent 2px),
    radial-gradient(circle at 70% 12%, #fff 0 1px, transparent 2px),
    radial-gradient(circle at 45% 30%, #fff 0 1px, transparent 2px),
    radial-gradient(circle at 85% 32%, #fff 0 1px, transparent 2px),
    radial-gradient(circle at 32% 8%, #fff 0 1px, transparent 2px),
    linear-gradient(180deg, #0b1633 0%, #1d3766 55%, #3d5f8f 100%);
}

.art-mountain::before {
  content: '';
  position: absolute;
  inset: 38% 0 0;
  background: linear-gradient(180deg, #e9eef7 0%, #9fb2cf 45%, #4a5f80 100%);
  clip-path: polygon(0 70%, 22% 30%, 34% 48%, 55% 0, 76% 42%, 88% 26%, 100% 55%, 100% 100%, 0 100%);
}

.art-mountain::after {
  content: '';
  position: absolute;
  bottom: 10%;
  left: 44%;
  width: 16%;
  height: 12%;
  background: #f59e42;
  clip-path: polygon(50% 0, 100% 100%, 0 100%);
  box-shadow: 0 0 18px #f59e42;
}

.art-city {
  background:
    linear-gradient(90deg, transparent 0 8%, #2b1f4a 8% 22%, transparent 22% 30%, #3a2a63 30% 46%, transparent 46% 54%, #22183d 54% 72%, transparent 72% 80%, #33245a 80% 94%, transparent 94%) bottom / 100% 70% no-repeat,
    repeating-linear-gradient(100deg, rgba(160, 200, 255, 0.18) 0 1px, transparent 1px 7px),
    linear-gradient(180deg, #0d0a1f 0%, #3b1450 55%, #e0418f 100%);
}

.art-city::after {
  content: '';
  position: absolute;
  inset: auto 12% 30% 12%;
  height: 6%;
  border-radius: 3px;
  background: linear-gradient(90deg, #22d3ee, #e0418f);
  box-shadow: 0 0 14px #e0418f;
}

.art-cat {
  background:
    radial-gradient(circle at 80% 18%, rgba(34, 211, 238, 0.55), transparent 32%),
    radial-gradient(circle at 18% 22%, rgba(224, 65, 143, 0.55), transparent 36%),
    linear-gradient(180deg, #120d2b 0%, #24124a 60%, #0c0a1c 100%);
}

.cat-glow {
  position: absolute;
  inset: auto 10% 12% 10%;
  height: 30%;
  border-radius: 50%;
  background: radial-gradient(ellipse at center, rgba(34, 211, 238, 0.45), transparent 70%);
}

.cat-head {
  position: absolute;
  left: 30%;
  top: 26%;
  width: 40%;
  height: 34%;
  border-radius: 48% 48% 44% 44%;
  background:
    radial-gradient(circle at 34% 50%, #22d3ee 0 7%, transparent 8%),
    radial-gradient(circle at 66% 50%, #22d3ee 0 7%, transparent 8%),
    linear-gradient(180deg, #f8a54b, #e7802c);
}

.cat-head::before {
  content: '';
  position: absolute;
  left: 2%;
  right: 2%;
  top: -26%;
  height: 34%;
  background: #f29a41;
  clip-path: polygon(0 100%, 16% 0, 36% 100%, 64% 100%, 84% 0, 100% 100%);
}

.cat-laptop {
  position: absolute;
  left: 22%;
  right: 22%;
  bottom: 16%;
  height: 20%;
  border-radius: 4px 4px 2px 2px;
  background: linear-gradient(180deg, #1f2a44, #0f172a);
  box-shadow: 0 -2px 16px rgba(34, 211, 238, 0.55);
}

.cat-laptop::after {
  content: '';
  position: absolute;
  left: 18%;
  right: 18%;
  top: 30%;
  height: 12%;
  border-radius: 2px;
  background: repeating-linear-gradient(90deg, #22d3ee 0 6px, transparent 6px 9px);
  opacity: 0.8;
}

.reveal {
  animation: reveal 0.6s ease-out;
}

@keyframes reveal {
  from {
    opacity: 0;
    filter: blur(6px);
  }
  to {
    opacity: 1;
    filter: blur(0);
  }
}
</style>
