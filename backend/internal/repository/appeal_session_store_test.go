package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newAppealSessionTestStore(t *testing.T) (*miniredis.Miniredis, service.AppealSessionStore) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, NewAppealSessionStore(rdb)
}

func TestAppealSessionStore_SavesUnderHashOnly(t *testing.T) {
	mr, store := newAppealSessionTestStore(t)
	ctx := context.Background()
	token := "apl_secret-token-value"
	hash := service.AppealTokenHash(token)

	require.NoError(t, store.Save(ctx, hash, &service.AppealSessionRecord{UserID: 7, Email: "u@example.com", IssuedAt: time.Now()}, 2*time.Hour))

	for _, key := range mr.Keys() {
		require.NotContains(t, key, "secret-token-value", "plaintext token never reaches redis keys")
		value, _ := mr.Get(key)
		require.NotContains(t, value, "secret-token-value", "plaintext token never reaches redis values")
	}
	require.True(t, mr.Exists("appeal_session:"+hash))
	require.True(t, mr.Exists("appeal_session_user:7"))
	require.Equal(t, 2*time.Hour, mr.TTL("appeal_session:"+hash))

	record, ttl, err := store.Load(ctx, hash)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, int64(7), record.UserID)
	require.Greater(t, ttl, time.Duration(0))
}

func TestAppealSessionStore_NewSessionReplacesOld(t *testing.T) {
	mr, store := newAppealSessionTestStore(t)
	ctx := context.Background()
	oldHash := service.AppealTokenHash("apl_old")
	newHash := service.AppealTokenHash("apl_new")

	require.NoError(t, store.Save(ctx, oldHash, &service.AppealSessionRecord{UserID: 7}, time.Hour))
	require.NoError(t, store.Save(ctx, newHash, &service.AppealSessionRecord{UserID: 7}, time.Hour))

	require.False(t, mr.Exists("appeal_session:"+oldHash))
	record, _, err := store.Load(ctx, oldHash)
	require.NoError(t, err)
	require.Nil(t, record)
	record, _, err = store.Load(ctx, newHash)
	require.NoError(t, err)
	require.NotNil(t, record)
}

func TestAppealSessionStore_RevokeAndRevokeUser(t *testing.T) {
	mr, store := newAppealSessionTestStore(t)
	ctx := context.Background()
	hash := service.AppealTokenHash("apl_x")

	require.NoError(t, store.Save(ctx, hash, &service.AppealSessionRecord{UserID: 7}, time.Hour))
	require.NoError(t, store.Revoke(ctx, hash))
	require.False(t, mr.Exists("appeal_session:"+hash))
	require.False(t, mr.Exists("appeal_session_user:7"))

	require.NoError(t, store.Save(ctx, hash, &service.AppealSessionRecord{UserID: 7}, time.Hour))
	require.NoError(t, store.RevokeUser(ctx, 7))
	record, _, err := store.Load(ctx, hash)
	require.NoError(t, err)
	require.Nil(t, record)
	require.Empty(t, mr.Keys())
}

func TestAppealSessionStore_ExpiresAndRejectsGarbage(t *testing.T) {
	mr, store := newAppealSessionTestStore(t)
	ctx := context.Background()
	hash := service.AppealTokenHash("apl_y")

	require.NoError(t, store.Save(ctx, hash, &service.AppealSessionRecord{UserID: 8}, time.Minute))
	mr.FastForward(2 * time.Minute)
	record, _, err := store.Load(ctx, hash)
	require.NoError(t, err)
	require.Nil(t, record)

	require.NoError(t, mr.Set("appeal_session:"+strings.Repeat("a", 64), "not json"))
	record, _, err = store.Load(ctx, strings.Repeat("a", 64))
	require.NoError(t, err)
	require.Nil(t, record)

	require.Error(t, store.Save(ctx, hash, nil, time.Minute))
}
