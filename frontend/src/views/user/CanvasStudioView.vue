<script setup lang="ts">
/**
 * CanvasStudioView.vue 是画布工作台页面。
 * 合成图复用 /v1/images/edits/async，空白画布复用 generations/async；
 * 任务、计费、内容审计和对象存储都由既有网关链路处理。
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import CanvasEditor from '@/components/canvas/CanvasEditor.vue'
import CanvasToolbar from '@/components/canvas/CanvasToolbar.vue'
import CanvasPromptPanel from '@/components/canvas/CanvasPromptPanel.vue'
import type { CanvasTool } from '@/components/canvas/types'
import {
  extractOutputImages,
  fetchOutputImage,
  getCanvasTask,
  submitCanvasTask,
  type CanvasTask,
} from '@/api/canvas'
import { keysAPI } from '@/api/keys'
import { getCatalog, type ModelCatalogItem } from '@/api/modelCatalog'
import type { ApiKey } from '@/types'
import { loadScene, saveAsset, saveScene } from '@/utils/canvasLocalStore'

const { t } = useI18n()
const MAX_IMPORT_BYTES = 50 * 1024 * 1024
const MAX_HISTORY = 20

const editor = shallowRef<InstanceType<typeof CanvasEditor> | null>(null)
const tool = ref<CanvasTool>('select')
const brushSize = ref(24)
const maskColor = ref('rgba(239, 68, 68, 0.45)')
const textColor = ref('#111827')
const background = ref('#ffffff')
const canDelete = ref(false)
const canUndo = ref(false)
const canRedo = ref(false)

const apiKeys = ref<ApiKey[]>([])
const apiKeyId = ref<number | null>(null)
const catalogItems = ref<ModelCatalogItem[]>([])
const model = ref('')
const prompt = ref('')
const size = ref('1024x1024')
const quality = ref('')
const count = ref(1)

const submitting = ref(false)
const errorMessage = ref('')
const currentTask = ref<CanvasTask | null>(null)
const history = ref<Array<{ taskId: string; status: string; images: string[] }>>([])
let pollTimer: number | null = null
let saveTimer: number | null = null

const busy = computed(
  () => submitting.value || currentTask.value?.status === 'processing',
)

const eligibleKeys = computed(() =>
  apiKeys.value.filter(
    (key) =>
      key.status === 'active' &&
      (key.group?.platform === 'openai' || key.group?.platform === 'grok') &&
      key.group?.allow_image_generation === true,
  ),
)

const selectedGroupId = computed(() => {
  const key = eligibleKeys.value.find((item) => item.id === apiKeyId.value)
  return key?.group?.id ?? key?.group_id ?? null
})

/** 只列当前 Key 所在分组真正可用的模型，避免提交后才得到 no available channel。 */
const imageModels = computed(() => {
  const names = new Set<string>()
  for (const item of catalogItems.value) {
    if (item.billing_mode !== 'image' || !item.model) continue
    if (selectedGroupId.value !== null) {
      const inBestGroup = item.best_group?.id === selectedGroupId.value
      const inOtherGroups = (item.other_groups || []).some(
        (companion) => companion.group?.id === selectedGroupId.value,
      )
      if (!inBestGroup && !inOtherGroups) continue
    }
    names.add(item.model)
  }
  return [...names].sort()
})

async function loadApiKeys(): Promise<void> {
  const items: ApiKey[] = []
  try {
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, 100, {
        status: 'active',
        sort_by: 'created_at',
        sort_order: 'desc',
      })
      items.push(...(response.items || []))
      if (page >= response.pages || (response.items || []).length === 0) break
      page += 1
    }
  } catch {
    // 页面会显示空 Key 列表；不把一个列表错误掩盖成提交错误。
  }
  apiKeys.value = items
  if (!eligibleKeys.value.some((key) => key.id === apiKeyId.value)) {
    apiKeyId.value = eligibleKeys.value[0]?.id ?? null
  }
}

async function loadModels(): Promise<void> {
  try {
    const catalog = await getCatalog()
    catalogItems.value = catalog.items || []
  } catch {
    catalogItems.value = []
  }
}

function onSelect(payload: { count: number }): void {
  canDelete.value = payload.count > 0
}

function onHistory(payload: { canUndo: boolean; canRedo: boolean }): void {
  canUndo.value = payload.canUndo
  canRedo.value = payload.canRedo
}

function onCanvasChange(): void {
  if (saveTimer !== null) window.clearTimeout(saveTimer)
  saveTimer = window.setTimeout(() => void persistScene(), 800)
}

async function persistScene(): Promise<void> {
  const instance = editor.value
  if (!instance) return
  try {
    const json = instance.serialize()
    if (json) await saveScene('default', json)
  } catch {
    // IndexedDB 配额或隐私模式失败时，画布本身仍可使用。
  }
}

async function restoreScene(): Promise<void> {
  const instance = editor.value
  if (!instance) return
  try {
    const record = await loadScene('default')
    if (record) await instance.restore(record.json)
  } catch {
    // 损坏或旧版本快照不阻塞新画布。
  }
}

async function onFileSelected(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  errorMessage.value = ''
  if (file.size > MAX_IMPORT_BYTES) {
    errorMessage.value = t('canvas.errors.fileTooLarge')
    return
  }
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    errorMessage.value = t('canvas.errors.importFailed')
    return
  }
  try {
    await editor.value?.addImage(file)
  } catch {
    errorMessage.value = t('canvas.errors.importFailed')
  }
}

function onTextSubmit(): void {
  const text = window.prompt(t('canvas.textPrompt'))
  if (text) editor.value?.addText(text)
}

async function submit(): Promise<void> {
  if (busy.value) return
  errorMessage.value = ''
  const instance = editor.value
  const key = eligibleKeys.value.find((item) => item.id === apiKeyId.value)
  if (!instance || !key) {
    errorMessage.value = t('canvas.errors.noApiKey')
    return
  }

  submitting.value = true
  stopPolling()
  try {
    const hasContent = instance.hasContent()
    const result = hasContent ? await instance.composite() : null
    if (hasContent && !result) {
      errorMessage.value = t('canvas.errors.compositeFailed')
      return
    }
    const task = await submitCanvasTask(key.key, {
      image: result?.blob ?? null,
      mask: result?.mask ?? null,
      prompt: prompt.value,
      model: model.value,
      size: size.value,
      n: count.value,
      quality: quality.value || undefined,
    })
    currentTask.value = { ...task }
    startPolling(key.key, task.id)
  } catch (error) {
    currentTask.value = null
    errorMessage.value = error instanceof Error ? error.message : String(error)
  } finally {
    submitting.value = false
  }
}

function startPolling(apiKey: string, taskId: string): void {
  stopPolling(false)
  let inFlight = false
  let failures = 0
  const tick = async (): Promise<void> => {
    if (inFlight) return
    inFlight = true
    try {
      const task = await getCanvasTask(apiKey, taskId)
      currentTask.value = task
      failures = 0
      if (task.status === 'completed') {
        await handleCompleted(task)
        stopPolling()
        return
      }
      if (task.status === 'failed') {
        errorMessage.value = task.error?.message || t('canvas.errors.taskFailed')
        stopPolling()
        return
      }
    } catch (error) {
      // 单次网络抖动不应丢掉已经付费且仍在执行的任务。
      failures += 1
      if (failures >= 5) {
        errorMessage.value = error instanceof Error ? error.message : String(error)
        stopPolling()
      }
    } finally {
      inFlight = false
    }
  }
  void tick()
  pollTimer = window.setInterval(() => void tick(), 3000)
}

function stopPolling(clearTask = true): void {
  if (pollTimer !== null) window.clearInterval(pollTimer)
  pollTimer = null
  // processing 是瞬时状态；停止轮询后解除页面锁定，但不伪报成功或失败。
  if (clearTask && currentTask.value?.status === 'processing') currentTask.value = null
}

async function handleCompleted(task: CanvasTask): Promise<void> {
  const images = extractOutputImages(task)
  const urls: string[] = []
  for (let index = 0; index < images.length; index += 1) {
    const blob = await fetchOutputImage(images[index])
    if (!blob) continue
    urls.push(URL.createObjectURL(blob))
    try {
      await saveAsset({
        key: `${task.id}:${index}`,
        blob,
        runId: task.id,
        outputIndex: index,
        createdAt: Date.now(),
      })
    } catch {
      // 本地缓存失败不影响在线结果。
    }
  }
  if (urls.length === 0) {
    errorMessage.value = t('canvas.errors.noResults')
    return
  }
  const next = [{ taskId: task.id, status: task.status, images: urls }, ...history.value]
  const evicted = next.slice(MAX_HISTORY)
  history.value = next.slice(0, MAX_HISTORY)
  for (const item of evicted) {
    for (const url of item.images) URL.revokeObjectURL(url)
  }
}

async function useResult(url: string): Promise<void> {
  errorMessage.value = ''
  try {
    const response = await fetch(url)
    if (!response.ok) throw new Error('fetch failed')
    await editor.value?.loadResultImage(await response.blob())
  } catch {
    errorMessage.value = t('canvas.errors.importFailed')
  }
}

onMounted(async () => {
  await nextTick()
  await Promise.all([loadApiKeys(), loadModels()])
  await restoreScene()
})

onBeforeUnmount(() => {
  stopPolling()
  if (saveTimer !== null) window.clearTimeout(saveTimer)
  for (const item of history.value) {
    for (const url of item.images) URL.revokeObjectURL(url)
  }
})

watch(imageModels, (models) => {
  if (!models.includes(model.value)) model.value = models[0] ?? ''
})
watch(tool, (value) => {
  if (value === 'text') {
    onTextSubmit()
    tool.value = 'select'
  }
})
</script>

<template>
  <AppLayout>
    <div class="flex h-full min-h-[640px] flex-col lg:flex-row">
      <div class="flex min-h-0 flex-1 flex-col">
        <CanvasToolbar
          v-model:tool="tool"
          v-model:brush-size="brushSize"
          :can-undo="canUndo"
          :can-redo="canRedo"
          :can-delete="canDelete"
          :disabled="busy"
          @undo="editor?.undo()"
          @redo="editor?.redo()"
          @delete="editor?.removeSelected()"
          @clear="editor?.clearCanvas()"
        />

        <div class="flex flex-wrap items-center gap-2 border-b border-gray-200 px-3 py-2 dark:border-dark-700">
          <label
            class="btn btn-secondary btn-sm cursor-pointer"
            :class="busy ? 'pointer-events-none opacity-50' : ''"
          >
            <Icon name="upload" size="sm" class="mr-1.5" />
            {{ $t('canvas.actions.importImage') }}
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp"
              class="hidden"
              :disabled="busy"
              @change="onFileSelected"
            />
          </label>
          <span class="text-xs text-gray-400 dark:text-gray-500">{{ $t('canvas.actions.importHint') }}</span>
          <span
            v-if="currentTask?.status === 'processing'"
            class="ml-auto inline-flex items-center text-xs text-primary-600 dark:text-primary-400"
          >
            <LoadingSpinner size="sm" class="mr-1.5" />
            {{ $t('canvas.status.processing') }}
          </span>
        </div>

        <div class="min-h-[420px] flex-1">
          <CanvasEditor
            ref="editor"
            :tool="tool"
            :brush-size="brushSize"
            :mask-color="maskColor"
            :text-color="textColor"
            :background="background"
            :size="size"
            @change="onCanvasChange"
            @select="onSelect"
            @history="onHistory"
          />
        </div>

        <div v-if="history.length > 0" class="border-t border-gray-200 px-3 py-2 dark:border-dark-700">
          <p class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-400">
            {{ $t('canvas.history.title') }}
          </p>
          <div class="flex gap-2 overflow-x-auto pb-1">
            <div v-for="item in history" :key="item.taskId" class="flex gap-2">
              <button
                v-for="(url, index) in item.images"
                :key="url"
                type="button"
                class="group relative h-16 w-16 shrink-0 overflow-hidden rounded-lg border border-gray-200 transition-colors hover:border-primary-400 dark:border-dark-700"
                :title="$t('canvas.history.useAsSource')"
                @click="useResult(url)"
              >
                <img :src="url" :alt="`${item.taskId}-${index}`" class="h-full w-full object-cover" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <aside class="w-full shrink-0 border-t border-gray-200 lg:w-80 lg:border-l lg:border-t-0 dark:border-dark-700">
        <CanvasPromptPanel
          :api-keys="eligibleKeys"
          :api-key-id="apiKeyId"
          :models="imageModels"
          :model="model"
          :prompt="prompt"
          :size="size"
          :quality="quality"
          :count="count"
          :submitting="busy"
          :error-message="errorMessage"
          @update:api-key-id="(value) => (apiKeyId = value)"
          @update:model="(value) => (model = value)"
          @update:prompt="(value) => (prompt = value)"
          @update:size="(value) => (size = value)"
          @update:quality="(value) => (quality = value)"
          @update:count="(value) => (count = value)"
          @submit="submit"
        />
      </aside>
    </div>
  </AppLayout>
</template>
