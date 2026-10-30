<template>
  <AppLayout :full-viewport="true">
    <div class="canvas-view" :class="{ 'drawer-open': historyOpen }">
      <CanvasEditor
        v-if="studio.ready.value && studio.scope.value"
        :key="editorKey"
        ref="editor"
        :scope="studio.scope.value"
        :operation="studio.operation.value"
        :tool="tool"
        :brush-size="brushSize"
        :brush-shape="brushShape"
        :text-color="textColor"
        :text-size="textSize"
        :default-text="t('canvas.textDefault')"
        :crop-handler="onCropRequest"
        @state="onEditorState"
        @error="onEditorError"
        @update:tool="tool = $event"
      />
      <CanvasToolbar
        v-model:tool="tool"
        :brush-size="brushSize"
        :brush-shape="brushShape"
        :text-color="textColor"
        :text-size="textSize"
        :state="editorState"
        @update:brush-size="brushSize = $event"
        @update:brush-shape="brushShape = $event"
        @update:text-color="textColor = $event"
        @update:text-size="textSize = $event"
        @undo="editor?.undo()"
        @redo="editor?.redo()"
        @remove-selected="editor?.removeSelected()"
        @clear-mask="editor?.clearMask()"
        @flatten="editor?.flattenSelected()"
        @download="editor?.downloadSelected()"
        @fit="editor?.fitToContent()"
        @zoom="editor?.zoomBy($event)"
        @upload="editor?.openFilePicker()"
      />
      <div class="corner-left">
        <button class="corner-btn" :title="t('canvas.actions.resetHint')" @click="resetConfirm = true"><Icon name="refresh" size="sm" /></button>
      </div>
      <div class="corner-right">
        <button class="corner-btn" :title="t('canvas.actions.history')" @click="historyOpen = !historyOpen">
          <Icon name="clock" size="sm" />
          <span v-if="studio.activeRunCount.value > 0" class="badge">{{ studio.activeRunCount.value }}</span>
        </button>
      </div>
      <CanvasHistoryDrawer
        :open="historyOpen"
        :runs="studio.history.value"
        :storage-warning="studio.storageWarning.value"
        :take-output="studio.takeOutput"
        @close="historyOpen = false"
        @resume="studio.resumeRun"
        @retry="studio.retryOutputs"
        @apply="studio.applyRunParams"
        @delete="onDeleteRun"
        @clear-local="clearConfirm = true"
      />
      <CanvasPromptPanel
        :operation="studio.operation.value"
        :prompt="studio.prompt.value"
        :api-key-id="studio.apiKeyId.value"
        :model="studio.model.value"
        :size="studio.size.value"
        :quality="studio.quality.value"
        :count="studio.count.value"
        :eligible-keys="studio.eligibleKeys.value"
        :image-models="studio.imageModels.value"
        :submitting="studio.submitting.value || capturing"
        :ready="studio.ready.value"
        @update:operation="studio.operation.value = $event"
        @update:prompt="studio.prompt.value = $event"
        @update:api-key-id="studio.apiKeyId.value = $event"
        @update:model="studio.model.value = $event"
        @update:size="studio.size.value = $event"
        @update:quality="studio.quality.value = $event"
        @update:count="studio.count.value = $event"
        @submit="onSubmit"
      />
      <div v-if="emptyBoard" class="empty-guide">
        <p class="empty-title">{{ t('canvas.empty.title') }}</p>
        <p class="empty-hint">{{ t('canvas.empty.hint') }}</p>
        <button class="upload-btn" @click="editor?.openFilePicker()">
          <Icon name="upload" size="sm" /> {{ t('canvas.actions.upload') }}
        </button>
        <p class="empty-hint">{{ t('canvas.empty.tryPrompt') }}</p>
        <button v-for="ex in examples" :key="ex" class="example" @click="studio.prompt.value = ex">{{ ex }}</button>
      </div>
      <CropImageModal
        v-if="cropQueue.length"
        :queue="cropQueue"
        @done="onCropDone"
        @cancel="cropQueue = []"
      />
      <ConfirmDialog
        v-if="resetConfirm"
        :show="true"
        :title="t('canvas.actions.reset')"
        :message="t('canvas.actions.resetHint')"
        @confirm="editor?.reset(); resetConfirm = false"
        @cancel="resetConfirm = false"
      />
      <ConfirmDialog
        v-if="deleteRunId"
        :show="true"
        :title="t('canvas.history.delete')"
        :message="t('canvas.history.deleteConfirm')"
        @confirm="onConfirmDelete"
        @cancel="deleteRunId = ''"
      />
      <ConfirmDialog
        v-if="clearConfirm"
        :show="true"
        :title="t('canvas.history.clearLocal')"
        :message="t('canvas.history.clearConfirm')"
        @confirm="onClearLocal"
        @cancel="clearConfirm = false"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
/**
 * CanvasStudioView is a thin shell: it binds the singleton useCanvasStudio
 * composable to the editor, routes the output-placement handler, and hosts
 * the toolbar / composer / drawer / crop modal. All heavy lifting lives in
 * the composable and CanvasEditor.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { Icon } from '@/components/icons'
import { useAppStore } from '@/stores'
import { useCanvasStudio } from '@/composables/useCanvasStudio'
import CanvasEditor from '@/components/canvas/CanvasEditor.vue'
import CanvasToolbar from '@/components/canvas/CanvasToolbar.vue'
import CanvasPromptPanel from '@/components/canvas/CanvasPromptPanel.vue'
import CanvasHistoryDrawer from '@/components/canvas/CanvasHistoryDrawer.vue'
import CropImageModal from '@/components/canvas/CropImageModal.vue'
import type {
  CanvasTool,
  CropRequestPayload,
  EditorStateSnapshot,
  PlaceOutputInput,
} from '@/components/canvas/types'
import type { CanvasRun, CanvasRunOutput } from '@/types/canvas'

const { t } = useI18n()
const appStore = useAppStore()
const studio = useCanvasStudio()

const editor = ref<InstanceType<typeof CanvasEditor> | null>(null)
const tool = ref<CanvasTool>('select')
const brushSize = ref(40)
const brushShape = ref<'round' | 'square'>('round')
const textColor = ref('#1f2937')
const textSize = ref(24)
const historyOpen = ref(false)
const resetConfirm = ref(false)
const deleteRunId = ref('')
const clearConfirm = ref(false)
const capturing = ref(false)
const cropQueue = ref<Blob[]>([])
const cropAt = ref<{ x: number; y: number } | null>(null)
const editorReady = ref(false)

const editorState = ref<EditorStateSnapshot>({
  ready: false, imageCount: 0, selectedCount: 0, selectedImageCount: 0,
  hasMask: false, canUndo: false, canRedo: false, saveBlocked: false, zoom: 1,
})

const examples = computed(() => [
  t('canvas.empty.example1'),
  t('canvas.empty.example2'),
  t('canvas.empty.example3'),
])

const editorKey = computed(() => {
  const s = studio.scope.value
  return s ? `u${s.userId}:w${s.workspaceId}` : 'none'
})

const emptyBoard = computed(() => editorReady.value && editorState.value.imageCount === 0)

let unregister: (() => void) | null = null

// Register the placement handler only once the editor reports ready — a
// handler mounted earlier would publish into a half-restored scene.
watch(editorReady, (ready) => {
  if (!ready || unregister) return
  unregister = studio.setOutputHandler(async (run: CanvasRun, output: CanvasRunOutput, blob: Blob | null) => {
    if (!blob || !editor.value) return
    const input: PlaceOutputInput = {
      blob,
      runId: run.id,
      outputIndex: output.index,
      assetKey: output.assetKey,
    }
    // placeOutput throws when the scene write fails, so a failure leaves the
    // output unplaced and retryable instead of marked durable.
    await editor.value.placeOutput(input)
    studio.markOutputPlaced(run.id, output.index)
  })
})

onMounted(() => {
  void studio.initialize()
})

onBeforeUnmount(() => {
  unregister?.()
  unregister = null
})

watch(editorKey, () => {
  editorReady.value = false
  unregister?.()
  unregister = null
})

function onEditorState(snapshot: EditorStateSnapshot): void {
  editorState.value = snapshot
  if (snapshot.ready) editorReady.value = true
}

function onEditorError(code: string): void {
  appStore.showError(t(`canvas.errors.${code}`, code))
}

async function onSubmit(): Promise<void> {
  // A double-click during the async capture must not issue a second submit.
  if (capturing.value || studio.submitting.value) return
  capturing.value = true
  try {
    const { images, mask } = await editor.value!.exportInput(studio.operation.value)
    const run = await studio.submit({ images, mask })
    if (!run && studio.errorMessage.value) {
      appStore.showError(t(`canvas.errors.${studio.errorMessage.value}`, studio.errorMessage.value))
    }
  } catch (error) {
    appStore.showError(
      t(`canvas.errors.${(error as Error).message}`, (error as Error).message),
    )
  } finally {
    capturing.value = false
  }
}

function onCropRequest(payload: CropRequestPayload): void {
  cropAt.value = payload.at
  cropQueue.value = payload.blobs
}

async function onCropDone(blobs: Blob[]): Promise<void> {
  cropQueue.value = []
  if (blobs.length) await editor.value?.importBlobs(blobs, cropAt.value)
}

async function onDeleteRun(runId: string): Promise<void> {
  await editor.value?.flush()
  deleteRunId.value = runId
}

async function onConfirmDelete(): Promise<void> {
  const id = deleteRunId.value
  deleteRunId.value = ''
  await studio.deleteRun(id)
}

async function onClearLocal(): Promise<void> {
  clearConfirm.value = false
  await editor.value?.flush()
  const ok = await studio.clearLocalData()
  if (ok) appStore.showSuccess(t('canvas.history.clearSuccess'))
  else appStore.showError(t('canvas.errors.clear-failed'))
}
</script>

<style scoped>
/* Docked drawer: shrink the board container itself — absolute children
 * (editor, toolbar, composer) re-anchor to its smaller box and the editor's
 * ResizeObserver resizes the canvases automatically. */
.canvas-view { position: absolute; inset: 0; overflow: hidden; }
@media (min-width: 768px) {
  .canvas-view.drawer-open { right: 320px; }
}
.corner-left { position: absolute; top: 12px; left: 12px; z-index: 20; display: flex; gap: 6px; }
.corner-right { position: absolute; top: 12px; right: 12px; z-index: 20; display: flex; gap: 6px; }
.corner-btn {
  position: relative; display: flex; align-items: center; justify-content: center;
  width: 36px; height: 36px; border-radius: 10px;
  background: rgba(255,255,255,0.92); color: #4b5563;
  box-shadow: 0 4px 16px rgba(0,0,0,0.1);
}
.dark .corner-btn { background: rgba(17,24,39,0.92); color: #9ca3af; }
.badge {
  position: absolute; top: -4px; right: -4px; min-width: 16px; height: 16px;
  border-radius: 999px; background: #14b8a6; color: #fff;
  font-size: 10px; line-height: 16px; text-align: center; padding: 0 4px;
}
.empty-guide {
  position: absolute; inset: 0; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 10px;
  pointer-events: none; z-index: 10;
}
.empty-guide > * { pointer-events: auto; }
.empty-title { font-size: 17px; font-weight: 600; color: #374151; }
.dark .empty-title { color: #d1d5db; }
.empty-hint { font-size: 13px; color: #6b7280; max-width: 400px; text-align: center; }
.upload-btn {
  display: flex; align-items: center; gap: 6px; padding: 9px 20px;
  border-radius: 10px; background: #14b8a6; color: #fff; font-size: 13px; font-weight: 600;
}
.example {
  font-size: 12px; color: #14b8a6; padding: 4px 10px; border-radius: 8px;
  border: 1px solid rgba(20,184,166,0.3);
}
</style>
