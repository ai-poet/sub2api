//go:build integration

package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestGroupRateSiteMessagesPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("SUB2API_TEST_DATABASE_URL is required for the PostgreSQL migration test")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	schema := fmt.Sprintf("grsm_254_test_%d", time.Now().UnixNano())
	_, err = tx.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema))
	require.NoError(t, err)
	_, err = tx.Exec(`SET LOCAL search_path TO ` + pq.QuoteIdentifier(schema))
	require.NoError(t, err)

	_, err = tx.Exec(`
CREATE TABLE groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    image_rate_independent BOOLEAN NOT NULL DEFAULT false,
    image_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    video_rate_independent BOOLEAN NOT NULL DEFAULT false,
    video_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    peak_rate_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    peak_start VARCHAR(5) NOT NULL DEFAULT '',
    peak_end VARCHAR(5) NOT NULL DEFAULT '',
    peak_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    model_pricing JSONB,
    is_exclusive BOOLEAN NOT NULL DEFAULT FALSE,
    subscription_type VARCHAR(20) NOT NULL DEFAULT 'standard',
    platform VARCHAR(50) NOT NULL DEFAULT 'anthropic',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    restrict_public_groups BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL DEFAULT '',
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE user_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(user_id, group_id)
);
CREATE TABLE user_allowed_groups (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);
CREATE TABLE user_group_rate_multipliers (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NULL,
    rpm_override integer NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);
`)
	require.NoError(t, err)

	siteMessagesSQL, err := FS.ReadFile("246_site_messages.sql")
	require.NoError(t, err)
	_, err = tx.Exec(string(siteMessagesSQL))
	require.NoError(t, err)

	userSeq := 0
	keySeq := 0

	mustExec := func(t *testing.T, query string, args ...any) {
		t.Helper()
		_, err := tx.Exec(query, args...)
		require.NoError(t, err, query)
	}
	type groupSeed struct {
		rate, imageRate, videoRate        float64
		imageIndep, videoIndep, exclusive bool
		subscriptionType, status          string
		deleted                           bool
	}
	newGroup := func(t *testing.T, name string, configure func(*groupSeed)) int64 {
		t.Helper()
		s := groupSeed{rate: 1, imageRate: 1, videoRate: 1, subscriptionType: "standard", status: "active"}
		if configure != nil {
			configure(&s)
		}
		var deletedAt *time.Time
		if s.deleted {
			now := time.Now()
			deletedAt = &now
		}
		var id int64
		require.NoError(t, tx.QueryRow(`INSERT INTO groups (name, rate_multiplier, image_rate_independent, image_rate_multiplier, video_rate_independent, video_rate_multiplier, is_exclusive, subscription_type, status, deleted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
			name, s.rate, s.imageIndep, s.imageRate, s.videoIndep, s.videoRate, s.exclusive, s.subscriptionType, s.status, deletedAt).Scan(&id))
		return id
	}
	type userSeed struct {
		status   string
		restrict bool
		deleted  bool
	}
	newUser := func(t *testing.T, configure func(*userSeed)) int64 {
		t.Helper()
		userSeq++
		s := userSeed{status: "active"}
		if configure != nil {
			configure(&s)
		}
		var deletedAt *time.Time
		if s.deleted {
			now := time.Now()
			deletedAt = &now
		}
		var id int64
		require.NoError(t, tx.QueryRow(`INSERT INTO users (email, status, restrict_public_groups, deleted_at) VALUES ($1,$2,$3,$4) RETURNING id`,
			fmt.Sprintf("grsm-user-%d@example.com", userSeq), s.status, s.restrict, deletedAt).Scan(&id))
		return id
	}
	addKey := func(t *testing.T, userID, groupID int64, status string, expiresAt *time.Time, deleted bool) {
		t.Helper()
		keySeq++
		var deletedAt *time.Time
		if deleted {
			now := time.Now()
			deletedAt = &now
		}
		mustExec(t, `INSERT INTO api_keys (user_id, key, group_id, status, expires_at, deleted_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			userID, fmt.Sprintf("sk-grsm-%d", keySeq), groupID, status, expiresAt, deletedAt)
	}
	addSub := func(t *testing.T, userID, groupID int64, status string, startsAt, expiresAt time.Time, deleted bool) {
		t.Helper()
		var deletedAt *time.Time
		if deleted {
			now := time.Now()
			deletedAt = &now
		}
		mustExec(t, `INSERT INTO user_subscriptions (user_id, group_id, starts_at, expires_at, status, deleted_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			userID, groupID, startsAt, expiresAt, status, deletedAt)
	}
	allowGroup := func(t *testing.T, userID, groupID int64) {
		t.Helper()
		mustExec(t, `INSERT INTO user_allowed_groups (user_id, group_id) VALUES ($1,$2)`, userID, groupID)
	}
	setOverride := func(t *testing.T, userID, groupID int64, rate any, rpm any) {
		t.Helper()
		mustExec(t, `INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, rpm_override) VALUES ($1,$2,$3,$4)`,
			userID, groupID, rate, rpm)
	}
	type siteMsg struct {
		category, title, content, sourceType, sourceID string
		senderID                                       sql.NullInt64
		readAt                                         sql.NullTime
		createdAt                                      time.Time
	}
	listMessages := func(t *testing.T, userID int64) []siteMsg {
		t.Helper()
		rows, err := tx.Query(`SELECT category, title, content, source_type, source_id, sender_user_id, read_at, created_at FROM site_messages WHERE user_id = $1 ORDER BY id`, userID)
		require.NoError(t, err)
		defer func() { require.NoError(t, rows.Close()) }()
		var out []siteMsg
		for rows.Next() {
			var m siteMsg
			require.NoError(t, rows.Scan(&m.category, &m.title, &m.content, &m.sourceType, &m.sourceID, &m.senderID, &m.readAt, &m.createdAt))
			out = append(out, m)
		}
		require.NoError(t, rows.Err())
		return out
	}
	totalMessages := func(t *testing.T) int {
		t.Helper()
		var n int
		require.NoError(t, tx.QueryRow(`SELECT count(*) FROM site_messages`).Scan(&n))
		return n
	}
	groupRate := func(t *testing.T, groupID int64) float64 {
		t.Helper()
		var r float64
		require.NoError(t, tx.QueryRow(`SELECT rate_multiplier FROM groups WHERE id = $1`, groupID).Scan(&r))
		return r
	}
	spaceRe := regexp.MustCompile(`[[:space:]]+`)
	mdEscape := func(value string) string {
		normalized := spaceRe.ReplaceAllString(value, " ")
		runes := []rune(normalized)
		if len(runes) > 120 {
			runes = runes[:120]
		}
		const specials = "\\`*_{}[]()#+-.!|<>~"
		var b strings.Builder
		for _, r := range runes {
			if strings.ContainsRune(specials, r) {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		return b.String()
	}

	preUser := newUser(t, nil)
	preGroup := newGroup(t, "grp-preinstall", nil)
	addKey(t, preUser, preGroup, "active", nil, false)

	migration254, err := FS.ReadFile("254_group_rate_site_messages.sql")
	require.NoError(t, err)
	_, err = tx.Exec(string(migration254))
	require.NoError(t, err)
	_, err = tx.Exec(string(migration254))
	require.NoError(t, err, "migration must remain safe to retry")
	require.Zero(t, totalMessages(t), "installing the migration must not send historical mail")
	var triggerCount int
	require.NoError(t, tx.QueryRow(`SELECT count(*) FROM pg_trigger t JOIN pg_class c ON c.oid = t.tgrelid JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = $1 AND t.tgname IN ('trg_groups_rate_site_messages','trg_user_group_rates_site_messages') AND NOT t.tgisinternal`, schema).Scan(&triggerCount))
	require.Equal(t, 2, triggerCount)

	t.Run("recipient_selection_filters", func(t *testing.T) {
		gid := newGroup(t, "grp-recipients", nil)
		past := time.Now().Add(-time.Hour)
		future := time.Now().Add(48 * time.Hour)

		activeKeyUser := newUser(t, nil)
		addKey(t, activeKeyUser, gid, "active", nil, false)
		subOnlyUser := newUser(t, nil)
		addSub(t, subOnlyUser, gid, "active", past, future, false)
		multiUser := newUser(t, nil)
		addKey(t, multiUser, gid, "active", nil, false)
		addKey(t, multiUser, gid, "active", nil, false)
		addSub(t, multiUser, gid, "active", past, future, false)
		noAccessUser := newUser(t, nil)
		expiredSubUser := newUser(t, nil)
		addSub(t, expiredSubUser, gid, "active", past.Add(-72*time.Hour), past, false)
		futureSubUser := newUser(t, nil)
		addSub(t, futureSubUser, gid, "active", future, future.Add(48*time.Hour), false)
		suspendedSubUser := newUser(t, nil)
		addSub(t, suspendedSubUser, gid, "suspended", past, future, false)
		deletedSubUser := newUser(t, nil)
		addSub(t, deletedSubUser, gid, "active", past, future, true)
		expiredKeyUser := newUser(t, nil)
		addKey(t, expiredKeyUser, gid, "active", &past, false)
		disabledKeyUser := newUser(t, nil)
		addKey(t, disabledKeyUser, gid, "disabled", nil, false)
		deletedKeyUser := newUser(t, nil)
		addKey(t, deletedKeyUser, gid, "active", nil, true)
		disabledUser := newUser(t, func(s *userSeed) { s.status = "disabled" })
		addKey(t, disabledUser, gid, "active", nil, false)
		deletedUser := newUser(t, func(s *userSeed) { s.deleted = true })
		addKey(t, deletedUser, gid, "active", nil, false)

		mustExec(t, `UPDATE groups SET rate_multiplier = 2 WHERE id = $1`, gid)

		require.Len(t, listMessages(t, activeKeyUser), 1)
		require.Len(t, listMessages(t, subOnlyUser), 1)
		require.Len(t, listMessages(t, multiUser), 1, "multiple keys + subscription must dedupe to one message")
		for _, uid := range []int64{noAccessUser, expiredSubUser, futureSubUser, suspendedSubUser, deletedSubUser, expiredKeyUser, disabledKeyUser, deletedKeyUser, disabledUser, deletedUser} {
			require.Empty(t, listMessages(t, uid), "user %d must not be notified", uid)
		}
	})

	t.Run("exclusive_and_restricted_visibility", func(t *testing.T) {
		exclusiveGroup := newGroup(t, "grp-exclusive", func(s *groupSeed) { s.exclusive = true })
		noAllow := newUser(t, nil)
		addKey(t, noAllow, exclusiveGroup, "active", nil, false)
		allowed := newUser(t, nil)
		addKey(t, allowed, exclusiveGroup, "active", nil, false)
		allowGroup(t, allowed, exclusiveGroup)
		mustExec(t, `UPDATE groups SET rate_multiplier = 2 WHERE id = $1`, exclusiveGroup)
		require.Empty(t, listMessages(t, noAllow))
		require.Len(t, listMessages(t, allowed), 1)

		publicGroup := newGroup(t, "grp-public", nil)
		restricted := newUser(t, func(s *userSeed) { s.restrict = true })
		addKey(t, restricted, publicGroup, "active", nil, false)
		restrictedAllowed := newUser(t, func(s *userSeed) { s.restrict = true })
		addKey(t, restrictedAllowed, publicGroup, "active", nil, false)
		allowGroup(t, restrictedAllowed, publicGroup)
		mustExec(t, `UPDATE groups SET rate_multiplier = 3 WHERE id = $1`, publicGroup)
		require.Empty(t, listMessages(t, restricted))
		require.Len(t, listMessages(t, restrictedAllowed), 1)

		subscriptionGroup := newGroup(t, "grp-subscription", func(s *groupSeed) {
			s.subscriptionType = "subscription"
			s.exclusive = true
		})
		subUser := newUser(t, nil)
		addSub(t, subUser, subscriptionGroup, "active", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour), false)
		mustExec(t, `UPDATE groups SET rate_multiplier = 4 WHERE id = $1`, subscriptionGroup)
		require.Len(t, listMessages(t, subUser), 1, "subscription group consumers bypass the standard allowlist")
	})

	t.Run("inactive_and_deleted_groups", func(t *testing.T) {
		disabledGroup := newGroup(t, "grp-disabled", func(s *groupSeed) { s.status = "disabled" })
		u1 := newUser(t, nil)
		addKey(t, u1, disabledGroup, "active", nil, false)
		mustExec(t, `UPDATE groups SET rate_multiplier = 2 WHERE id = $1`, disabledGroup)
		require.Empty(t, listMessages(t, u1))

		softDeleted := newGroup(t, "grp-softdeleted", nil)
		u2 := newUser(t, nil)
		addKey(t, u2, softDeleted, "active", nil, false)
		mustExec(t, `UPDATE groups SET rate_multiplier = 2, deleted_at = NOW() WHERE id = $1`, softDeleted)
		require.Empty(t, listMessages(t, u2))
	})

	t.Run("base_rate_change_message", func(t *testing.T) {
		gid := newGroup(t, "grp-base", nil)
		uid := newUser(t, nil)
		addKey(t, uid, gid, "active", nil, false)
		mustExec(t, `UPDATE groups SET rate_multiplier = 2 WHERE id = $1`, gid)
		msgs := listMessages(t, uid)
		require.Len(t, msgs, 1)
		m := msgs[0]
		require.Equal(t, "system", m.category)
		require.Equal(t, "分组倍率调整通知 / Group rate update", m.title)
		require.Equal(t, "group_rate_changed", m.sourceType)
		require.True(t, strings.HasPrefix(m.sourceID, fmt.Sprintf("group:%d:tx:", gid)), m.sourceID)
		require.False(t, m.senderID.Valid)
		require.False(t, m.readAt.Valid)
		require.False(t, m.createdAt.IsZero())
		require.Contains(t, m.content, "| 基础倍率（不含高峰加成） | 1× | 2× |")
		require.Contains(t, m.content, "| Base multiplier (before peak adjustment) | 1× | 2× |")
		require.Contains(t, m.content, "| 图片倍率 | 1×（跟随基础） | 2×（跟随基础） |")
		require.Contains(t, m.content, "| 视频倍率 | 1×（跟随基础） | 2×（跟随基础） |")
		require.Contains(t, m.content, "| Image multiplier | 1× (follows base) | 2× (follows base) |")
		require.Contains(t, m.content, "| Video multiplier | 1× (follows base) | 2× (follows base) |")
		require.Contains(t, m.content, "「"+mdEscape("grp-base")+"」")
		require.Contains(t, m.content, "“"+mdEscape("grp-base")+"”")
		require.Regexp(t, `调整时间：\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} UTC`, m.content)
		require.Regexp(t, `Updated at: \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} UTC`, m.content)
		require.Contains(t, m.content, "(/models)")
	})

	t.Run("override_and_consolidated_update", func(t *testing.T) {
		g1 := newGroup(t, "grp-override-skip", nil)
		plain1 := newUser(t, nil)
		addKey(t, plain1, g1, "active", nil, false)
		overridden := newUser(t, nil)
		addKey(t, overridden, g1, "active", nil, false)
		setOverride(t, overridden, g1, 0.5, nil)
		rpmOnly := newUser(t, nil)
		addKey(t, rpmOnly, g1, "active", nil, false)
		setOverride(t, rpmOnly, g1, nil, 50)

		mustExec(t, `UPDATE groups SET rate_multiplier = 2 WHERE id = $1`, g1)
		require.Len(t, listMessages(t, plain1), 1)
		overriddenMsgs := listMessages(t, overridden)
		require.Len(t, overriddenMsgs, 1, "fixed override suppresses default-only change; only the personal notice remains")
		require.Equal(t, "user_group_rate_changed", overriddenMsgs[0].sourceType)
		rpmMsgs := listMessages(t, rpmOnly)
		require.Len(t, rpmMsgs, 1, "NULL rate override still inherits and is notified")
		require.Contains(t, rpmMsgs[0].content, "| 基础倍率（不含高峰加成） | 1× | 2× |")

		g2 := newGroup(t, "grp-combined", nil)
		plain2 := newUser(t, nil)
		addKey(t, plain2, g2, "active", nil, false)
		overridden2 := newUser(t, nil)
		addKey(t, overridden2, g2, "active", nil, false)
		setOverride(t, overridden2, g2, 0.5, nil)

		mustExec(t, `UPDATE groups SET rate_multiplier = 2, image_rate_independent = true, image_rate_multiplier = 3, video_rate_independent = true, video_rate_multiplier = 4 WHERE id = $1`, g2)

		pm := listMessages(t, plain2)
		require.Len(t, pm, 1, "a single group save yields one consolidated mail")
		require.Contains(t, pm[0].content, "| 基础倍率（不含高峰加成） | 1× | 2× |")
		require.Contains(t, pm[0].content, "| 图片倍率 | 1×（跟随基础） | 3×（独立） |")
		require.Contains(t, pm[0].content, "| 视频倍率 | 1×（跟随基础） | 4×（独立） |")
		require.Contains(t, pm[0].content, "| Image multiplier | 1× (follows base) | 3× (independent) |")
		require.Contains(t, pm[0].content, "| Video multiplier | 1× (follows base) | 4× (independent) |")

		om := listMessages(t, overridden2)
		require.Len(t, om, 2)
		groupMsg := om[1]
		require.Equal(t, "group_rate_changed", groupMsg.sourceType)
		require.NotContains(t, groupMsg.content, "基础倍率（不含高峰加成）")
		require.NotContains(t, groupMsg.content, "| 2×", "default base price must not leak into an overridden user's notice")
		require.Contains(t, groupMsg.content, "| 图片倍率 | 0.5×（跟随基础） | 3×（独立） |")
		require.Contains(t, groupMsg.content, "| 视频倍率 | 0.5×（跟随基础） | 4×（独立） |")
	})

	t.Run("independent_mode_edges", func(t *testing.T) {
		disabledModes := newGroup(t, "grp-modes-off", nil)
		u1 := newUser(t, nil)
		addKey(t, u1, disabledModes, "active", nil, false)
		mustExec(t, `UPDATE groups SET image_rate_multiplier = 5, video_rate_multiplier = 6 WHERE id = $1`, disabledModes)
		require.Empty(t, listMessages(t, u1), "saved image/video values produce no mail while modes are disabled")

		enableSame := newGroup(t, "grp-enable-same", nil)
		u2 := newUser(t, nil)
		addKey(t, u2, enableSame, "active", nil, false)
		u2o := newUser(t, nil)
		addKey(t, u2o, enableSame, "active", nil, false)
		setOverride(t, u2o, enableSame, 0.5, nil)
		mustExec(t, `UPDATE groups SET image_rate_independent = true WHERE id = $1`, enableSame)
		m2 := listMessages(t, u2)
		require.Len(t, m2, 1)
		require.Contains(t, m2[0].content, "| 图片倍率 | 1×（跟随基础） | 1×（独立） |", "same numeric still records the mode change")
		m2o := listMessages(t, u2o)
		require.Len(t, m2o, 2)
		require.Contains(t, m2o[1].content, "| 图片倍率 | 0.5×（跟随基础） | 1×（独立） |")

		freeMedia := newGroup(t, "grp-free-media", func(s *groupSeed) {
			s.imageIndep = true
			s.imageRate = 3
		})
		u3 := newUser(t, nil)
		addKey(t, u3, freeMedia, "active", nil, false)
		mustExec(t, `UPDATE groups SET image_rate_multiplier = 0 WHERE id = $1`, freeMedia)
		m3 := listMessages(t, u3)
		require.Len(t, m3, 1)
		require.Contains(t, m3[0].content, "| 图片倍率 | 3×（独立） | 0×（独立） |")

		disableMode := newGroup(t, "grp-disable-mode", func(s *groupSeed) {
			s.imageIndep = true
			s.imageRate = 3
		})
		u4 := newUser(t, nil)
		addKey(t, u4, disableMode, "active", nil, false)
		u4o := newUser(t, nil)
		addKey(t, u4o, disableMode, "active", nil, false)
		setOverride(t, u4o, disableMode, 0.5, nil)
		mustExec(t, `UPDATE groups SET image_rate_independent = false WHERE id = $1`, disableMode)
		m4 := listMessages(t, u4)
		require.Len(t, m4, 1)
		require.Contains(t, m4[0].content, "| 图片倍率 | 3×（独立） | 1×（跟随基础） |")
		m4o := listMessages(t, u4o)
		require.Len(t, m4o, 2)
		require.Contains(t, m4o[1].content, "| 图片倍率 | 3×（独立） | 0.5×（跟随基础） |")
	})

	t.Run("personal_override_lifecycle", func(t *testing.T) {
		gid := newGroup(t, "grp-personal", nil)
		uid := newUser(t, nil)
		addKey(t, uid, gid, "active", nil, false)
		other := newUser(t, nil)
		addKey(t, other, gid, "active", nil, false)

		setOverride(t, uid, gid, 0.5, nil)
		msgs := listMessages(t, uid)
		require.Len(t, msgs, 1)
		require.Equal(t, "system", msgs[0].category)
		require.Equal(t, "专属倍率调整通知 / Personal rate update", msgs[0].title)
		require.Equal(t, "user_group_rate_changed", msgs[0].sourceType)
		require.True(t, strings.HasPrefix(msgs[0].sourceID, fmt.Sprintf("group:%d:tx:", gid)), msgs[0].sourceID)
		require.Contains(t, msgs[0].content, "| 基础倍率（不含高峰加成） | 1× | 0.5× |")
		require.Contains(t, msgs[0].content, "| 图片倍率 | 1×（跟随基础） | 0.5×（跟随基础） |")
		require.False(t, msgs[0].senderID.Valid)
		require.False(t, msgs[0].readAt.Valid)

		mustExec(t, `UPDATE user_group_rate_multipliers SET rate_multiplier = 0.7 WHERE user_id = $1 AND group_id = $2`, uid, gid)
		msgs = listMessages(t, uid)
		require.Len(t, msgs, 2)
		require.Contains(t, msgs[1].content, "| 基础倍率（不含高峰加成） | 0.5× | 0.7× |")

		mustExec(t, `UPDATE user_group_rate_multipliers SET rate_multiplier = NULL WHERE user_id = $1 AND group_id = $2`, uid, gid)
		msgs = listMessages(t, uid)
		require.Len(t, msgs, 3)
		require.Contains(t, msgs[2].content, "| 基础倍率（不含高峰加成） | 0.7× | 1× |")

		mustExec(t, `DELETE FROM user_group_rate_multipliers WHERE user_id = $1 AND group_id = $2`, uid, gid)
		require.Len(t, listMessages(t, uid), 3, "NULL -> delete must not notify")
		require.Empty(t, listMessages(t, other), "personal adjustments only notify the affected consumer")

		upsertUser := newUser(t, nil)
		addKey(t, upsertUser, gid, "active", nil, false)
		setOverride(t, upsertUser, gid, 0.5, nil)
		mustExec(t, `INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier) VALUES ($1,$2,0.9) ON CONFLICT (user_id, group_id) DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier`, upsertUser, gid)
		umsgs := listMessages(t, upsertUser)
		require.Len(t, umsgs, 2)
		require.Contains(t, umsgs[1].content, "| 基础倍率（不含高峰加成） | 0.5× | 0.9× |")
		mustExec(t, `DELETE FROM user_group_rate_multipliers WHERE user_id = $1 AND group_id = $2`, upsertUser, gid)
		umsgs = listMessages(t, upsertUser)
		require.Len(t, umsgs, 3)
		require.Contains(t, umsgs[2].content, "| 基础倍率（不含高峰加成） | 0.9× | 1× |", "delete falls back to the group default")

		sameAsDefault := newUser(t, nil)
		addKey(t, sameAsDefault, gid, "active", nil, false)
		setOverride(t, sameAsDefault, gid, 1.0, nil)
		require.Empty(t, listMessages(t, sameAsDefault), "no effective change must not notify")

		rpmOnlyUser := newUser(t, nil)
		addKey(t, rpmOnlyUser, gid, "active", nil, false)
		setOverride(t, rpmOnlyUser, gid, nil, 40)
		mustExec(t, `UPDATE user_group_rate_multipliers SET rpm_override = 90 WHERE user_id = $1 AND group_id = $2`, rpmOnlyUser, gid)
		require.Empty(t, listMessages(t, rpmOnlyUser), "RPM-only rows and updates must not notify")

		nonConsumer := newUser(t, nil)
		setOverride(t, nonConsumer, gid, 0.5, nil)
		require.Empty(t, listMessages(t, nonConsumer), "non-consumers are not notified")

		goneGroup := newGroup(t, "grp-gone", nil)
		goneUser := newUser(t, nil)
		addKey(t, goneUser, goneGroup, "active", nil, false)
		setOverride(t, goneUser, goneGroup, 0.5, nil)
		mustExec(t, `DELETE FROM groups WHERE id = $1`, goneGroup)
		require.Len(t, listMessages(t, goneUser), 1, "cascade cleanup on parent delete must not send a second notice")

		inactiveGroup := newGroup(t, "grp-inactive-personal", func(s *groupSeed) { s.status = "disabled" })
		inactiveUser := newUser(t, nil)
		addKey(t, inactiveUser, inactiveGroup, "active", nil, false)
		setOverride(t, inactiveUser, inactiveGroup, 0.5, nil)
		require.Empty(t, listMessages(t, inactiveUser))
	})

	t.Run("noop_and_unrelated_updates", func(t *testing.T) {
		gid := newGroup(t, "grp-noop", nil)
		uid := newUser(t, nil)
		addKey(t, uid, gid, "active", nil, false)
		mustExec(t, `UPDATE groups SET rate_multiplier = 1 WHERE id = $1`, gid)
		mustExec(t, `UPDATE groups SET rate_multiplier = 1.00001 WHERE id = $1`, gid)
		mustExec(t, `UPDATE groups SET name = 'grp-renamed', description = 'new description' WHERE id = $1`, gid)
		mustExec(t, `UPDATE groups SET peak_rate_enabled = true, peak_start = '09:00', peak_end = '11:00', peak_rate_multiplier = 2 WHERE id = $1`, gid)
		mustExec(t, `UPDATE groups SET model_pricing = '{"gpt-x":{"input":1}}'::jsonb WHERE id = $1`, gid)
		require.Empty(t, listMessages(t, uid), "same-value saves, 4-decimal rounding no-ops, and unrelated column changes must not notify")

		freshGroup := newGroup(t, "grp-fresh", nil)
		freshUser := newUser(t, nil)
		addKey(t, freshUser, freshGroup, "active", nil, false)
		require.Empty(t, listMessages(t, freshUser), "new group inserts must not notify")
	})

	t.Run("transactional_atomicity", func(t *testing.T) {
		gid := newGroup(t, "grp-atomic", nil)
		uid := newUser(t, nil)
		addKey(t, uid, gid, "active", nil, false)

		mustExec(t, `SAVEPOINT grsm_sp1`)
		mustExec(t, `UPDATE groups SET rate_multiplier = 3 WHERE id = $1`, gid)
		require.Len(t, listMessages(t, uid), 1)
		mustExec(t, `ROLLBACK TO SAVEPOINT grsm_sp1`)
		require.Empty(t, listMessages(t, uid))
		require.Equal(t, 1.0, groupRate(t, gid))

		mustExec(t, `CREATE FUNCTION grsm_block_site_message() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'blocked site message insert'; END; $$`)
		mustExec(t, `CREATE TRIGGER grsm_block_site_message BEFORE INSERT ON site_messages FOR EACH ROW EXECUTE FUNCTION grsm_block_site_message()`)
		mustExec(t, `SAVEPOINT grsm_sp2`)
		_, err := tx.Exec(`UPDATE groups SET rate_multiplier = 5 WHERE id = $1`, gid)
		require.Error(t, err, "site message insert failures must not be swallowed")
		require.Contains(t, err.Error(), "blocked site message insert")
		mustExec(t, `ROLLBACK TO SAVEPOINT grsm_sp2`)
		require.Equal(t, 1.0, groupRate(t, gid), "failed notification rolls back the rate update")
		require.Empty(t, listMessages(t, uid))
		mustExec(t, `DROP TRIGGER grsm_block_site_message ON site_messages`)
		mustExec(t, `DROP FUNCTION grsm_block_site_message()`)
	})

	t.Run("hostile_group_name_escaping", func(t *testing.T) {
		name := "evil | [`x`](http://h) <script>\n{z} ~! done\\"
		gid := newGroup(t, name, nil)
		uid := newUser(t, nil)
		mustExec(t, `INSERT INTO api_keys (user_id, key, group_id) VALUES ($1,'sk-secret-do-not-leak-99',$2)`, uid, gid)
		mustExec(t, `UPDATE groups SET rate_multiplier = 2 WHERE id = $1`, gid)
		msgs := listMessages(t, uid)
		require.Len(t, msgs, 1)
		content := msgs[0].content
		escaped := mdEscape(name)
		require.Contains(t, content, "「"+escaped+"」")
		require.Contains(t, content, "“"+escaped+"”")
		require.NotContains(t, content, "sk-secret-do-not-leak-99")
		require.NotContains(t, content, "grsm-user-")
		require.NotContains(t, content, "<script>")
		require.LessOrEqual(t, utf8.RuneCountInString(escaped), 240)
		require.LessOrEqual(t, utf8.RuneCountInString(content), 5000)
	})
}
