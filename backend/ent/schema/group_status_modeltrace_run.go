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

// GroupStatusModelTraceRun ModelTrace 指纹验证（数字分布指纹）的每次运行记录，本 fork 自有功能。
//
// Deprecated: dormant since 243 —— ModelTrace 已并入 meow 指纹验证，表保留给旧镜像，等清理迁移一起删除。
type GroupStatusModelTraceRun struct {
	ent.Schema
}

func (GroupStatusModelTraceRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_modeltrace_runs"},
	}
}

func (GroupStatusModelTraceRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id"),
		field.Int64("config_id"),
		field.String("platform").Default(""),
		field.String("bank_sha256").Default(""),
		field.String("bank_built_at").Default(""),
		field.String("expected_model").Default(""),
		field.String("request_model").Default(""),
		field.Int64("account_id").Optional().Nillable(),
		field.String("account_type").Default(""),
		field.Int("round").Default(1),
		field.String("verdict"),
		field.String("outcome").Default(""),
		field.String("top_model").Default(""),
		field.Float("top_probability").Default(0),
		field.Float("expected_probability").Optional().Nillable(),
		field.Int("calibration_queries").Default(0),
		field.Float("beta").Default(0),
		field.JSON("ranking", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("family_probabilities", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("reasons", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("outputs", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int("attempts_planned").Default(0),
		field.Int("attempts_made").Default(0),
		field.Int("valid_outputs").Default(0),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("reasoning_tokens").Default(0),
		field.Float("cost_usd").Default(0),
		field.Int64("latency_ms").Optional().Nillable(),
		field.Int("http_code").Optional().Nillable(),
		field.String("error_detail").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("started_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("finished_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GroupStatusModelTraceRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "finished_at"),
	}
}
