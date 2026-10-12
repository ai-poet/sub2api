// fork：模型广场按分组分区（参考 TokenRouter ModelMarketplaceView）。
// 目录接口按「分组 × 模型」逐条下发，每条带该分组自己的价格，这里按分组归拢成分区。

import type { ModelCatalogBillingMode, ModelCatalogItem } from '@/api/modelCatalog'

export type CatalogPricingFilter = 'all' | ModelCatalogBillingMode

export interface CatalogGroupSection {
  id: number
  name: string
  description: string
  platform: string
  /** 分组倍率：优先取按 token 计费的模型（图片 / 视频可能配了独立倍率） */
  rateMultiplier: number
  /** user_override 表示这是当前用户的专属倍率 */
  rateSource: string
  sortOrder: number
  /** 相对官方参考价的最高节省比例（0..1），没有参考价时为 null */
  maxSavings: number | null
  models: ModelCatalogItem[]
}

export interface CatalogGroupFilters {
  search: string
  platform: string
  pricing: CatalogPricingFilter
  groupId: number | 'all'
}

/** 把目录条目按分组归拢；分组按排序值、再按 ID 排列，组内模型按显示名排列。 */
export function groupCatalogItems(items: ModelCatalogItem[]): CatalogGroupSection[] {
  const sections = new Map<number, CatalogGroupSection>()

  for (const item of items) {
    const group = item.best_group
    let section = sections.get(group.id)
    if (!section) {
      section = {
        id: group.id,
        name: group.name,
        description: group.description?.trim() ?? '',
        platform: item.platform,
        rateMultiplier: group.rate_multiplier,
        rateSource: group.rate_source,
        sortOrder: group.sort_order ?? 0,
        maxSavings: null,
        models: [],
      }
      sections.set(group.id, section)
    }
    // 图片 / 视频模型的倍率可能是独立倍率，分组头部展示的倍率以文本模型为准
    if (item.billing_mode === 'token' && section.models.every(model => model.billing_mode !== 'token')) {
      section.rateMultiplier = group.rate_multiplier
      section.rateSource = group.rate_source
    }
    const savings = item.official_pricing.has_reference ? item.comparison.savings_percent : null
    if (typeof savings === 'number' && savings > 0 && (section.maxSavings === null || savings > section.maxSavings)) {
      section.maxSavings = savings
    }
    section.models.push(item)
  }

  const result = [...sections.values()]
  for (const section of result) {
    section.models.sort((a, b) =>
      prettifyModelName(a.display_name || a.model).localeCompare(prettifyModelName(b.display_name || b.model), undefined, {
        numeric: true,
        sensitivity: 'base',
      }),
    )
  }
  return result.sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
}

/** 搜索命中分组名 / 描述 / 平台时整组显示，否则只留模型名命中的模型；空分区不返回。 */
export function filterCatalogGroups(
  sections: CatalogGroupSection[],
  filters: CatalogGroupFilters,
  platformLabel: (platform: string) => string = (platform) => platform,
): CatalogGroupSection[] {
  const keyword = filters.search.trim().toLowerCase()

  return sections.flatMap((section) => {
    if (filters.platform !== 'all' && section.platform !== filters.platform) return []
    if (filters.groupId !== 'all' && section.id !== filters.groupId) return []

    const groupMatches =
      !keyword ||
      [section.name, section.description, section.platform, platformLabel(section.platform)]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(keyword))

    const models = section.models.filter((model) => {
      if (filters.pricing !== 'all' && model.billing_mode !== filters.pricing) return false
      if (groupMatches) return true
      return [model.model, model.display_name, prettifyModelName(model.model)]
        .some((value) => value.toLowerCase().includes(keyword))
    })

    return models.length > 0 ? [{ ...section, models }] : []
  })
}

const BRAND_WORDS: Record<string, string> = {
  gpt: 'GPT',
  glm: 'GLM',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  qwen: 'Qwen',
  qwq: 'QwQ',
  kimi: 'Kimi',
  grok: 'Grok',
  claude: 'Claude',
  gemini: 'Gemini',
  gemma: 'Gemma',
  llama: 'Llama',
  mistral: 'Mistral',
  doubao: 'Doubao',
  imagen: 'Imagen',
  veo: 'Veo',
  sora: 'Sora',
  dall: 'DALL',
  tts: 'TTS',
  vl: 'VL',
  ai: 'AI',
}

/**
 * 把模型 ID 排版成卡片标题：claude-opus-4-6 → Claude Opus 4.6，gpt-5.5 → GPT 5.5。
 * 相邻的纯数字段合并成版本号，末尾的 8 位日期快照去掉；ID 本身仍原样显示在下面一行。
 */
export function prettifyModelName(id: string): string {
  const parts = id.trim().split(/[-_\s/]+/).filter(Boolean)
  if (parts.length === 0) return id
  if (parts.length > 1 && /^\d{8}$/.test(parts[parts.length - 1])) parts.pop()

  const words: string[] = []
  for (const part of parts) {
    const lower = part.toLowerCase()
    const previous = words[words.length - 1]
    if (/^\d{1,2}$/.test(part) && previous && /^\d+(\.\d+)*$/.test(previous)) {
      words[words.length - 1] = `${previous}.${part}`
      continue
    }
    if (BRAND_WORDS[lower]) {
      words.push(BRAND_WORDS[lower])
    } else if (/^[a-z]?\d/.test(lower)) {
      // o3 / v3.2 / 4o 这类带数字的片段：版本号首字母大写（V3.2），其余保持原样
      words.push(/^v\d/.test(lower) ? `V${lower.slice(1)}` : lower)
    } else {
      words.push(lower.charAt(0).toUpperCase() + lower.slice(1))
    }
  }
  return words.join(' ')
}

/** 「最高省 X%」：比例 0..1，四舍五入到整数，太小或缺失时返回 null。 */
export function formatMaxSavings(savings: number | null): number | null {
  if (savings === null || !Number.isFinite(savings) || savings <= 0 || savings >= 1) return null
  const percent = Math.round(savings * 100)
  return percent > 0 ? percent : null
}

/** 分组倍率文案：x1、x0.5、x0.05。 */
export function formatGroupMultiplier(multiplier: number): string {
  if (!Number.isFinite(multiplier)) return 'x1'
  const fixed = multiplier % 1 === 0 ? multiplier.toFixed(0) : multiplier.toFixed(multiplier >= 0.1 ? 2 : 3)
  return `x${fixed.replace(/(\.\d*?)0+$/, '$1').replace(/\.$/, '')}`
}
