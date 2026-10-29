package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// UserAffiliate mirrors the fork-local affiliate profile table. Monetary
// transitions are intentionally implemented with explicit SQL in repository.
type UserAffiliate struct{ ent.Schema }

func (UserAffiliate) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "user_affiliates"}}
}

func (UserAffiliate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Unique(),
		field.String("aff_code").MaxLen(32).Unique(),
		field.Int64("inviter_id").Optional().Nillable(),
		field.Int("aff_count").Default(0),
		field.Float("aff_quota").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
		field.Float("aff_frozen_quota").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
		field.Float("aff_history_quota").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
		field.Float("aff_rebate_rate_percent").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(5,2)"}),
		field.Bool("aff_code_custom").Default(false),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}
