package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserReferral holds the schema definition for the UserReferral entity.
type UserReferral struct {
	ent.Schema
}

func (UserReferral) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_referrals"},
	}
}

func (UserReferral) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("referrer_id"),
		field.Int64("referee_id"),
		field.String("status").
			MaxLen(20).
			Default("pending"),

		// 推荐人奖励快照
		field.Float("referrer_balance_reward").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Int64("referrer_group_id").
			Optional().
			Nillable(),
		field.Int("referrer_subscription_days").
			Default(0),
		field.Time("referrer_rewarded_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		// 被推荐人奖励快照
		field.Float("referee_balance_reward").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Int64("referee_group_id").
			Optional().
			Nillable(),
		field.Int("referee_subscription_days").
			Default(0),
		field.Time("referee_rewarded_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UserReferral) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("referrer", User.Type).
			Ref("referrals_made").
			Field("referrer_id").
			Unique().
			Required(),
		edge.From("referee", User.Type).
			Ref("referral_received").
			Field("referee_id").
			Unique().
			Required(),
	}
}

func (UserReferral) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("referee_id").Unique(),
		index.Fields("referrer_id"),
		index.Fields("status"),
	}
}
