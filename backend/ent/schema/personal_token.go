package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// PersonalToken 运维管理员个人令牌（fork 本地功能，见 docs/PERSONAL_TOKENS.md）。
// 真源是 migrations/241_personal_tokens.sql，这里只为模型可读性；仓储走原生 SQL。
type PersonalToken struct {
	ent.Schema
}

func (PersonalToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "personal_tokens"},
	}
}

func (PersonalToken) Fields() []ent.Field {
	tstz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		// 每人至多一个，重新生成即覆盖
		field.Int64("user_id").Unique(),
		// SHA-256(hex)；明文从不落库
		field.String("token_hash").MaxLen(64).Unique(),
		field.String("token_hint").MaxLen(32),
		// 签发时的密码 / 邮箱指纹（service.User.TokenVersion），不一致即失效
		field.Int64("user_token_version"),
		// 空表示永不过期
		field.Time("expires_at").Optional().Nillable().SchemaType(tstz),
		field.Time("last_used_at").Optional().Nillable().SchemaType(tstz),
		field.String("last_used_ip").Default("").MaxLen(64),
		field.String("created_ip").Default("").MaxLen(64),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(tstz),
	}
}
