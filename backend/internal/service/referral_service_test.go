package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

var _ SettingRepository = (*stubSettingRepoForReferralService)(nil)

type stubSettingRepoForReferralService struct {
	values         map[string]string
	setMultipleErr error
}

func (r *stubSettingRepoForReferralService) Get(_ context.Context, key string) (*Setting, error) {
	if value, ok := r.values[key]; ok { return &Setting{Key: key, Value: value}, nil }
	return nil, ErrSettingNotFound
}
func (r *stubSettingRepoForReferralService) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok { return value, nil }
	return "", ErrSettingNotFound
}
func (r *stubSettingRepoForReferralService) Set(_ context.Context, key, value string) error { r.values[key] = value; return nil }
func (r *stubSettingRepoForReferralService) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys { if value, ok := r.values[key]; ok { result[key] = value } }
	return result, nil
}
func (r *stubSettingRepoForReferralService) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.setMultipleErr != nil { return r.setMultipleErr }
	for key, value := range settings { r.values[key] = value }
	return nil
}
func (r *stubSettingRepoForReferralService) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(r.values)); for key, value := range r.values { result[key] = value }; return result, nil
}
func (r *stubSettingRepoForReferralService) Delete(_ context.Context, key string) error { delete(r.values, key); return nil }

// affiliateRepoStub implements the invariants used by service tests. Tests for
// row locking and SQL snapshots live in repository integration tests.
type affiliateRepoStub struct {
	AffiliateRepository
	profiles   map[int64]*AffiliateSummary
	ledger     map[string]float64
	pairTotals map[string]float64
	ensureCalls int
}

func newAffiliateRepoStub() *affiliateRepoStub {
	return &affiliateRepoStub{profiles: map[int64]*AffiliateSummary{}, ledger: map[string]float64{}, pairTotals: map[string]float64{}}
}
func affiliatePairKey(inviter, invitee int64) string { return fmt.Sprintf("%d:%d", inviter, invitee) }
func (r *affiliateRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	r.ensureCalls++
	if profile, ok := r.profiles[userID]; ok { copy := *profile; return &copy, nil }
	profile := &AffiliateSummary{UserID: userID, AffCode: fmt.Sprintf("CODE%04d", userID), CreatedAt: time.Now()}
	r.profiles[userID] = profile
	copy := *profile
	return &copy, nil
}
func (r *affiliateRepoStub) GetAffiliateByCode(_ context.Context, code string) (*AffiliateSummary, error) {
	for _, profile := range r.profiles { if profile.AffCode == code { copy := *profile; return &copy, nil } }
	return nil, ErrAffiliateNotFound
}
func (r *affiliateRepoStub) BindInviter(_ context.Context, userID, inviterID int64) (bool, error) {
	profile := r.profiles[userID]
	if profile.InviterID != nil { return false, nil }
	profile.InviterID = &inviterID
	r.profiles[inviterID].AffCount++
	return true, nil
}
func (r *affiliateRepoStub) WithLockedInviter(ctx context.Context, _ int64, fn func(context.Context) error) error { return fn(ctx) }
func (r *affiliateRepoStub) RunInTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }
func (r *affiliateRepoStub) GetAccruedRebateFromInvitee(_ context.Context, inviterID, inviteeID int64) (float64, error) { return r.pairTotals[affiliatePairKey(inviterID, inviteeID)], nil }
func (r *affiliateRepoStub) AccrueQuota(_ context.Context, inviterID, inviteeID int64, amount float64, freezeHours int, ref *string) (bool, error) {
	if ref != nil { if _, ok := r.ledger[*ref]; ok { return false, nil }; r.ledger[*ref] = amount }
	profile := r.profiles[inviterID]
	if freezeHours > 0 { profile.AffFrozenQuota += amount } else { profile.AffQuota += amount }
	profile.AffHistoryQuota += amount
	r.pairTotals[affiliatePairKey(inviterID, inviteeID)] += amount
	return true, nil
}
func (r *affiliateRepoStub) ThawFrozenQuota(context.Context, int64) (float64, error) { return 0, nil }
func (r *affiliateRepoStub) ListInvitees(context.Context, int64, int) ([]AffiliateInvitee, error) { return []AffiliateInvitee{}, nil }
func (r *affiliateRepoStub) ListLedger(context.Context, AffiliateRecordFilter) ([]AffiliateLedgerEntry, *pagination.PaginationResult, error) { return []AffiliateLedgerEntry{}, &pagination.PaginationResult{}, nil }
func (r *affiliateRepoStub) ListRelations(context.Context, AffiliateRecordFilter) ([]AffiliateSummary, *pagination.PaginationResult, error) { return []AffiliateSummary{}, &pagination.PaginationResult{}, nil }

func affiliateTestSettings() *stubSettingRepoForReferralService {
	return &stubSettingRepoForReferralService{values: map[string]string{
		SettingKeyAffiliateEnabled: "true", SettingKeyAffiliateRebateRate: "20",
		SettingKeyAffiliateRebateFreezeHours: "0", SettingKeyAffiliateRebateDurationDays: "0",
		SettingKeyAffiliateRebatePerInviteeCap: "0", SettingKeyAPIBaseURL: "https://example.com",
	}}
}

func TestReferralService_UpdateReferralSettings_TriggersCallbackOnSuccess(t *testing.T) {
	settingRepo := affiliateTestSettings()
	svc := NewReferralService(nil, nil, nil, settingRepo, nil, nil)
	callbackCalled := 0
	svc.SetOnSettingsUpdateCallback(func() { callbackCalled++ })
	err := svc.UpdateReferralSettings(context.Background(), &ReferralSettings{Enabled: true, RebateRatePercent: 12.5, RebateFreezeHours: 24, RebateDurationDays: 30, RebatePerInviteeCap: 99})
	require.NoError(t, err)
	require.Equal(t, 1, callbackCalled)
	require.Equal(t, "true", settingRepo.values[SettingKeyAffiliateEnabled])
	require.Equal(t, "12.50000000", settingRepo.values[SettingKeyAffiliateRebateRate])
	require.Equal(t, "99.00000000", settingRepo.values[SettingKeyAffiliateRebatePerInviteeCap])
}

func TestReferralService_UpdateReferralSettings_DoesNotTriggerCallbackOnFailure(t *testing.T) {
	settingRepo := affiliateTestSettings(); settingRepo.setMultipleErr = errors.New("set settings failed")
	svc := NewReferralService(nil, nil, nil, settingRepo, nil, nil)
	callbackCalled := 0; svc.SetOnSettingsUpdateCallback(func() { callbackCalled++ })
	require.Error(t, svc.UpdateReferralSettings(context.Background(), &ReferralSettings{}))
	require.Zero(t, callbackCalled)
}

func TestAffiliateAccrual_RechargesAndOrderIdempotency(t *testing.T) {
	repo := newAffiliateRepoStub(); settings := affiliateTestSettings()
	_, _ = repo.EnsureUserAffiliate(context.Background(), 1)
	_, _ = repo.EnsureUserAffiliate(context.Background(), 2)
	require.NoError(t, (&ReferralService{referralRepo: repo, settingRepo: settings}).BindInviterByCode(context.Background(), "CODE0001", 2))
	svc := NewReferralService(repo, nil, nil, settings, nil, nil)
	first, err := svc.AccrueInviteRebate(context.Background(), 2, 100, "pay-order:first")
	require.NoError(t, err); require.InDelta(t, 20, first, 1e-8)
	second, err := svc.AccrueInviteRebate(context.Background(), 2, 50, "pay-order:second")
	require.NoError(t, err); require.InDelta(t, 10, second, 1e-8)
	duplicate, err := svc.AccrueInviteRebate(context.Background(), 2, 100, "pay-order:first")
	require.NoError(t, err); require.Zero(t, duplicate)
	require.InDelta(t, 30, repo.profiles[1].AffQuota, 1e-8)
	require.InDelta(t, 30, repo.profiles[1].AffHistoryQuota, 1e-8)
}

func TestAffiliateAccrual_CustomRateCapAndFreeze(t *testing.T) {
	repo := newAffiliateRepoStub(); settings := affiliateTestSettings()
	settings.values[SettingKeyAffiliateRebatePerInviteeCap] = "12"
	settings.values[SettingKeyAffiliateRebateFreezeHours] = "24"
	_, _ = repo.EnsureUserAffiliate(context.Background(), 1)
	_, _ = repo.EnsureUserAffiliate(context.Background(), 2)
	inviter := repo.profiles[1]; inviter.AffRebateRatePercent = new(50.0)
	invitee := repo.profiles[2]; invitee.InviterID = new(int64(1))
	svc := NewReferralService(repo, nil, nil, settings, nil, nil)
	credited, err := svc.AccrueInviteRebate(context.Background(), 2, 100, "pay-order:capped")
	require.NoError(t, err); require.InDelta(t, 12, credited, 1e-8)
	require.Zero(t, repo.profiles[1].AffQuota)
	require.InDelta(t, 12, repo.profiles[1].AffFrozenQuota, 1e-8)
	zero := 0.0; inviter.AffRebateRatePercent = &zero
	credited, err = svc.AccrueInviteRebate(context.Background(), 2, 100, "pay-order:disabled")
	require.NoError(t, err); require.Zero(t, credited)
}

func TestAffiliateCodeValidationAndRateClamp(t *testing.T) {
	for _, code := range []string{"ABCD", "A_BC-D123", "1234"} { require.True(t, IsValidAffiliateCodeFormat(code)) }
	for _, code := range []string{"abc", "abcd", "A B C", "你好邀请", "ABC"} { require.False(t, IsValidAffiliateCodeFormat(code)) }
	require.Equal(t, 0.0, ClampAffiliateRebateRate(-1))
	require.Equal(t, 100.0, ClampAffiliateRebateRate(101))
}
