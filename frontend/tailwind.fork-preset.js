// fork：动效档位（见 src/styles/fork-motion.css）。
// duration-fast / normal / layout 与 ease-standard / exit 只引用 CSS 变量，
// 系统开启「减少动态效果」时变量缩到 1ms，用到这些工具类的过渡随之失效。
// 不改 Tailwind 的默认 duration / ease，上游组件的过渡手感保持不变。

/** @type {import('tailwindcss').Config} */
export default {
  theme: {
    extend: {
      transitionDuration: {
        fast: 'var(--motion-fast)',
        normal: 'var(--motion-normal)',
        layout: 'var(--motion-layout)'
      },
      transitionTimingFunction: {
        standard: 'var(--motion-ease)',
        exit: 'var(--motion-ease-exit)'
      }
    }
  }
}
