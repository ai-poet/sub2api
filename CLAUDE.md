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

### meow 指纹验证 (Group status meow fingerprint check, multi-model, local implementation)

- The identity probe for OpenAI **and** Anthropic groups on top of the fork-local group runtime status; historically "Astra 指纹验证", so tables, routes, fields and i18n keys keep the `astra_check_*` / `astraCheck` names. It replaced both the former "纯 Sol 验证" (Sol Juice) and the short-lived ModelTrace check. A group checks **several expected models** (`astra_check_models`, a JSONB list of `{expected_model, request_model}`, at most 8) picked from a fixed per-platform target list (`astraCheckTargets` in `group_status_astra_benchmark.go`: `gpt-5.6-sol`, `gpt-6-sol`, `gpt-6-astra`, `gpt-6.1-sol`; `claude-opus-5.5`, `claude-opus-5`, `claude-fable-5.1`; each target has a `Method`: `sol_juice` for `gpt-5.6-sol`, `modeltrace` for `gpt-6.1-sol` / `claude-opus-5.5` / `claude-opus-5`, `meow` for `gpt-6-sol` / `gpt-6-astra` / `claude-fable-5.1`; tests pin every meow target to its package and every modeltrace target to the ModelTrace bank). **GPT-5.6 Sol uses the Juice reading, not the benchmark** (`group_status_sol_juice.go`, restored from the former 纯 Sol 验证): one Responses request at `reasoning=high` asks for the internal "Juice" number; 40 (or 40 followed by digits) is Sol and a match, 32 / 48 / 96 / 64 are Terra / Luna / GPT-5.5·5.4 / GPT-5.4 mini and a mismatch pointing at that model, anything else is insufficient (`juice_inconclusive`). It is one request, ignores the tier, is not retried on a bad answer, and keeps the former check's leniency: it always takes the scheduler's first-choice account (no idle-account preference, `preferIdle=false`), its first-mismatch confirmation re-selects instead of pinning the same account, and its badge shows the stable verdict first (`normalizeAstraStateStatus` in `utils/groupStatus.ts` — a single mismatch or unreadable answer keeps a green badge green). It writes into the same per-model state, run table, confirmation re-run and pushes as the meow targets (a single cell `sol_juice`, scoring version `sol-juice-v1`). Only the fingerprint values come from the git-ignored `gpt56_api_detector/` reference; prompt, normalizer and classifier are ours. With official keys GPT-5.6 Sol drifted into strong directions on other models under the meow 4.5.3 package, which is why it was moved back to Juice. **Claude Opus 5.5, Claude Opus 5 and GPT-6.1 Sol use ModelTrace** (`group_status_modeltrace.go` challenges + Guard classifier, `group_status_modeltrace_run.go` execution, `group_status_modeltrace_bank.go` / `group_status_modeltrace_scoring.go` restored from the former ModelTrace check): 3 (topped up to at most 6) challenges asking for 292–332 first-instinct integers in 1..355, worded exactly like the official web version (https://xqy2006.github.io/ModelTrace/, `challenge-browser.js`; a test pins the pools), sent to Claude as one user message with `max_tokens` 4096 and no temperature / thinking (OAuth adds only the one-line Claude Code system + `metadata.user_id`); answers are attributed against the embedded MIT bank `backend/resources/modeltrace-bank/unified_bank.json` (same sha256 `1c2cb74d…` as the web version, `-text`, provenance in its `NOTICE.md`) with the Go port of `fingerprint-core.js` (parity to 1e-9 pinned by `testdata/modeltrace/`), and classified with Guard thresholds (match: expected top at ≥ 0.5; mismatch: another model ≥ 0.8 with expected ≤ 0.15 and gap ≥ 0.65; < 2 valid outputs is insufficient). The bank id `claude-opus-5-5` maps to the target id `claude-opus-5.5` via `TraceModelID` (`claude-opus-5` is the same id on both sides; its reference answers and Opus 5.5's are told apart, pinned by `TestModelTraceProbe_Opus5AndOpus55TellEachOtherApart`); Fable 5.1 is not in the bank and stays on meow. GPT-6.1 Sol is not in the bank either, but its number fingerprint is nearly identical to GPT-6 Astra's, so its target sets `TraceModelID: "gpt-6-astra"` with `TraceProxy`: attribution to Astra counts as a match (Astra is the expected row, on the 50% line), while `modelTraceCandidateID` never renames `gpt-6-astra` to 6.1 Sol in other targets' results; the state carries the derived `trace_proxy_model` so `GroupRuntimeStatusDialog.vue` colours that row green and says why (`modeltraceProxyHint`). A test fails once the bank enrolls GPT-6.1 Sol, so the target can move onto its own fingerprint. GPT challenges go through Responses in the environment the bank's GPT data was collected in (official Codex): the model's real Codex base instructions (`openai.CodexBaseInstructionsForModel`, on API-key accounts too), Codex's default `low` effort and `low` verbosity, `max_output_tokens` 8192 only as a runaway guard, no temperature (`createOpenAIModelTracePayload`; `parseOpenAIModelTraceStream` maps `max_output_tokens`, refusals and content filtering onto the same rejections as Claude). It ignores the tier and writes into the same per-model state, runs, confirmation re-run and pushes. For each meow model one run sends a tier-sized batch of the package's fixed short-answer prompts to one account of the group (GPT: Responses, system `.`, `reasoning=low`, 128 output tokens; Claude: Messages with the `claude-code` contract — system `.`, `max_tokens` 128, adaptive thinking + `output_config.effort`; OAuth adds only the one-line Claude Code system prompt, `metadata.user_id` and Claude Code headers), normalizes the answers (trim, casefold, unknown → `__UNSEEN_IN_TRAINING__`) and scores them with the **v3-predictive engine written in this repo** (`meow-fingerprint-v3-predictive`): per-cell Dirichlet-multinomial log evidence summed per source, "other" = the single nearest reference source over the whole run, displayed score = sigmoid((own − best rival) / valid answers); only the unique argmax that clears its own tier threshold on an eligible, calibrated sample (≥ `completion_ratio` of planned answers per cell and overall) is a strong direction. match = strong direction on the expected model; mismatch = strong direction on another candidate; anything else is insufficient and changes nothing.
- State is **per (group, expected model)** in `group_status_astra_check_states` (unique index); two consecutive mismatches flip that model's badge and push `astra_mismatch` / `astra_recovered` with `sub_status = <expected>:winner_<winner>`; the first mismatch is re-checked immediately **on the same account**. A run checks up to 4 models in parallel and the runner checks due groups in parallel; each model still locks one account (preferring accounts no other parallel run is using), and all probe requests on one account share a service-wide cap of min(account concurrency, 16) in flight. The default tier is **medium** (meow recommends it; on low, models with spread-out answers such as Sol drift into strong directions on other models); migration 244 only changed the column default, saved groups keep their tier. The runner only runs models whose last check is older than the interval; the admin can check all models or one (`POST .../astra-check/probe` with optional `{expected_model}`).
- Files: `backend/internal/service/group_status_astra_benchmark.go` (package parsing/validation, registry, targets), `group_status_astra_check.go` (normalizer, v3 scoring, transition, events), `group_status_astra_check_probe.go` (per-model execution, confirmation re-run, async start), `group_status_astra_accounts.go` (service-wide per-account in-flight slots + busy-account registry), `group_status_astra_progress.go` (live progress, one tracker per running model), the shared probe channels `group_status_probe_openai.go` / `group_status_probe_anthropic.go` (liveness keeps the minimal-header mode unchanged), the `astra_check_*` fields in `group_status.go` / `group_status_service.go` / `group_status_runner_service.go` / `group_status_notify_service.go`, `backend/internal/repository/group_status_repo.go` (+ `group_status_repo_astra_test.go`), migrations `backend/migrations/243_group_status_meow_multi_model.sql`, `244_group_status_astra_check_default_medium.sql` and `245_group_status_astra_check_reset_changed_methods.sql` (clears the meow-era state left on `gpt-5.6-sol` / `claude-opus-5.5` so the new methods run at once and start clean; `ComputeAstraCheckTransition` also silently resets the streak whenever a model's result source — `benchmark_package_id` — changes), ent schemas `group_status_config.go` / `group_status_astra_check_state.go` / `group_status_astra_check_run.go`, the `astra-check/probe` route and `ProbeRuntimeStatusAstraCheck` handler; frontend `GroupRuntimeStatusDialog.vue` (model picker, per-model results, polling; `__tests__/GroupRuntimeStatusDialog.meow.spec.ts`), `ModelStatusView.vue` (one badge per model, plus `components/user/FingerprintBenchmarkNotes.vue` at the page foot whenever a listed group has fingerprint checks on: per family the method, the models it judges and the open-source project — meow-llm-detector / ModelTrace, Juice marked as implemented here — from `FINGERPRINT_BENCHMARKS` in `utils/groupStatus.ts`, which must follow `astraCheckTargets`; keys `modelStatus.benchmarks.*`), `utils/groupStatus.ts`, `astraCheck` i18n keys in `frontend/src/i18n/locales/{zh,en}/fork.ts`.
- Benchmark data: two byte-exact v3 packages in `backend/resources/astra-benchmark/` (GPT 4.5.4 efficient for GPT-6 Sol / Astra, Claude 4.5.4 efficient; the GPT 4.5.3 package was dropped when GPT-5.6 Sol moved to Juice), embedded via `go:embed`, `-text` in `.gitattributes`, provenance / sha256 / PolyForm Noncommercial license in its `NOTICE.md`. Only the data is taken from meow; never import or copy code from the git-ignored `meow-llm-detector-main/` (or `gpt56_api_detector/`, `ModelTrace-main/`) references.
- Retired, dormant but kept (SKIP_SETUP / image rollback need older images to keep working; drop them in a later cleanup migration): the 235 `sol_juice_*` columns and `group_status_juice_records`, the 242 `modeltrace_*` columns and `group_status_modeltrace_runs`, and since 243 `group_status_configs.astra_check_request_model` and `group_status_states.astra_check_*`. Migration 243 moved enabled Astra / ModelTrace configs into `astra_check_models` and switched `modeltrace_enabled` off; migrations 235 / 242 are published and must not be edited. Old `sol_juice_*` / `modeltrace_*` events still render in the history. Keep this feature on upstream merges.

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
  `SUB2API_BRAND_NAME`), the Grok family and every GPT generation from 5 on
  admitted to the reasoning-model list so their effort picker reaches the
  request at all (`query/src/runner/provider_options.rs` — `gpt-6-*` fell
  through a `gpt-5` prefix and went out with no effort), and truncation on character
  rather than byte boundaries in `tools/src/{pty_bash,powershell,web_fetch}.rs`
  (the byte slices panicked on any long non-ASCII output), plan mode widened
  from "reads only" to also allow the tools planning itself needs and any
  shell invocation the classifier proves read-only
  (`core/src/lib.rs`, `core/src/bash_classifier.rs`,
  `tools/src/{lib,pty_bash}.rs` — without this the agent could not take
  notes, ask a question, or leave plan mode, and `ls -la | head` was
  refused; the classifier parses quoting and redirections and judges each
  simple command by its own rules — `cd`, `2>&1`, `>/dev/null`, `sed -n`,
  `awk`, `xargs` over readers, git's listing forms — and in plan mode an
  "always allow" rule no longer opens the shell or an editing tool,
  `plan_mode_overrides_allow_rule`), and refusals that carry their reason
  (`tools/src/lib.rs::denial_message` plus the exported
  `PLAN_MODE_DENIAL_SUFFIX` / `KEEP_PLANNING_DENIAL` markers, which
  `waku-core`'s driver matches to say them in the user's language), leaving
  plan mode gated on the user seeing the plan (`tools/src/exit_plan_mode.rs`
  declares `self_gates` and asks for permission itself, passing the plan:
  the whole plan is its required `plan` parameter, as in Claude Code and
  ZCode, and a call without one gets `MISSING_PLAN_ERROR` instead of a dialog),
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
  mapping where it never ran), the command queue's messages appended after
  the conversation rather than prepended to it
  (`query/src/lib.rs::inject_command_queue` — a steering message landed in
  front of the prompt it steered), compaction on every route through the
  session's own adapter, gated on the settings' `auto_compact` /
  `compact_threshold` and reported as `QueryEvent::Compaction`
  (`query/src/{lib,compact}.rs` — only the Messages route used to compact,
  so Responses and Chat sessions overflowed), the Responses usage no longer
  counting cached tokens twice (`api/src/providers/codex.rs::responses_usage`),
  and the keyword effort / persona read from the last message a person wrote
  (`query/src/lib.rs::last_written_user_message`), and prompt-cache
  breakpoints on every Messages request, placed where Claude Code places
  them — the last tool, the system prompt, the last message and the user
  message before it (`api/src/prompt_cache.rs`, called from
  `api/src/providers/anthropic.rs::build_request` and from the request the
  Anthropic route actually sends in `query/src/lib.rs` — that route bypasses
  the provider adapter; the engine never set one, so a route straight to
  Anthropic, such as a Claude Max group, re-read the whole conversation at
  full price on every call while Claude Code was cached on the same route),
  and the reasoning effort sent to current Claude families as adaptive
  thinking plus `output_config.effort`, the way Claude Code sends it, instead
  of a thinking budget (`api/src/claude_effort.rs`, a `CreateMessageRequest`
  `output_config` field, `query/src/lib.rs` — Opus 5.5 rejects a budget, and
  gateways read the effort from that field; older models keep the budget;
  Sonnet 5.5 has its own row and a dotted `claude-sonnet-5.5` reads as the
  dashed id, as the gateway's `effort_catalog.go` does),
  and request URLs built by `api/src/endpoint.rs::versioned_url`, which
  appends `/v1` only when the base does not already end in a version segment
  (`api/src/lib.rs`, `api/src/providers/openai.rs`, `api/src/registry.rs` —
  Zhipu's `…/api/paas/v4` and Volcengine Ark's `…/api/v3` became
  `…/v4/v1/…` and answered 404, which the engine reported as "Model not
  found"; `client/crates/sub2api/src/gateway.rs::versioned_url` mirrors it
  for the probes and the CLI writers),
  and the reasoning made visible on every route: adaptive thinking sent with
  `display: "summarized"` as Claude Code sends it (a `display` field on
  `ThinkingConfig` in `api/src/lib.rs`, `ThinkingConfig::summarized`,
  `claude_effort.rs` — these families default to `"omitted"`, whose thinking
  blocks stream empty), the Responses stream reading
  `response.reasoning_summary_text.delta` (`api/src/providers/codex.rs` — the
  summary it asked for fell into `_ => {}`) with plain thinking text no longer
  replayed as a `reasoning` input item (`api/src/providers/copilot.rs` — an
  item with no id the API cannot place), the Chat Completions stream reading
  `reasoning_content` / `reasoning` (`api/src/providers/openai.rs`), and an
  unsigned thinking block — a Chat model's reasoning kept for the transcript —
  left out of an Anthropic request (`ApiMessage::from` in `api/src/lib.rs`),
  and image blocks sent to every model instead of being swapped for
  "[Image not supported by this model]" wherever the registry or adapter
  claims no vision (`query/src/lib.rs` — the Responses adapter claims it for
  every model, so GPT never saw a picture; a model without vision now answers
  with its API's error). The pictures themselves come from the bridge
  (`waku-agent-bridge/src/images.rs`): the engine never read the `@path`
  mentions the composer appends per attachment, so the bridge sends each
  mentioned PNG / JPEG / GIF / WebP as an image block ahead of the text.
  One more departure: on the Responses and Chat Completions routes the calls
  of one assistant message no longer run strictly one after another — each
  stretch of calls that may overlap (sub-agents and read-only tools,
  `runner/tools.rs::{runs_concurrently, concurrency_runs}`) runs as one
  `run_tool_batch`, the way the Messages route runs a message's calls, while
  writes, commands and state switches still run alone and in order, results
  keep the calls' order, and a cancel abandons the batch in flight
  (`query/src/lib.rs` provider branch — sub-agents asked for together ran back
  to back on every GPT, Grok and Chinese model).
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
- Sub-agents are the bridge's own `Agent` tool
  (`client/crates/waku-agent-bridge/src/subagent.rs`), **not** the engine's
  `claurst_query::AgentTool`, which stays vendored and unused — do not put it
  back into `builtin_tools`. Ours runs the public `run_query_loop` with a real
  event channel and the parent's client, registries, tool set (MCP included,
  minus `Agent` / `AskUserQuestion` / plan and worktree switches /
  `GoalComplete`) and session rules, and forwards what the child does as
  `AgentEvent::Subagent`; `native.rs` turns that into one `Subagent`
  background-work entry keyed by the parent call's id plus
  `BackgroundWorkEvent::Transcript` entries (`driver/subagent.rs`). Claude
  Code's `parent_tool_use_id` messages feed the same record
  (`driver/claude_subagent.rs`). The desktop shows a sub-agent call as one
  summary line (`src/app/subagent_row.rs`) that opens its live record in the
  right panel (`subagent_panel.rs`, store in `subagent_transcript.rs`). A
  foreground child runs on a Tokio task of its own (as a background one
  does): the engine polls a message's calls on the parent's one task, so a
  child waiting on an approval dialog or a synchronous tool froze its
  siblings; the call's drop guard cancels the child's token so it cannot
  outlive the call.
- Context windows: the gateway catalog's `context_window` (and windows a user
  declared on their own endpoint) are written as
  `provider_configs.*.options.context_windows` in the engine's
  `settings.json`; the bridge sizes the usage meter by them and lays them over
  the bundled model table as `modelOverrides` keyed under the model's *owning*
  provider (never `codex/` or the route's — a synthesized entry there would
  switch thinking off), so auto-compact fires at the real window.
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
  gateway catalog **and from every registry entry that routes and declares
  models** (`ProviderEntry::offers_models_to_agent`), ZCode-style: each such
  endpoint is a vendor-column entry of its own, its models listed as
  `custom:<provider id>::<model>` in the endpoint's declared format (no name
  rule overrules it — `native_wire_format`, `WireFormat::resolve`), with no
  binding needed; the sections end in "Manage models…". The writer files
  those endpoints under `provider_configs.anthropic.options.endpoints`, and
  the bridge's `select_route` routes such a session by that entry alone,
  rewriting all three engine entries to it and removing the gateway key table
  and the endpoint table from the session's config — a gateway key must never
  reach a third-party endpoint; never add a path that consults `gateway_keys`
  for a `custom:` platform. The three Settings → Agent lines stay whole-line
  overrides; a bound endpoint with models hides the gateway rows of its format
  (`shadowed_formats`), and binding one still empties the gateway key tables.
  Model providers has a per-model Test (`client/crates/sub2api/src/model_test.rs`),
  the request URL and the picker status; a 404 names model, route and URL
  (`AgentEvent::RouteNotFound`). On the managed gateway each model goes out with the key of
  the group that serves it: `client/crates/sub2api/src/model_routing.rs` picks
  it (the CLI slot's group for Claude, GPT and Grok; the Chinese models'
  own slot for DeepSeek, GLM, Kimi, MiniMax and Qwen, falling back to
  pay-as-you-go while a subscription picked there is spent; else a
  subscription group with room left, else a pay-as-you-go group, else a spent
  subscription) and `client/src/app/cloud_subscriptions.rs` refreshes it
  together with the subscriptions Settings → Cloud Account lists; the
  per-model keys ride in `gateway_keys.models`. A Chinese model with no keyed
  route yet is looked up on its own before its turn goes out
  (`sub2api::route_one_model`, `native_route_need` / `ensure_native_model_route`
  in `cloud_subscriptions.rs`) — otherwise the first turn after sign-in rides
  the general key to the wrong group. The Chinese models' groups are
  `openai` groups on the live gateway, so the group picker files groups by
  lane, not platform (`model_routing::group_lane`, `DOMESTIC_LANE`,
  `Credentials::domestic_group_id`, `pending_group_bindings` in
  `client/src/app/cloud_groups.rs`, which also moves a Codex or general slot
  off such a group); never offer them under Codex again — a Codex slot on
  one breaks every GPT request. 0.2.3's second "pay as you go" picker row
  (`<platform>+payg::<model>`) is gone; `migrate_legacy_pay_as_you_go`
  rewrites saved ids on launch and the bridge's `split_model` still strips
  the mark. Do not go back to one key per platform — a group
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

### 画图页 (Client image studio, local implementation)

- An "Images" (画图) row under Search in the desktop client's sidebar opens a page in the main column: one field where a prompt alone draws (`/v1/images/generations`) and a prompt plus dropped / pasted / picked pictures edits them (`/v1/images/edits`, multipart `image[]`). Model, group, size (priced from the catalog's `pricing_details.media_tiers`), quality and count sit under the field; pictures go to the user's Pictures folder (`Pictures/CheapRouter/YYYY-MM/`), and the gallery, its history and references live in `~/.cheaprouter/image-studio/`. Files: `client/src/app/image_studio.rs` (state, store, job driver), `client/src/app/image_studio_view.rs`, the shared request layer `client/crates/sub2api/src/images.rs` (also used by the agents' `generate_image` in `client/src/js_repl_image.rs`), `form_text` / `form_file` / `download_to` in `client/crates/sub2api/src/http.rs`, and the hook points listed in `client/docs/FORK.md`.
- **No backend change.** A job submits to `/v1/images/*/async` and polls `/v1/images/tasks/:id`; when the gateway answers `async image tasks are not enabled` (no object storage configured) it streams the synchronous endpoint instead, one picture per call, relying on the gateway's ten-second SSE keep-alive to get past Cloudflare's first-byte timeout. OpenAI groups default to `allow_image_generation = false` while the model catalog still lists image models under them, so a job walks the candidate groups (`images::image_route_candidates`) past a `403 Image generation is not enabled for this group`, remembers refusals for a day, and records the group that drew in `Credentials::image_groups`; `refresh_model_routes` keeps that group's key and routes the model there, so `gateway_keys.models` sends the agents' image calls to the same group. Keep this feature on upstream merges.

### 智能体团队 AgentTeams (Client agent teams, local implementation)

- **What it is.** A built-in agent session becomes a team's captain:
  - It drafts members and a task DAG, then waits for approval.
  - A scheduler then hands each ready task to a member: a persistent sub-agent with its own route, conversation and permission scope.
  - Members talk to the captain and to each other through mailboxes.
  - Every task passes the full quality gates.
- **Origin and licence.** Translated from dsh-agent-teams (`@nanmicoder/dsh-agent-teams` 0.1.22, MIT, licence in `client/NOTICE.md`). Design notes: `client/docs/agent-teams.md`.
- **Scope.** Built-in agent only, and no wire protocol change.
- **It replaced Settings → Workflow.** That page and `client/crates/workflow-engine` are deleted. Never let a merge bring them back.
- **Files.**
  - `client/crates/agent-teams/**`: pure logic, plus the synchronous runtime behind a `Host` trait.
  - `client/crates/waku-agent-bridge/src/team/**` (`TeamHost`, `BridgeHost`, `MemberRuntime`, `TeamTool`).
  - `client/crates/waku-agent-bridge/src/session_team.rs`, mounted in `session.rs` as `team_seam`.
  - The hooks in the bridge's `session.rs`, `permission.rs` (`MemberScope`, `release_where` / `release_member`), `subagent.rs` and `lib.rs`.
  - `member_feeds` and the member permission title in `client/crates/waku-core/src/driver/native.rs`; the `/agent-teams` entries in `composer_complete.rs`.
  - App: `client/src/app/team_panel.rs` (the right-panel Team surface), `client/src/app/agent_teams_settings.rs` (Settings → Agent → Teams), the `RightPanelSurface::Team` arms in `right_panel.rs` / `surface_bar.rs`, the member-permission retains in `streaming.rs` / `runtime.rs` / `sessions.rs`, and `assets/icons/users.svg`.
  - The `team.*` keys in `client/locales/{app,zh-CN,ja}.yml`.
- **Where things are stored.**
  - Team state: `<workspace>/.agent-teams/`.
  - Configuration: `agent-teams.json`, beside the engine's `settings.json`. Never put it in `settings.json`: the engine's save drops unknown keys.
- **Rules.**
  - **Lazy activation.** A session carries neither the team tools nor the captain protocol until a message starts with `/agent-teams`, or until an unarchived team it leads is found on disk. After that, the section is byte-identical on every turn.
  - **Member permission ids.** A member's permission request uses a `team:<uuid>:<member>` id. The app accepts it with no captain turn running and keeps it across `TurnFinished` and Stop. A captain cancel releases only non-team dialogs.
  - **Team mail is never steered.** Mail to a running captain is injected silently. It must never go through `push_steer`, which echoes into the transcript as the user's own message.
  - **Running members in every snapshot.** A running member must appear in every background-work snapshot. Otherwise `reconcile_live` marks it lost.
  - **Panel controls.** Panel buttons send a `<waku:agent-teams>` control that the bridge intercepts before a turn opens. They never post a chat message.
- **On upstream merges.** Keep this feature, and resolve conflicts on the paths above in favour of the local version.

### 计划审阅面板 (Client plan review surface, local implementation)

- When Claude Code or the built-in agent finishes planning (`ExitPlanMode`, a permission titled `plan.ready_title`), the plan opens in the right panel's Plan surface (`client/src/app/plan_review.rs`).
- **What the surface shows.**
  - The whole plan, rendered and selectable.
  - Every version the session submitted, with a line diff against the previous one (`similar`).
  - A multi-line note. "Quote selection" drops the selected plan text into it as a Markdown quote.
  - Sending the plan back refuses it and steers the note into the turn (`deny_permission_with_note` in `permission_card.rs`).
- **The card above the composer** keeps one row: the title, "View plan", and the two answers with their digit keys.
- **The built-in agent's plan text.** The dialog body is exactly `ExitPlanMode`'s required `plan` parameter (`client/crates/waku-agent/tools/src/exit_plan_mode.rs`). Never guess the plan from the model's reply: an earlier version did and showed "Let me present the plan." as the plan.
- **Hooks.**
  - `RightPanelSurface::Plan` arms and the `plan_review` field.
  - `plan_requested` after a permission is queued (`streaming.rs`).
  - `note_plan_answer` at the top of `respond_permission` (`sessions.rs`).
  - `plan_selected_text` in the copy chain.
- **On upstream merges.** Keep this feature, and resolve conflicts in favour of the local version.

### GitHub OAuth login (local implementation)

- GitHub OAuth is a fork-local feature and must not be changed by upstream syncs: `backend/internal/handler/auth_github_oauth.go` (+ `_test.go`), `backend/internal/handler/auth_email_oauth.go`, `backend/internal/service/github_oauth_fork.go`, `backend/internal/service/setting_oauth.go`, the `github_oauth_*` settings/config in `backend/internal/config/config.go` and `backend/internal/handler/admin/setting_handler_*.go`; frontend `frontend/src/components/auth/EmailOAuthButtons.vue` (+ spec), the GitHub parts of `frontend/src/api/auth.ts`, `frontend/src/views/auth/LoginView.vue` / `RegisterView.vue`, and `frontend/src/components/admin/settings/ForkSettingsSection.vue`.
- On conflict keep the local version. Do not apply upstream changes to these files (including small "fixes" such as extra OAuth start parameters) unless explicitly asked.
- The two `// fork：申诉` lines in `auth_github_oauth.go` (callback redirect + complete-registration JSON) and the two in `auth_email_oauth.go` belong to the appeal session (see 封禁申诉会话); keep them.

### LinuxDo OAuth login (local registration flow)

- The LinuxDo OAuth registration/binding flow is locally customized: `backend/internal/handler/auth_linuxdo_oauth.go` (+ `_test.go`), `backend/internal/handler/auth_oauth_pending_flow.go` (+ test), `backend/internal/service/auth_oauth_email_flow.go`, the `linuxdo_*` settings in `backend/internal/service/setting_*.go` / `backend/internal/config/config.go`; frontend `frontend/src/components/auth/LinuxDoOAuthSection.vue`, `frontend/src/views/auth/LinuxDoCallbackView.vue`, `frontend/src/components/auth/PendingOAuthCreateAccountForm.vue` (+ spec), and the LinuxDo parts of `frontend/src/views/auth/RegisterView.vue` / `LoginView.vue`.
- The `// fork：申诉` lines in `auth_linuxdo_oauth.go` (two callback redirects), `auth_oauth_pending_flow.go` (pending exchange) and the identity-carrying `findActiveUserByID` belong to the appeal session (see 封禁申诉会话); keep them.
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
- 运维个人令牌 (operator personal token, local implementation): an operator generates one token on the profile page (current password required; 30/90/180/365 days or never; regenerating overwrites) and scripts call the admin API with `Authorization: Bearer pat-<64hex>`. The token only replaces the login step: `adminAuth` routes `pat-` bearers to `validatePersonalTokenForAdmin` (`middleware/admin_auth_personal_token.go`), which authenticates via `service.PersonalTokenService.Authenticate` and then goes through `authorizeConsoleUser` in `middleware/admin_auth.go` — the single role gate shared with operator JWT sessions (allowlist, approval capture, refusals). Only `role == operator` passes (checked live in the service and again in the middleware; admins and users are 401). SHA-256 only in `personal_tokens` (`migrations/241_personal_tokens.sql`, `ent/schema/personal_token.go`, `repository/personal_token_repo.go`); the token dies on password/email change (`user_token_version` vs `User.TokenVersion`), expiry, revocation, the site switch `personal_token_enabled` (default off, plumbed like `ticket_notify_serverchan_enabled`), disabling, or leaving the operator role (`service/admin_user_personal_token.go` hook, injected by `ProvidePersonalTokenService`). A token never carries a session id, is rejected by `enforceStepUp` regardless of the switch, and cannot approve (`verifyApprover`). User-side (`jwtAuth`) routes also accept the token — `jwtAuthWithPersonalTokens` in `middleware/jwt_auth.go` → `validatePersonalTokenForUser` — because they only touch the operator's own account; the only exceptions are account security writes, denied by prefix in `middleware/personal_token_user_scope.go` (password, TOTP, passkeys, account bindings, auth identities, `/user/personal-token` itself, and every `/auth/*` write such as desktop-session / revoke-all-sessions) with 403 `PERSONAL_TOKEN_ROUTE_FORBIDDEN` + audit; reads are always allowed. Management endpoints `GET/POST/DELETE /api/v1/user/personal-token` therefore stay browser-session only (tokens cannot manage tokens); the admin view `GET /api/v1/admin/personal-tokens`, `DELETE /api/v1/admin/personal-tokens/:user_id` is `AdminOnly()` and pinned in `operatorForbiddenPrefixes`. Audit: `auth_method=personal_token` + `personal_token_id`. The guarantee is pinned by `routes/personal_token_scope_equivalence_test.go` (every adminAuth route: operator JWT vs the same operator's token must give identical status and handler reachability; every jwtAuth route across auth / user / pay / page registrations: allowed ones authenticate, account security writes get 403, and every deny prefix must match a registered write route). Frontend: `api/personalToken.ts`, `api/admin/personalTokens.ts`, `components/user/profile/ProfilePersonalTokenCard.vue` (operators only, plaintext shown once), `components/admin/settings/PersonalTokensPanel.vue` + toggle in `ForkSettingsSection.vue`, the `personal_token` option in `views/admin/AuditLogView.vue`, `operator.personalToken.*` / `admin.settings.site.personalToken.*` / `admin.audit.filters.personalToken` keys in `fork.ts`. Details in `docs/PERSONAL_TOKENS.md`. Keep on upstream merges; never add a token-only allow path on admin routes, never shrink the account-security deny list, and never accept tokens outside the Authorization header.
- Design notes and the allowlist table live in `docs/OPERATOR_ROLE.md`. Keep this feature on upstream merges: resolve conflicts on the paths above in favour of the local version, never widen the allowlist while merging, and re-run the middleware / routes / handler tests afterwards.

### 工单系统 (Support tickets, local implementation)

- Users open support tickets (title + fixed category + Markdown body) and talk to staff in one thread; the single `admin` and the fork-local `operator` role are **equal peers** here: both see every ticket and reply / close / reopen directly. Operator writes are explicitly allowed in `operatorWriteScope` (never queued through the approval flow) and fully audited (`admin.tickets.reply|close|reopen`, extras `ticket_id` / `ticket_status` / `ticket_user_id`). State machine `open` → (staff reply) `replied` → (user reply) `open`, `close` from either non-closed state, `reopen` from `closed`; all transitions are atomic `UPDATE ... WHERE status` in the repo. Limits: title ≤ 200 runes, body ≤ 5000 runes, 5 active tickets per user, `panelRateLimiter.Heavy()` on create / reply. Badges: staff = `open` count, user = `user_unread` count (set by staff replies, cleared when the user fetches the detail). Optional Server酱³ push on new tickets / user replies via `ticket_notify_serverchan_enabled` (reuses the group-status UID / SendKey; staff replies never push).
- Files: `backend/migrations/240_support_tickets.sql` (+ `support_tickets_migration_test.go`), ent schemas `support_ticket.go` / `support_ticket_message.go` (model parity only, repo is raw SQL), `backend/internal/service/ticket.go` / `ticket_service.go` / `ticket_notify_service.go` (+ tests), `ProvideTicketService` in `service/wire_local.go`, `backend/internal/repository/support_ticket_repo.go`, `backend/internal/handler/dto/ticket.go`, `backend/internal/handler/ticket_handler.go` (user side, + unit test), `backend/internal/handler/admin/ticket_handler.go` (staff side, + test), the `/tickets` group in `routes/user.go` and `registerTicketRoutes` in `routes/admin.go`, the eight ticket entries in `middleware/console_scope.go` (pinned by `routes/console_scope_coverage_test.go` and `middleware/console_scope_test.go`), the audit action overrides / extra keys in `middleware/audit_log.go` + `service/audit_log.go`, and the `ticket_notify_serverchan_enabled` setting plumbed through the same eight files as `approval_notify_serverchan_enabled`. Frontend: `frontend/src/api/tickets.ts`, `api/admin/tickets.ts`, `stores/tickets.ts`, `utils/tickets.ts`, `components/tickets/TicketThread.vue`, `views/user/TicketsView.vue` (`/tickets`), `views/admin/TicketsView.vue` (`/admin/tickets`, `operatorAllowed`), the tickets nav entries + `.sidebar-badge` styles in `components/layout/AppSidebar.vue`, the polling lifecycle in `App.vue`, the toggle in `components/admin/settings/ForkSettingsSection.vue`, and the `tickets` / `nav.tickets` / `admin.settings.site.ticketNotify` keys in `i18n/locales/{zh,en}/fork.ts` (+ view / store / sidebar specs).
- 图片附件 (ticket image attachments, local): the body stays plain Markdown — images are written as `![image](ticket-attachment://<key>)` with no attachment table. The content endpoint is header-authenticated like every other route, and a browser sends no `Authorization` header for `<img src>`, so a same-origin URL can **never** be hung directly off an `<img>` (it 401s and renders a broken image). `useTicketAttachmentImages` fetches the bytes through `apiClient`, wraps them in `URL.createObjectURL`, and `rewriteTicketAttachmentUrls` swaps the scheme for that `blob:` URL before `MarkdownRenderer` (revoked on unmount; a 1×1 transparent data URI stands in while loading). DOMPurify rejects `blob:` by default, so `renderSafeMarkdown` takes an `allowBlobImages` flag that only the ticket thread passes — do not widen it to the default path. Object keys are built entirely server-side (`<prefix><uid>/<yyyymm>/<rand8hex><ext>`, staff under `<prefix>staff/<uid>/`), the extension comes only from a `image/{jpeg,png,webp,gif}` Content-Type whitelist that `http.DetectContentType` must also agree with (a declared type can lie, and the bytes are served back from our own origin — letting HTML through would be a stored XSS), 5 MiB per image with `RequestBodyLimit` on the route, and the prefix **is** the read authorization boundary: a user may read `<prefix><own uid>/` plus any `<prefix>staff/` key that a staff-authored message in one of their own tickets references (`SupportTicketRepository.StaffAttachmentReferencedForUser`, the only way staff-pasted images reach the ticket owner; anything else is a flat 404, never a leak), staff may read the whole prefix, and `..` / `path.Clean` mismatches are rejected. Files: `backend/internal/service/ticket_attachment_service.go` / `ticket_attachment_storage_settings.go`, `repository/ticket_attachment_s3_store.go` (+ `wire.go` factory), `handler/ticket_attachment_handler.go`, `handler/admin/ticket_attachment_handler.go`, the storage-config trio in `handler/admin/backup_handler.go`, the attachment routes in `routes/user.go` / `admin.go`, the two `auditBodyOmittedRoutes` entries in `middleware/audit_log.go` (+ tests throughout); frontend `components/tickets/useTicketAttachments.ts` (upload) and `useTicketAttachmentImages.ts` (authenticated fetch → object URL → revoke), the `allowBlobImages` option in `utils/markdown.ts` + the matching prop on `components/common/MarkdownRenderer.vue` (pinned by its spec), the `ticketAttachment*` / `rewriteTicketAttachmentUrls` helpers in `utils/tickets.ts`, the `side` prop on `TicketThread.vue`, `uploadAttachment` / `fetchAttachment` in `api/tickets.ts` / `api/admin/tickets.ts`, the storage card in `views/admin/BackupView.vue` + `api/admin/backup.ts`, and the `tickets.attachments.*` / `admin.backup.ticketAttachmentStorage.*` keys.
- Attachment storage config (setting key `ticket_attachment_storage_config`, `/admin/backups/ticket-attachment-storage`) is **admin-only** — unlike the ticket routes themselves, operators are not allowlisted — and its PUT is step-up 2FA gated on both sides (`gin.HandlerFunc(stepUpAuth)` on the route, `backupStepUp.run(...)` in `BackupView.vue`): the prefix is the content endpoint's authorization boundary, so retargeting it changes what staff can read. The settings service additionally refuses a prefix overlapping the database-backup prefix, which would otherwise turn the content endpoint into a backup-download channel. Never drop either guard on a merge.
- Category `appeal` belongs to the appeal session (see 封禁申诉会话): only `CreateAppeal` creates it, the normal create rejects it, and the admin detail shows the requester status with a restore button.
- Design notes in `docs/TICKETS.md`. Keep this feature on upstream merges: resolve conflicts on the paths above in favour of the local version and never let a merge move the operator ticket writes out of `operatorWriteScope` or behind `AdminOnly()`.

### 客户端登录码 (Desktop login code, local implementation)

- For desktop clients that cannot reach the `http://127.0.0.1:<port>/callback` redirect of the `/auth/paseo` bridge page: the signed-in page asks for a one-time code, the user pastes it into the client, and the client exchanges it. Codes are **one-time** (Redis `GETDEL`), live **10 minutes**, and are **bound to an S256 PKCE challenge** whose verifier exists only in the client, so a leaked code is useless. The code (8 chars from `23456789ABCDEFGHJKMNPQRSTUVWXYZ`, shown as `XXXX-XXXX`, normalized by uppercasing and dropping non-alphanumerics) never reaches Redis in clear — only its SHA-256 is used as the key (`desktop_login_code:<hex>`). The record holds the user id, challenge and the page's gateway API keys; **tokens are not stored**, they are minted at exchange time through `AuthService.GenerateDesktopTokenPairWithUser` (own unbound session family, same as `/auth/desktop-session`). Exchange consumes the code *before* checking the verifier, so a wrong verifier burns it; every code-related failure (unknown, expired, reused, verifier mismatch or malformed, bad body) is the single `400 DESKTOP_LOGIN_CODE_INVALID`.
- Routes: `POST /api/v1/auth/desktop-session/code` (JWT, `authenticated` group, rate limiter `desktop-login-code` 10/min fail-close; body `code_challenge` / `code_challenge_method: "S256"` / `api_key` / optional `claude_api_key` / `codex_api_key` → `{code, expires_in}`; `DESKTOP_LOGIN_CHALLENGE_INVALID`, `DESKTOP_LOGIN_API_KEY_INVALID`) and `POST /api/v1/auth/desktop-session/exchange` (public `auth` group, rate limiter `desktop-login-exchange` 20/min fail-close; body `code` / `code_verifier` → `{access_token, refresh_token, expires_in, token_type, api_key, claude_api_key?, codex_api_key?}`; non-privileged users get 403 in backend mode, mirroring `RefreshToken` — `BackendModeAuthGuard` already blocks the route then, and it must **not** be added to that guard's allowlist). Both bodies are in `auditBodyOmittedRoutes`.
- Backend files: `backend/internal/service/desktop_login.go` (+ `desktop_login_test.go`), `GenerateDesktopTokenPairWithUser` in `backend/internal/service/auth_desktop_session.go`, `backend/internal/repository/desktop_login_code_store.go` (+ `_test.go`, miniredis), `backend/internal/handler/desktop_login_handler.go` (+ `_test.go`), the two routes in `backend/internal/server/routes/auth.go` (+ `auth_rate_limit_test.go` entries), the two `auditBodyOmittedRoutes` entries in `backend/internal/server/middleware/audit_log.go` (+ `audit_log_desktop_login_test.go`), and the wiring in `repository/wire.go` (`NewDesktopLoginCodeStore`), `service/wire.go` (`NewDesktopLoginService`), `handler/handler.go` / `handler/wire.go` (`DesktopLogin`), `backend/cmd/server/wire_gen.go`. Consumers: the web bridge page `frontend/src/views/auth/PaseoBridgeView.vue` + `frontend/src/views/auth/paseo-bridge.ts`, and the client `client/crates/sub2api/src/auth.rs`.
- Web page: `/auth/paseo` enters code mode only when the query carries a valid 43-char `code_challenge` and `code_challenge_method=S256` (`readDesktopCodeRequest`); otherwise the legacy token-fragment flow runs unchanged, because clients ≤ 0.2.5 still use it. In code mode it shows the code (copy button, countdown, a "never send this code to anyone" warning — PKCE stops a leaked code, not a user talked into handing theirs over), keeps it in sessionStorage `sub2api_desktop_login:<challenge>`, and after 1.5 s navigates to the loopback `redirect_to#code=…&endpoint=…`. Coming Back from a callback page that failed to load shows the stored code without navigating again or minting another. It never guesses the `agentdesk://` / `paseo://` schemes in code mode. Keys: `auth.desktopBridge.code.*` in `frontend/src/i18n/locales/{zh,en}/fork.ts`; specs `frontend/src/views/auth/__tests__/{paseo-bridge,PaseoBridgeView}.spec.ts`.
- Client (separate git repo under `client/`): `sub2api::auth` builds the PKCE pair, sends the challenge in the login URL, redeems a delivered or pasted code (`exchange_login_code`, `parse_pasted_login` accepts the bare code or the whole address-bar link, including an old bridge's `#access_token=` fragment), and refuses `/deliver` POSTs from any other origin. The sign-in window (`client/src/app/cloud_sign_in.rs`, state in `client/src/app/cloud_account.rs::SignInAttempt`) takes the paste. Sign-out is protected by `Credentials.session_id`: `Credentials::save` and `refresh_if_needed` refuse a copy whose session is no longer the stored one (`SignedOut`), and every async completion in `client/src/app/` checks `cloud_session_is` before touching state, so a request in flight at sign-out can no longer bring the account back. Sign-out also revokes the refresh token via `POST /api/v1/auth/logout`.
- Keep this feature on upstream merges: resolve conflicts on the paths above in favour of the local version.

### 站内信 (Site messages, local implementation)

- Point-to-point messages to exactly one account (user / operator / admin) — modelled on announcements but never broadcast or targeted. Named "site message" on purpose: `UserMessageQueueService` already exists upstream and `/v1/messages` is the gateway path. Three sources: `security` (content-moderation notices), `admin` (written in user management), `system` (e.g. "账户已恢复 / Account restored" from the appeal flow). Unread messages **pop up by default** (`components/common/SiteMessagePopup.vue`, after any announcement popup, never on `/appeal`): on login and whenever the 60 s unread-count poll grows, the store queues up to 10 unread messages oldest-first, remembers popped ids per session (`sub2api_site_msg_popped:<uid>` in sessionStorage) and "知道了" marks read. Header envelope with a numeric badge + list / Markdown detail (`components/common/SiteMessageInbox.vue`, `MarkdownRenderer`).
- Content moderation: the setting `content_moderation_config.site_message_on_hit` (default **on**, toggle "命中时发送站内信" next to "命中时发送邮件" in `RiskControlView.vue`; missing in old JSON → default true because `parseContentModerationConfig` overlays defaults) gates every moderation site message: violation notice on each hit, account-disabled notice when an auto-ban was just applied, cyber notice unless `LogOnly`; log-only modes and the allowlist send nothing. Independent of `email_on_hit` and of SMTP / the user having an email. Bodies are rendered server-side as bilingual Markdown (zh, `---`, en — recipient locale is never recorded, `RememberRecipientLocale` has no callers) with every interpolated value Markdown-escaped and the cyber detail fenced (`content_moderation_site_message.go`). The disabled notice tells the user to sign in and appeal.
- Upstream hooks (each commented `// fork：站内信`): the `siteMessageNotifier` field, `s.deliverFlaggedSiteMessages(...)` after `sendFlaggedNotificationSideEffects` in `persistContentModerationLog`, `s.deliverCyberPolicySiteMessages(...)` at the end of `RecordCyberPolicyEvent`, and the `SiteMessageOnHit` line next to every `EmailOnHit` (config struct, view, update input, default, `UpdateConfig`, view mapping) in `backend/internal/service/content_moderation.go`; the request field + mapping in `handler/admin/content_moderation_handler.go`; `site_message_on_hit` next to `email_on_hit` in `frontend/src/api/admin/riskControl.ts` / `views/admin/RiskControlView.vue`. `ProvideContentModerationService` (`service/wire_local.go`) replaces `NewContentModerationService` in `service/wire.go`.
- Sending from user management: `POST /api/v1/admin/users/:id/site-messages` (idempotent via `executeAdminIdempotentJSON`; JSON `{title ≤200, content ≤5000 runes}`) and `GET …/site-messages` (history with read state). Operators: GET is in `operatorReadScope`, POST in `operatorApprovalScope` (202 → admin approves → replay); on replay the handler reads `ApprovalReplayFromContext` so the row records the operator as sender plus `approval_id` (`source_id=approval:<id>`). Users never see staff identity (`from: system|staff`). Push label `发送站内信` in `approvalActionLabels`, describe case in `utils/approvalDescribe.ts` (falls back to a "body too long" line when the 16 KiB redacted body was truncated).
- Files: `backend/migrations/246_site_messages.sql` (+ `site_messages_migration_test.go`; no FK to soft-deleted users, no dedup index — request ids are client-controllable), ent schema `site_message.go` (parity only; regenerate with `go generate ./ent` — on Windows run it in a clean worktree if the IDE holds mapped files), `service/site_message.go` / `site_message_service.go` / `content_moderation_site_message.go` (+ tests), `repository/site_message_repo.go` (+ sqlmock test), `handler/dto/site_message.go`, `handler/site_message_handler.go` (+ test), `handler/admin/user_site_message_handler.go` (+ test), the `/site-messages` group in `routes/user.go`, `registerUserSiteMessageRoutes` in `routes/admin.go`, the two entries in `middleware/console_scope.go` (pinned by `routes/console_scope_coverage_test.go` / `middleware/console_scope_test.go`), `site_message_id` in `auditExtraAllowedKeys`, wiring in `repository/wire.go`, `service/wire.go`, `handler/handler.go`, `handler/wire.go`, `cmd/server/wire_gen.go`. Frontend: `api/siteMessages.ts`, `sendSiteMessage` / `listSiteMessages` in `api/admin/users.ts`, `stores/siteMessages.ts`, `SiteMessageInbox.vue` (in `AppHeader.vue`), `SiteMessagePopup.vue` + lifecycle in `App.vue`, `components/admin/user/UserSiteMessageModal.vue` + the "More" menu item in `views/admin/UsersView.vue`, `utils/approval.ts`, and the `siteMessages.*` / `admin.users.siteMessage.*` / `admin.riskControl.siteMessageOnHit*` / `operator.approval.*.userSiteMessage` keys in `i18n/locales/{zh,en}/fork.ts` (+ store / popup / modal / describe specs). Keep this feature on upstream merges; never let an operator send bypass the approval capture.

### 网关错误中英双语 (Bilingual gateway errors, local implementation)

- API-key auth, risk-control and billing rejections on gateway routes are returned as one `中文 / English` string (the `cyberSessionBlockedClientMsg` convention); `code` / `type` / status are unchanged. Text lives in the fork-local leaf package `backend/internal/pkg/bilingual` (`gateway_messages.go` table, `Gateway`, `TruncateUTF8`): lookups are case-insensitive and idempotent, `Access denied. Your IP is <ip>` is a prefix rule, a `（hash: …）` suffix is kept, unknown / admin-customized messages pass through.
- **Rule:** the legacy text stays verbatim at the start of its language half — `handler/ops_error_logger.go` and the desktop client's `client/crates/sub2api/src/paywall.rs` classify by those lowercase substrings, and paywall ignores any message containing "upstream" (so English halves never say it). `bilingual_test.go` pins both.
- Upstream hooks (one line each, `// fork：网关双语`): `AbortWithError`, `abortWithOpenAIQuotaError`, `RequireGroupAssignment` in `server/middleware/middleware.go`; `abortWithGoogleError` in `api_key_auth_google.go`; the deferred wrap in `billingErrorDetails` (`handler/gateway_handler.go`) and `securityAuditMessage` (`handler/security_audit_helper.go`); `writeContentModerationWSError` and `closeOpenAIClientWS` (UTF-8-safe 120-byte cut) in `handler/openai_gateway_handler.go`; `handler/gateway_web_search.go`. `middleware/bilingual_gateway_local.go` skips `/api/v1/` and `/api/internal/` (panel routes keep code-based i18n; `/api/v3` is a gateway prefix). The subscription-limit path used to return `validateErr.Error()` (an `error: code=… reason=…` debug string); it now returns the message (`gatewayErrorMessage`).
- The content-moderation default block message stays in Go; stored zh / en defaults are upgraded at output time, and the frontend default `admin.riskControl.defaultBlockMessage` (`i18n/locales/{zh,en}/admin/channels.ts`) is the bilingual string.
- **Merge landmine:** `frontend/src/api/client.ts` must keep `reason` and `metadata` on both reject paths (upstream `faee59ee1`, lost once because it was labelled `fix(payment)`); login's `auth.errors.USER_NOT_ACTIVE`, the appeal token, ticket error maps and many `extractApiErrorCode` callers depend on it (pinned in `api/__tests__/client.spec.ts`).

### 封禁申诉会话 (Appeal session for disabled accounts, local implementation)

- A disabled account that authenticates successfully gets `403 USER_NOT_ACTIVE` with `metadata.appeal_token` / `expires_in` (password login — accounts with TOTP only after Login2FA; passkey — `service/passkey.go` lets `disabled` through via `passkeyLoginAccountUsable`; pending OAuth exchange), or a `#appeal_token=…&expires_in=…` redirect from LinuxDo / GitHub / Google / OIDC / WeChat callbacks. DingTalk discards its lookup error and is not covered. A wrong password still returns `INVALID_CREDENTIALS`, so account state is only revealed after authentication.
- Identity travels in `service.UserDisabledError` (`user_disabled_error.go`, unwraps to `ErrUserNotActive` so every existing check behaves the same), constructed only after credentials are verified: `AuthService.Login`, `LoginOrRegisterOAuth` and the OAuth token pair in `auth_service.go`, `auth_email_oauth_auto.go`, `findActiveUserByID` in `auth_oauth_pending_flow.go`, `ensureLoginUserActive` in `auth_handler.go`. Never use it in `ValidatePasswordCredentials` (status before password). Issue points are one-line hooks (`// fork：申诉`) calling `respondAppealIfDisabled` / `h.redirectAppealIfDisabled` / `handleDisabledPasswordLogin` / `rejectInactivePasskeyUser` from `handler/auth_appeal.go` in `auth_handler.go`, `passkey_handler.go`, `auth_oauth_pending_flow.go`, `auth_linuxdo_oauth.go`, `auth_github_oauth.go`, `auth_email_oauth.go`, `auth_oidc_oauth.go`, `auth_wechat_oauth.go`; `appealService` fields on `AuthHandler` / `PasskeyHandler` injected by `ProvideAuthHandler` / `ProvidePasskeyHandler` (`handler/wire_local.go`, replacing the `New*` providers in `handler/wire.go`). Nil service = previous behaviour.
- Token: opaque `apl_` + 32 random bytes, only its SHA-256 in Redis (`appeal_session:<hex>` + `appeal_session_user:<uid>`, `repository/appeal_session_store.go`), 2 h fixed, one per user, sent only in `X-Appeal-Token` (CORS allowed). Never accepted by `jwtAuth` / `adminAuth`, never exchanged for a JWT. `AppealService.Authenticate` re-reads the user on every request: active → revoke + 409 `APPEAL_ACCOUNT_ACTIVE`, deleted / other status → 401 `APPEAL_TOKEN_INVALID`. Backend mode issues no appeal tokens at all (the session runs as role `user`, which `BackendModeUserGuard` would block; the admin restores accounts directly there).
- Routes `/api/v1/appeal/*` (`routes/appeal.go`, registered in `server/router.go`): default-deny allowlist `middleware/appeal_scope.go` pinned by `routes/appeal_scope_coverage_test.go` (golden list == registered routes, no privileged segments); `middleware/appeal_auth.go` sets the same context keys as jwtAuth with role forced to `user` and `auth_method=appeal_token`, so `TicketHandler`, `TicketAttachmentHandler` and `SiteMessageHandler` are reused as-is. Limits: per-IP 120/min fail-closed, per-user create 1/min + 5/day, writes 30/h. Own handlers: `handler/appeal_handler.go` (`session`, `logout`, `tickets` create).
- Tickets: category `appeal` (`service/ticket.go`, `TicketUserCategories` excludes it), created only via `TicketService.CreateAppeal` (`ticket_appeal.go`: one open appeal per user, outside the 5-ticket quota; the normal create rejects it), push title prefixed 「【申诉】」, staff detail carries `user.status` (`handler/admin/ticket_handler_appeal.go`, `ProvideTicketHandler`). The admin ticket view shows the status and a "恢复账户" button (`PUT /admin/users/:id {status:'active'}`; operators go through approval).
- Restore hook: `UserStatusObserver` (`service/appeal.go`, `admin_user_status_observer.go`) — one field + one `notifyUserStatusChanged` line in upstream `admin_service.go` / `admin_user.go`, and one field + two lines in `ContentModerationService.UnbanUser`; registered by `ProvideAppealService`. Disabled → active revokes the appeal session and sends the "账户已恢复 / Account restored" site message.
- Frontend: `api/appeal.ts` is a separate axios instance (only `X-Appeal-Token`, no `Authorization`, no global 401 handling); `utils/appeal.ts` keeps the token in sessionStorage `sub2api_appeal_session` (never `auth_token`); `views/auth/AppealView.vue` at `/appeal` shows the notices, the appeal form / `TicketThread side="appeal"`, polls the session and returns to `/login` with a "restored" / "expired" notice. `LoginView.vue` (password / 2FA / passkey catches) and the LinuxDo / GitHub / OAuth / OIDC / WeChat callback views route there; ticket composables have an `appeal` branch; audit filter `appeal_token`; keys `appeal.*`, `tickets.category.appeal`, `tickets.admin.{userStatus,restore*}`, `admin.audit.filters.appealToken`. Specs: `utils/__tests__/appeal.spec.ts`, `api/__tests__/appeal.spec.ts`, `views/auth/__tests__/AppealView.spec.ts`, callback / admin ticket specs; backend `handler/auth_appeal_test.go`, `service/appeal_service_test.go`, `repository/appeal_session_store_test.go`, `middleware/appeal_auth_test.go`.
- Keep this feature on upstream merges; never widen the appeal allowlist casually, never accept the token in `Authorization`, and never mint a session from an appeal route.

### 内容自动翻译 (Content auto-translation, local implementation)

- Admin-written text that i18n cannot cover (group names / descriptions, active announcements, `site_subtitle`, `contact_info`, user-visible custom menu labels, custom endpoint name / description, login agreement documents, active channel descriptions, and the pay service's for-sale plans / promotions / channels / `PAY_HELP_TEXT`) is translated into zh / en / ja with **one of the admin's own gateway API keys** and a cheap model, through the gateway's own `{base_url or http://127.0.0.1:<server.port>}/v1/chat/completions`. Not translated: `site_name`, `home_content` (unsanitized HTML), site messages, tickets. Contract and rules: `docs/CONTENT_TRANSLATION.md`.
- **Display only.** API responses and the structs routing reads are never changed; every surface sends the original text to `POST /api/v1/content-translations/lookup` (public, `PublicIP` rate limit, ≤ 200 texts / 256 KiB) and shows the translation when one exists. The desktop client classifies the Chinese-model lane from raw group / plan names ("国模"), so never substitute text server-side. Admin pages (`/admin/*`) always show originals.
- **Cache rules (do not weaken):** the DB table `content_translations` (migration `247_content_translations.sql`, key `sha256(normalized source) + target_lang`) is the only persistent cache; memory is a mirror loaded at start. Lookup is a pure cache read and never calls the model (unknown text only triggers a debounced ≥ 60 s rescan of the sources). A scan (30 s after start, every 10 min, lookup miss, admin "sync now", pay registration) only reads the DB and translates pairs with no row; restarts, model changes, language toggles, text that disappears and comes back, or reverts to an old wording never re-request. No automatic cleanup (`last_seen_at` only marks "not in use" in the admin list). Only new / changed text, admin-deleted rows ("clear machine translations" keeps manual rows) and failed pairs (in-memory backoff 10 min → 24 h; "sync now" ignores it) call the model. `manual = TRUE` rows (admin edits) are never overwritten by `Upsert`.
- The setting `content_translation_config` (JSON: `enabled`, `api_key_id`, `model`, `base_url`, `languages`) stores **only the key id**; the key is read from `api_keys` at call time and must be active and owned by a `RoleAdmin` user. Never copy the key into settings.
- Backend files: `backend/internal/service/content_translation.go` (types, lang / script detection, repo interface), `content_translation_config.go`, `content_translation_sources.go` (read-only collectors), `content_translation_llm.go` (batched chat-completions client: JSON `{"t": [...]}` batches ≤ 40 items / 6 000 chars, mismatch → one by one, > 2 000 chars alone, no temperature), `content_translation_service.go` (+ `_test.go` / `content_translation_llm_test.go`, which count model requests), `ProvideContentTranslationService` in `service/wire_local.go` (starts the loop; `Stop` in `cmd/server/wire.go` `provideCleanup`), `repository/content_translation_repo.go` (+ sqlmock test), `handler/dto/content_translation.go`, `handler/content_translation_handler.go` (public lookup + pay registration), `handler/admin/content_translation_handler.go`, `server/routes/content_translation.go` (+ test; admin routes under `/admin/content-translations` are `AdminOnly`, operators are not allowlisted), the one-line hooks in `server/router.go`, `routes/admin.go`, `routes/pay_integration.go` (`PUT /api/internal/pay/content-translations/sources`, internal token only, replaces namespace `pay`), wiring in `repository/wire.go`, `service/wire.go`, `handler/handler.go`, `handler/wire.go`, `cmd/server/wire_gen.go`, ent parity schemas `content_translation.go` / `content_translation_source.go`, migration test `migrations/content_translations_migration_test.go`.
- Web frontend: `api/contentTranslations.ts`, `api/admin/contentTranslations.ts`, `stores/contentTranslations.ts` (batched lookup, pending re-asks, script-based skip; loads the api module lazily on purpose), `composables/useContentTranslation.ts` (`tx` for components, `translateContent` for plain TS such as `router/title.ts`; reads `$route` / `$i18n` global properties instead of importing `vue-router` / `@/i18n`, which many component specs mock partially; any `/admin` path returns the original), `App.vue` (re-runs the document title when translations arrive), the wrapped display sites (`GroupBadge`, `GroupOptionItem`, `KeysView` tooltip, `SubscriptionsView`, `SubscriptionProgressMini`, `ModelStatusView`, `ModelCatalogView`, `HomePricingSection`, `RedeemView`, `ProfileView`, `UsageTable`, `GroupDistributionChart`, `UserErrorRequestsTable`, `BatchImageGuideView`, `AvailableChannelsTable`, `AnnouncementPopup` / `AnnouncementBell` (markdown translated before rendering), `AuthLayout` subtitle, `AppHeader` / `AppSidebar` contact info and custom menu labels, `EndpointPopover`, `LoginAgreementPrompt` / `LegalDocumentView` — the agreement revision fallback and icon matching keep the original title; search / filter / select labels and v-for keys keep raw values), `components/admin/settings/ContentTranslationSettingsCard.vue` (its config is not part of the SettingsView form: the enable switch saves at once and reverts on failure, the card listens to the enclosing `<form>`'s `submit` so the page's Save button also saves it, and "sync now" warns when the server config is disabled) + `ContentTranslationsDialog.vue` mounted from `ForkSettingsSection.vue` (+ specs under `components/admin/settings/__tests__/`, `stores/__tests__/contentTranslations.spec.ts`, `composables/__tests__/useContentTranslation.spec.ts`), `admin.settings.site.contentTranslation.*` keys in `i18n/locales/{zh,en}/fork.ts`.
- Pay service (`sub2apipay/`): `src/lib/sub2api/content-translations.ts` (collect / register sources, debounced and ≤ 10 min stale sync, lookup), `src/app/api/content-translations/route.ts` (forwards lookups), `src/lib/use-content-translations.ts`, display-only copies in `src/app/pay/page.tsx`, sync calls after admin writes in `src/app/api/admin/{subscription-plans,promotions,channels}/**` and stale sync in `src/app/api/{subscription-plans,user}/route.ts`.
- Desktop client (`client/`, separate repo): `crates/sub2api/src/content_translations.rs` (lookup, script check, disk cache `~/.cheaprouter/content-translations.json`), `src/app/content_translations.rs` (`Waku::tx`, batched background lookup flushed after renders), display-site hooks recorded in `client/docs/FORK.md`.
- Keep this feature on upstream merges.

### 扣费失败重试与 API Key 删建防护 (Billing retry queue + API key churn guard, local implementation)

- **Incident (2026-10-05/06):** a user scripted "create a key with `quota` 0.01 → send a long request → delete the key ~3 s later". The post-billing transaction updated the key's `quota_used` with `WHERE id = $2 AND deleted_at IS NULL`, got `ErrAPIKeyNotFound`, and rolled back the whole transaction (balance charge + dedup claim); the gateway then zeroed `actual_cost` and wrote the usage log. 1 479 requests (list $559, $111.90 at the group multiplier) went unbilled; a DB slowdown on 10-03 did the same to 76 requests of 24 users.
- **Upstream fixes this builds on (in since the v0.2.13 merge `7e8483df2`):** PR #7816 `c2d5bbd93` + `432a6a461` (`applyUsageBillingEffects` ignores `ErrAPIKeyNotFound` from the key quota / rate-limit statements; the statements keep `deleted_at IS NULL`), PR #7752 `9ecb34082` + `017bbcb90` (`api_key_create.max_active_per_user` / `max_per_user_per_hour`, `IncrementCreateCount` in `repository/api_key_cache.go`, `checkAPIKeyCreateLimits`, delete / update no longer reset the create-attempt counter) and PR #7681 (in-flight balance reservation; `syncBalanceCacheAfterDeduction` deducts the balance cache synchronously, while a retried settlement only invalidates it).
- **Fork additions on top:**
  - Retry queue: `backend/migrations/248_usage_billing_retries.sql` (+ `_migration_test.go`; also seeds an enabled `billing_failure_count` alert rule), ent parity `ent/schema/usage_billing_retry.go`, `service/usage_billing_retry.go` (types, `UsageBillingRetryRepository`, `BillingRetryEnqueuer`, `BillingFailureSource`), `service/usage_billing_retry_service.go` (+ `_test.go`: enqueue, 30 s timing-wheel replay via the same `UsageBillingRepository.Apply`, backoff 30 s → 1 h, 24 attempts, `ErrAccountNotFound` retried once without the account quota, permanent errors → `failed`, `Annotate`, Server酱 digest), `repository/usage_billing_retry_repo.go` (+ sqlmock test; `ClaimDue` uses `FOR UPDATE SKIP LOCKED` + a lease), `ProvideUsageBillingRetryService` in `service/wire_local.go` (hooks `SetBillingRetryEnqueuer` on both gateway services and `SetBillingFailureSource` on the alert evaluator), the `billingRetry` field + `enqueueBillingRetry` call in the failure branch of `recordUsageCore` (`gateway_usage_billing.go`) and `OpenAIGatewayService.RecordUsage` (`openai_gateway_usage.go`) — **`usageLog.ActualCost` is no longer zeroed on billing failure**; the log keeps the amount owed and the queue settles it. Read-only `BillingStatus` / `BillingError` / `BillingAttempts` on `service.UsageLog` + `dto.AdminUsageLog` (`billing_status`, filled by `UsageHandler.SetBillingRetryAnnotator` via `admin.ProvideUsageHandler` in `handler/admin/wire_local.go`, replacing `admin.NewUsageHandler` in `handler/wire.go`), `billing_failure_count` / `billing_unsettled_count` metric cases in `ops_alert_evaluator_service.go`, Stop entry in `cmd/server/wire.go`.
  - Setting `billing_failure_notify_serverchan_enabled` (default off; reuses the group-status UID / SendKey, one digest per 10 min per instance) plumbed through the same eight files as `ticket_notify_serverchan_enabled`.
  - Delete rate limit: `api_key_create.max_deletes_per_user_per_hour` (fork field on `APIKeyCreateConfig`), `IncrementDeleteCount` in `repository/api_key_cache.go` (+ miniredis test), `service/api_key_service_limits.go` (`APIKeyDeleteCounter`, `ErrAPIKeyDeleteLimited`, `checkAPIKeyDeleteLimit`, + `_delete_limit_test.go`), the check in `APIKeyService.Delete`, and admin / operator exemption from all three limits (the `IsPrivilegedRole` wrapper around upstream's `checkAPIKeyCreateLimits` in `Create`). Fork defaults are tighter than upstream: 50 active keys, 30 creates/h, 30 deletes/h (legit maximum observed: 17 keys, 10 creates/h, 8 deletes/h); documented in `deploy/config.example.yaml`.
  - Frontend: `billing_status` on `AdminUsageLog` (`types/index.ts`), the badge in `components/admin/usage/UsageTable.vue` (+ spec), the two metrics in `views/admin/ops/components/OpsAlertRulesCard.vue` + `api/admin/ops.ts`, the toggle in `ForkSettingsSection.vue` + `SettingsView.vue` + `api/admin/settings.ts`, keys `usage.billingStatus.*`, `admin.ops.alertRules.metrics.billing*`, `admin.settings.site.billingFailureNotify.*` in `i18n/locales/{zh,en}/fork.ts`.
- Keep on upstream merges: never reintroduce `usageLog.ActualCost = 0` in the failure branches, keep the enqueue hook next to `applyUsageBilling`, and keep the dedup key `(request_id, api_key_id)` shared between `usage_billing_dedup` and `usage_billing_retries` (that is what makes replay idempotent).

## Working Boundary

- During upstream syncs or conflict resolution, prioritize gateway-service changes and leave payment customizations untouched unless explicitly instructed.
- The same boundary applies to the local features listed above (模型广场 / LinuxDo OAuth): do not let an upstream merge silently overwrite or re-introduce their upstream counterparts.

## Client references (客户端参考)

- Before porting DeepSeek Harness plugin UI or logic into the desktop client, read `client/AGENTS.md` § "DeepSeek Harness reference" and `client/docs/deepseek-harness.md` (中文：`client/docs/deepseek-harness.zh.md`). 移植 DeepSeek Harness 插件的界面或逻辑到客户端之前，先读这两份文档。
- `client/CLAUDE.md` is a git symlink to `client/AGENTS.md`; a Windows checkout without `core.symlinks` turns it into a one-line text file, so Windows sessions never load `client/AGENTS.md`. This pointer stands in for it — read `client/AGENTS.md` directly when working in `client/`. Windows 检出里该符号链接只是一行文本，客户端规则需直接读 `client/AGENTS.md`。
