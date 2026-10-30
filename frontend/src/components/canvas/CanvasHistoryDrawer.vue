<template>
    <div v-if="open" class="backdrop md:hidden" @click="$emit('close')" />
    <aside v-if="open" class="drawer" role="complementary">
      <header class="drawer-head">
        <h3 class="drawer-title">{{ t('canvas.history.title') }}</h3>
        <div class="drawer-head-actions">
          <button class="link-btn" :title="t('canvas.history.clearLocal')" @click="$emit('clear-local')">
            <Icon name="trash" size="xs" />
          </button>
          <button class="link-btn" @click="$emit('close')"><Icon name="x" size="sm" /></button>
        </div>
      </header>
      <p v-if="storageWarning" class="warn">{{ t('canvas.history.storageWarning') }} ({{ storageWarning }})</p>
      <p v-if="runs.length === 0" class="empty">{{ t('canvas.history.empty') }}</p>
      <div v-for="run in runs" :key="run.id" class="card">
        <div class="card-head">
          <span class="badge" :class="`st-${run.status}`">
            {{ t(`canvas.status.${run.paused ? 'paused' : run.status}`) }}
          </span>
          <span class="time">{{ formatTime(run.createdAt) }}</span>
        </div>
        <p class="prompt" :title="run.params?.prompt">{{ run.params?.prompt }}</p>
        <p class="meta">{{ run.params?.model }} · {{ run.params?.size }} · x{{ run.params?.count }}</p>
        <div class="thumbs">
          <template v-for="out in run.outputs" :key="out.index">
            <img
              v-if="thumbSrc(run.id, out.index)"
              :src="thumbSrc(run.id, out.index)!"
              class="thumb"
              :draggable="out.state === 'local' && !!out.assetKey"
              @dragstart="onDragStart($event, out)"
            />
          </template>
          <template v-for="out in run.outputs" :key="`r-${out.index}`">
            <a
              v-if="out.state === 'remote' && out.remoteUrl"
              :href="out.remoteUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="remote-link"
            ><Icon name="externalLink" size="xs" /> {{ t('canvas.history.notImportable') }}</a>
          </template>
        </div>
        <div class="actions">
          <button v-if="run.status === 'processing' && run.paused" class="act" @click="$emit('resume', run.id)">
            {{ t('canvas.history.resume') }}
          </button>
          <button
            v-if="run.status === 'completed' && run.outputs.some((o) => o.state !== 'local')"
            class="act"
            @click="$emit('retry', run.id)"
          >{{ t('canvas.history.retry') }}</button>
          <button class="act" @click="copy(run.params?.prompt)">{{ t('canvas.history.copyPrompt') }}</button>
          <button class="act" @click="$emit('apply', run)">{{ t('canvas.history.applyParams') }}</button>
          <button
            v-for="out in run.outputs.filter((o) => o.state === 'local')"
            :key="`dl-${out.index}`"
            class="act"
            @click="download(run.id, out.index)"
          >{{ t('canvas.history.download') }} {{ out.index + 1 }}</button>
          <button v-if="run.status !== 'processing'" class="act danger" @click="$emit('delete', run.id)">
            {{ t('canvas.history.delete') }}
          </button>
        </div>
      </div>
    </aside>
</template>

<script setup lang="ts">
/**
 * History drawer: durable run metadata + cached output thumbnails. Object
 * URLs are cached per run:index and revoked only when the drawer unmounts
 * or a run disappears — never on every recompute.
 */
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import type { CanvasRun, CanvasRunOutput } from '@/types/canvas'

const props = defineProps<{
  open: boolean
  runs: CanvasRun[]
  storageWarning: string
  takeOutput: (runId: string, index: number) => Promise<Blob | null>
}>()

const emit = defineEmits<{
  close: []
  resume: [runId: string]
  retry: [runId: string]
  apply: [run: CanvasRun]
  delete: [runId: string]
  'clear-local': []
}>()

const { t } = useI18n()

const urls = new Map<string, string>()
const thumbUrls = ref<Record<string, string>>({})

function thumbSrc(runId: string, index: number): string | undefined {
  return thumbUrls.value[`${runId}:${index}`]
}

async function loadThumb(runId: string, index: number): Promise<void> {
  const key = `${runId}:${index}`
  if (urls.has(key)) return
  const blob = await props.takeOutput(runId, index).catch(() => null)
  if (!blob) return
  urls.set(key, URL.createObjectURL(blob))
  thumbUrls.value = { ...thumbUrls.value, [key]: urls.get(key)! }
}

watch(
  () => [props.open, props.runs] as const,
  ([open, runs]) => {
    if (!open) return
    for (const run of runs) {
      for (const out of run.outputs) {
        if (out.state === 'local') void loadThumb(run.id, out.index)
      }
    }
    // Revoke entries whose run disappeared.
    const live = new Set(runs.map((r) => r.id))
    for (const key of [...urls.keys()]) {
      if (!live.has(key.split(':')[0])) {
        URL.revokeObjectURL(urls.get(key)!)
        urls.delete(key)
        delete thumbUrls.value[key]
      }
    }
  },
  { immediate: true, deep: true },
)

onBeforeUnmount(() => {
  for (const url of urls.values()) URL.revokeObjectURL(url)
})

function formatTime(ms: number): string {
  if (!ms) return ''
  return new Date(ms).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function copy(text?: string): void {
  if (text) void navigator.clipboard?.writeText(text)
}

function onDragStart(e: DragEvent, out: CanvasRunOutput): void {
  if (out.state !== 'local' || !out.assetKey || !e.dataTransfer) return
  e.dataTransfer.setData('application/x-canvas-asset', JSON.stringify({ assetKey: out.assetKey }))
}

async function download(runId: string, index: number): Promise<void> {
  const blob = await props.takeOutput(runId, index).catch(() => null)
  if (!blob) return
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `canvas-${runId}-${index}.png`
  a.click()
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

void emit
</script>

<style scoped>
/* Mobile: full overlay above the composer with a backdrop. */
.backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.4); z-index: 44; }
.drawer {
  position: fixed; top: 0; right: 0; bottom: 0; width: 320px; max-width: 90vw;
  background: #fff; z-index: 45; overflow-y: auto; padding: 14px;
  box-shadow: -4px 0 24px rgba(0,0,0,0.15);
}
.dark .drawer { background: #111827; }
/* Desktop: dock inside .canvas-view (its right edge is the padded-away
 * strip), so the AppHeader stays visible and the board keeps the rest. */
@media (min-width: 768px) {
  /* Fixed so it keeps hugging the viewport's right edge while the board
   * container shrinks to `right: 320px` — top clears the 64px AppHeader. */
  .drawer {
    position: fixed; top: 64px; right: 0; bottom: 0; z-index: 30;
    box-shadow: none; border-left: 1px solid rgba(0,0,0,0.08);
  }
  .dark .drawer { border-left-color: rgba(255,255,255,0.08); }
}
.drawer-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.drawer-title { font-size: 14px; font-weight: 600; }
.drawer-head-actions { display: flex; gap: 6px; }
.link-btn { color: #6b7280; padding: 4px; }
.warn { font-size: 11px; color: #b45309; margin-bottom: 8px; }
.empty { font-size: 12px; color: #9ca3af; }
.card { border: 1px solid rgba(0,0,0,0.08); border-radius: 12px; padding: 10px; margin-bottom: 10px; }
.dark .card { border-color: rgba(255,255,255,0.1); }
.card-head { display: flex; justify-content: space-between; align-items: center; }
.badge { font-size: 10px; padding: 2px 8px; border-radius: 999px; font-weight: 600; }
.st-processing { background: #dbeafe; color: #1d4ed8; }
.st-completed { background: #dcfce7; color: #15803d; }
.st-failed, .st-expired, .st-unavailable { background: #fee2e2; color: #b91c1c; }
.time { font-size: 11px; color: #9ca3af; }
.prompt { font-size: 12px; margin-top: 6px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.meta { font-size: 11px; color: #6b7280; margin-top: 2px; }
.thumbs { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.thumb { width: 56px; height: 56px; object-fit: cover; border-radius: 8px; cursor: grab; border: 1px solid rgba(0,0,0,0.08); }
.remote-link { font-size: 11px; color: #14b8a6; display: inline-flex; align-items: center; gap: 3px; }
.actions { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
.act { font-size: 11px; color: #14b8a6; padding: 2px 6px; border-radius: 6px; }
.act:hover { background: rgba(20,184,166,0.1); }
.act.danger { color: #dc2626; }
</style>
