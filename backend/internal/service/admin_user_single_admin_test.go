//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// singleAdminUserRepo 在通用桩之上补一个按角色过滤的 ListWithFilters，用来模拟"系统里已有哪些管理员"。
type singleAdminUserRepo struct {
	userRepoStub
	admins []User
}

func (r *singleAdminUserRepo) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	if filters.Role != RoleAdmin {
		panic("single-admin guard must only query admins")
	}
	out := append([]User(nil), r.admins...)
	if params.PageSize > 0 && len(out) > params.PageSize {
		out = out[:params.PageSize]
	}
	return out, &pagination.PaginationResult{}, nil
}

func newSingleAdminService(admins ...User) (*adminServiceImpl, *singleAdminUserRepo) {
	repo := &singleAdminUserRepo{admins: admins}
	repo.usersByID = map[int64]*User{
		1: {ID: 1, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive},
		2: {ID: 2, Email: "user@example.com", Role: RoleUser, Status: StatusActive},
		3: {ID: 3, Email: "ops@example.com", Role: RoleOperator, Status: StatusActive},
	}
	return &adminServiceImpl{userRepo: repo}, repo
}

func TestSingleAdmin_CreateSecondAdminRejected(t *testing.T) {
	svc, repo := newSingleAdminService(User{ID: 1, Role: RoleAdmin})

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{Email: "second@example.com", Password: "Passw0rd!", Role: RoleAdmin})
	require.ErrorIs(t, err, ErrAdminAlreadyExists)
	require.Empty(t, repo.created, "no user row must be written when the guard fires")
}

func TestSingleAdmin_CreateOperatorOrUserStillAllowed(t *testing.T) {
	svc, repo := newSingleAdminService(User{ID: 1, Role: RoleAdmin})

	for _, role := range []string{RoleOperator, RoleUser, ""} {
		_, err := svc.CreateUser(context.Background(), &CreateUserInput{Email: role + "@example.com", Password: "Passw0rd!", Role: role})
		require.NoErrorf(t, err, "role %q", role)
	}
	require.Len(t, repo.created, 3)
	require.Equal(t, RoleOperator, repo.created[0].Role)
	require.Equal(t, RoleUser, repo.created[2].Role, "empty role defaults to user")
}

func TestSingleAdmin_FirstAdminAllowedWhenNoneExists(t *testing.T) {
	svc, repo := newSingleAdminService()

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{Email: "first@example.com", Password: "Passw0rd!", Role: RoleAdmin})
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	require.Equal(t, RoleAdmin, repo.created[0].Role)
}

func TestSingleAdmin_PromoteAnotherUserRejected(t *testing.T) {
	svc, repo := newSingleAdminService(User{ID: 1, Role: RoleAdmin})

	for _, id := range []int64{2, 3} {
		_, err := svc.UpdateUser(context.Background(), id, &UpdateUserInput{Role: RoleAdmin})
		require.ErrorIsf(t, err, ErrAdminAlreadyExists, "user %d", id)
	}
	require.Empty(t, repo.updated)
}

func TestSingleAdmin_EditingTheAdminItselfIsNotBlocked(t *testing.T) {
	svc, repo := newSingleAdminService(User{ID: 1, Role: RoleAdmin})

	// 前端编辑表单总是携带 role；目标本来就是 admin 时不算"第二个管理员"。
	notes := "edited"
	updated, err := svc.UpdateUser(context.Background(), 1, &UpdateUserInput{Role: RoleAdmin, Notes: &notes})
	require.NoError(t, err)
	require.Equal(t, RoleAdmin, updated.Role)
	require.Len(t, repo.updated, 1)
}

func TestSingleAdmin_DemoteToOperatorStillAllowedForOthers(t *testing.T) {
	svc, repo := newSingleAdminService(User{ID: 1, Role: RoleAdmin})

	updated, err := svc.UpdateUser(context.Background(), 2, &UpdateUserInput{Role: RoleOperator})
	require.NoError(t, err)
	require.Equal(t, RoleOperator, updated.Role)
	require.Len(t, repo.updated, 1)
}

func TestSingleAdmin_ErrorIsConflict(t *testing.T) {
	status, _ := infraerrors.ToHTTP(ErrAdminAlreadyExists)
	require.Equal(t, 409, status)
	require.True(t, errors.Is(fmt.Errorf("wrap: %w", ErrAdminAlreadyExists), ErrAdminAlreadyExists))
}
