<div align="center">

# CheapRouter Server

**The server side of CheapRouter — an AI model gateway built on [Sub2API](https://github.com/Wei-Shaw/sub2api)**

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

English | [中文](README_CN.md) | [日本語](README_JA.md)

</div>

## Overview

This repository is the **server side of CheapRouter**. CheapRouter consists of two parts:

- **Server (this repository)**: an AI model gateway that exposes a unified API for Claude, OpenAI/Codex, Gemini, Grok and other models, with user management, subscription billing, payments, and an admin console.
- **Client ([client/](client/) submodule)**: a native desktop Agent workbench written in Rust + GPUI.

The server is built on the open-source project [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api). It continuously syncs gateway capabilities from upstream while maintaining this project's own customizations on top:

- **Payment service `sub2apipay/`**: a Next.js + Prisma + Stripe service for top-ups, subscription purchases and invoicing, integrated with the gateway through an internal bridge API (maintained locally, never merged from upstream)
- **Referral rewards**: invite-link registration with two-way rewards (balance / subscription) triggered by the invitee's first top-up
- **Client integration**: client download distribution, changelog, desktop callback protocol (`agentdesk://`)
- Site customizations such as group runtime status, community links, and a public pricing catalog

## Repository Layout

```
backend/       — Go backend (Gin + Ent ORM + PostgreSQL + Redis)
frontend/      — Admin frontend (Vue 3 + Vite + TypeScript + pnpm)
sub2apipay/    — Payment service (Next.js + Prisma + Stripe)
client/        — CheapRouter desktop client submodule
deploy/        — Deployment configs (Docker Compose + Caddy)
tools/         — Helper scripts
```

## Quick Start

### Docker Deployment

```bash
# Build the image (defaults to goproxy.cn / npmmirror mirrors for builds in China)
./deploy/build_image.sh
```

See [deploy/DOCKER.md](deploy/DOCKER.md) and [deploy/config.example.yaml](deploy/config.example.yaml) for configuration, then start everything with [deploy/docker-compose.standalone.yml](deploy/docker-compose.standalone.yml).

### Local Development

```bash
# Backend
cd backend && make build          # Build
cd backend && make test-unit      # Unit tests
cd backend && make generate       # Generate Ent schema & Wire

# Frontend
cd frontend && pnpm dev           # Dev server
cd frontend && pnpm lint          # ESLint
cd frontend && pnpm typecheck     # Type check

# Payment service
cd sub2apipay && pnpm dev         # Dev server
cd sub2apipay && pnpm test        # Tests
```

## Upstream & License

- Upstream project: [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- License: see [LICENSE](LICENSE), consistent with upstream

## Disclaimer

- Proxying upstream AI services with this project may violate the terms of service of Anthropic and other providers. Verify before use; all risks are borne by the user.
- Use this project only in compliance with the laws and regulations of your country or region.
- This project is provided for technical learning and research purposes only. The authors assume no liability for account bans, service interruptions, data loss, or any other direct or indirect damages resulting from its use.
