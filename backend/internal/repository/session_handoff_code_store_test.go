package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newSessionHandoffCodeTestStore(t *testing.T) (*miniredis.Miniredis, service.SessionHandoffCodeStore) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, NewSessionHandoffCodeStore(rdb)
}

var sessionHandoffCodeFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func TestSessionHandoffCodeStore_StoresUnderHashAndConsumesOnce(t *testing.T) {
	mr, store := newSessionHandoffCodeTestStore(t)
	record := &service.SessionHandoffCode{
		UserID:        41,
		CodeChallenge: strings.Repeat("A", 43),
		TargetOrigin:  "https://cdn.cheaprouter.cc",
		CreatedAt:     time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
	}

	code, err := store.Store(context.Background(), record, 2*time.Minute)
	require.NoError(t, err)
	require.Regexp(t, sessionHandoffCodeFormat, code)

	sum := sha256.Sum256([]byte(code))
	key := sessionHandoffCodePrefix + hex.EncodeToString(sum[:])
	require.True(t, mr.Exists(key), "stored under the code's SHA-256")
	require.Equal(t, 2*time.Minute, mr.TTL(key))
	for _, k := range mr.Keys() {
		require.NotContains(t, k, code, "the plain code never appears in a key")
	}

	got, err := store.Consume(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, record.UserID, got.UserID)
	require.Equal(t, record.TargetOrigin, got.TargetOrigin)
	require.Equal(t, record.CodeChallenge, got.CodeChallenge)
	require.True(t, record.CreatedAt.Equal(got.CreatedAt))
	require.False(t, mr.Exists(key))

	_, err = store.Consume(context.Background(), code)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid)
}

func TestSessionHandoffCodeStore_RejectsMalformedAndExpiredCodes(t *testing.T) {
	mr, store := newSessionHandoffCodeTestStore(t)
	for _, code := range []string{"", "short", strings.Repeat("A", 42), strings.Repeat("A", 44), strings.Repeat("+", 43)} {
		_, err := store.Consume(context.Background(), code)
		require.ErrorIsf(t, err, service.ErrSessionHandoffCodeInvalid, "code %q", code)
	}

	code, err := store.Store(context.Background(), &service.SessionHandoffCode{UserID: 1, CreatedAt: time.Now()}, time.Minute)
	require.NoError(t, err)
	mr.FastForward(2 * time.Minute)
	_, err = store.Consume(context.Background(), code)
	require.ErrorIs(t, err, service.ErrSessionHandoffCodeInvalid)

	_, err = store.Store(context.Background(), nil, time.Minute)
	require.Error(t, err)
}
