<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.changelog.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.changelog.description') }}
      </p>
    </div>
    <div class="space-y-4 p-6">
      <!-- Add Entry Button -->
      <button
        type="button"
        data-test="changelog-add-entry"
        @click="addChangelogEntry"
        class="btn btn-secondary btn-sm"
      >
        <svg class="mr-1 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
        {{ t('admin.settings.changelog.addEntry') }}
      </button>

      <!-- Empty hint -->
      <div
        v-if="entries.length === 0"
        data-test="changelog-empty-hint"
        class="rounded-lg border border-dashed border-gray-200 p-6 text-center dark:border-dark-600"
      >
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.changelog.emptyHint') }}
        </p>
      </div>

      <!-- Entry list -->
      <div v-else class="space-y-4">
        <div
          v-for="(entry, index) in entries"
          :key="index"
          data-test="changelog-entry"
          :data-changelog-entry="index"
          class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
        >
          <!-- Entry header -->
          <div class="mb-3 flex items-center justify-between">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              #{{ index + 1 }}
            </span>
            <div class="flex items-center gap-2">
              <!-- Move up -->
              <button
                v-if="index > 0"
                type="button"
                data-test="changelog-move-up"
                class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                :title="t('admin.settings.changelog.moveUp')"
                @click="moveChangelogEntry(index, -1)"
              >
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" /></svg>
              </button>
              <!-- Move down -->
              <button
                v-if="index < entries.length - 1"
                type="button"
                data-test="changelog-move-down"
                class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                :title="t('admin.settings.changelog.moveDown')"
                @click="moveChangelogEntry(index, 1)"
              >
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" /></svg>
              </button>
              <!-- Enabled toggle -->
              <label class="inline-flex cursor-pointer items-center gap-1.5">
                <input
                  v-model="entry.enabled"
                  type="checkbox"
                  data-test="changelog-enabled-toggle"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-xs text-gray-600 dark:text-gray-400">{{ t('admin.settings.changelog.enabled') }}</span>
              </label>
              <!-- Delete -->
              <button
                type="button"
                data-test="changelog-delete-entry"
                class="rounded p-1 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                :title="t('admin.settings.changelog.delete')"
                @click="removeChangelogEntry(index)"
              >
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
              </button>
            </div>
          </div>

          <!-- Fields -->
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.changelog.version') }}
              </label>
              <input
                v-model="entry.version"
                type="text"
                data-test="changelog-version-input"
                class="input text-sm"
                :placeholder="t('admin.settings.changelog.versionPlaceholder')"
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.changelog.publishedAt') }}
              </label>
              <input
                v-model="entry.published_at"
                type="date"
                data-test="changelog-published-at-input"
                class="input text-sm"
              />
            </div>
          </div>

          <div class="mt-3">
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.changelog.titleLabel') }}
            </label>
            <input
              v-model="entry.title"
              type="text"
              data-test="changelog-title-input"
              class="input text-sm"
              :placeholder="t('admin.settings.changelog.titlePlaceholder')"
            />
          </div>

          <!-- Items -->
          <div class="mt-3">
            <div class="mb-1.5 flex items-center justify-between">
              <label class="text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.changelog.items') }}
              </label>
              <button
                type="button"
                data-test="changelog-add-item"
                @click="addChangelogItem(index)"
                class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
              >
                + {{ t('admin.settings.changelog.addItem') }}
              </button>
            </div>
            <div class="space-y-2">
              <div
                v-for="(item, itemIdx) in entry.items"
                :key="itemIdx"
                class="flex items-start gap-2"
              >
                <div class="flex-1">
                  <textarea
                    v-if="changelogPreviewIndex !== `${index}-${itemIdx}`"
                    v-model="entry.items[itemIdx]"
                    rows="2"
                    maxlength="5000"
                    data-test="changelog-item-textarea"
                    class="input w-full text-sm"
                    :placeholder="t('admin.settings.changelog.itemPlaceholder')"
                  ></textarea>
                  <p
                    v-if="changelogPreviewIndex !== `${index}-${itemIdx}`"
                    class="mt-1 text-right text-xs text-gray-400"
                  >
                    {{ entry.items[itemIdx]?.length || 0 }} / 5000
                  </p>
                  <MarkdownRenderer
                    v-else
                    :content="item"
                    class-name="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
                  />
                </div>
                <div class="flex flex-col gap-1">
                  <button
                    type="button"
                    data-test="changelog-preview-toggle"
                    class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                    :title="changelogPreviewIndex === `${index}-${itemIdx}` ? t('admin.settings.changelog.edit') : t('admin.settings.changelog.preview')"
                    @click="toggleChangelogPreview(index, itemIdx)"
                  >
                    <svg v-if="changelogPreviewIndex !== `${index}-${itemIdx}`" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                    <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                  </button>
                  <button
                    type="button"
                    data-test="changelog-delete-item"
                    class="rounded p-1 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                    :title="t('admin.settings.changelog.delete')"
                    @click="removeChangelogItem(index, itemIdx)"
                  >
                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ClientChangelogEntry } from '@/types'

// fork 自有的「客户端更新日志」编辑器。抽成独立组件，避免与上游 SettingsView 冲突。
/* eslint-disable vue/no-mutating-props --
   entries 是父组件 SettingsView 传入的共享 reactive 数组：本编辑器按约定就地增删排序
   （与 SettingsView 其余内联设置区块一致），保存动作仍由父组件统一提交。 */
const props = defineProps<{ entries: ClientChangelogEntry[] }>()
const { t } = useI18n()

const changelogPreviewIndex = ref<string | null>(null)

function addChangelogEntry() {
  props.entries.unshift({
    version: '',
    published_at: new Date().toISOString().slice(0, 10),
    title: '',
    items: [''],
    enabled: true,
  })
}


function removeChangelogEntry(index: number) {
  if (!confirm(t('admin.settings.changelog.deleteConfirm'))) return
  props.entries.splice(index, 1)
}

function moveChangelogEntry(index: number, direction: -1 | 1) {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= props.entries.length) return
  const entries = props.entries
  const temp = entries[index]
  entries[index] = entries[targetIndex]
  entries[targetIndex] = temp
}

function addChangelogItem(entryIndex: number) {
  props.entries[entryIndex].items.push('')
  nextTick(() => {
    const entryEl = document.querySelector(`[data-changelog-entry="${entryIndex}"]`)
    if (entryEl) {
      const textareas = entryEl.querySelectorAll('textarea')
      const last = textareas[textareas.length - 1] as HTMLTextAreaElement | undefined
      last?.focus()
    }
  })
}

function removeChangelogItem(entryIndex: number, itemIndex: number) {
  props.entries[entryIndex].items.splice(itemIndex, 1)
}

function toggleChangelogPreview(entryIndex: number, itemIndex: number) {
  const key = `${entryIndex}-${itemIndex}`
  changelogPreviewIndex.value = changelogPreviewIndex.value === key ? null : key
}
</script>
