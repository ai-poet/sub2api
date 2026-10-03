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

// SiteMessage 站内信（fork 本地功能）。
// 真源是 migrations/246_site_messages.sql，这里只为模型可读性；仓储走原生 SQL。
type SiteMessage struct {
	ent.Schema
}

func (SiteMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "site_messages"},
	}
}

func (SiteMessage) Fields() []ent.Field {
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		field.Int64("user_id"),
		// security | admin | system
		field.String("category").Default("admin").MaxLen(20),
		field.String("title").MaxLen(200),
		// Markdown
		field.Text("content"),
		// content_moderation | content_moderation_ban | cyber_policy | cyber_policy_ban | admin | appeal_restore
		field.String("source_type").Default("").MaxLen(32),
		// log:<id> | req:<request_id> | approval:<id>
		field.String("source_id").Default("").MaxLen(128),
		// 写信的工作人员（经审批时为发起的 operator）；nil 表示系统
		field.Int64("sender_user_id").Optional().Nillable(),
		// admin | operator | ''
		field.String("sender_role").Default("").MaxLen(20),
		field.Int64("approval_id").Optional().Nillable(),
		field.Time("read_at").Optional().Nillable().SchemaType(tstz),
		field.Time("created_at").Default(time.Now).Immutable().SchemaType(tstz),
	}
}

func (SiteMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
	}
}
