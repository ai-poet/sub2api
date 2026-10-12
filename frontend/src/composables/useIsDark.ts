import { onScopeDispose, ref, type Ref } from 'vue'

// fork：响应式的深色模式状态（TokenRouter 的 useTheme().isDark 的等价物）。
// 本项目的主题切换只改 <html> 的 dark 类（utils/themeSwitch.ts），这里监听这个类，
// 切换主题后依赖它的图表配置、背景动画随之更新；所在作用域销毁时停止监听。

const readDark = (): boolean =>
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark')

export function useIsDark(): Ref<boolean> {
  const isDark = ref(readDark())
  if (typeof MutationObserver !== 'undefined' && typeof document !== 'undefined') {
    const observer = new MutationObserver(() => {
      isDark.value = readDark()
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    onScopeDispose(() => observer.disconnect())
  }
  return isDark
}
