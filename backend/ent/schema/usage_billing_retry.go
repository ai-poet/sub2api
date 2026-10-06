package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UsageBillingRetry 扣费失败重试队列（fork 本地功能）。
// 真源是 migrations/248_usage_billing_retries.sql，这里只为模型可读性；仓储走原生 SQL。
type UsageBillingRetry struct {
	ent.Schema
}

func (UsageBillingRetry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "usage_billing_retries"},
	}
}

func (UsageBillingRetry) Fields() []ent.Field {
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	numeric := map[string]string{dialect.Postgres: "numeric(20,8)"}
	return []ent.Field{
		field.String("request_id").MaxLen(255),
		field.Int64("api_key_id"),
		field.Int64("user_id"),
		field.Int64("account_id"),
		field.Int64("group_id").Optional().Nillable(),
		field.String("platform").Default("").MaxLen(64),
		// 应扣金额快照（倍率后），仅展示与告警用
		field.Float("actual_cost").Default(0).SchemaType(numeric),
		// 序列化的 service.UsageBillingCommand
		field.JSON("command", map[string]any{}),
		// pending | settled | failed
		field.String("status").Default("pending").MaxLen(16),
		field.Int("attempts").Default(0),
		field.Text("last_error").Default(""),
		field.Time("next_retry_at").Default(time.Now).SchemaType(tstz),
		field.Time("settled_at").Optional().Nillable().SchemaType(tstz),
		field.Time("created_at").Default(time.Now).Immutable().SchemaType(tstz),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(tstz),
	}
}

func (UsageBillingRetry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id", "api_key_id").Unique(),
		index.Fields("next_retry_at"),
		index.Fields("created_at"),
		index.Fields("user_id", "created_at"),
	}
}
