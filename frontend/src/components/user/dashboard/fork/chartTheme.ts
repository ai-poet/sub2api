import { computed } from 'vue'
import { useIsDark } from '@/composables/useIsDark'

// fork：仪表盘图表主题（参考 TokenRouter composables/useChartTheme.ts）。
// 深色模式状态来自 useIsDark（监听 <html> 的 dark 类），切换主题后图表配置随之重算。
// 颜色取值与 TokenRouter 一致：刻度/网格走 zinc 三档（text / muted / grid），序列色与 12 色调色板逐项相同。
// 唯一例外是深色网格：TokenRouter 的深色卡片是 #0F0F10，网格 #27272A；本项目深色卡片是 #262626，
// 照搬会看不见网格，这里取与卡片同样反差的 #3d3d3d。

/** token 用量趋势序列的语义色（输入/输出/缓存创建/缓存读取/缓存命中率）。 */
export const CHART_SERIES_COLORS = {
  input: '#3b82f6',
  output: '#10b981',
  cacheCreation: '#f59e0b',
  cacheRead: '#06b6d4',
  cacheHitRate: '#8b5cf6',
} as const

/** 分布图（圆环）的 12 色调色板，按排名依次取色；管理员仪表盘的分布图和用户趋势共用。 */
export const CHART_PALETTE = [
  '#3b82f6',
  '#10b981',
  '#f59e0b',
  '#ef4444',
  '#8b5cf6',
  '#ec4899',
  '#00D2FF',
  '#f97316',
  '#6366f1',
  '#84cc16',
  '#06b6d4',
  '#a855f7',
] as const

/** 分布图「其他」聚合扇区的固定灰。 */
export const CHART_OTHER_COLOR = '#94a3b8'

/** 坐标刻度字号。 */
export const CHART_TICK_FONT_SIZE = 10

/** 图例字号。 */
export const CHART_LEGEND_FONT_SIZE = 11

export function useChartTheme() {
  const isDark = useIsDark()

  const colors = computed(() =>
    isDark.value
      ? { text: '#DEE0E2', muted: '#A1A1AA', grid: '#3d3d3d' }
      : { text: '#3F3F46', muted: '#71717A', grid: '#E4E4E7' },
  )

  return { isDark, colors }
}
