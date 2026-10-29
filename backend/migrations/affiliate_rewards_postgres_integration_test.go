//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestAffiliateRewardsMigrationPostgres executes the complete migration against
// the legacy settings shape used by existing installations (no created_at).
func TestAffiliateRewardsMigrationPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SUB2API_TEST_DATABASE_URL is required for the PostgreSQL migration test")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	schema := fmt.Sprintf("affiliate_253_test_%d", time.Now().UnixNano())
	quotedSchema := pq.QuoteIdentifier(schema)
	_, err = db.Exec(`CREATE SCHEMA ` + quotedSchema)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupDB, cleanupErr := sql.Open("postgres", dsn)
		require.NoError(t, cleanupErr)
		defer func() { _ = cleanupDB.Close() }()
		_, cleanupErr = cleanupDB.Exec(`DROP SCHEMA ` + quotedSchema + ` CASCADE`)
		require.NoError(t, cleanupErr)
	})
	_, err = db.Exec(`SET search_path TO ` + quotedSchema)
	require.NoError(t, err)

	_, err = db.Exec(`
CREATE TABLE settings (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    referral_code VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE user_referrals (
    id BIGSERIAL PRIMARY KEY,
    referrer_id BIGINT NOT NULL,
    referee_id BIGINT NOT NULL
);
INSERT INTO users (id, referral_code) VALUES (1, 'INVITER1'), (2, 'INVITEE1');
INSERT INTO user_referrals (referrer_id, referee_id) VALUES (1, 2);
INSERT INTO settings (key, value) VALUES ('referral_enabled', 'true');
`)
	require.NoError(t, err)

	content, err := FS.ReadFile("253_affiliate_rewards.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), string(content))
	require.NoError(t, err)

	var enabled string
	require.NoError(t, db.QueryRow(`SELECT value FROM settings WHERE key='affiliate_enabled'`).Scan(&enabled))
	require.Equal(t, "true", enabled)
	var affCount int
	require.NoError(t, db.QueryRow(`SELECT aff_count FROM user_affiliates WHERE user_id=1`).Scan(&affCount))
	require.Equal(t, 1, affCount)
	var inviterID int64
	require.NoError(t, db.QueryRow(`SELECT inviter_id FROM user_affiliates WHERE user_id=2`).Scan(&inviterID))
	require.Equal(t, int64(1), inviterID)
	var custom bool
	require.NoError(t, db.QueryRow(`SELECT aff_code_custom FROM user_affiliates WHERE user_id=1`).Scan(&custom))
	require.False(t, custom)

	_, err = db.Exec(string(content))
	require.NoError(t, err, "migration must remain safe to retry")
}
