# CLAUDE.md

## Project Overview

sub2api 是一个 AI 模型网关服务，提供统一的 API 接口代理多种 AI 模型（OpenAI、Claude、Codex 等），附带用户管理、订阅计费和管理后台。

## Architecture

```
backend/       — Go 后端（Gin + Ent ORM + PostgreSQL + Redis）
frontend/      — 管理前端（Vue 3 + Vite + TypeScript + pnpm）
sub2apipay/    — 支付服务（Next.js + Prisma + Stripe）
client/        — Paseo 客户端子模块（移动端 agent 管理）
deploy/        — 部署配置（Docker Compose + Caddy）
tools/         — 辅助工具脚本
```

## Tech Stack

### Backend (Go 1.26)
- Web 框架: Gin
- ORM: Ent (entgo.io)
- 数据库: PostgreSQL + Redis
- 依赖注入: Wire
- HTTP 客户端: req/v3
- WebSocket: gorilla/websocket, coder/websocket
- 定时任务: robfig/cron
- 测试: testify + testcontainers

### Frontend (Vue 3)
- 构建: Vite
- 状态管理: Pinia
- UI: Ant Design Vue (implied by @lobehub/icons, chart.js)
- 国际化: vue-i18n
- 包管理: pnpm

### Payment Service (Next.js)
- ORM: Prisma
- 支付: Stripe
- 包管理: pnpm

## Common Commands

```bash
# Backend
cd backend && make build          # 编译
cd backend && make test           # 运行全部测试
cd backend && make generate       # 生成 Ent schema 和 Wire

# Frontend
cd frontend && pnpm dev           # 开发服务器
cd frontend && pnpm build         # 构建
cd frontend && pnpm lint          # ESLint
cd frontend && pnpm typecheck     # 类型检查

# Payment
cd sub2apipay && pnpm dev         # 开发服务器
cd sub2apipay && pnpm build       # 构建
cd sub2apipay && pnpm test        # 测试
```

## Development Guidelines

- 后端遵循 Go 标准项目布局，业务逻辑在 `backend/internal/` 下
- 前端使用 Vue 3 Composition API + TypeScript，组件使用 `<script setup>` 语法
- 提交信息使用 Conventional Commits 格式（feat/fix/chore/refactor/docs）
- 数据库变更通过 Ent schema 定义，运行 `make generate` 生成代码
- 部署使用 Docker Compose，配置参考 `deploy/config.example.yaml`

## Merge Rule

- When resolving conflicts against upstream, never merge upstream payment-related code into this project.
- Only merge upstream changes that are directly related to the gateway service itself.
- Treat payment code as locally maintained customization unless an explicit task says otherwise.
- During upstream merges or cherry-picks, local payment code always wins over upstream payment code.
- If an upstream commit mixes gateway fixes with payment changes, merge only the gateway-service portion and keep the local payment implementation unchanged.
- For upstream sync work, do not directly merge `upstream/main` by default. Compare the commit range first, then cherry-pick only the upstream commits that are safe and relevant to the gateway service.
- Prefer cherry-picking small, self-contained gateway fixes. Do not pull in payment-focused commits, broad refactors, or large mixed changes unless the task explicitly says to do so.
- If an upstream commit contains both gateway and payment changes, do not cherry-pick it wholesale. Keep only the gateway-related portion or skip it.
- If a cherry-pick becomes empty because the fix already exists locally, skip it rather than forcing a duplicate commit.

## Working Boundary

- During upstream syncs or conflict resolution, prioritize gateway-service changes and leave payment customizations untouched unless explicitly instructed.
