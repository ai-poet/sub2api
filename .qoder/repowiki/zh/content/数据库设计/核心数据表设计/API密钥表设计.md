# API密钥表设计

<cite>
**本文档引用的文件**
- [api_key.go](file://backend/ent/schema/api_key.go)
- [apikey.go](file://backend/ent/apikey.go)
- [apikey_create.go](file://backend/ent/apikey_create.go)
- [apikey_update.go](file://backend/ent/apikey_update.go)
- [apikey_delete.go](file://backend/ent/apikey_delete.go)
- [api_key_handler.go](file://backend/internal/handler/api_key_handler.go)
- [api_key_service.go](file://backend/internal/service/api_key_service.go)
- [api_key_auth.go](file://backend/internal/server/middleware/api_key_auth.go)
- [api_key_repo.go](file://backend/internal/repository/api_key_repo.go)
</cite>

## 目录
1. [项目概述](#项目概述)
2. [表结构设计](#表结构设计)
3. [安全设计](#安全设计)
4. [生命周期管理](#生命周期管理)
5. [权限体系](#权限体系)
6. [密钥生成算法](#密钥生成算法)
7. [存储与加密](#存储与加密)
8. [访问控制机制](#访问控制机制)
9. [密钥轮换](#密钥轮换)
10. [审计日志](#审计日志)
11. [密钥分发与使用统计](#密钥分发与使用统计)
12. [配额控制](#配额控制)
13. [与用户表的关系](#与用户表的关系)
14. [与用量日志表的关系](#与用量日志表的关系)
15. [架构图](#架构图)
16. [总结](#总结)

## 项目概述

API密钥表设计是sub2api系统中的核心组件之一，负责管理用户API密钥的完整生命周期。该设计实现了企业级的安全密钥管理功能，包括密钥生成、存储、访问控制、配额管理和审计跟踪等关键特性。

## 表结构设计

### 核心字段定义

基于Ent框架的schema定义，API密钥表包含以下核心字段：

```mermaid
erDiagram
API_KEYS {
bigint id PK
bigint user_id FK
string key UK
string name
bigint group_id
string status
timestamp last_used_at
json ip_whitelist
json ip_blacklist
decimal quota
decimal quota_used
timestamp expires_at
decimal rate_limit_5h
decimal rate_limit_1d
decimal rate_limit_7d
decimal usage_5h
decimal usage_1d
decimal usage_7d
timestamp window_5h_start
timestamp window_1d_start
timestamp window_7d_start
timestamp created_at
timestamp updated_at
timestamp deleted_at
}
USERS {
bigint id PK
string email UK
string username UK
string role
decimal balance
integer concurrency
string status
}
GROUPS {
bigint id PK
string name
string platform
decimal rate_multiplier
boolean subscription_type
string status
}
USAGE_LOGS {
bigint id PK
bigint api_key_id FK
string endpoint
decimal cost_usd
timestamp created_at
}
API_KEYS ||--|| USERS : "belongs_to"
API_KEYS ||--o{ GROUPS : "belongs_to"
API_KEYS ||--o{ USAGE_LOGS : "has_many"
```

**图表来源**
- [api_key.go:34-119](file://backend/ent/schema/api_key.go#L34-L119)
- [apikey.go:18-73](file://backend/ent/apikey.go#L18-L73)

### 字段详细说明

| 字段名 | 类型 | 约束 | 描述 |
|--------|------|------|------|
| id | bigint | 主键 | 自增ID |
| user_id | bigint | 外键 | 关联用户表 |
| key | string | 唯一键 | API密钥值，最大128字符 |
| name | string | 必填 | 密钥名称，最大100字符 |
| group_id | bigint | 可空 | 所属分组ID |
| status | string | 默认"active" | 密钥状态 |
| last_used_at | timestamp | 可空 | 最后使用时间 |
| ip_whitelist | json | 默认[] | IP白名单数组 |
| ip_blacklist | json | 默认[] | IP黑名单数组 |
| quota | decimal | 默认0 | 配额上限(美元) |
| quota_used | decimal | 默认0 | 已用配额(美元) |
| expires_at | timestamp | 可空 | 过期时间 |
| rate_limit_5h | decimal | 默认0 | 5小时限流(美元) |
| rate_limit_1d | decimal | 默认0 | 日限流(美元) |
| rate_limit_7d | decimal | 默认0 | 7天限流(美元) |
| usage_5h | decimal | 默认0 | 当前5小时用量 |
| usage_1d | decimal | 默认0 | 当前日用量 |
| usage_7d | decimal | 默认0 | 当前7天用量 |
| window_5h_start | timestamp | 可空 | 5小时窗口开始时间 |
| window_1d_start | timestamp | 可空 | 日窗口开始时间 |
| window_7d_start | timestamp | 可空 | 7天窗口开始时间 |

**章节来源**
- [api_key.go:34-119](file://backend/ent/schema/api_key.go#L34-L119)
- [apikey.go:18-73](file://backend/ent/apikey.go#L18-L73)

## 安全设计

### 数据完整性约束

系统采用软删除机制，通过deleted_at字段实现数据恢复能力：

```mermaid
flowchart TD
CREATE["创建API密钥"] --> VALIDATE["验证输入参数"]
VALIDATE --> GENERATE["生成密钥"]
GENERATE --> STORE["存储到数据库"]
STORE --> ACTIVE["状态设为active"]
UPDATE["更新API密钥"] --> CHECK_STATUS{"检查状态"}
CHECK_STATUS --> |正常| MODIFY["修改字段"]
CHECK_STATUS --> |软删除| RESTORE["恢复密钥"]
MODIFY --> SAVE["保存更改"]
RESTORE --> SAVE
DELETE["删除API密钥"] --> SOFT_DELETE["软删除"]
SOFT_DELETE --> TOMBSTONE["生成墓碑键"]
```

**图表来源**
- [api_key.go:27-32](file://backend/ent/schema/api_key.go#L27-L32)
- [api_key_repo.go:256-284](file://backend/internal/repository/api_key_repo.go#L256-L284)

### 错误处理机制

系统定义了完整的错误处理策略：

- **密钥不存在**: `ErrAPIKeyNotFound` - API密钥不存在
- **权限不足**: `ErrInsufficientPerms` - 用户无权操作该密钥
- **密钥冲突**: `ErrAPIKeyExists` - 密钥已存在
- **格式错误**: `ErrAPIKeyInvalidChars` - 密钥格式无效
- **配额超限**: `ErrAPIKeyQuotaExhausted` - 配额已用完
- **过期**: `ErrAPIKeyExpired` - 密钥已过期

**章节来源**
- [api_key_service.go:22-39](file://backend/internal/service/api_key_service.go#L22-L39)

## 生命周期管理

### 密钥状态流转

```mermaid
stateDiagram-v2
[*] --> Active : 创建时默认
Active --> Inactive : 手动禁用
Active --> Expired : 到期自动
Active --> QuotaExhausted : 配额用尽
Inactive --> Active : 解封
Expired --> Active : 续期
QuotaExhausted --> Active : 充值配额
Active --> [*] : 删除
note right of Active
正常使用状态
可进行API调用
end note
note right of Inactive
管理员手动禁用
拒绝所有请求
end note
note right of Expired
自动检测过期
拒绝所有请求
end note
note right of QuotaExhausted
配额用尽
仅限查询用量
end note
```

**图表来源**
- [api_key_service.go:666-690](file://backend/internal/service/api_key_service.go#L666-L690)
- [api_key_auth.go:154-173](file://backend/internal/server/middleware/api_key_auth.go#L154-L173)

### 状态检查流程

```mermaid
sequenceDiagram
participant Client as 客户端
participant Middleware as 认证中间件
participant Service as API密钥服务
participant Repo as 数据仓库
participant DB as 数据库
Client->>Middleware : 发送API请求
Middleware->>Service : GetByKey(key)
Service->>Repo : 查询密钥
Repo->>DB : SELECT FROM api_keys
DB-->>Repo : 返回密钥信息
Repo-->>Service : 密钥实体
Service-->>Middleware : 验证结果
alt 密钥状态正常
Middleware->>Middleware : 检查IP限制
Middleware->>Middleware : 检查用户状态
Middleware->>Middleware : 计费执行
Middleware-->>Client : 允许访问
else 密钥状态异常
Middleware-->>Client : 拒绝访问
end
```

**图表来源**
- [api_key_auth.go:69-110](file://backend/internal/server/middleware/api_key_auth.go#L69-L110)
- [api_key_service.go:666-690](file://backend/internal/service/api_key_service.go#L666-L690)

**章节来源**
- [api_key_auth.go:21-221](file://backend/internal/server/middleware/api_key_auth.go#L21-L221)
- [api_key_service.go:328-428](file://backend/internal/service/api_key_service.go#L328-L428)

## 权限体系

### IP地址白名单/黑名单

系统支持灵活的IP访问控制：

```mermaid
flowchart TD
REQUEST["API请求"] --> GET_IP["获取客户端IP"]
GET_IP --> CHECK_WHITELIST{"检查白名单"}
CHECK_WHITELIST --> |匹配| ALLOW["允许访问"]
CHECK_WHITELIST --> |不匹配| CHECK_BLACKLIST{"检查黑名单"}
CHECK_BLACKLIST --> |匹配| DENY["拒绝访问"]
CHECK_BLACKLIST --> |不匹配| ALLOW
subgraph "IP规则编译"
WHITELIST["IP白名单规则"] --> COMPILE_WL["编译为规则集"]
BLACKLIST["IP黑名单规则"] --> COMPILE_BL["编译为规则集"]
end
GET_IP --> COMPILE_WL
GET_IP --> COMPILE_BL
```

**图表来源**
- [api_key_service.go:240-246](file://backend/internal/service/api_key_service.go#L240-L246)
- [api_key_auth.go:89-98](file://backend/internal/server/middleware/api_key_auth.go#L89-L98)

### 分组权限控制

系统支持基于分组的权限管理：

| 分组类型 | 权限规则 | 订阅要求 |
|----------|----------|----------|
| 标准分组 | 用户可绑定的公开分组 | 无 |
| 订阅分组 | 需要有效订阅才能绑定 | 需要有效订阅 |

**章节来源**
- [api_key_service.go:318-326](file://backend/internal/service/api_key_service.go#L318-L326)
- [api_key_handler.go:328-326](file://backend/internal/handler/api_key_handler.go#L328-L326)

## 密钥生成算法

### 随机密钥生成

系统采用安全的随机密钥生成机制：

```mermaid
flowchart TD
START["开始生成密钥"] --> GENERATE_BYTES["生成32字节随机数据"]
GENERATE_BYTES --> READ_RAND["从熵源读取随机数"]
READ_RAND --> CHECK_SUCCESS{"读取成功?"}
CHECK_SUCCESS --> |否| ERROR["抛出错误"]
CHECK_SUCCESS --> |是| ADD_PREFIX["添加前缀"]
ADD_PREFIX --> HEX_ENCODE["十六进制编码"]
HEX_ENCODE --> COMPOSE_KEY["组合最终密钥"]
COMPOSE_KEY --> RETURN["返回密钥"]
subgraph "前缀配置"
DEFAULT_PREFIX["默认前缀: sk-"]
CONFIG_PREFIX["配置前缀: 从配置读取"]
end
ADD_PREFIX --> DEFAULT_PREFIX
ADD_PREFIX --> CONFIG_PREFIX
```

**图表来源**
- [api_key_service.go:248-264](file://backend/internal/service/api_key_service.go#L248-L264)

### 自定义密钥验证

对于用户提供的自定义密钥，系统执行严格验证：

- **长度检查**: 至少16个字符
- **字符验证**: 仅允许字母、数字、下划线、连字符
- **冲突检测**: 检查数据库中是否存在重复密钥
- **速率限制**: 防止恶意尝试生成重复密钥

**章节来源**
- [api_key_service.go:267-285](file://backend/internal/service/api_key_service.go#L267-L285)
- [api_key_service.go:366-396](file://backend/internal/service/api_key_service.go#L366-L396)

## 存储与加密

### 数据库存储

API密钥以明文形式存储在数据库中，但通过以下机制保证安全性：

1. **唯一性约束**: key字段具有唯一索引
2. **软删除**: 使用deleted_at字段实现可恢复删除
3. **索引优化**: 为常用查询字段建立索引

### 缓存策略

系统采用多层缓存机制：

```mermaid
graph TB
subgraph "缓存层次"
L1_CACHE["本地缓存(Ristretto)"]
L2_CACHE["分布式缓存(Redis)"]
DB_CACHE["数据库"]
end
subgraph "缓存内容"
AUTH_CACHE["认证缓存"]
RATE_CACHE["限流缓存"]
ATTEMPT_CACHE["尝试次数缓存"]
end
L1_CACHE --> L2_CACHE
L2_CACHE --> DB_CACHE
AUTH_CACHE --> L1_CACHE
RATE_CACHE --> L2_CACHE
ATTEMPT_CACHE --> L2_CACHE
```

**图表来源**
- [api_key_service.go:125-148](file://backend/internal/service/api_key_service.go#L125-L148)

**章节来源**
- [api_key_repo.go:450-489](file://backend/internal/repository/api_key_repo.go#L450-L489)

## 访问控制机制

### 认证中间件

API密钥认证中间件执行以下检查：

1. **密钥提取**: 支持多种头部格式
2. **基础验证**: 检查密钥存在性和用户状态
3. **IP限制**: 应用白名单/黑名单规则
4. **订阅验证**: 对订阅类型分组执行额外检查

### 多头部支持

系统支持三种密钥传递方式：

| 头部名称 | 格式 | 用途 |
|----------|------|------|
| Authorization | Bearer {key} | 标准OAuth2方式 |
| x-api-key | {key} | 简单API密钥方式 |
| x-goog-api-key | {key} | Gemini CLI兼容方式 |

**章节来源**
- [api_key_auth.go:28-66](file://backend/internal/server/middleware/api_key_auth.go#L28-L66)
- [api_key_auth.go:89-110](file://backend/internal/server/middleware/api_key_auth.go#L89-L110)

## 密钥轮换

### 自动轮换机制

系统支持密钥的自动轮换和管理：

```mermaid
sequenceDiagram
participant Admin as 管理员
participant Handler as 处理器
participant Service as 服务层
participant Repo as 仓库层
participant Audit as 审计日志
Admin->>Handler : 请求轮换密钥
Handler->>Service : CreateAPIKeyRequest
Service->>Service : 验证现有密钥
Service->>Service : 生成新密钥
Service->>Repo : 创建新密钥记录
Repo->>Repo : 存储新密钥
Service->>Audit : 记录轮换事件
Audit-->>Service : 确认审计
Service-->>Handler : 返回新密钥
Handler-->>Admin : 返回新密钥
```

**图表来源**
- [api_key_handler.go:136-179](file://backend/internal/handler/api_key_handler.go#L136-L179)
- [api_key_service.go:328-428](file://backend/internal/service/api_key_service.go#L328-L428)

### 轮换最佳实践

- **渐进式切换**: 建议先创建新密钥，再切换使用
- **监控使用**: 跟踪旧密钥的使用情况
- **及时清理**: 删除不再使用的旧密钥
- **通知用户**: 重要变更时通知相关用户

## 审计日志

### 审计事件类型

系统记录以下关键审计事件：

| 事件类型 | 触发条件 | 记录内容 |
|----------|----------|----------|
| 密钥创建 | 新密钥生成 | 创建者、创建时间、密钥标识 |
| 密钥更新 | 属性变更 | 修改者、修改时间、变更详情 |
| 密钥删除 | 密钥销毁 | 删除者、删除时间、密钥标识 |
| 访问尝试 | API请求 | 客户端IP、时间戳、结果 |
| 配额变更 | 用量更新 | 用量、时间、成本 |

### 审计数据存储

审计信息存储在独立的日志表中，支持：

- **实时查询**: 快速检索最近的审计事件
- **历史归档**: 长期保存审计记录
- **合规要求**: 满足监管审计需求

**章节来源**
- [api_key_service.go:692-722](file://backend/internal/service/api_key_service.go#L692-L722)

## 密钥分发与使用统计

### 分发机制

系统提供多种密钥分发方式：

1. **在线创建**: 通过Web界面即时生成
2. **批量导入**: 支持批量创建多个密钥
3. **API分发**: 通过REST API程序化创建
4. **导出功能**: 支持安全导出密钥列表

### 使用统计

系统提供全面的使用统计功能：

```mermaid
flowchart TD
REQUEST["API请求"] --> SERVICE["服务层处理"]
SERVICE --> QUOTA_CHECK["配额检查"]
QUOTA_CHECK --> RATE_CHECK["限流检查"]
RATE_CHECK --> UPDATE_USAGE["更新用量"]
UPDATE_USAGE --> LOG_USAGE["记录用量日志"]
LOG_USAGE --> RESPONSE["返回响应"]
subgraph "统计指标"
DAILY_STATS["日统计"]
WEEKLY_STATS["周统计"]
MONTHLY_STATS["月统计"]
TOTAL_STATS["累计统计"]
end
UPDATE_USAGE --> DAILY_STATS
UPDATE_USAGE --> WEEKLY_STATS
UPDATE_USAGE --> MONTHLY_STATS
UPDATE_USAGE --> TOTAL_STATS
```

**图表来源**
- [api_key_service.go:724-736](file://backend/internal/service/api_key_service.go#L724-L736)

**章节来源**
- [api_key_handler.go:65-104](file://backend/internal/handler/api_key_handler.go#L65-L104)

## 配额控制

### 多维度配额管理

系统实现三层配额控制：

```mermaid
graph TB
subgraph "配额层次"
GLOBAL_QUOTA["全局配额"]
GROUP_QUOTA["分组配额"]
INDIVIDUAL_QUOTA["个人配额"]
end
subgraph "时间维度"
HOURLY["小时配额"]
DAILY["日配额"]
MONTHLY["月配额"]
ANNUAL["年配额"]
end
subgraph "检查顺序"
CHECK_GLOBAL["检查全局配额"]
CHECK_GROUP["检查分组配额"]
CHECK_INDIVIDUAL["检查个人配额"]
CHECK_TIME["检查时间窗口"]
end
CHECK_GLOBAL --> CHECK_GROUP
CHECK_GROUP --> CHECK_INDIVIDUAL
CHECK_INDIVIDUAL --> CHECK_TIME
```

**图表来源**
- [api_key_service.go:82-114](file://backend/internal/service/api_key_service.go#L82-L114)

### 限流算法

系统采用滑动窗口限流算法：

| 时间窗口 | 限额类型 | 重置机制 |
|----------|----------|----------|
| 5小时 | 流量配额 | 窗口到期自动重置 |
| 1天 | 流量配额 | 每日重置 |
| 7天 | 流量配额 | 每周重置 |

**章节来源**
- [api_key_repo.go:508-537](file://backend/internal/repository/api_key_repo.go#L508-L537)

## 与用户表的关系

### 关联关系设计

API密钥与用户表建立了一对多的关系：

```mermaid
erDiagram
USERS {
bigint id PK
string email UK
string username UK
string role
decimal balance
integer concurrency
string status
}
API_KEYS {
bigint id PK
bigint user_id FK
string key UK
string name
string status
decimal quota
decimal quota_used
timestamp expires_at
}
USERS ||--o{ API_KEYS : "拥有"
```

**图表来源**
- [api_key.go:121-133](file://backend/ent/schema/api_key.go#L121-L133)

### 用户状态影响

用户状态直接影响API密钥的有效性：

- **激活用户**: 可正常使用所有API密钥
- **停用用户**: 所有API密钥立即失效
- **冻结账户**: 禁止所有API访问

**章节来源**
- [api_key_auth.go:100-110](file://backend/internal/server/middleware/api_key_auth.go#L100-L110)

## 与用量日志表的关系

### 用量追踪机制

系统通过用量日志表追踪每个API密钥的使用情况：

```mermaid
erDiagram
API_KEYS {
bigint id PK
string key UK
string name
decimal quota
decimal quota_used
}
USAGE_LOGS {
bigint id PK
bigint api_key_id FK
string endpoint
decimal cost_usd
timestamp created_at
}
API_KEYS ||--o{ USAGE_LOGS : "产生"
```

**图表来源**
- [api_key.go:132-133](file://backend/ent/schema/api_key.go#L132-L133)

### 日志记录策略

系统记录详细的用量信息：

| 日志字段 | 描述 | 用途 |
|----------|------|------|
| endpoint | API端点 | 分析使用模式 |
| cost_usd | 成本(美元) | 计费和统计 |
| created_at | 时间戳 | 时间序列分析 |
| api_key_id | 密钥标识 | 关联查询 |

**章节来源**
- [api_key_repo.go:132-177](file://backend/internal/repository/api_key_repo.go#L132-L177)

## 架构图

### 整体架构设计

```mermaid
graph TB
subgraph "客户端层"
WEB_APP["Web应用"]
MOBILE_APP["移动应用"]
DESKTOP_APP["桌面应用"]
PROGRAM["程序化客户端"]
end
subgraph "网关层"
API_GATEWAY["API网关"]
AUTH_MIDDLEWARE["认证中间件"]
RATE_LIMIT["限流中间件"]
end
subgraph "业务逻辑层"
API_KEY_HANDLER["API密钥处理器"]
API_KEY_SERVICE["API密钥服务"]
SUBSCRIPTION_SERVICE["订阅服务"]
end
subgraph "数据持久层"
API_KEY_REPO["API密钥仓库"]
USER_REPO["用户仓库"]
GROUP_REPO["分组仓库"]
USAGE_REPO["用量仓库"]
end
subgraph "数据存储"
POSTGRES_DB["PostgreSQL数据库"]
REDIS_CACHE["Redis缓存"]
AUDIT_LOGS["审计日志"]
end
WEB_APP --> API_GATEWAY
MOBILE_APP --> API_GATEWAY
DESKTOP_APP --> API_GATEWAY
PROGRAM --> API_GATEWAY
API_GATEWAY --> AUTH_MIDDLEWARE
AUTH_MIDDLEWARE --> RATE_LIMIT
RATE_LIMIT --> API_KEY_HANDLER
API_KEY_HANDLER --> API_KEY_SERVICE
API_KEY_SERVICE --> SUBSCRIPTION_SERVICE
API_KEY_SERVICE --> API_KEY_REPO
API_KEY_SERVICE --> USER_REPO
API_KEY_SERVICE --> GROUP_REPO
API_KEY_SERVICE --> USAGE_REPO
API_KEY_REPO --> POSTGRES_DB
USER_REPO --> POSTGRES_DB
GROUP_REPO --> POSTGRES_DB
USAGE_REPO --> POSTGRES_DB
API_KEY_SERVICE --> REDIS_CACHE
API_KEY_SERVICE --> AUDIT_LOGS
```

**图表来源**
- [api_key_handler.go:19-29](file://backend/internal/handler/api_key_handler.go#L19-L29)
- [api_key_service.go:195-229](file://backend/internal/service/api_key_service.go#L195-L229)

## 总结

API密钥表设计实现了企业级的安全密钥管理功能，具有以下特点：

### 核心优势

1. **安全性**: 多层防护机制，包括IP限制、配额控制、审计日志
2. **灵活性**: 支持多种密钥生成方式和分组权限模型
3. **可扩展性**: 模块化设计，易于扩展新功能
4. **可观测性**: 完善的统计和审计功能
5. **可靠性**: 软删除机制和错误处理策略

### 技术特色

- **Ent框架集成**: 强类型的ORM设计
- **多层缓存**: L1/L2缓存策略提升性能
- **滑动窗口限流**: 精确的流量控制
- **订阅集成**: 与订阅系统的深度整合
- **审计完备**: 全面的使用追踪和合规支持

该设计为企业提供了安全、可靠、易用的API密钥管理解决方案，能够满足各种规模企业的API治理需求。