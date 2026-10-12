import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { keysAPI } from '@/api/keys'
import { userGroupsAPI } from '@/api/groups'
import { usageAPI, type TrendParams } from '@/api/usage'
import type { SelectOption } from '@/components/common/Select.vue'
import type { ModelStat, UsageRequestType } from '@/types'
import { requestTypeToLegacyStream } from '@/utils/usageRequestType'
import { queryDates, type UsageRange } from './usageChartData'

// fork：仪表盘用量图的筛选条件（移植自 TokenRouter useUsageChartFilters.ts）。
// 本项目没有团队密钥，候选项来源与使用记录页（views/user/UsageView.vue）一致：个人密钥 + 可用分组。

// UsageChartFilters 与使用记录页的筛选条件一一对应，null 表示不限。
export interface UsageChartFilters {
  api_key_id: number | null
  model: string | null
  group_id: number | null
  request_type: UsageRequestType | null
  billing_type: number | null
  billing_mode: string | null
}

type NamedOption = { id: number; name: string }

// 密钥候选最多翻这么多页（每页 100 个），与使用记录页的上限相当。
const MAX_KEY_PAGES = 10

const emptyFilters = (): UsageChartFilters => ({
  api_key_id: null,
  model: null,
  group_id: null,
  request_type: null,
  billing_type: null,
  billing_mode: null,
})

// useUsageChartFilters 管理用量图的筛选条件和候选项。
export function useUsageChartFilters() {
  const { t } = useI18n()
  const filters = ref<UsageChartFilters>(emptyFilters())
  const apiKeys = ref<NamedOption[]>([])
  const groups = ref<NamedOption[]>([])
  const modelNames = ref<string[]>([])

  const activeCount = computed(() => Object.values(filters.value).filter((value) => value !== null).length)

  // queryParams 只带上已选的条件；请求类型同时换算出旧版 stream 参数，与使用记录页一致。
  const queryParams = computed<Partial<TrendParams>>(() => {
    const params: Partial<TrendParams> = {}
    const current = filters.value
    if (current.api_key_id !== null) params.api_key_id = current.api_key_id
    if (current.model) params.model = current.model
    if (current.group_id !== null) params.group_id = current.group_id
    if (current.request_type) {
      params.request_type = current.request_type
      const stream = requestTypeToLegacyStream(current.request_type)
      if (typeof stream === 'boolean') params.stream = stream
    }
    if (current.billing_type !== null) params.billing_type = current.billing_type
    if (current.billing_mode) params.billing_mode = current.billing_mode
    return params
  })

  const apiKeyOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('usage.allApiKeys') },
    ...apiKeys.value.map((key) => ({ value: key.id, label: key.name })),
  ])
  const modelOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('admin.usage.allModels') },
    ...modelNames.value.map((name) => ({ value: name, label: name })),
  ])
  const groupOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('admin.usage.allGroups') },
    ...groups.value.map((group) => ({ value: group.id, label: group.name })),
  ])
  const requestTypeOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('admin.usage.allTypes') },
    { value: 'ws_v2', label: t('usage.ws') },
    { value: 'live', label: t('usage.live') },
    { value: 'stream', label: t('usage.stream') },
    { value: 'sync', label: t('usage.sync') },
  ])
  const billingTypeOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('admin.usage.allBillingTypes') },
    { value: 0, label: t('admin.usage.billingTypeBalance') },
    { value: 1, label: t('admin.usage.billingTypeSubscription') },
  ])
  const billingModeOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('admin.usage.allBillingModes') },
    { value: 'token', label: t('admin.usage.billingModeToken') },
    { value: 'per_request', label: t('admin.usage.billingModePerRequest') },
    { value: 'image', label: t('admin.usage.billingModeImage') },
    { value: 'video', label: t('admin.usage.billingModeVideo') },
  ])

  // loadKeyAndGroupOptions 读取自己的全部密钥和可用分组。
  const loadKeyAndGroupOptions = async () => {
    try {
      const loadKeys = async () => {
        const first = await keysAPI.list(1, 100)
        const keys = [...(first.items || [])]
        const pages = Math.min(first.pages || 1, MAX_KEY_PAGES)
        for (let page = 2; page <= pages; page++) {
          const next = await keysAPI.list(page, 100)
          if (!next.items?.length) break
          keys.push(...next.items)
        }
        return keys
      }
      const [keys, availableGroups] = await Promise.all([loadKeys(), userGroupsAPI.getAvailable()])
      apiKeys.value = keys.map((key) => ({ id: key.id, name: key.name }))
      groups.value = (availableGroups || []).map((group) => ({ id: group.id, name: group.name }))
    } catch (error) {
      console.error('Failed to load usage filter options:', error)
    }
  }

  // setModelOptions 根据模型统计刷新候选项，并保留当前选中的模型。
  const setModelOptions = (models: ModelStat[]) => {
    const names = new Set(models.map((item) => item.model).filter(Boolean))
    if (filters.value.model) names.add(filters.value.model)
    modelNames.value = [...names].sort()
  }

  // 用递增序号丢弃过期的模型候选响应。
  let modelSeq = 0

  // loadModelOptions 按时间范围读取用过的模型，其他筛选条件独立于候选查询。
  const loadModelOptions = async (range: UsageRange) => {
    const seq = ++modelSeq
    try {
      const response = await usageAPI.getDashboardModels(queryDates(range))
      if (seq !== modelSeq) return
      setModelOptions(response.models || [])
    } catch (error) {
      console.error('Failed to load usage model options:', error)
    }
  }

  const reset = () => {
    filters.value = emptyFilters()
  }

  return {
    filters,
    activeCount,
    queryParams,
    apiKeyOptions,
    modelOptions,
    groupOptions,
    requestTypeOptions,
    billingTypeOptions,
    billingModeOptions,
    loadKeyAndGroupOptions,
    setModelOptions,
    loadModelOptions,
    reset,
  }
}
