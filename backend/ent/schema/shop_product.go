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

type ShopProduct struct {
	ent.Schema
}

func (ShopProduct) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "shop_products"},
	}
}

func (ShopProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("description").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Float("price").SchemaType(map[string]string{dialect.Postgres: "decimal(10,2)"}),
		field.String("currency").MaxLen(10).Default("CNY"),
		field.String("redeem_type").MaxLen(20),
		field.Float("redeem_value").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).Default(0),
		field.Int64("group_id").Optional().Nillable(),
		field.Int("validity_days").Default(30),
		field.Int("stock_count").Default(0),
		field.Bool("is_active").Default(true),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ShopProduct) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_active", "sort_order"),
	}
}
