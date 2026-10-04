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

// ContentTranslationSource 外部登记的待翻译文案（fork 本地功能，目前只有支付服务 namespace=pay）。
// 真源是 migrations/247_content_translations.sql（主键是 (namespace, source_hash)），
// 这里只为模型可读性；仓储走原生 SQL。
type ContentTranslationSource struct {
	ent.Schema
}

func (ContentTranslationSource) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "content_translation_sources"},
	}
}

func (ContentTranslationSource) Fields() []ent.Field {
	return []ent.Field{
		field.String("namespace").MaxLen(32),
		field.String("source_hash").MaxLen(64).SchemaType(map[string]string{dialect.Postgres: "char(64)"}),
		field.Text("source_text"),
		field.Time("updated_at").Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ContentTranslationSource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "source_hash").Unique(),
	}
}
