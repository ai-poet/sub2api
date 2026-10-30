<template>
  <div class="composer" :class="{ collapsed }">
    <div class="composer-row">
      <div class="seg" role="radiogroup">
        <button
          v-for="op in operations"
          :key="op"
          class="seg-btn"
          :class="{ active: operation === op }"
          @click="updateField('operation', op)"
        >{{ t(`canvas.composer.${op}`) || opLabel(op) }}</button>
      </div>
      <button class="collapse-btn" @click="collapsed = !collapsed">
        {{ collapsed ? t('canvas.composer.expandParams') : t('canvas.composer.collapseParams') }}
      </button>
    </div>
    <div v-show="!collapsed" class="composer-body">
      <textarea
        v-model="promptText"
        class="prompt-input"
        rows="2"
        :placeholder="t('canvas.composer.promptPlaceholder')"
        @keydown="onKeydown"
      />
      <div class="param-row">
        <select :value="apiKeyId" class="param" @change="updateField('apiKeyId', Number(($event.target as HTMLSelectElement).value))">
          <option :value="null" disabled>{{ t('canvas.composer.apiKeyPlaceholder') }}</option>
          <option v-for="k in eligibleKeys" :key="k.id" :value="k.id">{{ k.name }}</option>
        </select>
        <select :value="model" class="param" :disabled="imageModels.length === 0" @change="updateField('model', ($event.target as HTMLSelectElement).value)">
          <option v-if="imageModels.length === 0" value="">{{ t('canvas.composer.modelEmpty') }}</option>
          <option v-for="m in imageModels" :key="m" :value="m">{{ m }}</option>
        </select>
        <select :value="size" class="param" @change="updateField('size', ($event.target as HTMLSelectElement).value)">
          <option v-for="s in ['1024x1024','1024x1536','1536x1024']" :key="s" :value="s">{{ s }}</option>
        </select>
        <select :value="quality" class="param" @change="updateField('quality', ($event.target as HTMLSelectElement).value)">
          <option v-for="q in ['','low','medium','high']" :key="q" :value="q">{{ t(`canvas.quality.${q || 'default'}`) }}</option>
        </select>
        <select :value="count" class="param narrow" @change="updateField('count', Number(($event.target as HTMLSelectElement).value))">
          <option v-for="n in [1,2,3,4]" :key="n" :value="n">{{ n }}</option>
        </select>
        <button class="submit-btn" :disabled="submitting || !ready" @click="$emit('submit')">
          {{ submitting ? t('canvas.composer.submitting') : t('canvas.composer.submit') }}
        </button>
      </div>
      <p class="guide">{{ guide }}</p>
      <p v-if="operation !== 'generate'" class="guide dim">{{ t('canvas.composer.annotationsNote') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * Floating composer: operation segmented control, prompt, and the
 * key/model/size/quality/count params bound straight to the studio refs.
 * Submit emits once; the in-flight POST lock lives in useCanvasStudio.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ApiKey } from '@/types'
import type { CanvasOperation } from '@/types/canvas'

const props = defineProps<{
  operation: CanvasOperation
  prompt: string
  apiKeyId: number | null
  model: string
  size: string
  quality: string
  count: number
  eligibleKeys: ApiKey[]
  imageModels: string[]
  submitting: boolean
  ready: boolean
}>()

const emit = defineEmits<{
  'update:operation': [v: CanvasOperation]
  'update:prompt': [v: string]
  'update:apiKeyId': [v: number]
  'update:model': [v: string]
  'update:size': [v: string]
  'update:quality': [v: string]
  'update:count': [v: number]
  submit: []
}>()

const { t } = useI18n()
const collapsed = ref(typeof window !== 'undefined' && window.innerWidth < 768)
const operations: CanvasOperation[] = ['generate', 'edit', 'inpaint']

const promptText = computed({
  get: () => props.prompt,
  set: (v: string) => emit('update:prompt', v),
})

function opLabel(op: CanvasOperation): string {
  return op
}

function updateField(field: string, value: unknown): void {
  emit(`update:${field}` as never, value as never)
}

const guide = computed(() => {
  if (props.operation === 'edit') return t('canvas.composer.guideEdit')
  if (props.operation === 'inpaint') return t('canvas.composer.guideInpaint')
  return t('canvas.composer.guideGenerate')
})

function onKeydown(e: KeyboardEvent): void {
  // Cmd/Ctrl+Enter submits; never while an IME composition is active.
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && !e.isComposing) {
    e.preventDefault()
    emit('submit')
  }
}
</script>

<style scoped>
.composer {
  position: absolute;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  width: min(680px, calc(100% - 32px));
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.95);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.14);
  backdrop-filter: blur(10px);
  padding: 10px 14px;
  z-index: 20;
}
.dark .composer { background: rgba(17, 24, 39, 0.95); }
.composer-row { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.seg { display: flex; gap: 4px; background: rgba(0,0,0,0.05); border-radius: 10px; padding: 3px; }
.dark .seg { background: rgba(255,255,255,0.06); }
.seg-btn { padding: 4px 12px; border-radius: 8px; font-size: 13px; color: #4b5563; }
.dark .seg-btn { color: #9ca3af; }
.seg-btn.active { background: #14b8a6; color: #fff; }
.collapse-btn { font-size: 12px; color: #6b7280; }
.composer-body { margin-top: 8px; }
.prompt-input {
  width: 100%; resize: none; border: 1px solid rgba(0,0,0,0.1); border-radius: 10px;
  padding: 8px 10px; font-size: 14px; background: transparent; color: inherit;
}
.dark .prompt-input { border-color: rgba(255,255,255,0.12); }
.param-row { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; align-items: center; }
.param {
  font-size: 12px; padding: 5px 8px; border-radius: 8px;
  border: 1px solid rgba(0,0,0,0.1); background: transparent; color: inherit;
  max-width: 160px;
}
.dark .param { border-color: rgba(255,255,255,0.12); background: #1f2937; }
.param.narrow { width: 60px; }
.submit-btn {
  margin-left: auto; padding: 7px 20px; border-radius: 10px;
  background: #14b8a6; color: #fff; font-size: 13px; font-weight: 600;
}
.submit-btn:disabled { opacity: 0.5; }
.guide { font-size: 11px; color: #6b7280; margin-top: 6px; }
.guide.dim { opacity: 0.8; }
@media (max-width: 640px) {
  .param-row .param { flex: 1 1 40%; max-width: none; }
}
</style>
