import { computed, onScopeDispose, ref } from 'vue'

// fork：仪表盘图表主题（参考 TokenRouter composables/useChartTheme.ts）。
// 本项目没有响应式的主题 composable，这里监听 <html> 的 dark 类，切换主题后图表配置随之重算。
// 刻度/网格色取自 tailwind.config.js 的中性色阶，保证与卡片底色搭配。

/** token 用量趋势序列的语义色（输入/输出/缓存创建/缓存读取/缓存命中率）。 */
export const CHART_SERIES_COLORS = {
  input: '#3b82f6',
  output: '#10b981',
  cacheCreation: '#f59e0b',
  cacheRead: '#06b6d4',
  cacheHitRate: '#8b5cf6',
} as const

/** 坐标刻度字号。 */
export const CHART_TICK_FONT_SIZE = 10

const readDark = (): boolean =>
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark')

export function useChartTheme() {
  const isDark = ref(readDark())
  let observer: MutationObserver | null = null
  if (typeof MutationObserver !== 'undefined' && typeof document !== 'undefined') {
    observer = new MutationObserver(() => {
      isDark.value = readDark()
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  }
  onScopeDispose(() => observer?.disconnect())

  const colors = computed(() =>
    isDark.value
      ? { text: '#d4d4d4', muted: '#a3a3a3', grid: '#3d3d3d' }
      : { text: '#525252', muted: '#737373', grid: '#e6e6e6' },
  )

  return { isDark, colors }
}
