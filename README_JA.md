<div align="center">

# CheapRouter サーバー

**CheapRouter Server — [Sub2API](https://github.com/Wei-Shaw/sub2api) をベースに構築された AI モデルゲートウェイサーバー**

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D.svg)](https://vuejs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

[English](README.md) | [中文](README_CN.md) | 日本語

</div>

## プロジェクト概要

本リポジトリは **CheapRouter** のサーバーサイドです。CheapRouter は次の 2 つの部分で構成されています：

- **サーバー（本リポジトリ）**：Claude、OpenAI/Codex、Gemini、Grok などのモデルを統一 API でプロキシする AI モデルゲートウェイ。ユーザー管理、サブスクリプション課金、チャージ決済、管理コンソールを備えています。
- **クライアント（[client/](client/) サブモジュール）**：Rust + GPUI で書かれたネイティブデスクトップ Agent ワークベンチ。

サーバーはオープンソースプロジェクト [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) をベースに構築されており、上流のゲートウェイ機能を継続的に取り込みながら、本プロジェクト独自のカスタマイズを維持しています：

- **決済サービス `sub2apipay/`**：Next.js + Prisma + Stripe によるチャージ / サブスクリプション購入 / 請求書サービス。内部ブリッジ API でゲートウェイと統合（ローカル管理、上流とは同期しない）
- **紹介・招待リワード**：招待リンク経由の登録、被招待者の初回チャージで双方向リワード（残高 / サブスクリプション）を付与
- **クライアント統合**：クライアントのダウンロード配布、更新履歴、デスクトップコールバックプロトコル（`agentdesk://`）
- グループ稼働ステータス、コミュニティリンク、公開価格カタログなどのサイトカスタマイズ

## ディレクトリ構成

```
backend/       — Go バックエンド（Gin + Ent ORM + PostgreSQL + Redis）
frontend/      — 管理フロントエンド（Vue 3 + Vite + TypeScript + pnpm）
sub2apipay/    — 決済サービス（Next.js + Prisma + Stripe）
client/        — CheapRouter デスクトップクライアントサブモジュール
deploy/        — デプロイ設定（Docker Compose + Caddy）
tools/         — 補助ツールスクリプト
```

## クイックスタート

### Docker デプロイ

```bash
# イメージのビルド（中国国内環境では既定で goproxy.cn / npmmirror ミラーを使用）
./deploy/build_image.sh
```

設定は [deploy/DOCKER.md](deploy/DOCKER.md) と [deploy/config.example.yaml](deploy/config.example.yaml) を参照し、[deploy/docker-compose.standalone.yml](deploy/docker-compose.standalone.yml) で一括起動できます。

### ローカル開発

```bash
# バックエンド
cd backend && make build          # ビルド
cd backend && make test-unit      # ユニットテスト
cd backend && make generate       # Ent スキーマと Wire の生成

# フロントエンド
cd frontend && pnpm dev           # 開発サーバー
cd frontend && pnpm lint          # ESLint
cd frontend && pnpm typecheck     # 型チェック

# 決済サービス
cd sub2apipay && pnpm dev         # 開発サーバー
cd sub2apipay && pnpm test        # テスト
```

## 上流プロジェクトとライセンス

- 上流プロジェクト：[Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
- ライセンス：[LICENSE](LICENSE) を参照（上流と同一）

## 免責事項

- 本プロジェクトで上流 AI サービスをプロキシすることは、Anthropic などのプロバイダーの利用規約に違反する可能性があります。使用前にご自身で確認し、リスクはすべて利用者が負うものとします。
- お住まいの国・地域の法令を遵守できる場合にのみ本プロジェクトを使用してください。
- 本プロジェクトは技術学習・研究目的でのみ提供されています。本プロジェクトの使用に起因するアカウント停止、サービス中断、データ損失などの直接・間接の損害について、作者は一切の責任を負いません。
