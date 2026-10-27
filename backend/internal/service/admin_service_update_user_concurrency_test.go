//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newConcurrencyAdminService(user *User) (*adminServiceImpl, *userRepoStub) {
	repo := &userRepoStub{user: user}
	cfg := &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyDefaultConcurrency: "9",
	}}, cfg)
	return &adminServiceImpl{
		userRepo:       repo,
		settingService: settings,
		redeemCodeRepo: &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}},
	}, repo
}

func TestAdminService_UpdateUser_ConcurrencyOverrideAndRestore(t *testing.T) {
	t.Run("explicit value becomes a custom override", func(t *testing.T) {
		svc, repo := newConcurrencyAdminService(&User{ID: 1, Role: RoleUser, Status: StatusActive, Concurrency: 9, ConcurrencyIsDefault: true})
		value := 3
		user, err := svc.UpdateUser(context.Background(), 1, &UpdateUserInput{Concurrency: &value})
		require.NoError(t, err)
		require.Equal(t, 3, user.Concurrency)
		require.False(t, user.ConcurrencyIsDefault)
		require.True(t, repo.updatedFields[0].Concurrency)
		require.True(t, repo.updatedFields[0].ConcurrencyIsDefault)
	})

	t.Run("restore uses current global default", func(t *testing.T) {
		svc, _ := newConcurrencyAdminService(&User{ID: 1, Role: RoleUser, Status: StatusActive, Concurrency: 3})
		useDefault := true
		user, err := svc.UpdateUser(context.Background(), 1, &UpdateUserInput{UseDefaultConcurrency: &useDefault})
		require.NoError(t, err)
		require.Equal(t, 9, user.Concurrency)
		require.True(t, user.ConcurrencyIsDefault)
	})

	t.Run("explicit zero remains a custom override", func(t *testing.T) {
		svc, _ := newConcurrencyAdminService(&User{ID: 1, Role: RoleUser, Status: StatusActive, Concurrency: 9, ConcurrencyIsDefault: true})
		zero := 0
		user, err := svc.UpdateUser(context.Background(), 1, &UpdateUserInput{Concurrency: &zero})
		require.NoError(t, err)
		require.Equal(t, 0, user.Concurrency)
		require.False(t, user.ConcurrencyIsDefault)
	})
}

func TestAdminService_UpdateUser_LegacyUnchangedConcurrencyKeepsDefault(t *testing.T) {
	svc, repo := newConcurrencyAdminService(&User{ID: 1, Role: RoleUser, Status: StatusActive, Concurrency: 9, ConcurrencyIsDefault: true})
	value := 9
	notes := "only notes changed"
	user, err := svc.UpdateUser(context.Background(), 1, &UpdateUserInput{Notes: &notes, Concurrency: &value})
	require.NoError(t, err)
	require.True(t, user.ConcurrencyIsDefault)
	require.False(t, repo.updatedFields[0].Concurrency)
	require.False(t, repo.updatedFields[0].ConcurrencyIsDefault)
	require.True(t, repo.updatedFields[0].Notes)
}

func TestAdminService_UpdateUser_RejectsAmbiguousDefaultRequest(t *testing.T) {
	value := 4
	useDefault := true
	svc, _ := newConcurrencyAdminService(&User{ID: 1, Role: RoleUser, Status: StatusActive})
	_, err := svc.UpdateUser(context.Background(), 1, &UpdateUserInput{
		Concurrency: &value, UseDefaultConcurrency: &useDefault,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be combined")

	useDefault = false
	_, err = svc.UpdateUser(context.Background(), 1, &UpdateUserInput{UseDefaultConcurrency: &useDefault})
	require.Error(t, err)
	require.Contains(t, err.Error(), "concurrency is required")
}
