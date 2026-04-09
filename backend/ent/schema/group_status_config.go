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

type GroupStatusConfig struct {
	ent.Schema
}

func (GroupStatusConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_configs"},
	}
}

func (GroupStatusConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (GroupStatusConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id").Unique(),
		field.Bool("enabled").Default(false),
		field.String("probe_model").Default(""),
		field.String("probe_prompt").
			Default("").
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("validation_mode").Default("non_empty"),
		field.JSON("expected_keywords", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int("interval_seconds").Default(60),
		field.Int("timeout_seconds").Default(30),
		field.Int64("slow_latency_ms").Default(15000),
	}
}

func (GroupStatusConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("enabled"),
	}
}
