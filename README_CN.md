<div align="center">

# CheapRouter 服务端

**CheapRouter Server — 基于 [Sub2API](https://github.com/Wei-Shaw/sub2api) 构建的 AI 模型网关服务端**

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

[English](README.md) | 中文 | [日本語](README_JA.md)

</div>

## 项目简介

本仓库是 **CheapRouter** 的服务端。CheapRouter 由两部分组成：

- **服务端（本仓库）**：AI 模型网关，提供统一的 API 接口代理 Claude、OpenAI/Codex、Gemini、Grok 等模型，附带用户管理、订阅计费、充值支付与管理后台。
- **客户端（[client/](client/) 子模块）**：Rust + GPUI 编写的原生桌面 Agent 工作台。

服务端基于开源项目 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) 构建，持续同步上游的网关能力，并在其上维护本项目自有的定制功能：

- **支付服务 `sub2apipay/`**：Next.js + Prisma + Stripe 的充值 / 订阅购买 / 发票服务，通过内部桥接接口与网关集成（本地维护，不随上游同步）
- **推荐邀请奖励**：邀请链接注册、首充触发双向奖励（余额 / 订阅）
- **客户端集成**：客户端下载分发、更新日志、桌面回调协议（`agentdesk://`）
- 分组运行状态、社群入口、公开定价目录等站点定制

## 目录结构

```
backend/       — Go 后端（Gin + Ent ORM + PostgreSQL + Redis）
frontend/      — 管理前端（Vue 3 + Vite + TypeScript + pnpm）
sub2apipay/    — 支付服务（Next.js + Prisma + Stripe）
client/        — CheapRouter 桌面客户端子模块
deploy/        — 部署配置（Docker Compose + Caddy）
tools/         — 辅助工具脚本
```

## 快速开始

### Docker 部署

```bash
# 构建镜像（国内环境默认走 goproxy.cn / npmmirror 镜像源）
./deploy/build_image.sh
```

参考 [deploy/DOCKER.md](deploy/DOCKER.md) 与 [deploy/config.example.yaml](deploy/config.example.yaml) 完成配置，使用 [deploy/docker-compose.standalone.yml](deploy/docker-compose.standalone.yml) 一键启动。

### 本地开发

```bash
# 后端
cd backend && make build          # 编译
cd backend && make test-unit      # 单元测试
cd backend && make generate       # 生成 Ent schema 与 Wire

# 前端
cd frontend && pnpm dev           # 开发服务器
cd frontend && pnpm lint          # ESLint
cd frontend && pnpm typecheck     # 类型检查

# 支付服务
cd sub2apipay && pnpm dev         # 开发服务器
cd sub2apipay && pnpm test        # 测试
```

## 上游与许可

- 上游项目：[Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- 许可证：见 [LICENSE](LICENSE)，与上游保持一致

## 免责声明

- 使用本项目代理上游 AI 服务可能违反 Anthropic 等提供商的服务条款，请在使用前自行确认，风险自负。
- 请仅在符合所在国家或地区法律法规的前提下使用本项目。
- 本项目仅供技术学习与研究，因使用本项目导致的封号、服务中断、数据丢失等任何直接或间接损失，作者不承担责任。
