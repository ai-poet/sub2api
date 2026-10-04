<template>
  <BaseDialog
    :show="show"
    :title="t('admin.settings.site.contentTranslation.dialog.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <select
          v-model="lang"
          class="input text-sm sm:w-40"
          data-test="content-translations-lang"
          @change="reload"
        >
          <option value="">{{ t('admin.settings.site.contentTranslation.dialog.allLanguages') }}</option>
          <option v-for="code in LANGUAGES" :key="code" :value="code">{{ languageLabel(code) }}</option>
        </select>
        <input
          v-model="search"
          type="search"
          class="input flex-1 text-sm"
          :placeholder="t('admin.settings.site.contentTranslation.dialog.searchPlaceholder')"
          data-test="content-translations-search"
          @input="handleSearchInput"
          @keydown.enter.prevent="reload"
        />
        <div class="flex gap-2">
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="loading"
            :title="t('admin.settings.site.contentTranslation.dialog.refresh')"
            @click="load"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button
            type="button"
            class="btn btn-danger btn-sm"
            :disabled="clearing"
            data-test="content-translations-clear"
            @click="confirmClear = true"
          >
            {{ t('admin.settings.site.contentTranslation.dialog.clearMachine') }}
          </button>
        </div>
      </div>

      <p v-if="loadFailed" class="text-sm text-red-600 dark:text-red-400">
        {{ t('admin.settings.site.contentTranslation.dialog.loadFailed') }}
      </p>

      <DataTable :columns="columns" :data="items" :loading="loading" row-key="id">
        <template #cell-source_text="{ row }">
          <p class="max-w-xs whitespace-pre-wrap break-words text-sm text-gray-900 line-clamp-4 dark:text-gray-100" :title="row.source_text">
            {{ row.source_text }}
          </p>
        </template>

        <template #cell-translated_text="{ row }">
          <div v-if="editingId === row.id" class="min-w-[16rem] space-y-2">
            <textarea
              v-model="editingText"
              rows="3"
              class="input text-sm"
              data-test="content-translations-edit-input"
            ></textarea>
            <div class="flex gap-2">
              <button
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="savingEdit"
                data-test="content-translations-edit-save"
                @click="saveEdit(row)"
              >
                {{ t('admin.settings.site.contentTranslation.dialog.save') }}
              </button>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="savingEdit" @click="cancelEdit">
                {{ t('admin.settings.site.contentTranslation.dialog.cancel') }}
              </button>
            </div>
          </div>
          <p
            v-else
            class="max-w-sm whitespace-pre-wrap break-words text-sm text-gray-700 line-clamp-4 dark:text-gray-300"
            :title="row.translated_text"
          >
            {{ row.translated_text }}
          </p>
        </template>

        <template #cell-target_lang="{ row }">
          <span class="whitespace-nowrap text-sm">{{ languageLabel(row.target_lang) }}</span>
        </template>

        <template #cell-model="{ row }">
          <span class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ row.model || '-' }}</span>
        </template>

        <template #cell-flags="{ row }">
          <div class="flex flex-wrap gap-1">
            <span v-if="row.manual" class="badge badge-primary" data-test="content-translations-manual">
              {{ t('admin.settings.site.contentTranslation.dialog.manual') }}
            </span>
            <span
              :class="['badge', row.in_use ? 'badge-success' : 'badge-gray']"
              :title="row.in_use ? undefined : t('admin.settings.site.contentTranslation.dialog.notInUseHint')"
            >
              {{
                row.in_use
                  ? t('admin.settings.site.contentTranslation.dialog.inUse')
                  : t('admin.settings.site.contentTranslation.dialog.notInUse')
              }}
            </span>
          </div>
        </template>

        <template #cell-actions="{ row }">
          <div class="flex items-center gap-1">
            <button
              type="button"
              class="btn btn-ghost btn-sm"
              :disabled="editingId !== null"
              data-test="content-translations-edit"
              @click="startEdit(row)"
            >
              {{ t('admin.settings.site.contentTranslation.dialog.edit') }}
            </button>
            <button
              type="button"
              class="btn btn-ghost btn-sm text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950/30"
              data-test="content-translations-delete"
              @click="deleteTarget = row"
            >
              {{ t('admin.settings.site.contentTranslation.dialog.delete') }}
            </button>
          </div>
        </template>
      </DataTable>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="handleClose">{{ t('common.close') }}</button>
      </div>
    </template>

    <ConfirmDialog
      :show="deleteTarget !== null"
      :title="t('admin.settings.site.contentTranslation.dialog.deleteTitle')"
      :message="t('admin.settings.site.contentTranslation.dialog.deleteConfirm')"
      danger
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
    />
    <ConfirmDialog
      :show="confirmClear"
      :title="t('admin.settings.site.contentTranslation.dialog.clearTitle')"
      :message="t('admin.settings.site.contentTranslation.dialog.clearConfirm')"
      danger
      @confirm="clearMachine"
      @cancel="confirmClear = false"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import contentTranslationsAPI, {
  type ContentTranslationItem,
  type ContentTranslationLang
} from '@/api/admin/contentTranslations'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

// 内容自动翻译（fork 本地功能）：译文管理。改写后后端标为人工译文；删除 / 清空后下一轮重新翻译。

const LANGUAGES: ContentTranslationLang[] = ['zh', 'en', 'ja']
const SEARCH_DEBOUNCE_MS = 300

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<ContentTranslationItem[]>([])
const lang = ref<ContentTranslationLang | ''>('')
const search = ref('')
const loading = ref(false)
const loadFailed = ref(false)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const editingId = ref<number | null>(null)
const editingText = ref('')
const savingEdit = ref(false)
const deleteTarget = ref<ContentTranslationItem | null>(null)
const confirmClear = ref(false)
const clearing = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let requestSeq = 0

const columns = computed<Column[]>(() => [
  { key: 'source_text', label: t('admin.settings.site.contentTranslation.dialog.columns.source') },
  { key: 'translated_text', label: t('admin.settings.site.contentTranslation.dialog.columns.translation') },
  { key: 'target_lang', label: t('admin.settings.site.contentTranslation.dialog.columns.lang') },
  { key: 'model', label: t('admin.settings.site.contentTranslation.dialog.columns.model') },
  { key: 'flags', label: t('admin.settings.site.contentTranslation.dialog.columns.flags') },
  { key: 'actions', label: t('admin.settings.site.contentTranslation.dialog.columns.actions') }
])

function languageLabel(code: string): string {
  return LANGUAGES.includes(code as ContentTranslationLang)
    ? t(`admin.settings.site.contentTranslation.languageNames.${code}`)
    : code
}

function clearSearchTimer() {
  if (searchTimer !== null) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
}

async function load() {
  if (!props.show) return
  const seq = ++requestSeq
  loading.value = true
  loadFailed.value = false
  try {
    const result = await contentTranslationsAPI.list({
      lang: lang.value,
      q: search.value,
      page: pagination.page,
      page_size: pagination.page_size
    })
    if (seq !== requestSeq) return
    items.value = result.items ?? []
    pagination.total = result.total ?? 0
    // 删除最后一条后当前页可能越界：退回最后一页
    const pages = result.pages ?? 0
    if (items.value.length === 0 && pagination.page > 1 && pages > 0 && pagination.page > pages) {
      pagination.page = pages
      void load()
    }
  } catch {
    if (seq !== requestSeq) return
    loadFailed.value = true
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

function reload() {
  clearSearchTimer()
  pagination.page = 1
  void load()
}

function handleSearchInput() {
  clearSearchTimer()
  searchTimer = setTimeout(() => {
    searchTimer = null
    reload()
  }, SEARCH_DEBOUNCE_MS)
}

function handlePageChange(page: number) {
  pagination.page = page
  void load()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void load()
}

function startEdit(item: ContentTranslationItem) {
  editingId.value = item.id
  editingText.value = item.translated_text
}

function cancelEdit() {
  editingId.value = null
  editingText.value = ''
}

async function saveEdit(item: ContentTranslationItem) {
  if (savingEdit.value) return
  const text = editingText.value.trim()
  if (!text) {
    appStore.showError(t('admin.settings.site.contentTranslation.dialog.emptyTranslation'))
    return
  }
  savingEdit.value = true
  try {
    const updated = await contentTranslationsAPI.updateItem(item.id, text)
    const index = items.value.findIndex((row) => row.id === item.id)
    if (index >= 0) {
      items.value[index] =
        updated && typeof updated === 'object' && updated.id === item.id
          ? updated
          : { ...item, translated_text: text, manual: true }
    }
    cancelEdit()
    appStore.showSuccess(t('admin.settings.site.contentTranslation.dialog.updated'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.contentTranslation.dialog.updateFailed')))
  } finally {
    savingEdit.value = false
  }
}

async function confirmDelete() {
  const target = deleteTarget.value
  deleteTarget.value = null
  if (!target) return
  try {
    await contentTranslationsAPI.deleteItem(target.id)
    if (editingId.value === target.id) cancelEdit()
    appStore.showSuccess(t('admin.settings.site.contentTranslation.dialog.deleted'))
    void load()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.contentTranslation.dialog.deleteFailed')))
  }
}

async function clearMachine() {
  confirmClear.value = false
  if (clearing.value) return
  clearing.value = true
  try {
    const result = await contentTranslationsAPI.clearMachine()
    cancelEdit()
    appStore.showSuccess(
      t('admin.settings.site.contentTranslation.dialog.cleared', { count: result?.deleted ?? 0 })
    )
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.contentTranslation.dialog.clearFailed')))
  } finally {
    clearing.value = false
  }
}

function handleClose() {
  clearSearchTimer()
  cancelEdit()
  emit('close')
}

watch(
  () => props.show,
  (open) => {
    if (open) {
      reload()
    } else {
      clearSearchTimer()
      requestSeq++
    }
  },
  { immediate: true }
)

onBeforeUnmount(clearSearchTimer)
</script>
