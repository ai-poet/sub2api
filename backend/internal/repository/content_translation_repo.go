package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// 内容自动翻译的译文缓存与外部文案源（fork 本地功能，原生 SQL，风格同 site_message_repo.go）。

const contentTranslationColumns = `id, source_hash, target_lang, source_text, translated_text, model, manual,
	created_at, updated_at, last_seen_at`

type contentTranslationRepository struct {
	db *sql.DB
}

// NewContentTranslationRepository 构造译文缓存仓储。
func NewContentTranslationRepository(db *sql.DB) service.ContentTranslationRepository {
	return &contentTranslationRepository{db: db}
}

func (r *contentTranslationRepository) ListAll(ctx context.Context) ([]*service.ContentTranslation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+contentTranslationColumns+` FROM content_translations`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanContentTranslations(rows)
}

// Upsert 写入机器译文。已存在的人工译文不覆盖：ON CONFLICT 的 WHERE 不成立时 RETURNING 没有行，
// 这时再读出库里那一行返回，调用方据此更新内存镜像。
func (r *contentTranslationRepository) Upsert(ctx context.Context, entry *service.ContentTranslation) (*service.ContentTranslation, error) {
	if entry == nil {
		return nil, errors.New("content translation is nil")
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO content_translations (source_hash, target_lang, source_text, translated_text, model, manual)
		VALUES ($1, $2, $3, $4, $5, FALSE)
		ON CONFLICT (source_hash, target_lang) DO UPDATE SET
			source_text = EXCLUDED.source_text,
			translated_text = EXCLUDED.translated_text,
			model = EXCLUDED.model,
			updated_at = NOW(),
			last_seen_at = NOW()
		WHERE content_translations.manual = FALSE
		RETURNING `+contentTranslationColumns,
		entry.SourceHash, entry.TargetLang, entry.SourceText, entry.TranslatedText, entry.Model,
	)
	out, err := scanContentTranslation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return r.getByHashLang(ctx, entry.SourceHash, entry.TargetLang)
	}
	return out, err
}

func (r *contentTranslationRepository) getByHashLang(ctx context.Context, hash, lang string) (*service.ContentTranslation, error) {
	out, err := scanContentTranslation(r.db.QueryRowContext(ctx,
		`SELECT `+contentTranslationColumns+` FROM content_translations WHERE source_hash = $1 AND target_lang = $2`,
		hash, lang))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrContentTranslationNotFound
	}
	return out, err
}

func (r *contentTranslationRepository) GetByID(ctx context.Context, id int64) (*service.ContentTranslation, error) {
	out, err := scanContentTranslation(r.db.QueryRowContext(ctx,
		`SELECT `+contentTranslationColumns+` FROM content_translations WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrContentTranslationNotFound
	}
	return out, err
}

func (r *contentTranslationRepository) UpdateManual(ctx context.Context, id int64, translated string) (*service.ContentTranslation, error) {
	out, err := scanContentTranslation(r.db.QueryRowContext(ctx, `
		UPDATE content_translations
		SET translated_text = $2, manual = TRUE, updated_at = NOW()
		WHERE id = $1
		RETURNING `+contentTranslationColumns, id, translated))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrContentTranslationNotFound
	}
	return out, err
}

func (r *contentTranslationRepository) Delete(ctx context.Context, id int64) (*service.ContentTranslation, error) {
	out, err := scanContentTranslation(r.db.QueryRowContext(ctx,
		`DELETE FROM content_translations WHERE id = $1 RETURNING `+contentTranslationColumns, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrContentTranslationNotFound
	}
	return out, err
}

func (r *contentTranslationRepository) DeleteMachine(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM content_translations WHERE manual = FALSE`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *contentTranslationRepository) TouchSeen(ctx context.Context, hashes []string, at time.Time) error {
	if len(hashes) == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE content_translations SET last_seen_at = $2 WHERE source_hash = ANY($1)`,
		pq.Array(hashes), at)
	return err
}

func (r *contentTranslationRepository) List(ctx context.Context, filter service.ContentTranslationFilter) ([]*service.ContentTranslation, int64, error) {
	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	where := []string{"TRUE"}
	args := []any{}
	if lang := strings.TrimSpace(filter.Lang); lang != "" {
		args = append(args, lang)
		where = append(where, fmt.Sprintf("target_lang = $%d", len(args)))
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		args = append(args, "%"+escapeContentTranslationLike(q)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf("(source_text ILIKE $%d OR translated_text ILIKE $%d)", n, n))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM content_translations WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+contentTranslationColumns+`
		FROM content_translations
		WHERE `+whereSQL+`
		ORDER BY updated_at DESC, id DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2),
		limitArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanContentTranslations(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ReplaceNamespaceSources 在一个事务里整体替换某个 namespace 的登记：删掉不再登记的，补上新增的。
func (r *contentTranslationRepository) ReplaceNamespaceSources(ctx context.Context, namespace string, sources map[string]string) error {
	hashes := make([]string, 0, len(sources))
	for hash := range sources {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM content_translation_sources WHERE namespace = $1 AND NOT (source_hash = ANY($2))`,
		namespace, pq.Array(hashes)); err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO content_translation_sources (namespace, source_hash, source_text)
			VALUES ($1, $2, $3)
			ON CONFLICT (namespace, source_hash) DO NOTHING`,
			namespace, hash, sources[hash]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *contentTranslationRepository) ListNamespaceSources(ctx context.Context) ([]service.ContentTranslationSourceText, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT namespace, source_hash, source_text FROM content_translation_sources`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.ContentTranslationSourceText
	for rows.Next() {
		var item service.ContentTranslationSourceText
		if err := rows.Scan(&item.Namespace, &item.SourceHash, &item.SourceText); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanContentTranslations(rows *sql.Rows) ([]*service.ContentTranslation, error) {
	var out []*service.ContentTranslation
	for rows.Next() {
		item, err := scanContentTranslation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanContentTranslation(row scannable) (*service.ContentTranslation, error) {
	var out service.ContentTranslation
	if err := row.Scan(
		&out.ID, &out.SourceHash, &out.TargetLang, &out.SourceText, &out.TranslatedText, &out.Model, &out.Manual,
		&out.CreatedAt, &out.UpdatedAt, &out.LastSeenAt,
	); err != nil {
		return nil, err
	}
	out.SourceHash = strings.TrimSpace(out.SourceHash)
	return &out, nil
}

// escapeContentTranslationLike 转义 ILIKE 的通配符，搜索词按字面匹配。
func escapeContentTranslationLike(q string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(q)
}
