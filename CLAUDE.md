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

## Local Features (always keep)

The features below are locally maintained customizations of this fork. During upstream merges, cherry-picks, or conflict resolution the local implementation always wins; never replace them with the upstream counterpart unless an explicit task says otherwise.

### 模型广场 (Model Catalog, local implementation)

- Local implementation lives at the `/models` route: `frontend/src/views/user/ModelCatalogView.vue`, `frontend/src/api/modelCatalog.ts` (+ spec), `frontend/src/api/pricing.ts`, the `modelCatalog` i18n keys in `frontend/src/i18n/locales/{zh,en}/fork.ts`, the sidebar entry in `frontend/src/components/layout/AppSidebar.vue`; backend `backend/internal/service/model_catalog_service.go` (+ test), `backend/internal/handler/model_catalog_handler.go`, `backend/internal/handler/public_pricing_handler.go` (+ test), the routes in `backend/internal/server/routes/user.go`, and their wiring in `backend/internal/handler/wire.go`, `backend/internal/handler/handler.go`, `backend/internal/service/wire.go`, `backend/cmd/server/wire_gen.go`.
- Upstream ships its own "model plaza" (`backend/internal/handler/model_plaza_*`, `backend/internal/service/model_plaza_*`, `frontend/src/api/modelPlaza.ts`, `frontend/src/components/modelPlaza/**`, plus plaza entries in `frontend/src/views/HomeView.vue` and `frontend/src/utils/featureFlags.ts`). It was intentionally removed from this fork. Keep it removed: resolve modify/delete conflicts on those paths by deleting, drop new upstream files under those paths, and do not re-add plaza wiring or routes.

### GitHub OAuth login (local implementation)

- GitHub OAuth is a fork-local feature and must not be changed by upstream syncs: `backend/internal/handler/auth_github_oauth.go` (+ `_test.go`), `backend/internal/handler/auth_email_oauth.go`, `backend/internal/service/github_oauth_fork.go`, `backend/internal/service/setting_oauth.go`, the `github_oauth_*` settings/config in `backend/internal/config/config.go` and `backend/internal/handler/admin/setting_handler_*.go`; frontend `frontend/src/components/auth/EmailOAuthButtons.vue` (+ spec), the GitHub parts of `frontend/src/api/auth.ts`, `frontend/src/views/auth/LoginView.vue` / `RegisterView.vue`, and `frontend/src/components/admin/settings/ForkSettingsSection.vue`.
- On conflict keep the local version. Do not apply upstream changes to these files (including small "fixes" such as extra OAuth start parameters) unless explicitly asked.

### LinuxDo OAuth login (local registration flow)

- The LinuxDo OAuth registration/binding flow is locally customized: `backend/internal/handler/auth_linuxdo_oauth.go` (+ `_test.go`), `backend/internal/handler/auth_oauth_pending_flow.go` (+ test), `backend/internal/service/auth_oauth_email_flow.go`, the `linuxdo_*` settings in `backend/internal/service/setting_*.go` / `backend/internal/config/config.go`; frontend `frontend/src/components/auth/LinuxDoOAuthSection.vue`, `frontend/src/views/auth/LinuxDoCallbackView.vue`, `frontend/src/components/auth/PendingOAuthCreateAccountForm.vue` (+ spec), and the LinuxDo parts of `frontend/src/views/auth/RegisterView.vue` / `LoginView.vue`.
- Upstream also ships LinuxDo OAuth. On conflict keep the local version. Port upstream changes to these files only when explicitly asked, and re-run `backend/internal/handler/auth_linuxdo_oauth_test.go` and the frontend auth specs afterwards.

## Working Boundary

- During upstream syncs or conflict resolution, prioritize gateway-service changes and leave payment customizations untouched unless explicitly instructed.
- The same boundary applies to the local features listed above (模型广场 / LinuxDo OAuth): do not let an upstream merge silently overwrite or re-introduce their upstream counterparts.
