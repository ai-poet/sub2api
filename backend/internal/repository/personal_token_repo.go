package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 运维管理员个人令牌仓储（fork 本地功能，原生 SQL，风格同 passkey_repo.go / admin_approval_repo.go）。
// 只存 SHA-256；按 user_id 唯一，重新生成即整行覆盖。

const personalTokenColumns = `id, user_id, token_hash, token_hint, user_token_version, expires_at,
	last_used_at, last_used_ip, created_ip, created_at`

type personalTokenRepository struct {
	db *sql.DB
}

// NewPersonalTokenRepository 构造个人令牌仓储。
func NewPersonalTokenRepository(db *sql.DB) service.PersonalTokenRepository {
	return &personalTokenRepository{db: db}
}

func (r *personalTokenRepository) Upsert(ctx context.Context, token *service.PersonalToken) (*service.PersonalToken, error) {
	if token == nil {
		return nil, errors.New("personal token is nil")
	}
	createdAt := token.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	// 覆盖时连同最后使用信息一起清空：新令牌从未被使用过
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO personal_tokens (
			user_id, token_hash, token_hint, user_token_version, expires_at,
			last_used_at, last_used_ip, created_ip, created_at
		) VALUES ($1, $2, $3, $4, $5, NULL, '', $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			token_hash         = EXCLUDED.token_hash,
			token_hint         = EXCLUDED.token_hint,
			user_token_version = EXCLUDED.user_token_version,
			expires_at         = EXCLUDED.expires_at,
			last_used_at       = NULL,
			last_used_ip       = '',
			created_ip         = EXCLUDED.created_ip,
			created_at         = EXCLUDED.created_at
		RETURNING `+personalTokenColumns,
		token.UserID, token.TokenHash, token.TokenHint, token.UserTokenVersion, token.ExpiresAt,
		token.CreatedIP, createdAt,
	)
	return scanPersonalToken(row)
}

func (r *personalTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*service.PersonalToken, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+personalTokenColumns+` FROM personal_tokens WHERE token_hash = $1`, tokenHash)
	return scanPersonalToken(row)
}

func (r *personalTokenRepository) GetByUserID(ctx context.Context, userID int64) (*service.PersonalToken, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+personalTokenColumns+` FROM personal_tokens WHERE user_id = $1`, userID)
	return scanPersonalToken(row)
}

func (r *personalTokenRepository) DeleteByUserID(ctx context.Context, userID int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM personal_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *personalTokenRepository) List(ctx context.Context) ([]*service.PersonalToken, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+personalTokenColumns+` FROM personal_tokens ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.PersonalToken, 0)
	for rows.Next() {
		token, err := scanPersonalToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, token)
	}
	return out, rows.Err()
}

func (r *personalTokenRepository) TouchLastUsed(ctx context.Context, id int64, ip string, now time.Time, minInterval time.Duration) error {
	// 条件更新：距上次写入不足 minInterval 时不写，避免高频脚本每个请求都写库
	_, err := r.db.ExecContext(ctx, `
		UPDATE personal_tokens
		SET last_used_at = $2, last_used_ip = $3
		WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < $4 OR last_used_ip <> $3)`,
		id, now, ip, now.Add(-minInterval),
	)
	return err
}

type personalTokenScanner interface {
	Scan(dest ...any) error
}

func scanPersonalToken(row personalTokenScanner) (*service.PersonalToken, error) {
	var (
		token      service.PersonalToken
		expiresAt  sql.NullTime
		lastUsedAt sql.NullTime
	)
	err := row.Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.TokenHint, &token.UserTokenVersion, &expiresAt,
		&lastUsedAt, &token.LastUsedIP, &token.CreatedIP, &token.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPersonalTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		token.ExpiresAt = &t
	}
	if lastUsedAt.Valid {
		t := lastUsedAt.Time
		token.LastUsedAt = &t
	}
	return &token, nil
}
