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

func newDesktopLoginCodeTestStore(t *testing.T) (*miniredis.Miniredis, service.DesktopLoginCodeStore) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, NewDesktopLoginCodeStore(rdb)
}

func desktopLoginTestRecord() *service.DesktopLoginCode {
	return &service.DesktopLoginCode{
		UserID:        41,
		CodeChallenge: strings.Repeat("A", 43),
		APIKey:        "sk-general",
		CodexAPIKey:   "sk-codex",
		CreatedAt:     time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
	}
}

var desktopLoginCodeFormat = regexp.MustCompile(`^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}$`)

func TestDesktopLoginCodeStore_StoresUnderHashAndConsumesOnce(t *testing.T) {
	mr, store := newDesktopLoginCodeTestStore(t)
	record := desktopLoginTestRecord()

	code, err := store.Store(context.Background(), record, 10*time.Minute)
	require.NoError(t, err)
	require.Regexp(t, desktopLoginCodeFormat, code)

	// 键只含规范化码的 SHA-256，码本身不出现在 Redis 里
	keys := mr.Keys()
	require.Len(t, keys, 1)
	normalized := strings.ReplaceAll(code, "-", "")
	sum := sha256.Sum256([]byte(normalized))
	require.Equal(t, "desktop_login_code:"+hex.EncodeToString(sum[:]), keys[0])
	require.NotContains(t, keys[0], normalized)
	stored, err := mr.Get(keys[0])
	require.NoError(t, err)
	require.NotContains(t, stored, normalized)
	require.Equal(t, 10*time.Minute, mr.TTL(keys[0]))

	got, err := store.Consume(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, record, got)
	require.Empty(t, mr.Keys(), "consume deletes the record")

	_, err = store.Consume(context.Background(), code)
	require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
}

// 用户手抄的码：小写、空格、没有连字符或多余分隔符都应认得。
func TestDesktopLoginCodeStore_ConsumeNormalizesTypedCode(t *testing.T) {
	for name, mangle := range map[string]func(string) string{
		"lowercase":    strings.ToLower,
		"no_dash":      func(c string) string { return strings.ReplaceAll(c, "-", "") },
		"spaces":       func(c string) string { return "  " + strings.ReplaceAll(c, "-", " ") + "\n" },
		"extra_dashes": func(c string) string { return strings.Join(strings.Split(strings.ReplaceAll(c, "-", ""), ""), "-") },
	} {
		t.Run(name, func(t *testing.T) {
			_, store := newDesktopLoginCodeTestStore(t)
			code, err := store.Store(context.Background(), desktopLoginTestRecord(), time.Minute)
			require.NoError(t, err)

			got, err := store.Consume(context.Background(), mangle(code))
			require.NoError(t, err)
			require.Equal(t, int64(41), got.UserID)
		})
	}
}

func TestDesktopLoginCodeStore_ExpiredOrMalformedIsInvalid(t *testing.T) {
	mr, store := newDesktopLoginCodeTestStore(t)
	code, err := store.Store(context.Background(), desktopLoginTestRecord(), time.Minute)
	require.NoError(t, err)

	for _, bad := range []string{
		"",
		"ABCD-EFG",   // 7 chars
		"ABCD-EFGHJ", // 9 chars
		"ABCD-EFG0",  // 0 is not in the alphabet
		"ABCD-EFGO",  // O is not in the alphabet
		"ABCD-EFG1",  // 1 is not in the alphabet
		strings.Repeat("A", 65),
	} {
		_, err := store.Consume(context.Background(), bad)
		require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid, "code=%q", bad)
	}

	mr.FastForward(2 * time.Minute)
	_, err = store.Consume(context.Background(), code)
	require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
}

func TestDesktopLoginCodeStore_CorruptPayloadIsInvalid(t *testing.T) {
	mr, store := newDesktopLoginCodeTestStore(t)
	require.NoError(t, mr.Set(desktopLoginCodeKey("ABCDEFGH"), "{not json"))

	_, err := store.Consume(context.Background(), "ABCD-EFGH")
	require.ErrorIs(t, err, service.ErrDesktopLoginCodeInvalid)
	require.Empty(t, mr.Keys(), "a corrupt record is consumed too")
}

func TestDesktopLoginCodeStore_RejectsInvalidInput(t *testing.T) {
	_, store := newDesktopLoginCodeTestStore(t)
	_, err := store.Store(context.Background(), nil, time.Minute)
	require.Error(t, err)
	_, err = store.Store(context.Background(), desktopLoginTestRecord(), 0)
	require.Error(t, err)
}

func TestRandomDesktopLoginCodeUsesOnlyTheAlphabet(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 2000; i++ {
		code, err := randomDesktopLoginCode()
		require.NoError(t, err)
		require.Len(t, code, desktopLoginCodeLength)
		normalized, ok := normalizeDesktopLoginCode(formatDesktopLoginCode(code))
		require.True(t, ok)
		require.Equal(t, code, normalized)
		seen[code] = struct{}{}
	}
	require.Greater(t, len(seen), 1990, "codes are random")
}
