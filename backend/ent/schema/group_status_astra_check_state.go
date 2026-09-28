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

// GroupStatusAstraCheckState meow 指纹验证每个（分组, 预期模型）的最近结果与稳定结论，本 fork 自有功能。
type GroupStatusAstraCheckState struct {
	ent.Schema
}

func (GroupStatusAstraCheckState) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_astra_check_states"},
	}
}

func (GroupStatusAstraCheckState) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id"),
		field.Int64("config_id"),
		field.String("expected_model"),
		field.String("verdict").Default(""),
		field.String("stable_status").Default(""),
		field.String("winner_model").Default(""),
		field.JSON("matches", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("reasons", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("checked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int("consecutive_mismatch").Default(0),
		field.Int("valid_samples").Default(0),
		field.Int("planned_samples").Default(0),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("reasoning_tokens").Default(0),
		field.Float("last_cost_usd").Default(0),
		field.Int64("last_run_id").Optional().Nillable(),
		field.String("benchmark_package_id").Default(""),
		field.String("benchmark_version").Default(""),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GroupStatusAstraCheckState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "expected_model").Unique(),
	}
}
