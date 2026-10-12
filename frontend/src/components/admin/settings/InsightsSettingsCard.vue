<template>
  <!-- 排行榜与首页公开统计（fork 本地功能）：独立加载、改一项就保存，不属于 SettingsView 的大表单 -->
  <div class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-600" data-test="insights-card">
    <div>
      <label class="font-medium text-gray-900 dark:text-white">{{ t('admin.settings.site.insights.title') }}</label>
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.site.insights.description') }}</p>
    </div>

    <p v-if="loadFailed" class="text-sm text-red-600 dark:text-red-400" data-test="insights-load-failed">
      {{ t('admin.settings.site.insights.loadFailed') }}
    </p>

    <div v-for="row in toggles" :key="row.key" class="flex items-center justify-between gap-4">
      <div>
        <div class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ t(`admin.settings.site.insights.${row.label}`) }}</div>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.settings.site.insights.${row.label}Hint`) }}</p>
      </div>
      <Toggle
        :model-value="form[row.key]"
        :disabled="loading || saving || (row.dependsOnUsers && !form.usage_ranking_enabled)"
        :data-test="`insights-${row.key}`"
        @update:model-value="(value: boolean) => toggle(row.key, value)"
      />
    </div>

    <div class="flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-800 dark:text-gray-200" for="insights-limit">
          {{ t('admin.settings.site.insights.usageRankingLimit') }}
        </label>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.site.insights.usageRankingLimitHint') }}</p>
      </div>
      <input
        id="insights-limit"
        v-model.number="form.usage_ranking_limit"
        type="number"
        min="3"
        max="50"
        class="input w-24 text-sm"
        :disabled="loading || saving || !form.usage_ranking_enabled"
        data-test="insights-limit"
        @change="saveLimit"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
// fork：排行榜 / 首页公开统计的后台开关。保存后刷新公开设置，侧栏入口和首页计数立即跟着变。
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { insightsAPI, type InsightsConfig } from '@/api/admin/insights'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

type ToggleKey = 'model_ranking_enabled' | 'usage_ranking_enabled' | 'usage_ranking_show_cost' | 'public_stats_enabled'

const { t } = useI18n()
const appStore = useAppStore()

const toggles: Array<{ key: ToggleKey; label: string; dependsOnUsers?: boolean }> = [
  { key: 'model_ranking_enabled', label: 'modelRanking' },
  { key: 'usage_ranking_enabled', label: 'usageRanking' },
  { key: 'usage_ranking_show_cost', label: 'usageRankingShowCost', dependsOnUsers: true },
  { key: 'public_stats_enabled', label: 'publicStats' },
]

const form = reactive<InsightsConfig>({
  model_ranking_enabled: true,
  usage_ranking_enabled: false,
  usage_ranking_limit: 20,
  usage_ranking_show_cost: false,
  public_stats_enabled: false,
})
const loading = ref(true)
const saving = ref(false)
const loadFailed = ref(false)

function apply(config: InsightsConfig) {
  Object.assign(form, config)
}

async function save(next: InsightsConfig): Promise<boolean> {
  saving.value = true
  try {
    apply(await insightsAPI.updateConfig(next))
    appStore.showSuccess(t('admin.settings.site.insights.saved'))
    void appStore.fetchPublicSettings(true)
    return true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.settings.site.insights.saveFailed')))
    return false
  } finally {
    saving.value = false
  }
}

/** 开关一改就保存，失败时弹回原状态 */
async function toggle(key: ToggleKey, value: boolean) {
  if (loading.value || saving.value) return
  const previous = form[key]
  form[key] = value
  if (!(await save({ ...form }))) {
    form[key] = previous
  }
}

async function saveLimit() {
  if (loading.value || saving.value) return
  const limit = Math.min(50, Math.max(3, Math.round(Number(form.usage_ranking_limit) || 20)))
  await save({ ...form, usage_ranking_limit: limit })
}

onMounted(async () => {
  try {
    apply(await insightsAPI.getConfig())
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
})
</script>
