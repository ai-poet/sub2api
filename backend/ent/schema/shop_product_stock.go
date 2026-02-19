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

type ShopProductStock struct {
	ent.Schema
}

func (ShopProductStock) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "shop_product_stocks"},
	}
}

func (ShopProductStock) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("product_id"),
		field.Int64("redeem_code_id"),
		field.String("status").MaxLen(20).Default("available"),
		field.Int64("order_id").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ShopProductStock) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "status"),
	}
}
