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

type ShopOrder struct {
	ent.Schema
}

func (ShopOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "shop_orders"},
	}
}

func (ShopOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_no").MaxLen(64).Unique(),
		field.Int64("user_id"),
		field.Int64("product_id"),
		field.String("product_name").MaxLen(100),
		field.Float("amount").SchemaType(map[string]string{dialect.Postgres: "decimal(10,2)"}),
		field.String("currency").MaxLen(10).Default("CNY"),
		field.String("payment_method").MaxLen(20).Optional().Nillable(),
		field.String("status").MaxLen(20).Default("pending"),
		field.Int64("redeem_code_id").Optional().Nillable(),
		field.Time("paid_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("expires_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ShopOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "status"),
		index.Fields("order_no"),
	}
}
