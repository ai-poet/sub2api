# 侧栏动态图标（fork）

侧栏菜单悬停时，图标播放自己的小动画（钥匙转动、柱状图重绘、齿轮旋转……）。实现移植自 TokenRouter 的 `frontend/src/components/icons/`（LGPL-3.0），这里只保留侧栏用到的图形。

- `AnimatedIcon.vue` 是入口，`registry.ts` 把语义名称映射到图形，`useIconAnimation.ts` 负责交互：动画绑定到最近的链接或按钮，悬停文字也会触发；键盘聚焦同样触发；触屏不模拟悬停；禁用、`inert`、加载转圈和系统的「减少动态效果」都会停止装饰动画；离开或卸载时取消动画序列。
- 侧栏通过 `components/layout/SidebarNavIcon.vue` 使用，按路径查 `components/layout/sidebarIconMap.ts`。`AnimatedIcon` 和 `motion-v` 懒加载并单独分包（`vite.config.ts` 的 `feature-motion-icons`），加载完成前和加载失败时显示原来的静态图标。
- 和上游 `components/icons/Icon.vue`（全站静态 heroicons）并存，互不替换。

## 来源与许可

- 动态图形来自 [Lucide Animated](https://github.com/pqoqubbw/icons/tree/072c38b1b04ea738d90a084485ccaad4b890ddca/icons)，版本 `072c38b1b04ea738d90a084485ccaad4b890ddca`，MIT 许可见 `LICENSE.lucide-animated`。
- 补充图形来自 [Lucide](https://github.com/lucide-icons/lucide/tree/66d8f9fc394b8530377e5f6112f0b8908ba01280/icons)，版本 `66d8f9fc394b8530377e5f6112f0b8908ba01280`，许可见 `LICENSE.lucide`。每个文件标注其来源。
- Vue 动画由 `motion-v`（固定 2.5.1）执行。

没有官方动画的 Lucide 图形使用 `createFallbackIcon`，在 400ms 内按 `1 → 1.06 → 1` 缩放一次。新增图标时先选择同语义的上游图形，复制对应的 artwork 文件，再在 `registry.ts` 登记。
