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

// GroupStatusAstraCheckRun meow 指纹验证（历史名 Astra）的每次运行记录，一次运行对应一个预期模型，本 fork 自有功能。
type GroupStatusAstraCheckRun struct {
	ent.Schema
}

func (GroupStatusAstraCheckRun) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_astra_check_runs"},
	}
}

func (GroupStatusAstraCheckRun) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id"),
		field.Int64("config_id"),
		field.String("platform").Default(""),
		field.String("expected_model").Default(""),
		field.Int("round").Default(1),
		field.String("scoring_version").Default(""),
		field.String("benchmark_package_id").Default(""),
		field.String("benchmark_version").Default(""),
		field.String("benchmark_sha256").Default(""),
		field.String("request_model").Default(""),
		field.String("tier").Default("low"),
		field.Int64("account_id").Optional().Nillable(),
		field.String("verdict"),
		field.String("winner_model").Default(""),
		field.JSON("matches", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("cells", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("reasons", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("samples", []map[string]any{}).
			Default([]map[string]any{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int("requests_planned").Default(0),
		field.Int("requests_completed").Default(0),
		field.Int("valid_samples").Default(0),
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

func (GroupStatusAstraCheckRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "finished_at"),
		index.Fields("group_id", "expected_model", "finished_at"),
	}
}
