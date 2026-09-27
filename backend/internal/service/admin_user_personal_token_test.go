//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type personalTokenRevokerSpy struct {
	userIDs []int64
}

func (s *personalTokenRevokerSpy) RevokeForUser(_ context.Context, userID int64) error {
	s.userIDs = append(s.userIDs, userID)
	return nil
}

// 角色离开 operator 时吊销其个人令牌，防止再次提升后旧令牌"复活"（fork 本地）。
func TestAdminService_UpdateUser_RevokesPersonalTokenWhenLeavingOperator(t *testing.T) {
	cases := []struct {
		name       string
		from, to   string
		wantRevoke bool
	}{
		{name: "operator_to_user", from: RoleOperator, to: RoleUser, wantRevoke: true},
		{name: "operator_to_admin", from: RoleOperator, to: RoleAdmin, wantRevoke: true},
		{name: "user_to_operator", from: RoleUser, to: RoleOperator, wantRevoke: false},
		{name: "operator_unchanged", from: RoleOperator, to: RoleOperator, wantRevoke: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := &userRepoStub{user: &User{ID: 42, Email: "ops@example.com", Role: tc.from}}
			repo := &rpmUserRepoStub{userRepoStub: base}
			spy := &personalTokenRevokerSpy{}
			svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}
			svc.SetPersonalTokenRevoker(spy)

			updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: tc.to})
			require.NoError(t, err)
			require.Equal(t, tc.to, updated.Role)
			if tc.wantRevoke {
				require.Equal(t, []int64{42}, spy.userIDs)
			} else {
				require.Empty(t, spy.userIDs)
			}
		})
	}
}

func TestAdminService_UpdateUser_WithoutRevokerStillWorks(t *testing.T) {
	base := &userRepoStub{user: &User{ID: 42, Email: "ops@example.com", Role: RoleOperator}}
	svc := &adminServiceImpl{userRepo: &rpmUserRepoStub{userRepoStub: base}, redeemCodeRepo: &redeemRepoStub{}}
	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: RoleUser})
	require.NoError(t, err)
	require.Equal(t, RoleUser, updated.Role)
}
