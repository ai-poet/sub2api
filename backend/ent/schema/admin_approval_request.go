package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AdminApprovalRequest 运维管理员写操作审批队列（fork 本地功能）。
// 真源是 migrations/239_admin_approval_requests.sql，这里只为模型可读性；仓储走原生 SQL。
type AdminApprovalRequest struct {
	ent.Schema
}

func (AdminApprovalRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "admin_approval_requests"},
	}
}

func (AdminApprovalRequest) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (AdminApprovalRequest) Fields() []ent.Field {
	text := map[string]string{dialect.Postgres: "text"}
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		// pending | executing | approved | failed | rejected | cancelled | expired
		field.String("status").Default("pending").MaxLen(20),
		// 审计动作名，如 admin.users.balance.create
		field.String("action").MaxLen(128),
		field.String("method").MaxLen(10),
		field.String("route_template").MaxLen(255),
		field.String("request_path").MaxLen(1024),
		field.String("request_query").Default("").SchemaType(text),
		field.String("content_type").Default("").MaxLen(128),
		// AES-256-GCM 加密的原始 body（重放用）；脱敏副本用于展示
		field.String("request_body_enc").Default("").SchemaType(text),
		field.String("request_body_redacted").Default("").SchemaType(text),
		field.String("request_body_sha256").Default("").MaxLen(64),
		// user | users_batch | subscription | api_key
		field.String("target_type").Default("").MaxLen(32),
		field.Int64("target_id").Optional().Nillable(),
		field.String("target_summary").Default("").MaxLen(255),
		field.Int64("requester_user_id"),
		field.String("requester_email").Default("").MaxLen(255),
		field.String("requester_ip").Default("").MaxLen(64),
		field.String("request_id").Default("").MaxLen(64),
		field.Int64("decided_by_user_id").Optional().Nillable(),
		field.String("decided_by_email").Default("").MaxLen(255),
		field.Time("decided_at").Optional().Nillable().SchemaType(tstz),
		field.String("decision_reason").Default("").SchemaType(text),
		field.Time("executed_at").Optional().Nillable().SchemaType(tstz),
		field.Int("result_status_code").Optional().Nillable(),
		field.String("result_body").Default("").SchemaType(text),
		field.String("result_error").Default("").MaxLen(255),
		field.Time("notified_at").Optional().Nillable().SchemaType(tstz),
		field.Time("expires_at").SchemaType(tstz),
	}
}

func (AdminApprovalRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("requester_user_id", "created_at"),
		index.Fields("expires_at"),
	}
}
