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
		field.Int64("total_latency_ms").Optional().Nillable(),
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
		// Deprecated: dormant since 242 —— 纯 Sol 验证（Juice）的旧列，保留给旧镜像
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
		// Deprecated: dormant since 243 —— ModelTrace 的旧列，保留给旧镜像
		field.String("modeltrace_verdict").Default(""),
		field.String("modeltrace_stable_status").Default(""),
		field.String("modeltrace_run_expected_model").Default(""),
		field.String("modeltrace_top_model").Default(""),
		field.Float("modeltrace_top_probability").Default(0),
		field.Float("modeltrace_expected_probability").Optional().Nillable(),
		field.JSON("modeltrace_ranking", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("modeltrace_reasons", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("modeltrace_detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("modeltrace_checked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int("modeltrace_consecutive_mismatch").Default(0),
		field.Int("modeltrace_valid_outputs").Default(0),
		field.Int64("modeltrace_input_tokens").Default(0),
		field.Int64("modeltrace_output_tokens").Default(0),
		field.Int64("modeltrace_reasoning_tokens").Default(0),
		field.Float("modeltrace_last_cost_usd").Default(0),
		field.Int64("modeltrace_last_run_id").Optional().Nillable(),
		// Deprecated: dormant since 243 —— 单模型 Astra 的旧列，按模型的状态在 group_status_astra_check_states，保留给旧镜像
		field.String("astra_check_verdict").Default(""),
		field.String("astra_check_stable_status").Default(""),
		field.String("astra_check_winner").Default(""),
		field.JSON("astra_check_matches", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("astra_check_reasons", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("astra_check_detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("astra_check_checked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int("astra_check_consecutive_mismatch").Default(0),
		field.Int("astra_check_valid_samples").Default(0),
		field.Int("astra_check_planned_samples").Default(0),
		field.Int64("astra_check_input_tokens").Default(0),
		field.Int64("astra_check_output_tokens").Default(0),
		field.Int64("astra_check_reasoning_tokens").Default(0),
		field.Int64("astra_check_last_run_id").Optional().Nillable(),
	}
}

func (GroupStatusState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("stable_status"),
	}
}
