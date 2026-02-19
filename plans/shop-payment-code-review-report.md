# 商店与支付系统代码审查报告

## 概述

本报告对 sub2api 商店和支付相关代码进行全面审查，涵盖安全性、鲁棒性、前端交互等方面。

---

## 1. 代码架构概览

### 1.1 后端架构

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Handlers                         │
├─────────────────────────────────────────────────────────────┤
│  shop_handler.go          │  admin/shop_handler.go          │
│  - ListProducts           │  - ListProducts                 │
│  - GetPaymentChannels     │  - CreateProduct                │
│  - CreateOrder            │  - UpdateProduct                │
│  - QueryOrder             │  - DeleteProduct                │
│  - EpayNotify             │  - GetStockList                 │
│  - CreemNotify            │  - AddStock                     │
│                           │  - DeleteStock                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Service Layer                         │
├─────────────────────────────────────────────────────────────┤
│  shop_service.go                                             │
│  - CreateOrder: 创建订单并生成支付链接                        │
│  - fulfillOrder: 原子性库存扣减+兑换码核销                    │
│  - HandlePaymentNotify: 处理易支付回调                        │
│  - HandleCreemWebhook: 处理Creem支付回调                      │
│  - GetPaymentChannels: 获取可用支付渠道                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Repository Layer                        │
├─────────────────────────────────────────────────────────────┤
│  shop_repo.go                                                │
│  - shopProductRepository: 商品CRUD                           │
│  - shopProductStockRepository: 库存管理（含原子操作）          │
│  - shopOrderRepository: 订单管理                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Payment Clients                         │
├─────────────────────────────────────────────────────────────┤
│  pkg/epay/client.go    │  pkg/creem/client.go               │
│  - Purchase            │  - CreateCheckout                  │
│  - Verify              │  - VerifyWebhook                   │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 前端架构

```
┌─────────────────────────────────────────────────────────────┐
│                        User Views                            │
├─────────────────────────────────────────────────────────────┤
│  ShopView.vue                                                │
│  - 商品列表展示                                               │
│  - 支付方式选择弹窗                                           │
│  - 订单创建并跳转支付                                         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Admin Views                            │
├─────────────────────────────────────────────────────────────┤
│  admin/ShopView.vue          │  SettingsView.vue            │
│  - 商品管理CRUD              │  - 支付渠道配置                │
│  - 库存管理                  │  - 易支付/Creem配置            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        API Layer                             │
├─────────────────────────────────────────────────────────────┤
│  api/shop.ts                 │  api/admin/shop.ts           │
│  - getProducts               │  - listProducts              │
│  - getChannels               │  - createProduct             │
│  - createOrder               │  - updateProduct             │
│  - queryOrder                │  - deleteProduct             │
│                              │  - getStockList              │
│                              │  - addStock/deleteStock      │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 安全性审查

### 2.1 ✅ 已实现的安全措施

| 安全措施 | 实现位置 | 说明 |
|---------|---------|------|
| 支付回调签名验证 | [`epay/client.go:73`](backend/internal/pkg/epay/client.go:73) | 使用 go-epay 库验证签名 |
| Webhook签名验证 | [`creem/client.go:101`](backend/internal/pkg/creem/client.go:101) | HMAC-SHA256 验证 |
| 管理接口认证 | [`routes/admin.go`](backend/internal/server/routes/admin.go) | AdminAuthMiddleware 保护 |
| 用户接口认证 | [`shop_handler.go:56`](backend/internal/handler/shop_handler.go:56) | 从 context 获取 user_id |
| SQL注入防护 | [`shop_repo.go`](backend/internal/repository/shop_repo.go) | 使用参数化查询 $1, $2... |
| 库存竞态条件 | [`shop_repo.go:156-168`](backend/internal/repository/shop_repo.go:156) | FOR UPDATE SKIP LOCKED |
| 事务原子性 | [`shop_service.go:308-343`](backend/internal/service/shop_service.go:308) | fulfillOrder 使用数据库事务 |
| 订单幂等处理 | [`shop_service.go:309-311`](backend/internal/service/shop_service.go:309) | 已支付订单直接返回 |

### 2.2 ⚠️ 安全风险与建议

#### 风险1: 支付回调无重放保护 (中等)

**位置**: [`shop_handler.go:92-115`](backend/internal/handler/shop_handler.go:92)

**问题**: 易支付回调没有记录已处理的订单号，理论上可能被重放攻击。

**当前状态**: 由于 `fulfillOrder` 有幂等检查（第309行），实际影响有限。

**建议**: 添加回调日志记录
```go
// 建议添加 shop_payment_callbacks 表记录已处理的回调
CREATE TABLE shop_payment_callbacks (
  id BIGSERIAL PRIMARY KEY,
  order_no VARCHAR(64) NOT NULL UNIQUE,
  callback_params JSONB,
  processed_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### 风险2: 订单号可预测性 (低)

**位置**: [`epay/client.go:88-90`](backend/internal/pkg/epay/client.go:88)

```go
func GenerateOrderNo(userID int64) string {
    return fmt.Sprintf("SHOP%d%d", userID, time.Now().UnixNano())
}
```

**问题**: 订单号包含用户ID和时间戳，理论上可预测。

**建议**: 添加随机因子
```go
func GenerateOrderNo(userID int64) string {
    randBytes := make([]byte, 4)
    rand.Read(randBytes)
    return fmt.Sprintf("SHOP%d%d%x", userID, time.Now().UnixNano(), randBytes)
}
```

#### 风险3: 支付渠道接口公开 (低)

**位置**: [`routes/common.go:39-40`](backend/internal/server/routes/common.go:39)

```go
// 支付渠道列表（公开，无需登录）
r.GET("/api/v1/shop/channels", h.Shop.GetPaymentChannels)
```

**问题**: 支付渠道配置对所有人可见。

**评估**: 这是设计决策，便于用户在登录前看到支付方式。风险较低。

#### 风险4: 缺少请求速率限制

**位置**: 所有商店API端点

**问题**: 创建订单等接口没有速率限制，可能被滥用。

**建议**: 添加基于用户/IP的速率限制中间件

---

## 3. 鲁棒性审查

### 3.1 ✅ 良好的鲁棒性设计

| 设计 | 位置 | 说明 |
|-----|------|------|
| 错误类型定义 | [`shop_service.go:17-24`](backend/internal/service/shop_service.go:17) | 明确的业务错误类型 |
| 库存原子扣减 | [`shop_repo.go:156-168`](backend/internal/repository/shop_repo.go:156) | 使用 SKIP LOCKED 防止超卖 |
| 事务回滚 | [`shop_service.go:316`](backend/internal/service/shop_service.go:316) | defer tx.Rollback() |
| Context传递 | 全局 | 支持请求取消和超时 |
| 配置校验 | [`epay/client.go:27-28`](backend/internal/pkg/epay/client.go:27) | 必要配置检查 |

### 3.2 ⚠️ 鲁棒性问题

#### 问题1: 订单过期未处理 (高)

**位置**: [`shop_service.go:240`](backend/internal/service/shop_service.go:240)

```go
expiresAt := time.Now().Add(30 * time.Minute)
```

**问题**: 订单设置了30分钟过期时间，但没有定时任务清理过期订单。

**影响**: 
- 过期订单会一直占用库存
- 用户无法再次购买同一商品（如果库存被锁定）

**建议**: 添加定时任务
```go
// 建议添加定时任务清理过期订单
func (s *ShopService) CleanupExpiredOrders(ctx context.Context) error {
    // UPDATE shop_orders SET status='expired' 
    // WHERE status='pending' AND expires_at < NOW()
    // 释放对应的库存
}
```

#### 问题2: 库存数量不一致风险 (中等)

**位置**: [`shop_repo.go:98-101`](backend/internal/repository/shop_repo.go:98) 和 [`shop_service.go:202`](backend/internal/service/shop_service.go:202)

**问题**: `shop_products.stock_count` 和 `shop_product_stocks` 表的实际可用库存可能不一致。

**场景**: 
1. 管理员添加10个库存 → stock_count=10, stocks表10条记录
2. 直接删除stocks表记录 → stock_count仍为10，但实际只有9个

**建议**: 
- 使用数据库触发器或计算列维护一致性
- 或在查询时动态计算库存

#### 问题3: 并发下单的库存检查 (中等)

**位置**: [`shop_service.go:231-233`](backend/internal/service/shop_service.go:231)

```go
if product.StockCount <= 0 {
    return nil, "", ErrProductOutOfStock
}
```

**问题**: 先检查 `StockCount`，后扣减库存，两者不在同一事务中。

**当前状态**: `TakeOne` 使用 `FOR UPDATE SKIP LOCKED`，实际不会超卖。

**建议**: 移除这个预检查，或改为在事务内检查。

#### 问题4: 支付失败无回调 (低)

**位置**: 支付回调处理

**问题**: 只处理支付成功的情况，支付失败没有记录。

**建议**: 添加支付失败日志记录，便于排查问题。

---

## 4. 前端交互审查

### 4.1 ✅ 良好的前端实践

| 实践 | 位置 | 说明 |
|-----|------|------|
| 加载状态 | [`ShopView.vue:38-44`](frontend/src/views/user/ShopView.vue:38) | 按钮禁用+加载提示 |
| 错误处理 | [`ShopView.vue:126-127`](frontend/src/views/user/ShopView.vue:126) | Toast 错误提示 |
| 支付渠道降级 | [`ShopView.vue:106-111`](frontend/src/views/user/ShopView.vue:106) | API失败时使用默认渠道 |
| 表单验证 | [`admin/ShopView.vue:66`](frontend/src/views/admin/ShopView.vue:66) | HTML5 表单验证 |

### 4.2 ⚠️ 前端问题

#### 问题1: 支付跳转后状态丢失 (中等)

**位置**: [`ShopView.vue:125`](frontend/src/views/user/ShopView.vue:125)

```typescript
window.location.href = result.pay_url
```

**问题**: 直接跳转到支付页面，用户返回时无法知道订单状态。

**建议**: 
1. 跳转前保存订单信息到 localStorage
2. 返回时检查订单状态并显示结果
3. 或使用弹窗打开支付页面

#### 问题2: 缺少订单确认流程 (低)

**位置**: [`ShopView.vue:118-131`](frontend/src/views/user/ShopView.vue:118)

**问题**: 选择支付方式后直接创建订单，没有确认步骤。

**建议**: 添加订单确认弹窗，显示商品信息、价格、支付方式。

#### 问题3: 库存数量显示可能不准 (低)

**位置**: [`ShopView.vue:32-36`](frontend/src/views/user/ShopView.vue:32)

**问题**: 显示的库存数量来自 `stock_count` 字段，可能与实际可用库存不一致。

**建议**: 后端返回实时计算的可用库存。

---

## 5. 代码质量审查

### 5.1 ✅ 良好的代码实践

- **清晰的分层架构**: Handler → Service → Repository
- **接口定义**: Repository 使用接口，便于测试和替换
- **DTO转换**: 使用专门的 DTO 类型进行数据转换
- **国际化支持**: 前端使用 i18n
- **错误处理**: 使用自定义错误类型

### 5.2 ⚠️ 代码改进建议

#### 建议1: 添加订单列表接口

**当前状态**: 用户无法查看自己的订单历史。

**建议**: 添加 `/api/v1/shop/orders` 接口返回用户订单列表。

#### 建议2: 添加订单取消功能

**当前状态**: 用户无法取消待支付订单。

**建议**: 添加取消订单接口，释放库存。

#### 建议3: 添加支付状态轮询

**当前状态**: 用户支付后需要手动刷新页面。

**建议**: 前端添加订单状态轮询或WebSocket通知。

---

## 6. 数据库设计审查

### 6.1 表结构

```sql
-- 商品表
shop_products (
  id, name, description, price, currency,
  redeem_type, redeem_value, group_id, validity_days,
  stock_count, is_active, sort_order, creem_product_id,
  created_at, updated_at
)

-- 库存表
shop_product_stocks (
  id, product_id, redeem_code_id,
  status, order_id, created_at
)

-- 订单表
shop_orders (
  id, order_no, user_id, product_id, product_name,
  amount, currency, payment_method, status,
  redeem_code_id, paid_at, expires_at,
  created_at, updated_at
)
```

### 6.2 ⚠️ 数据库设计问题

#### 问题1: 缺少支付记录表

**建议**: 添加 `shop_payments` 表记录支付详情

```sql
CREATE TABLE shop_payments (
  id BIGSERIAL PRIMARY KEY,
  order_id BIGINT NOT NULL REFERENCES shop_orders(id),
  payment_method VARCHAR(20) NOT NULL,
  transaction_id VARCHAR(100),  -- 第三方交易号
  amount DECIMAL(10,2) NOT NULL,
  status VARCHAR(20) NOT NULL,
  callback_params JSONB,        -- 原始回调参数
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### 问题2: 缺少软删除

**建议**: 商品和订单表添加 `deleted_at` 字段支持软删除。

---

## 7. 综合评估

### 7.1 评分

| 维度 | 评分 | 说明 |
|-----|------|------|
| 安全性 | ⭐⭐⭐⭐ (4/5) | 核心安全措施完善，有少量改进空间 |
| 鲁棒性 | ⭐⭐⭐ (3/5) | 缺少过期订单处理等关键功能 |
| 代码质量 | ⭐⭐⭐⭐ (4/5) | 架构清晰，可维护性好 |
| 前端交互 | ⭐⭐⭐ (3/5) | 基本功能完整，用户体验可提升 |
| 数据库设计 | ⭐⭐⭐ (3/5) | 核心表完整，缺少辅助表 |

### 7.2 优先级建议

| 优先级 | 问题 | 建议 |
|-------|------|------|
| 🔴 高 | 订单过期未处理 | 添加定时清理任务 |
| 🟡 中 | 库存数量不一致 | 使用触发器或计算列 |
| 🟡 中 | 支付跳转状态丢失 | 使用 localStorage 保存状态 |
| 🟡 中 | 并发库存检查 | 移到事务内或移除预检查 |
| 🟢 低 | 订单号可预测 | 添加随机因子 |
| 🟢 低 | 缺少订单列表 | 添加用户订单历史接口 |

---

## 8. 总结

sub2api 的商店和支付系统整体设计合理，核心安全措施（签名验证、事务处理、并发控制）已正确实现。USDT 支付已通过 `epay_channels` 配置正确整合。

**主要优点**:
1. 清晰的分层架构
2. 正确的支付签名验证
3. 使用 `FOR UPDATE SKIP LOCKED` 防止超卖
4. 订单处理幂等性

**需要改进**:
1. 添加过期订单定时清理
2. 完善库存一致性保障
3. 增强用户体验（订单历史、状态轮询）
4. 添加支付记录表

---

*报告生成时间: 2026-02-19*
