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

// GroupStatusJuiceRecord 纯 Sol 验证（Juice 指纹探测）的每次样本，本 fork 自有功能。
//
// Deprecated: dormant since 242 —— Juice 探测已由 ModelTrace 指纹验证替换，表保留给旧镜像，等清理迁移一起删除。
type GroupStatusJuiceRecord struct {
	ent.Schema
}

func (GroupStatusJuiceRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_juice_records"},
	}
}

func (GroupStatusJuiceRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id"),
		field.Int64("config_id"),
		field.String("model").Default(""),
		field.String("effort").Default("high"),
		field.String("classification"),
		field.String("normalized_value").Default(""),
		field.String("answer_excerpt").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int("http_code").Optional().Nillable(),
		field.Int64("latency_ms").Optional().Nillable(),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("reasoning_tokens").Default(0),
		field.String("error_detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("observed_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GroupStatusJuiceRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "observed_at"),
	}
}
