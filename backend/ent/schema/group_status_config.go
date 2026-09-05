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
		// 稳定状态变红 / 从红恢复时是否推送提醒（Server酱³）
		field.Bool("notify_enabled").Default(true),
		// 纯 Sol 验证（Juice 指纹探测），仅 OpenAI 分组
		field.Bool("sol_juice_enabled").Default(false),
		field.Int("sol_juice_interval_seconds").Default(900),
		field.String("sol_juice_model").Default("gpt-5.6-sol"),
	}
}

func (GroupStatusConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("enabled"),
	}
}
