# 登录页背景动画

`auth-bg-light.json` / `auth-bg-dark.json` 原样取自 TokenRouter（`frontend/src/assets/lottie/`，LGPL-3.0，与本项目相同的许可），
由 `components/auth/fork/AuthBackground.vue` 按主题懒加载，用 `lottie-web` 播放。

| 文件 | sha256 |
| --- | --- |
| auth-bg-light.json | `254d0d7794050faeabc98e7bd5a0a6ac82c4c001d42f334c44de2fb047a0cf6e` |
| auth-bg-dark.json | `6b58b6be15ac0f8f0f42612f07a1618650c75cf6b20404f26e95c1ee4948b83b` |

动画内容是一张浅灰点阵图，被几组移动的形状遮罩着逐块显现，没有品牌色，所以不需要按本项目的配色改色。
不要手改这两个文件；需要更新时整份替换并更新上表。
