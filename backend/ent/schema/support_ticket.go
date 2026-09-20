package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SupportTicket 工单（fork 本地功能）。
// 真源是 migrations/240_support_tickets.sql，这里只为模型可读性；仓储走原生 SQL。
type SupportTicket struct {
	ent.Schema
}

func (SupportTicket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "support_tickets"},
	}
}

func (SupportTicket) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (SupportTicket) Fields() []ent.Field {
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		field.Int64("user_id"),
		// 提交时的邮箱快照，管理端以 user_id 为准
		field.String("user_email").Default("").MaxLen(255),
		field.String("title").MaxLen(200),
		// account | billing | api | other
		field.String("category").Default("other").MaxLen(32),
		// open（等客服）| replied（客服已回复，等用户）| closed
		field.String("status").Default("open").MaxLen(20),
		// 客服回复后置 true，用户打开详情时清零（用户侧角标）
		field.Bool("user_unread").Default(false),
		field.Int("message_count").Default(0),
		field.Time("last_message_at").Default(time.Now).SchemaType(tstz),
		field.Time("closed_at").Optional().Nillable().SchemaType(tstz),
		field.Int64("closed_by_user_id").Optional().Nillable(),
		// user | admin | operator
		field.String("closed_by_role").Default("").MaxLen(20),
	}
}

func (SupportTicket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "last_message_at"),
		index.Fields("status", "last_message_at"),
	}
}
