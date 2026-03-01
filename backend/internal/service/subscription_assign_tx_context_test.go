package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type txContextUserSubRepoStub struct {
	sub                 *UserSubscription
	extendCalledInTx    bool
	updateStatusInTx    bool
	updateNotesInTx     bool
	updatedSubscription *UserSubscription
}

func (s *txContextUserSubRepoStub) Create(context.Context, *UserSubscription) error { return nil }

func (s *txContextUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if s.sub == nil || s.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *s.sub
	return &cp, nil
}

func (s *txContextUserSubRepoStub) GetByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	if s.sub == nil || s.sub.UserID != userID || s.sub.GroupID != groupID {
		return nil, ErrSubscriptionNotFound
	}
	cp := *s.sub
	return &cp, nil
}

func (s *txContextUserSubRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, ErrSubscriptionNotFound
}

func (s *txContextUserSubRepoStub) Update(context.Context, *UserSubscription) error { return nil }
func (s *txContextUserSubRepoStub) Delete(context.Context, int64) error             { return nil }

func (s *txContextUserSubRepoStub) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}

func (s *txContextUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}

func (s *txContextUserSubRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *txContextUserSubRepoStub) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *txContextUserSubRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return s.sub != nil, nil
}

func (s *txContextUserSubRepoStub) ExtendExpiry(ctx context.Context, id int64, expiresAt time.Time) error {
	if s.sub == nil || s.sub.ID != id {
		return ErrSubscriptionNotFound
	}
	s.extendCalledInTx = dbent.TxFromContext(ctx) != nil
	s.sub.ExpiresAt = expiresAt
	return nil
}

func (s *txContextUserSubRepoStub) UpdateStatus(ctx context.Context, id int64, status string) error {
	if s.sub == nil || s.sub.ID != id {
		return ErrSubscriptionNotFound
	}
	s.updateStatusInTx = dbent.TxFromContext(ctx) != nil
	s.sub.Status = status
	return nil
}

func (s *txContextUserSubRepoStub) UpdateNotes(ctx context.Context, id int64, notes string) error {
	if s.sub == nil || s.sub.ID != id {
		return ErrSubscriptionNotFound
	}
	s.updateNotesInTx = dbent.TxFromContext(ctx) != nil
	s.sub.Notes = notes
	return nil
}

func (s *txContextUserSubRepoStub) ActivateWindows(context.Context, int64, time.Time) error {
	return nil
}
func (s *txContextUserSubRepoStub) ResetDailyUsage(context.Context, int64, time.Time) error {
	return nil
}
func (s *txContextUserSubRepoStub) ResetWeeklyUsage(context.Context, int64, time.Time) error {
	return nil
}
func (s *txContextUserSubRepoStub) ResetMonthlyUsage(context.Context, int64, time.Time) error {
	return nil
}
func (s *txContextUserSubRepoStub) IncrementUsage(context.Context, int64, float64) error {
	return nil
}
func (s *txContextUserSubRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	return 0, nil
}

type txContextGroupRepoStub struct {
	group *Group
}

func (s *txContextGroupRepoStub) Create(context.Context, *Group) error { return nil }

func (s *txContextGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	return s.group, nil
}

func (s *txContextGroupRepoStub) GetByIDLite(context.Context, int64) (*Group, error) {
	return s.group, nil
}

func (s *txContextGroupRepoStub) Update(context.Context, *Group) error { return nil }
func (s *txContextGroupRepoStub) Delete(context.Context, int64) error  { return nil }
func (s *txContextGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (s *txContextGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *txContextGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *txContextGroupRepoStub) ListActive(context.Context) ([]Group, error) { return nil, nil }
func (s *txContextGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return nil, nil
}
func (s *txContextGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}
func (s *txContextGroupRepoStub) GetAccountCount(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *txContextGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *txContextGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}
func (s *txContextGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	return nil
}
func (s *txContextGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	return nil
}

func TestAssignOrExtendSubscription_ReuseOuterTransactionContext(t *testing.T) {
	groupRepo := &txContextGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	userSubRepo := &txContextUserSubRepoStub{
		sub: &UserSubscription{
			ID:        10,
			UserID:    42,
			GroupID:   1,
			Status:    SubscriptionStatusSuspended,
			Notes:     "existing",
			ExpiresAt: time.Now().AddDate(0, 0, 7),
		},
	}

	svc := NewSubscriptionService(groupRepo, userSubRepo, nil, nil, nil)
	ctx := dbent.NewTxContext(context.Background(), &dbent.Tx{})

	sub, extended, err := svc.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
		UserID:       42,
		GroupID:      1,
		ValidityDays: 3,
		Notes:        "referral reward",
	})

	require.NoError(t, err)
	require.True(t, extended)
	require.NotNil(t, sub)
	require.True(t, userSubRepo.extendCalledInTx)
	require.True(t, userSubRepo.updateStatusInTx)
	require.True(t, userSubRepo.updateNotesInTx)
	require.Equal(t, SubscriptionStatusActive, sub.Status)
	require.Contains(t, sub.Notes, "referral reward")
}
