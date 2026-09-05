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

type GroupStatusState struct {
	ent.Schema
}

func (GroupStatusState) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_states"},
	}
}

func (GroupStatusState) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (GroupStatusState) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id").Unique(),
		field.Int64("config_id"),
		field.String("latest_status").Default(""),
		field.String("stable_status").Default(""),
		field.String("response_excerpt").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int64("latency_ms").Optional().Nillable(),
		field.Int("http_code").Optional().Nillable(),
		field.String("sub_status").Default(""),
		field.String("error_detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("observed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int("consecutive_down").Default(0),
		field.Int("consecutive_non_down").Default(0),
		// 纯 Sol 验证（Juice 指纹探测）的最近结果与稳定结论
		field.String("sol_juice_status").Default(""),
		field.String("sol_juice_stable_status").Default(""),
		field.String("sol_juice_value").Default(""),
		field.String("sol_juice_detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("sol_juice_checked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int("sol_juice_consecutive_mismatch").Default(0),
		field.Int64("sol_juice_input_tokens").Default(0),
		field.Int64("sol_juice_output_tokens").Default(0),
		field.Int64("sol_juice_reasoning_tokens").Default(0),
	}
}

func (GroupStatusState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("stable_status"),
	}
}
