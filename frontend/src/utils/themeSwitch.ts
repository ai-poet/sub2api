// fork：主题切换时冻结过渡（移植自 TokenRouter composables/useTheme.ts 的 suspendThemeTransitions）。
// 切换前给 html 加 theme-switching，保留两个渲染帧，Vue 更新和浏览器绘制期间都不触发
// 零散的颜色过渡，所有表面在同一帧换色。样式见 styles/fork-motion.css。

const themeSwitchingClass = 'theme-switching'
let themeSwitchFrame: number | undefined

function suspendThemeTransitions() {
  if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') return
  const root = document.documentElement
  root.classList.add(themeSwitchingClass)
  if (themeSwitchFrame !== undefined) window.cancelAnimationFrame(themeSwitchFrame)
  themeSwitchFrame = window.requestAnimationFrame(() => {
    themeSwitchFrame = window.requestAnimationFrame(() => {
      root.classList.remove(themeSwitchingClass)
      themeSwitchFrame = undefined
    })
  })
}

/** 切换 html 上的 dark 类；主题按钮都经由这里，不直接改 classList。 */
export function applyDarkClass(isDark: boolean) {
  const root = document.documentElement
  if (root.classList.contains('dark') !== isDark) suspendThemeTransitions()
  root.classList.toggle('dark', isDark)
}
