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

### 分组运行状态 Server酱³ 推送 (Group status ServerChan notify, local implementation)

- Pushes a Server酱³ message when a group's stable runtime status turns `down` or recovers (`up` event), on top of the fork-local group runtime status feature. Files: `backend/internal/service/group_status_notify_service.go` (+ `_test.go`), the `SetTransitionNotifier` hook in `backend/internal/service/group_status_probe_service.go`, `ProvideGroupStatusProbeService` in `backend/internal/service/wire_local.go`, `backend/internal/handler/admin/setting_handler_group_status_notify.go` (+ `_test.go`), the `group_status_notify_serverchan_*` setting keys across `setting_parse.go` / `setting_update.go` / `settings_view.go` / `dto/settings.go` / `setting_handler*.go`, the `notify_enabled` column (`backend/migrations/234_group_status_config_notify_enabled.sql`, `backend/ent/schema/group_status_config.go`, `backend/internal/repository/group_status_repo.go`); frontend `frontend/src/components/admin/settings/ForkSettingsSection.vue`, `frontend/src/components/admin/group/GroupRuntimeStatusDialog.vue`, the `groupStatusNotify` / `notifyEnabled` i18n keys in `frontend/src/i18n/locales/{zh,en}/fork.ts`.
- The SendKey is stored only in the settings table, never echoed back (only `_configured`), and must never be added to public settings. Keep this feature on upstream merges.

### 纯 Sol 验证 (Group status Sol Juice probe, local implementation)

- A second, low-frequency identity probe for OpenAI groups on top of the fork-local group runtime status: one `reasoning=high` Responses request reads the model's internal "Juice" budget (Sol = 40, Terra = 32, Luna = 48) and turns a separate "Sol verified / Not Sol" badge red or green, with Server酱³ pushes on `sol_juice_mismatch` / `sol_juice_recovered`. Files: `backend/internal/service/group_status_sol_juice.go` (classifier + transition), `backend/internal/service/group_status_sol_juice_probe.go` (probe/payload/parser), the `sol_juice_*` fields in `group_status.go` / `group_status_service.go` / `group_status_runner_service.go` / `group_status_notify_service.go`, `backend/internal/repository/group_status_repo.go`, migration `backend/migrations/235_group_status_sol_juice.sql`, ent schemas `group_status_config.go` / `group_status_state.go` / `group_status_juice_record.go`, the `sol-juice/probe` route and `ProbeRuntimeStatusSolJuice` handler; frontend `GroupRuntimeStatusDialog.vue`, `ModelStatusView.vue`, `utils/groupStatus.ts`, and the `solJuice` i18n keys in `frontend/src/i18n/locales/{zh,en}/fork.ts`.
- The prompt wording, classifier and state machine are written in this repo; only the fingerprint values come from the (non-commercial licensed, git-ignored) `gpt56_api_detector/` reference. Never import or copy code from that directory. Keep this feature on upstream merges.

### Astra 指纹验证 (Group status Astra fingerprint check, local implementation)

- A third, independent probe for OpenAI groups next to liveness and Sol Juice: one run sends a tier-sized batch (20/50/100) of fixed short-answer prompts at `reasoning=low` to one account, normalizes the answers and scores them against the bundled meow benchmark (Astra / Sol / Terra / Luna distributions, family weights, per-tier thresholds); the verdict drives a separate "Astra 指纹" badge and `astra_mismatch` / `astra_recovered` pushes. Files: `backend/internal/service/group_status_astra_benchmark.go` (package parsing/validation, embedded loader), `group_status_astra_check.go` (normalizers, scoring, transition), `group_status_astra_check_probe.go` (run execution, async start), `group_status_astra_progress.go` (in-memory live progress + per-request samples exposed on the admin view), the `astra_check_*` fields in `group_status.go` / `group_status_service.go` / `group_status_runner_service.go` / `group_status_notify_service.go`, `backend/internal/repository/group_status_repo.go`, migrations `backend/migrations/236_group_status_astra_check.sql` and `237_group_status_astra_check_samples.sql`, ent schemas `group_status_config.go` / `group_status_state.go` / `group_status_astra_check_run.go`, the `astra-check/probe` route and `ProbeRuntimeStatusAstraCheck` handler; frontend `GroupRuntimeStatusDialog.vue` (card + polling), `ModelStatusView.vue`, `utils/groupStatus.ts`, `astraCheck` i18n keys in `frontend/src/i18n/locales/{zh,en}/fork.ts`.
- The benchmark data lives in `backend/resources/astra-benchmark/` (embedded via `go:embed`, provenance and PolyForm Noncommercial license in its `NOTICE.md`); the scoring engine is written in this repo. Never import or copy code from the git-ignored `meow-llm-detector-main/` reference. Sol Juice logic and wording stay unchanged. Keep this feature on upstream merges.

### 内置 Agent (Waku Agent / ProviderKind::Native, local implementation)

- The default provider is an agent engine compiled into the client, not a CLI
  it launches. It removes the Node + npm install step from first run entirely,
  which is why the onboarding checklist is two steps rather than three.
- Vendored engine: `client/crates/waku-agent/{core,api,tools,query,mcp,plugins}`
  — a copy of Claurst (GPL-3.0, same licence as this fork), kept as a pristine
  subtree so it can still be re-synced. **Do not hand-edit it** except for the
  three deliberate departures recorded in its manifests: `rusqlite` bumped to
  0.37 (`links = "sqlite3"` cannot coexist with `waku-core`'s copy), `wreq`/
  BoringSSL removed in favour of `reqwest` (`api/src/bun_tls.rs` — the fork
  routes through its own gateway and has no first-party client to imitate), and
  `enigo`/`xcap`/`image`/`cpal` left off (Waku has its own Computer Use).
- Adapter, which is ours and where changes belong:
  `client/crates/waku-agent-bridge/` (engine lifecycle, permission bridge,
  history ownership, steering, MCP tool wrapper, background-work snapshots,
  user questions, one-shot prompts) and
  `client/crates/waku-core/src/driver/native.rs` (`AgentEvent` → `DriverEvent`,
  `DriverControl`, transcript files). The bridge depends on neither
  `waku-core` nor `waku-protocol` on purpose.
- Settings surface: `client/src/app/agent_page.rs` (Settings → Agent) over
  `client/crates/sub2api/src/agent_settings.rs`, which edits only the keys it
  owns in the engine's `settings.json`; routing is written by
  `client/crates/sub2api/src/global_config/native.rs` like every other
  provider's. `client/src/app/native_agent.rs` feeds the picker from the
  gateway catalog.
- Upstream files carry only hook points: the `ProviderKind::Native` variant and
  its `is_builtin()` predicate (`waku-protocol/src/model.rs`), one match arm in
  `waku-core/src/driver/mod.rs`, and the built-in short-circuits in
  `provider_probe` / `provider_binary` / `driver_start_request_for_session` —
  every one of which exists because a built-in provider has no binary to find.
- Waku owns the transcript (`Vec<Message>` in
  `waku-agent-bridge/src/history.rs`), stored in `agent-sessions/` beside the
  daemon's state database. That is what makes rewind, branch and resume
  truncations rather than protocol calls; do not move it back into the engine.
- Keep this feature on upstream merges. Resolve conflicts on the paths above in
  favour of the local version, and re-read
  `client/docs/providers.md` § "Waku Agent (the built-in one)" before changing
  the driver contract for it.

### 守护进程断线重连与连接状态条 (Daemon socket reconnect + connection banner, local implementation)

- 本地 daemon 的 WebSocket 断开而进程仍在时，`client/crates/waku-client/src/process.rs` 的监督线程用 `connect_with_resume` 重连同一进程（拨号不在 `target` 锁内；连续 6 次被拒后回退到重启进程；短时间内反复断开按 1/2/4/8 s 退避），`DaemonSupervisor::restart` 供界面手动重启。桌面端用 `client/src/app/daemon_banner.rs` 的连接相位和顶部状态条替代反复弹出的 "Waku daemon disconnected" toast：`client/src/driver/mod.rs` 的 `transport_failure_notice`、`client/src/app/background_work.rs` 的 `should_refresh_background_work`、`client/src/app/runtime.rs::save`、`client/src/app/drafts.rs`、`client/src/app.rs` 的 `ToastState::refresh_if_same` 与 `show_toast_with_tone`；传输层错误文案常量与 `is_daemon_transport_error` 在 `client/crates/waku-client/src/client.rs`。同一改动还包括 `client/crates/waku-daemon/src/main.rs` 的 Windows 父进程存活判断（只有 `ERROR_INVALID_PARAMETER` 算父进程已死）和 `client/crates/waku-core/src/server.rs` accept 循环对瞬时错误的容忍。
- 上游 egoist/waku#218 是同一问题的未合并 PR（在 `target` 锁内阻塞拨号、无退避和回退）。上游合并时保留本地实现，不要用它替换；对应的 i18n 键 `daemon.restart` / `daemon.banner_*_detail` 在 `client/locales/{app,zh-CN,ja}.yml`。

### GitHub OAuth login (local implementation)

- GitHub OAuth is a fork-local feature and must not be changed by upstream syncs: `backend/internal/handler/auth_github_oauth.go` (+ `_test.go`), `backend/internal/handler/auth_email_oauth.go`, `backend/internal/service/github_oauth_fork.go`, `backend/internal/service/setting_oauth.go`, the `github_oauth_*` settings/config in `backend/internal/config/config.go` and `backend/internal/handler/admin/setting_handler_*.go`; frontend `frontend/src/components/auth/EmailOAuthButtons.vue` (+ spec), the GitHub parts of `frontend/src/api/auth.ts`, `frontend/src/views/auth/LoginView.vue` / `RegisterView.vue`, and `frontend/src/components/admin/settings/ForkSettingsSection.vue`.
- On conflict keep the local version. Do not apply upstream changes to these files (including small "fixes" such as extra OAuth start parameters) unless explicitly asked.

### LinuxDo OAuth login (local registration flow)

- The LinuxDo OAuth registration/binding flow is locally customized: `backend/internal/handler/auth_linuxdo_oauth.go` (+ `_test.go`), `backend/internal/handler/auth_oauth_pending_flow.go` (+ test), `backend/internal/service/auth_oauth_email_flow.go`, the `linuxdo_*` settings in `backend/internal/service/setting_*.go` / `backend/internal/config/config.go`; frontend `frontend/src/components/auth/LinuxDoOAuthSection.vue`, `frontend/src/views/auth/LinuxDoCallbackView.vue`, `frontend/src/components/auth/PendingOAuthCreateAccountForm.vue` (+ spec), and the LinuxDo parts of `frontend/src/views/auth/RegisterView.vue` / `LoginView.vue`.
- Upstream also ships LinuxDo OAuth. On conflict keep the local version. Port upstream changes to these files only when explicitly asked, and re-run `backend/internal/handler/auth_linuxdo_oauth_test.go` and the frontend auth specs afterwards.

### 启动期自动迁移与 SKIP_SETUP 只读接入 (startup migrations gate, local implementation)

- 上游只在安装路径（`AutoSetupFromEnv` / 向导安装）里执行迁移；本 fork 在 `backend/cmd/server/main.go` 的 `runMainServer` 里额外调用 `setup.MigrateOnStartup`（`backend/internal/setup/startup_migrations.go` + `_test.go`），让已安装实例升级重启时也对齐 schema。
- `SKIP_SETUP=true` 的实例被视为只读接入一个已初始化的库：既跳过安装流程，也跳过启动期迁移，不会对 schema 做任何改动。这类实例要求库的 schema 不低于其镜像版本。
- 上游合并时保留 `MigrateOnStartup` 调用与该开关，不要退回到"只在安装路径迁移"的上游行为。

### 运维管理员角色 (Operator role, local implementation)

- A read-only troubleshooting role `operator` (`domain.RoleOperator`, UI "运维管理员 / Operator") that can only reach the ops monitoring and global usage log pages. Authorization is a **default-deny static allowlist** of `METHOD + gin FullPath` entries in `backend/internal/server/middleware/console_scope.go`, enforced inside `validateJWTForAdmin` (`middleware/admin_auth.go`) so every reuse of `adminAuth` (pay probe, page routes, WebSocket handshake) denies operators by default; denials are audited as `admin.scope.denied`, and operator GET reads are always audited (`middleware/audit_log.go`). Golden test `backend/internal/server/routes/console_scope_coverage_test.go` pins the allowlist: any upstream route added later stays invisible to operators until explicitly allowlisted there.
- Response projection for operators: `backend/internal/handler/dto/operator_usage.go` (no plaintext key / balances / session id / account cost, masked IP via `internal/pkg/ip.MaskIP`), `backend/internal/handler/admin/ops_operator_redact.go` (masked `client_ip`), `handler/admin/console_handler.go` (`GET /admin/console/session`), plus `IsPrivilegedRole` / `IsConsoleUser` in `internal/service/user.go`, step-up + self-demotion guards in `handler/admin/user_handler.go`, and the last-admin + single-admin guards (`ensureNotLastAdmin` / `ensureNoOtherAdmin`) in `service/admin_user.go` — the system allows exactly one `admin`; everyone else must be `operator` or `user`. The admin usage list also stopped embedding plaintext API keys (`apiKeyWithoutSecret` in `dto/mappers.go`).
- Frontend: `isOperator` / `hasConsoleAccess` / `homePath` in `frontend/src/stores/auth.ts`, `meta.operatorAllowed` on `/admin/ops` and `/admin/usage` in `router/index.ts` (+ `meta.d.ts`, `setupRedirect.ts`), operator nav in `components/layout/AppSidebar.vue`, `api/admin/console.ts`, the operator branch in `stores/adminSettings.ts`, read-only mode in `views/admin/ops/OpsDashboard.vue` (+ `OpsDashboardHeader` / `OpsSystemLogTable` / `OpsAlertEventsCard`), `views/admin/UsageView.vue` (+ `UsageFilters`, `UsageTable`), admin-only IP geo lookups in `components/common/IpGeoCell.vue` / `IpGeoBatchToolbar.vue`, role labels `admin.users.roles.operator` and `operator.*` keys in `i18n/locales/{zh,en}/fork.ts`.
- Design notes and the allowlist table live in `docs/OPERATOR_ROLE.md`. Keep this feature on upstream merges: resolve conflicts on the paths above in favour of the local version, never widen the allowlist while merging, and re-run the middleware / routes / handler tests afterwards.

## Working Boundary

- During upstream syncs or conflict resolution, prioritize gateway-service changes and leave payment customizations untouched unless explicitly instructed.
- The same boundary applies to the local features listed above (模型广场 / LinuxDo OAuth): do not let an upstream merge silently overwrite or re-introduce their upstream counterparts.
