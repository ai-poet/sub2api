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

// ContentTranslation 内容自动翻译的译文缓存（fork 本地功能）。
// 真源是 migrations/247_content_translations.sql，这里只为模型可读性；仓储走原生 SQL。
type ContentTranslation struct {
	ent.Schema
}

func (ContentTranslation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "content_translations"},
	}
}

func (ContentTranslation) Fields() []ent.Field {
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		// sha256(规范化原文)，十六进制
		field.String("source_hash").MaxLen(64).SchemaType(map[string]string{dialect.Postgres: "char(64)"}),
		// zh | en | ja
		field.String("target_lang").MaxLen(8),
		field.Text("source_text"),
		field.Text("translated_text"),
		field.String("model").Default("").MaxLen(128),
		// 管理员改写过的译文，自动翻译不会覆盖
		field.Bool("manual").Default(false),
		field.Time("created_at").Default(time.Now).Immutable().SchemaType(tstz),
		field.Time("updated_at").Default(time.Now).SchemaType(tstz),
		field.Time("last_seen_at").Default(time.Now).SchemaType(tstz),
	}
}

func (ContentTranslation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_hash", "target_lang").Unique(),
	}
}
