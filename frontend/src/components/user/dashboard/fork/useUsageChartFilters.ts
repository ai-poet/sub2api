import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TrendParams } from '@/api/usage'
import type { SelectOption } from '@/components/common/Select.vue'
import type { ModelStat, UsageRequestType } from '@/types'
import { requestTypeToLegacyStream } from '@/utils/usageRequestType'
import { useDashboardDataSource, type DashboardDataSource, type DashboardOptionalFilter } from './dashboardDataSource'
import { queryDates, type UsageRange } from './usageChartData'

// fork：仪表盘用量图的筛选条件（移植自 TokenRouter useUsageChartFilters.ts）。
// 本项目没有团队密钥，候选项来源与使用记录页一致：用户口径是个人密钥 + 可用分组，管理员口径是全部分组（没有密钥筛选）。

// UsageChartFilters 与使用记录页的筛选条件一一对应，null 表示不限。
export interface UsageChartFilters {
  api_key_id: number | null
  model: string | null
  group_id: number | null
  request_type: UsageRequestType | null
  billing_type: number | null
  billing_mode: string | null
  native_compaction_v2: boolean | null
}

type NamedOption = { id: number; name: string }

const emptyFilters = (): UsageChartFilters => ({
  api_key_id: null,
  model: null,
  group_id: null,
  request_type: null,
  billing_type: null,
  billing_mode: null,
  native_compaction_v2: null,
})

// useUsageChartFilters 管理用量图的筛选条件和候选项。
export function useUsageChartFilters(source: DashboardDataSource = useDashboardDataSource()) {
  const { t } = useI18n()
  // 管理员口径看全站，没有「我的密钥」可选，工具栏不显示密钥筛选
  const hasApiKeyFilter = Boolean(source.loadApiKeys)
  // supportsFilter 判断数据源是否支持某个可选筛选项（管理员口径不支持计费模式和请求类别）
  const supportsFilter = (key: DashboardOptionalFilter): boolean => !source.unsupportedFilters?.includes(key)
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
    if (current.native_compaction_v2 !== null) params.native_compaction_v2 = current.native_compaction_v2
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
  // 请求类别的前两项沿用使用记录页的文案；第三项（native_compaction_v2=false）本项目没有现成文案，用 TokenRouter 的措辞补上。
  const compactionOptions = computed<SelectOption[]>(() => [
    { value: null, label: t('usage.allCompactionTypes') },
    { value: true, label: t('usage.compactionOnly') },
    { value: false, label: t('dashboard.fork.filters.legacyCompaction') },
  ])

  // loadKeyAndGroupOptions 读取密钥（仅用户口径）和分组候选。
  const loadKeyAndGroupOptions = async () => {
    try {
      const [keys, availableGroups] = await Promise.all([
        source.loadApiKeys ? source.loadApiKeys() : Promise.resolve([]),
        source.loadGroups(),
      ])
      apiKeys.value = keys
      groups.value = availableGroups
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
      const models = await source.getModels(queryDates(range))
      if (seq !== modelSeq) return
      setModelOptions(models)
    } catch (error) {
      console.error('Failed to load usage model options:', error)
    }
  }

  const reset = () => {
    filters.value = emptyFilters()
  }

  return {
    filters,
    hasApiKeyFilter,
    supportsFilter,
    activeCount,
    queryParams,
    apiKeyOptions,
    modelOptions,
    groupOptions,
    requestTypeOptions,
    billingTypeOptions,
    billingModeOptions,
    compactionOptions,
    loadKeyAndGroupOptions,
    setModelOptions,
    loadModelOptions,
    reset,
  }
}
