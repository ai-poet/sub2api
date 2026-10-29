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

// UserAffiliateLedger is the audit trail for rebate accrual and transfers.
type UserAffiliateLedger struct{ ent.Schema }

func (UserAffiliateLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "user_affiliate_ledger"}}
}

func (UserAffiliateLedger) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("action").MaxLen(32),
		field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int64("source_user_id").Optional().Nillable(),
		field.String("source_order_ref").Optional().Nillable().MaxLen(64),
		field.Time("frozen_until").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Float("balance_after").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("aff_quota_after").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("aff_frozen_quota_after").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("aff_history_quota_after").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UserAffiliateLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("source_user_id", "created_at"),
		index.Fields("action", "created_at"),
	}
}
