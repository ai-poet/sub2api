//go:build unit

package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/stretchr/testify/require"
)

// ptUsers 可变的用户表桩（模拟降级、禁用、改密等变化）。
type ptUsers struct {
	mu    sync.Mutex
	users map[int64]*service.User
}

func (u *ptUsers) GetByID(_ context.Context, id int64) (*service.User, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	user, ok := u.users[id]
	if !ok {
		return nil, service.ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (u *ptUsers) update(id int64, fn func(*service.User)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fn(u.users[id])
}

type ptFixture struct {
	svc      *service.PersonalTokenService
	repo     *testutil.MemoryPersonalTokenRepo
	users    *ptUsers
	settings *testutil.StaticPersonalTokenSettings
}

const ptPassword = "correct horse battery"

func newPTFixture(t *testing.T) *ptFixture {
	t.Helper()
	mk := func(id int64, email, role, status string) *service.User {
		u := &service.User{ID: id, Email: email, Role: role, Status: status, TokenVersion: 7, TokenVersionResolved: true}
		require.NoError(t, u.SetPassword(ptPassword))
		return u
	}
	users := &ptUsers{users: map[int64]*service.User{
		1: mk(1, "admin@example.com", service.RoleAdmin, service.StatusActive),
		2: mk(2, "ops@example.com", service.RoleOperator, service.StatusActive),
		3: mk(3, "user@example.com", service.RoleUser, service.StatusActive),
		4: mk(4, "ops-off@example.com", service.RoleOperator, service.StatusDisabled),
	}}
	repo := testutil.NewMemoryPersonalTokenRepo()
	settings := testutil.NewStaticPersonalTokenSettings(true)
	return &ptFixture{svc: service.NewPersonalTokenService(repo, users, settings), repo: repo, users: users, settings: settings}
}

func (f *ptFixture) generate(t *testing.T, userID int64, days int) *service.PersonalTokenIssued {
	t.Helper()
	issued, err := f.svc.Generate(context.Background(), service.PersonalTokenGenerateInput{
		UserID: userID, Password: ptPassword, ExpiresInDays: days, ClientIP: "198.51.100.7",
	})
	require.NoError(t, err)
	return issued
}

func TestPersonalTokenGenerate(t *testing.T) {
	ctx := context.Background()

	t.Run("stores_only_hash_and_returns_plaintext_once", func(t *testing.T) {
		f := newPTFixture(t)
		issued := f.generate(t, 2, 90)
		require.True(t, service.IsPersonalTokenFormat(issued.Token), issued.Token)
		require.True(t, strings.HasPrefix(issued.Token, "pat-"))

		stored, err := f.repo.GetByUserID(ctx, 2)
		require.NoError(t, err)
		require.Equal(t, service.HashPersonalToken(issued.Token), stored.TokenHash)
		require.NotEqual(t, issued.Token, stored.TokenHash)
		require.NotContains(t, stored.TokenHint, issued.Token[len("pat-")+6:len(issued.Token)-4], "hint must not reveal the secret body")
		require.Equal(t, int64(7), stored.UserTokenVersion)
		require.Equal(t, "198.51.100.7", stored.CreatedIP)
		require.NotNil(t, stored.ExpiresAt)
		require.WithinDuration(t, time.Now().Add(90*24*time.Hour), *stored.ExpiresAt, time.Minute)
	})

	t.Run("zero_days_never_expires", func(t *testing.T) {
		f := newPTFixture(t)
		f.generate(t, 2, 0)
		stored, err := f.repo.GetByUserID(ctx, 2)
		require.NoError(t, err)
		require.Nil(t, stored.ExpiresAt)
	})

	t.Run("regenerate_overwrites_and_kills_old_token", func(t *testing.T) {
		f := newPTFixture(t)
		first := f.generate(t, 2, 0)
		second := f.generate(t, 2, 30)
		require.NotEqual(t, first.Token, second.Token)

		_, _, err := f.svc.Authenticate(ctx, first.Token, "ip")
		require.ErrorIs(t, err, service.ErrPersonalTokenAuthInvalid)
		user, _, err := f.svc.Authenticate(ctx, second.Token, "ip")
		require.NoError(t, err)
		require.Equal(t, int64(2), user.ID)

		all, err := f.repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, all, 1, "one token per user")
	})

	rejects := []struct {
		name    string
		userID  int64
		pass    string
		days    int
		disable bool
		want    error
	}{
		{name: "feature_disabled", userID: 2, pass: ptPassword, days: 30, disable: true, want: service.ErrPersonalTokenDisabled},
		{name: "admin_not_eligible", userID: 1, pass: ptPassword, days: 30, want: service.ErrPersonalTokenNotEligible},
		{name: "user_not_eligible", userID: 3, pass: ptPassword, days: 30, want: service.ErrPersonalTokenNotEligible},
		{name: "disabled_operator", userID: 4, pass: ptPassword, days: 30, want: service.ErrUserNotActive},
		{name: "missing_password", userID: 2, pass: "", days: 30, want: service.ErrPasswordRequired},
		{name: "wrong_password", userID: 2, pass: "nope", days: 30, want: service.ErrPasswordIncorrect},
		{name: "invalid_expiry", userID: 2, pass: ptPassword, days: 7, want: service.ErrPersonalTokenExpiryInvalid},
		{name: "negative_expiry", userID: 2, pass: ptPassword, days: -1, want: service.ErrPersonalTokenExpiryInvalid},
	}
	for _, tc := range rejects {
		t.Run(tc.name, func(t *testing.T) {
			f := newPTFixture(t)
			if tc.disable {
				f.settings.Set(false)
			}
			issued, err := f.svc.Generate(ctx, service.PersonalTokenGenerateInput{UserID: tc.userID, Password: tc.pass, ExpiresInDays: tc.days})
			require.ErrorIs(t, err, tc.want)
			require.Nil(t, issued)
			all, listErr := f.repo.List(ctx)
			require.NoError(t, listErr)
			require.Empty(t, all, "nothing may be stored on rejection")
		})
	}
}

func TestPersonalTokenAuthenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("valid_token_returns_operator", func(t *testing.T) {
		f := newPTFixture(t)
		issued := f.generate(t, 2, 30)
		user, token, err := f.svc.Authenticate(ctx, issued.Token, "203.0.113.9")
		require.NoError(t, err)
		require.Equal(t, service.RoleOperator, user.Role)
		require.Equal(t, int64(2), token.UserID)

		stored, err := f.repo.GetByUserID(ctx, 2)
		require.NoError(t, err)
		require.NotNil(t, stored.LastUsedAt)
		require.Equal(t, "203.0.113.9", stored.LastUsedIP)
	})

	t.Run("last_used_is_throttled", func(t *testing.T) {
		f := newPTFixture(t)
		issued := f.generate(t, 2, 30)
		for i := 0; i < 5; i++ {
			_, _, err := f.svc.Authenticate(ctx, issued.Token, "203.0.113.9")
			require.NoError(t, err)
		}
		require.Equal(t, 1, f.repo.Touches)
	})

	cases := []struct {
		name   string
		mutate func(f *ptFixture, raw *string)
		want   error
	}{
		{name: "garbage", mutate: func(_ *ptFixture, raw *string) { *raw = "pat-not-a-token" }, want: service.ErrPersonalTokenAuthInvalid},
		{name: "uppercase_hex_rejected", mutate: func(_ *ptFixture, raw *string) { *raw = strings.ToUpper(*raw) }, want: service.ErrPersonalTokenAuthInvalid},
		{name: "unknown_token", mutate: func(_ *ptFixture, raw *string) { *raw = "pat-" + strings.Repeat("ab", 32) }, want: service.ErrPersonalTokenAuthInvalid},
		{name: "feature_switched_off", mutate: func(f *ptFixture, _ *string) { f.settings.Set(false) }, want: service.ErrPersonalTokenAuthDisabled},
		{name: "revoked_by_owner", mutate: func(f *ptFixture, _ *string) { _ = f.svc.Revoke(context.Background(), 2) }, want: service.ErrPersonalTokenAuthInvalid},
		{name: "demoted_to_user", mutate: func(f *ptFixture, _ *string) {
			f.users.update(2, func(u *service.User) { u.Role = service.RoleUser })
		}, want: service.ErrPersonalTokenAuthNotEligible},
		// 令牌永远不可能以 admin 身份通过
		{name: "promoted_to_admin", mutate: func(f *ptFixture, _ *string) {
			f.users.update(2, func(u *service.User) { u.Role = service.RoleAdmin })
		}, want: service.ErrPersonalTokenAuthNotEligible},
		{name: "user_disabled", mutate: func(f *ptFixture, _ *string) {
			f.users.update(2, func(u *service.User) { u.Status = service.StatusDisabled })
		}, want: service.ErrPersonalTokenAuthUserInvalid},
		{name: "password_or_email_changed", mutate: func(f *ptFixture, _ *string) {
			f.users.update(2, func(u *service.User) { u.TokenVersion++ })
		}, want: service.ErrPersonalTokenAuthRevoked},
		{name: "user_deleted", mutate: func(f *ptFixture, _ *string) {
			f.users.update(2, func(u *service.User) {})
			f.users.mu.Lock()
			delete(f.users.users, 2)
			f.users.mu.Unlock()
		}, want: service.ErrPersonalTokenAuthInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newPTFixture(t)
			raw := f.generate(t, 2, 30).Token
			tc.mutate(f, &raw)
			user, token, err := f.svc.Authenticate(ctx, raw, "ip")
			require.ErrorIs(t, err, tc.want)
			require.Nil(t, user)
			require.Nil(t, token)
		})
	}

	t.Run("expired", func(t *testing.T) {
		f := newPTFixture(t)
		raw := f.generate(t, 2, 30).Token
		stored, err := f.repo.GetByUserID(ctx, 2)
		require.NoError(t, err)
		past := time.Now().Add(-time.Second)
		stored.ExpiresAt = &past
		_, err = f.repo.Upsert(ctx, stored)
		require.NoError(t, err)
		_, _, err = f.svc.Authenticate(ctx, raw, "ip")
		require.ErrorIs(t, err, service.ErrPersonalTokenAuthExpired)
	})

	t.Run("auth_errors_are_401", func(t *testing.T) {
		for _, err := range []error{
			service.ErrPersonalTokenAuthInvalid, service.ErrPersonalTokenAuthExpired, service.ErrPersonalTokenAuthRevoked,
			service.ErrPersonalTokenAuthDisabled, service.ErrPersonalTokenAuthNotEligible, service.ErrPersonalTokenAuthUserInvalid,
		} {
			var appErr interface{ Error() string }
			require.True(t, errors.As(err, &appErr))
			require.Contains(t, err.Error(), "code=401")
		}
	})
}

func TestPersonalTokenRevokeAndList(t *testing.T) {
	ctx := context.Background()
	f := newPTFixture(t)

	require.ErrorIs(t, f.svc.Revoke(ctx, 2), service.ErrPersonalTokenNotFound)
	require.NoError(t, f.svc.RevokeForUser(ctx, 2), "admin / hook revoke is idempotent")

	f.generate(t, 2, 0)
	status, err := f.svc.Status(ctx, 2)
	require.NoError(t, err)
	require.True(t, status.FeatureEnabled)
	require.True(t, status.Eligible)
	require.NotNil(t, status.Token)
	require.False(t, status.Expired)

	userStatus, err := f.svc.Status(ctx, 3)
	require.NoError(t, err)
	require.False(t, userStatus.Eligible)
	require.Nil(t, userStatus.Token)

	f.users.update(2, func(u *service.User) { u.TokenVersion++ })
	items, err := f.svc.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, service.PersonalTokenStateRevoked, items[0].State)
	require.Equal(t, "ops@example.com", items[0].UserEmail)

	require.NoError(t, f.svc.Revoke(ctx, 2))
	items, err = f.svc.ListAll(ctx)
	require.NoError(t, err)
	require.Empty(t, items)
}
