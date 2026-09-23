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
  routes through its own gateway and has no first-party client to imitate),
  `enigo`/`xcap`/`image`/`cpal` left off (Waku has its own Computer Use), and
  the Responses adapter taught to take a gateway endpoint plus a bearer key
  (`api/src/providers/codex.rs::with_gateway`, wired in `api/src/registry.rs`)
  so any model can be driven through any of the three wire formats, a
  container-level `#[serde(default)]` on `Config` (`core/src/lib.rs`) so the
  partial `config` block Waku writes loads instead of failing the file, an
  explicit `config.provider` honoured even when it is `anthropic`
  (`query/src/lib.rs` — otherwise the engine's model-name family table
  re-routes `grok-*` to xai and `gemini-*` to google, neither configured
  here), a stream's first `error` event kept and used to end the turn
  (`api/src/lib.rs` + `query/src/lib.rs` — it used to be logged and dropped,
  surfacing as a turn that finished with nothing to say), and
  `attribution_text` naming the product instead of claiming to be Anthropic's
  official CLI (`core/src/system_prompt.rs`, brand from
  `SUB2API_BRAND_NAME`), the Grok family admitted to the reasoning-model list
  so its effort picker reaches the request at all
  (`query/src/runner/provider_options.rs`), and truncation on character
  rather than byte boundaries in `tools/src/{pty_bash,powershell,web_fetch}.rs`
  (the byte slices panicked on any long non-ASCII output), plan mode widened
  from "reads only" to also allow the tools planning itself needs and any
  shell invocation the classifier proves read-only
  (`core/src/lib.rs`, `core/src/bash_classifier.rs`,
  `tools/src/{lib,pty_bash}.rs` — without this the agent could not take
  notes, ask a question, or leave plan mode, and `ls -la | head` was
  refused), and refusals that carry their reason
  (`tools/src/lib.rs::denial_message` plus the exported
  `PLAN_MODE_DENIAL_SUFFIX` / `KEEP_PLANNING_DENIAL` markers, which
  `waku-core`'s driver matches to say them in the user's language), leaving
  plan mode gated on the user seeing the plan (`tools/src/exit_plan_mode.rs`
  declares `self_gates` and asks for permission itself, passing the summary),
  and a Responses `function_call`'s `arguments` read as the already-parsed
  object a normalizing gateway returns and not only as the string the API
  specifies (`api/src/providers/codex.rs::decode_tool_arguments` — the string
  form alone dropped the arguments and ran the tool with none, which is
  indistinguishable from a call that takes none; a parse failure still yields
  `{}` but now logs, in `api/src/provider_types.rs` and
  `query/src/runner/tools.rs`), and on Windows the Bash tool run through Git
  Bash rather than `cmd /C` (`core/src/shell.rs` finds it; `tools/src/pty_bash.rs`
  uses it and keeps the working directory with `pwd -W`), or through PowerShell
  when Git Bash is missing (same wrapper, `-EncodedCommand`, and a PowerShell
  read-only check for plan mode in `core/src/ps_classifier.rs`), with output read
  from both pipes at once, decoded per line in the console code page when it
  is not UTF-8, and process trees killed on timeout (`tools/src/capture.rs`,
  also used by `tools/src/powershell.rs`), and the environment block in
  `core/src/system_prompt.rs` stating the real shell and today's date, and a sub-agent
  started without `max_turns` left uncapped rather than stopped at ten tool
  rounds (`query/src/agent_tool.rs`; the main session's cap is lifted in the
  bridge's `build_query_config`, no engine change), and the effort level
  mapped onto DeepSeek's and GLM's thinking switch and Kimi K3's
  `reasoning_effort` on any OpenAI-compatible route
  (`query/src/runner/provider_options.rs` — upstream nested the DeepSeek
  mapping where it never ran).
- Computer Use and image generation reach the built-in agent with **no engine
  change at all**: the bridge pushes `waku_js_repl` into the session's
  `Config.mcp_servers`, `GuiPermissionHandler` promotes only *undecided*
  requests for those tools to `Allow` (plan mode and written deny rules still
  win), and the bundled skill is written to the engine's own config directory
  because its `Skill` tool reads flat `<name>.md` files rather than the
  `SKILL.md` directories the app ships. `generate_image` is a third tool on
  the REPL (`client/src/js_repl_image.rs`) so one implementation serves the
  built-in agent and every wired CLI; its key is the gateway's `openai` one
  (falling back to `default`, never `anthropic` — images dispatch on the
  key's group platform). A helper that is not installed only turns desktop
  control off for that session (`driver/support.rs::optional_computer_use`,
  used by every driver); it never fails a session or the message that
  started it.
- Adapter, which is ours and where changes belong:
  `client/crates/waku-agent-bridge/` (engine lifecycle, permission bridge,
  history ownership, steering, MCP tool wrapper, background-work snapshots,
  user questions, one-shot prompts) and
  `client/crates/waku-core/src/driver/native.rs` (`AgentEvent` → `DriverEvent`,
  `DriverControl`, transcript files). The bridge depends on neither
  `waku-core` nor `waku-protocol` on purpose.
- Settings surface: `client/src/app/agent_page.rs` (Settings → Agent) over
  `client/crates/sub2api/src/agent_settings.rs`, which edits only the keys it
  owns in the engine's `settings.json`. **Everything** about the built-in
  agent lives on that page — which endpoint serves each of its three APIs (a
  list beside one detail pane), behaviour, tools, MCP servers, permission
  rules and its enable switch. It deliberately has no card on the Providers
  page: no binary, no version, no installer. Do not let a merge re-add one.
  What an endpoint *is* — address, key, wire format, models — lives in the
  fork-local provider registry (`client/crates/sub2api/src/providers.rs`,
  edited on Settings → Model providers, `client/src/app/model_providers_page.rs`);
  every slot, CLI and built-in alike, only points at one by `provider_ref`,
  and `CustomApiConfig::resolved_endpoint` is the single place that
  resolution happens. Routing is written by
  `client/crates/sub2api/src/global_config/native.rs` like every other
  provider's. `client/src/app/native_agent.rs` feeds the picker from the
  gateway catalog. On the managed gateway each model goes out with the key of
  the group that serves it: `client/crates/sub2api/src/model_routing.rs` picks
  it (the CLI slot's group for Claude, GPT and Grok, else an active
  subscription group, else any group that lists it) and
  `client/src/app/cloud_subscriptions.rs` refreshes it together with the
  subscriptions Settings → Cloud Account lists; the per-model keys ride in
  `gateway_keys.models`. Do not go back to one key per platform — a group
  serves only the models its accounts map, so a DeepSeek model sent with the
  Codex group's key comes back "no available channel".
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

### 推荐邀请奖励 (Referral rewards, local implementation)

- Users share `/register?ref=<code>` links; the invitee's first balance recharge triggers two-way rewards. Backend: `backend/internal/service/referral_service.go` (+ tests, incl. `referral_code_stability_test.go`), `backend/internal/repository/referral_repo.go` / `referral_cache.go` / `user_repo_referral.go` (`GetByReferralCode` + `SetReferralCodeIfEmpty` conditional write), `backend/internal/handler/referral_handler.go` + `handler/admin/referral_handler.go`, the `referral_*` setting keys, migration `053_add_referral_system.sql` (partial unique index on `users.referral_code` — the ent schema deliberately has no `.Unique()`), and the reward trigger in `redeem_service.go`. Frontend: `views/user/ReferralView.vue`, the `?ref=` capture via `resolveAffiliateReferralCodeFromQuery` (`utils/oauthAffiliate.ts`, used by `RegisterView.vue`, the OAuth sections, and the logged-in redirect in `router/index.ts`).
- **Merge landmines (each has already been lost once in an upstream merge)**: the `ReferralCode: u.ReferralCode` line in `userEntityToService` (`backend/internal/repository/api_key_repo.go`) — losing it makes every referral-page visit regenerate and overwrite the user's code, killing all shared links; the `?ref=` query capture in `RegisterView.vue`; the `SetReferralCode` call in `user_repo.go`'s create builder. Regression tests `TestUserRepositoryGetByIDRoundTripsReferralCode` and `TestReferralService_GenerateReferralCode_StableAcrossCalls` pin these — keep them green after every sync.

### 启动期自动迁移与 SKIP_SETUP 只读接入 (startup migrations gate, local implementation)

- 上游只在安装路径（`AutoSetupFromEnv` / 向导安装）里执行迁移；本 fork 在 `backend/cmd/server/main.go` 的 `runMainServer` 里额外调用 `setup.MigrateOnStartup`（`backend/internal/setup/startup_migrations.go` + `_test.go`），让已安装实例升级重启时也对齐 schema。
- `SKIP_SETUP=true` 的实例被视为只读接入一个已初始化的库：既跳过安装流程，也跳过启动期迁移，不会对 schema 做任何改动。这类实例要求库的 schema 不低于其镜像版本。
- 上游合并时保留 `MigrateOnStartup` 调用与该开关，不要退回到"只在安装路径迁移"的上游行为。

### 运维管理员角色 (Operator role, local implementation)

- A read-only troubleshooting role `operator` (`domain.RoleOperator`, UI "运维管理员 / Operator") that can only reach the ops monitoring and global usage log pages. Authorization is a **default-deny static allowlist** of `METHOD + gin FullPath` entries in `backend/internal/server/middleware/console_scope.go`, enforced inside `validateJWTForAdmin` (`middleware/admin_auth.go`) so every reuse of `adminAuth` (pay probe, page routes, WebSocket handshake) denies operators by default; denials are audited as `admin.scope.denied`, and operator GET reads are always audited (`middleware/audit_log.go`). Golden test `backend/internal/server/routes/console_scope_coverage_test.go` pins the allowlist: any upstream route added later stays invisible to operators until explicitly allowlisted there.
- Response projection for operators: `backend/internal/handler/dto/operator_usage.go` (no plaintext key / balances / session id / account cost, masked IP via `internal/pkg/ip.MaskIP`), `backend/internal/handler/admin/ops_operator_redact.go` (masked `client_ip`), `handler/admin/console_handler.go` (`GET /admin/console/session`), plus `IsPrivilegedRole` / `IsConsoleUser` in `internal/service/user.go`, step-up + self-demotion guards in `handler/admin/user_handler.go`, and the last-admin + single-admin guards (`ensureNotLastAdmin` / `ensureNoOtherAdmin`) in `service/admin_user.go` — the system allows exactly one `admin`; everyone else must be `operator` or `user`. The admin usage list also stopped embedding plaintext API keys (`apiKeyWithoutSecret` in `dto/mappers.go`).
- Frontend: `isOperator` / `hasConsoleAccess` / `homePath` in `frontend/src/stores/auth.ts`, `meta.operatorAllowed` on `/admin/ops` and `/admin/usage` in `router/index.ts` (+ `meta.d.ts`, `setupRedirect.ts`), operator nav in `components/layout/AppSidebar.vue`, `api/admin/console.ts`, the operator branch in `stores/adminSettings.ts`, read-only mode in `views/admin/ops/OpsDashboard.vue` (+ `OpsDashboardHeader` / `OpsSystemLogTable` / `OpsAlertEventsCard`), `views/admin/UsageView.vue` (+ `UsageFilters`, `UsageTable`), admin-only IP geo lookups in `components/common/IpGeoCell.vue` / `IpGeoBatchToolbar.vue`, role labels `admin.users.roles.operator` and `operator.*` keys in `i18n/locales/{zh,en}/fork.ts`.
- 运维写操作审批 (operator write approval, local implementation): operators may open user management (`/admin/users`) and subscription management (`/admin/subscriptions`), but every write there is captured inside `validateJWTForAdmin` (`middleware/admin_auth.go` + `middleware/admin_auth_approval.go`; tables `operatorApprovalScope` / `operatorRefusedScope` in `console_scope.go`, pinned by the golden test), stored AES-encrypted in `admin_approval_requests` (`migrations/239_admin_approval_requests.sql`, `ent/schema/admin_approval_request.go`, `repository/admin_approval_repo.go`) and answered with HTTP 202. The single admin approves with one click, singly or in bulk (`POST /admin/approvals/:id/approve`, `POST /admin/approvals/batch-approve` for up to `approval_batch_limit` ids (site setting, default 50) replayed one by one; the per-operator pending cap is the sibling setting `approval_pending_limit_per_user` (default 20), both parsed in `service/admin_approval_limits.go` and returned by `pending-count` so the approvals page batches accordingly, `handler/admin/approval_handler.go`, `dto/approval.go`, routes in `routes/admin.go`) and `service/admin_approval_service.go` replays the original request through the gin engine (`SetDispatcher` in `server/router.go`) with the Go-context `ApprovalReplay` marker from `service/admin_approval_replay_ctx.go` (`auth_method=approval_replay`), so every existing guard runs as the admin; a missing gate fails closed (503). Deleting users and any role change are refused outright (403). New requests are pushed via Server酱³ (`service/approval_notify_service.go`, setting `approval_notify_serverchan_enabled`, toggle in `ForkSettingsSection.vue`). Operator projections on the users pages: `dto/operator_usage.go` (`APIKeyFromServiceOperator`, `AdminUserForOperator`), balance-history codes hidden. Frontend: `utils/approval.ts`, `utils/approvalDescribe.ts` (renders each queued request as plain language, never a raw JSON blob), the 202 branch in `api/client.ts`, `api/admin/approvals.ts`, `stores/approvals.ts`, `views/admin/ApprovalsView.vue`, the approvals nav entry + badge in `AppSidebar.vue`, `operatorAllowed` on the users / subscriptions / approvals routes, the readonly branches in `views/admin/UsersView.vue` / `SubscriptionsView.vue` and the `components/admin/user/*` modals (their group dropdowns come from `GET /admin/usage/filter-groups`, whose projection carries `status` / `subscription_type` / `is_exclusive` so subscription groups stay assignable), `operator.approval.*` / `nav.approvals` keys in `fork.ts`. Keep on upstream merges; never let a merge route operator writes around the capture.
- Design notes and the allowlist table live in `docs/OPERATOR_ROLE.md`. Keep this feature on upstream merges: resolve conflicts on the paths above in favour of the local version, never widen the allowlist while merging, and re-run the middleware / routes / handler tests afterwards.

### 工单系统 (Support tickets, local implementation)

- Users open support tickets (title + fixed category + Markdown body) and talk to staff in one thread; the single `admin` and the fork-local `operator` role are **equal peers** here: both see every ticket and reply / close / reopen directly. Operator writes are explicitly allowed in `operatorWriteScope` (never queued through the approval flow) and fully audited (`admin.tickets.reply|close|reopen`, extras `ticket_id` / `ticket_status` / `ticket_user_id`). State machine `open` → (staff reply) `replied` → (user reply) `open`, `close` from either non-closed state, `reopen` from `closed`; all transitions are atomic `UPDATE ... WHERE status` in the repo. Limits: title ≤ 200 runes, body ≤ 5000 runes, 5 active tickets per user, `panelRateLimiter.Heavy()` on create / reply. Badges: staff = `open` count, user = `user_unread` count (set by staff replies, cleared when the user fetches the detail). Optional Server酱³ push on new tickets / user replies via `ticket_notify_serverchan_enabled` (reuses the group-status UID / SendKey; staff replies never push).
- Files: `backend/migrations/240_support_tickets.sql` (+ `support_tickets_migration_test.go`), ent schemas `support_ticket.go` / `support_ticket_message.go` (model parity only, repo is raw SQL), `backend/internal/service/ticket.go` / `ticket_service.go` / `ticket_notify_service.go` (+ tests), `ProvideTicketService` in `service/wire_local.go`, `backend/internal/repository/support_ticket_repo.go`, `backend/internal/handler/dto/ticket.go`, `backend/internal/handler/ticket_handler.go` (user side, + unit test), `backend/internal/handler/admin/ticket_handler.go` (staff side, + test), the `/tickets` group in `routes/user.go` and `registerTicketRoutes` in `routes/admin.go`, the eight ticket entries in `middleware/console_scope.go` (pinned by `routes/console_scope_coverage_test.go` and `middleware/console_scope_test.go`), the audit action overrides / extra keys in `middleware/audit_log.go` + `service/audit_log.go`, and the `ticket_notify_serverchan_enabled` setting plumbed through the same eight files as `approval_notify_serverchan_enabled`. Frontend: `frontend/src/api/tickets.ts`, `api/admin/tickets.ts`, `stores/tickets.ts`, `utils/tickets.ts`, `components/tickets/TicketThread.vue`, `views/user/TicketsView.vue` (`/tickets`), `views/admin/TicketsView.vue` (`/admin/tickets`, `operatorAllowed`), the tickets nav entries + `.sidebar-badge` styles in `components/layout/AppSidebar.vue`, the polling lifecycle in `App.vue`, the toggle in `components/admin/settings/ForkSettingsSection.vue`, and the `tickets` / `nav.tickets` / `admin.settings.site.ticketNotify` keys in `i18n/locales/{zh,en}/fork.ts` (+ view / store / sidebar specs).
- 图片附件 (ticket image attachments, local): the body stays plain Markdown — images are written as `![image](ticket-attachment://<key>)` with no attachment table. The content endpoint is header-authenticated like every other route, and a browser sends no `Authorization` header for `<img src>`, so a same-origin URL can **never** be hung directly off an `<img>` (it 401s and renders a broken image). `useTicketAttachmentImages` fetches the bytes through `apiClient`, wraps them in `URL.createObjectURL`, and `rewriteTicketAttachmentUrls` swaps the scheme for that `blob:` URL before `MarkdownRenderer` (revoked on unmount; a 1×1 transparent data URI stands in while loading). DOMPurify rejects `blob:` by default, so `renderSafeMarkdown` takes an `allowBlobImages` flag that only the ticket thread passes — do not widen it to the default path. Object keys are built entirely server-side (`<prefix><uid>/<yyyymm>/<rand8hex><ext>`, staff under `<prefix>staff/<uid>/`), the extension comes only from a `image/{jpeg,png,webp,gif}` Content-Type whitelist that `http.DetectContentType` must also agree with (a declared type can lie, and the bytes are served back from our own origin — letting HTML through would be a stored XSS), 5 MiB per image with `RequestBodyLimit` on the route, and the prefix **is** the read authorization boundary: a user may read `<prefix><own uid>/` plus any `<prefix>staff/` key that a staff-authored message in one of their own tickets references (`SupportTicketRepository.StaffAttachmentReferencedForUser`, the only way staff-pasted images reach the ticket owner; anything else is a flat 404, never a leak), staff may read the whole prefix, and `..` / `path.Clean` mismatches are rejected. Files: `backend/internal/service/ticket_attachment_service.go` / `ticket_attachment_storage_settings.go`, `repository/ticket_attachment_s3_store.go` (+ `wire.go` factory), `handler/ticket_attachment_handler.go`, `handler/admin/ticket_attachment_handler.go`, the storage-config trio in `handler/admin/backup_handler.go`, the attachment routes in `routes/user.go` / `admin.go`, the two `auditBodyOmittedRoutes` entries in `middleware/audit_log.go` (+ tests throughout); frontend `components/tickets/useTicketAttachments.ts` (upload) and `useTicketAttachmentImages.ts` (authenticated fetch → object URL → revoke), the `allowBlobImages` option in `utils/markdown.ts` + the matching prop on `components/common/MarkdownRenderer.vue` (pinned by its spec), the `ticketAttachment*` / `rewriteTicketAttachmentUrls` helpers in `utils/tickets.ts`, the `side` prop on `TicketThread.vue`, `uploadAttachment` / `fetchAttachment` in `api/tickets.ts` / `api/admin/tickets.ts`, the storage card in `views/admin/BackupView.vue` + `api/admin/backup.ts`, and the `tickets.attachments.*` / `admin.backup.ticketAttachmentStorage.*` keys.
- Attachment storage config (setting key `ticket_attachment_storage_config`, `/admin/backups/ticket-attachment-storage`) is **admin-only** — unlike the ticket routes themselves, operators are not allowlisted — and its PUT is step-up 2FA gated on both sides (`gin.HandlerFunc(stepUpAuth)` on the route, `backupStepUp.run(...)` in `BackupView.vue`): the prefix is the content endpoint's authorization boundary, so retargeting it changes what staff can read. The settings service additionally refuses a prefix overlapping the database-backup prefix, which would otherwise turn the content endpoint into a backup-download channel. Never drop either guard on a merge.
- Design notes in `docs/TICKETS.md`. Keep this feature on upstream merges: resolve conflicts on the paths above in favour of the local version and never let a merge move the operator ticket writes out of `operatorWriteScope` or behind `AdminOnly()`.

## Working Boundary

- During upstream syncs or conflict resolution, prioritize gateway-service changes and leave payment customizations untouched unless explicitly instructed.
- The same boundary applies to the local features listed above (模型广场 / LinuxDo OAuth): do not let an upstream merge silently overwrite or re-introduce their upstream counterparts.
