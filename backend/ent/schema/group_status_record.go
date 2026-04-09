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

type GroupStatusRecord struct {
	ent.Schema
}

func (GroupStatusRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_status_records"},
	}
}

func (GroupStatusRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id"),
		field.Int64("config_id"),
		field.String("status"),
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
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GroupStatusRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id", "observed_at"),
		index.Fields("config_id", "observed_at"),
	}
}
