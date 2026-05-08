# AGENTS.md

## Project Overview

sub2api 是一个 AI 模型网关服务，统一代理 OpenAI、Claude、Codex 等多种 AI 模型 API，附带用户管理、订阅计费和管理后台。

## Repository Structure

```
backend/       — Go 后端服务（API 网关核心）
frontend/      — Vue 3 管理前端
sub2apipay/    — Next.js 支付服务
client/        — Paseo 客户端子模块
deploy/        — Docker 部署配置
tools/         — 辅助工具
```

## Key Technologies

- **Backend**: Go 1.26, Gin, Ent ORM, PostgreSQL, Redis, Wire DI
- **Frontend**: Vue 3, Vite, TypeScript, Pinia, pnpm
- **Payment**: Next.js, Prisma, Stripe, pnpm
- **Deploy**: Docker Compose, Caddy

## Build & Test

```bash
# Backend
cd backend && make build
cd backend && make test
cd backend && make generate

# Frontend
cd frontend && pnpm dev
cd frontend && pnpm build
cd frontend && pnpm lint
cd frontend && pnpm typecheck

# Payment
cd sub2apipay && pnpm dev
cd sub2apipay && pnpm test
```

## Coding Standards

- Go: 标准项目布局，业务逻辑在 `backend/internal/`，使用 Ent schema 管理数据模型
- Vue: Composition API + `<script setup>` + TypeScript
- Commits: Conventional Commits（feat/fix/chore/refactor/docs）
- 数据库变更通过 Ent schema，运行 `make generate` 生成代码

## Repository Rules

- When resolving conflicts against upstream, never merge upstream payment-related code into this project.
- Only merge upstream changes that are directly related to the gateway service itself.
- Treat payment code as locally maintained customization unless an explicit task says otherwise.
- During upstream merges or cherry-picks, local payment code always wins over upstream payment code.
- If an upstream commit mixes gateway fixes with payment changes, merge only the gateway-service portion and keep the local payment implementation unchanged.
- For upstream sync work, do not directly merge `upstream/main` by default. First compare the diverged commits, then cherry-pick only the specific upstream commits that are safe and relevant to the gateway service.
- Prefer small, self-contained, gateway-related fixes when cherry-picking. Skip commits that are payment-focused, broad refactors, or otherwise high-risk unless the task explicitly requires them.
- If a commit mixes gateway changes with payment changes, do not cherry-pick it wholesale. Either pick only the gateway-related hunks or skip the commit.
- If an upstream fix is already present locally or cherry-picks as an empty change, skip it instead of forcing a duplicate commit.

## Scope Reminder

- Keep merge and sync work limited to the gateway service itself unless the task explicitly expands scope.
