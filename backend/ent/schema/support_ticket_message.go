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

// SupportTicketMessage 工单线程里的一条消息（fork 本地功能）。
// 真源是 migrations/240_support_tickets.sql，这里只为模型可读性；仓储走原生 SQL。
type SupportTicketMessage struct {
	ent.Schema
}

func (SupportTicketMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "support_ticket_messages"},
	}
}

func (SupportTicketMessage) Fields() []ent.Field {
	text := map[string]string{dialect.Postgres: "text"}
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		field.Int64("ticket_id"),
		field.Int64("author_user_id"),
		field.String("author_email").Default("").MaxLen(255),
		// user（工单发起方）| admin | operator
		field.String("author_role").MaxLen(20),
		field.String("body").SchemaType(text),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(tstz),
	}
}

func (SupportTicketMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id", "created_at"),
	}
}
