package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	AffiliateCodeMinLength         = 4
	AffiliateCodeMaxLength         = 32
	AffiliateRebateRateDefault     = 20.0
	AffiliateRebateFreezeHoursMax  = 720
	AffiliateRebateDurationDaysMax = 3650
	AffiliateActionAccrue          = "accrue"
	AffiliateActionTransfer        = "transfer"
)

var (
	ErrAffiliateNotFound      = infraerrors.NotFound("AFFILIATE_PROFILE_NOT_FOUND", "affiliate profile not found")
	ErrAffiliateCodeInvalid   = infraerrors.BadRequest("AFFILIATE_CODE_INVALID", "invalid referral code")
	ErrAffiliateCodeTaken     = infraerrors.Conflict("AFFILIATE_CODE_TAKEN", "referral code already taken")
	ErrAffiliateAlreadyBound  = infraerrors.Conflict("AFFILIATE_ALREADY_BOUND", "user already has an inviter")
	ErrAffiliateQuotaEmpty    = infraerrors.BadRequest("AFFILIATE_QUOTA_EMPTY", "no available affiliate rebate")
	ErrAffiliateRateInvalid   = infraerrors.BadRequest("AFFILIATE_RATE_INVALID", "rebate rate must be between 0 and 100")
	ErrAffiliateSourceInvalid = infraerrors.BadRequest("AFFILIATE_SOURCE_INVALID", "invalid affiliate source reference")

	// Kept as aliases for authentication call sites and external error handling.
	ErrReferralCodeInvalid = ErrAffiliateCodeInvalid
	ErrReferralCodeTaken   = ErrAffiliateCodeTaken
)

// AffiliateSummary is the persisted profile and quota state of one user.
type AffiliateSummary struct {
	UserID               int64     `json:"user_id"`
	AffCode              string    `json:"aff_code"`
	InviterID            *int64    `json:"inviter_id,omitempty"`
	AffCount             int       `json:"aff_count"`
	AffQuota             float64   `json:"aff_quota"`
	AffFrozenQuota       float64   `json:"aff_frozen_quota"`
	AffHistoryQuota      float64   `json:"aff_history_quota"`
	AffRebateRatePercent *float64  `json:"aff_rebate_rate_percent,omitempty"`
	AffCodeCustom        bool      `json:"aff_code_custom"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	Email                string    `json:"email,omitempty"`
	Username             string    `json:"username,omitempty"`
	InviterEmail         string    `json:"inviter_email,omitempty"`
	InviterUsername      string    `json:"inviter_username,omitempty"`
}

type AffiliateInvitee struct {
	UserID      int64     `json:"user_id"`
	Email       string    `json:"email"`
	Username    string    `json:"username,omitempty"`
	TotalRebate float64   `json:"total_rebate"`
	CreatedAt   time.Time `json:"created_at"`
}

type AffiliateLedgerEntry struct {
	ID                   int64      `json:"id"`
	UserID               int64      `json:"user_id"`
	Action               string     `json:"action"`
	Amount               float64    `json:"amount"`
	SourceUserID         *int64     `json:"source_user_id,omitempty"`
	SourceOrderRef       *string    `json:"source_order_ref,omitempty"`
	FrozenUntil          *time.Time `json:"frozen_until,omitempty"`
	BalanceAfter         *float64   `json:"balance_after,omitempty"`
	AffQuotaAfter        *float64   `json:"aff_quota_after,omitempty"`
	AffFrozenQuotaAfter  *float64   `json:"aff_frozen_quota_after,omitempty"`
	AffHistoryQuotaAfter *float64   `json:"aff_history_quota_after,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	Email                string     `json:"email,omitempty"`
	Username             string     `json:"username,omitempty"`
	SourceEmail          string     `json:"source_email,omitempty"`
	SourceUsername       string     `json:"source_username,omitempty"`
}

type AffiliateRecordFilter struct {
	Search   string
	Action   string
	UserID   int64
	StartAt  *time.Time
	EndAt    *time.Time
	Page     int
	PageSize int
}

// AffiliateRepository contains all monetary operations. Implementations must
// keep cap reads and writes under the same inviter row lock.
type AffiliateRepository interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
	EnsureUserAffiliate(ctx context.Context, userID int64) (*AffiliateSummary, error)
	GetAffiliateByCode(ctx context.Context, code string) (*AffiliateSummary, error)
	BindInviter(ctx context.Context, userID, inviterID int64) (bool, error)
	WithLockedInviter(ctx context.Context, inviterID int64, fn func(context.Context) error) error
	GetAccruedRebateFromInvitee(ctx context.Context, inviterID, inviteeID int64) (float64, error)
	AccrueQuota(ctx context.Context, inviterID, inviteeID int64, amount float64, freezeHours int, sourceOrderRef *string) (bool, error)
	ThawFrozenQuota(ctx context.Context, userID int64) (float64, error)
	TransferQuotaToBalance(ctx context.Context, userID int64) (*AffiliateLedgerEntry, error)
	ListInvitees(ctx context.Context, inviterID int64, limit int) ([]AffiliateInvitee, error)
	ListLedger(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateLedgerEntry, *pagination.PaginationResult, error)
	ListRelations(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateSummary, *pagination.PaginationResult, error)
	UpdateUserAffCode(ctx context.Context, userID int64, code string) error
	ResetUserAffCode(ctx context.Context, userID int64) (string, error)
	SetUserRebateRate(ctx context.Context, userID int64, rate *float64) error
}

type AffiliateDetail struct {
	AffiliateSummary
	EffectiveRebateRatePercent float64            `json:"effective_rebate_rate_percent"`
	Invitees                   []AffiliateInvitee `json:"invitees"`
}

// ReferralInfo preserves the existing user response while exposing all quota fields.
type ReferralInfo struct {
	ReferralCode               string             `json:"referral_code"`
	ReferralLink               string             `json:"referral_link"`
	Stats                      ReferralStats      `json:"stats"`
	Rewards                    *ReferralSettings  `json:"rewards"`
	AffQuota                   float64            `json:"aff_quota"`
	AffFrozenQuota             float64            `json:"aff_frozen_quota"`
	AffHistoryQuota            float64            `json:"aff_history_quota"`
	EffectiveRebateRatePercent float64            `json:"effective_rebate_rate_percent"`
	Invitees                   []AffiliateInvitee `json:"invitees"`
}

type ReferralStats struct {
	TotalCount       int     `json:"total_count"`
	RewardedCount    int     `json:"rewarded_count"`
	PendingCount     int     `json:"pending_count"`
	TotalBalanceEarn float64 `json:"total_balance_earn"`
}

type ReferralSettings struct {
	Enabled              bool    `json:"enabled"`
	RebateRatePercent    float64 `json:"rebate_rate_percent"`
	RebateFreezeHours    int     `json:"rebate_freeze_hours"`
	RebateDurationDays   int     `json:"rebate_duration_days"`
	RebatePerInviteeCap  float64 `json:"rebate_per_invitee_cap"`
	AdminRechargeEnabled bool    `json:"admin_recharge_enabled"`
}

type ReferralService struct {
	referralRepo         AffiliateRepository
	userRepo             UserRepository
	settingRepo          SettingRepository
	onSettingsUpdate     func()
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
}

// NewReferralService retains its constructor shape so all authentication and
// Wire call sites keep working. The legacy cache, reward-record and subscription
// dependencies are no longer used by the proportional rebate model.
func NewReferralService(
	referralRepo AffiliateRepository,
	_ ReferralCache,
	userRepo UserRepository,
	settingRepo SettingRepository,
	_ ReferralRewardRecordRepository,
	_ *SubscriptionService,
) *ReferralService {
	return &ReferralService{referralRepo: referralRepo, userRepo: userRepo, settingRepo: settingRepo}
}

// ReferralCache is retained only for constructor compatibility with older Wire providers.
type ReferralCache interface {
	AcquireRewardLock(ctx context.Context, refereeID int64) (bool, error)
	ReleaseRewardLock(ctx context.Context, refereeID int64) error
}

// ReferralRewardRecordRepository remains in the constructor for Wire compatibility.
type ReferralRewardRecordRepository interface {
	Create(ctx context.Context, code *RedeemCode) error
}

func (s *ReferralService) SetOnSettingsUpdateCallback(callback func()) { s.onSettingsUpdate = callback }

func (s *ReferralService) SetCacheInvalidators(auth APIKeyAuthCacheInvalidator, billing *BillingCacheService) {
	s.authCacheInvalidator = auth
	s.billingCacheService = billing
}

func (s *ReferralService) IsReferralEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAffiliateEnabled)
	return err == nil && value == "true"
}

func (s *ReferralService) GenerateReferralCode(ctx context.Context, userID int64) (string, error) {
	profile, err := s.referralRepo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return "", err
	}
	return profile.AffCode, nil
}

func normalizeAffiliateCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

func IsValidAffiliateCodeFormat(code string) bool {
	if len(code) < AffiliateCodeMinLength || len(code) > AffiliateCodeMaxLength {
		return false
	}
	for _, c := range []byte(code) {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func ClampAffiliateRebateRate(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return AffiliateRebateRateDefault
	}
	return math.Max(0, math.Min(100, value))
}

func ValidateAffiliateSourceRef(ref string) error { return validateAffiliateSourceRef(ref) }

func validateAffiliateSourceRef(ref string) error {
	if len(ref) == 0 || len(ref) > 64 || strings.TrimSpace(ref) != ref {
		return ErrAffiliateSourceInvalid
	}
	for _, c := range ref {
		if c < 0x21 || c > 0x7e {
			return ErrAffiliateSourceInvalid
		}
	}
	return nil
}

func (s *ReferralService) BindInviterByCode(ctx context.Context, code string, userID int64) error {
	code = normalizeAffiliateCode(code)
	if code == "" || !s.IsReferralEnabled(ctx) {
		return nil
	}
	if !IsValidAffiliateCodeFormat(code) {
		return ErrAffiliateCodeInvalid
	}
	inviter, err := s.referralRepo.GetAffiliateByCode(ctx, code)
	if err != nil || inviter.UserID == userID {
		return ErrAffiliateCodeInvalid
	}
	if _, err := s.referralRepo.EnsureUserAffiliate(ctx, userID); err != nil {
		return err
	}
	bound, err := s.referralRepo.BindInviter(ctx, userID, inviter.UserID)
	if err != nil {
		return err
	}
	if !bound {
		profile, getErr := s.referralRepo.EnsureUserAffiliate(ctx, userID)
		if getErr != nil {
			return getErr
		}
		if profile.InviterID != nil && *profile.InviterID == inviter.UserID {
			return nil
		}
		return ErrAffiliateAlreadyBound
	}
	return nil
}

// RegisterReferral is deliberately fail-open: an invitation problem must never block signup.
func (s *ReferralService) RegisterReferral(ctx context.Context, code string, userID int64) error {
	if strings.TrimSpace(code) == "" {
		return nil
	}
	if err := s.BindInviterByCode(ctx, code, userID); err != nil {
		log.Printf("[Affiliate] invitation not bound for user %d: %v", userID, err)
	}
	return nil
}

func (s *ReferralService) GetAffiliateDetail(ctx context.Context, userID int64) (*AffiliateDetail, error) {
	if _, err := s.referralRepo.ThawFrozenQuota(ctx, userID); err != nil {
		return nil, err
	}
	profile, err := s.referralRepo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return nil, err
	}
	invitees, err := s.referralRepo.ListInvitees(ctx, userID, 100)
	if err != nil {
		return nil, err
	}
	return &AffiliateDetail{AffiliateSummary: *profile, EffectiveRebateRatePercent: s.effectiveRate(ctx, profile), Invitees: invitees}, nil
}

func (s *ReferralService) GetReferralInfo(ctx context.Context, userID int64) (*ReferralInfo, error) {
	detail, err := s.GetAffiliateDetail(ctx, userID)
	if err != nil {
		return nil, err
	}
	siteURL, _ := s.settingRepo.GetValue(ctx, SettingKeyAPIBaseURL)
	link := ""
	if siteURL != "" {
		link = strings.TrimRight(siteURL, "/") + "/register?ref=" + detail.AffCode
	}
	return &ReferralInfo{
		ReferralCode: detail.AffCode, ReferralLink: link,
		Stats:   ReferralStats{TotalCount: detail.AffCount, RewardedCount: detail.AffCount, TotalBalanceEarn: detail.AffHistoryQuota},
		Rewards: s.GetReferralSettings(ctx), AffQuota: detail.AffQuota,
		AffFrozenQuota: detail.AffFrozenQuota, AffHistoryQuota: detail.AffHistoryQuota,
		EffectiveRebateRatePercent: detail.EffectiveRebateRatePercent, Invitees: detail.Invitees,
	}, nil
}

func roundAffiliateAmount(value float64) float64 { return math.Round(value*1e8) / 1e8 }

func (s *ReferralService) effectiveRate(ctx context.Context, inviter *AffiliateSummary) float64 {
	if inviter.AffRebateRatePercent != nil && !math.IsNaN(*inviter.AffRebateRatePercent) && !math.IsInf(*inviter.AffRebateRatePercent, 0) {
		return ClampAffiliateRebateRate(*inviter.AffRebateRatePercent)
	}
	return s.GetReferralSettings(ctx).RebateRatePercent
}

// AccrueInviteRebate credits the inviter for one balance recharge. A source ref
// is the cross-process idempotency key and must be stable across fulfillment retries.
func (s *ReferralService) AccrueInviteRebate(ctx context.Context, inviteeUserID int64, amount float64, sourceOrderRef string) (float64, error) {
	if inviteeUserID <= 0 || amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) || !s.IsReferralEnabled(ctx) {
		return 0, nil
	}
	if err := validateAffiliateSourceRef(sourceOrderRef); err != nil {
		return 0, err
	}
	invitee, err := s.referralRepo.EnsureUserAffiliate(ctx, inviteeUserID)
	if err != nil || invitee.InviterID == nil || *invitee.InviterID <= 0 {
		return 0, err
	}
	settings := s.GetReferralSettings(ctx)
	if settings.RebateDurationDays > 0 && time.Now().After(invitee.CreatedAt.AddDate(0, 0, settings.RebateDurationDays)) {
		return 0, nil
	}
	inviter, err := s.referralRepo.EnsureUserAffiliate(ctx, *invitee.InviterID)
	if err != nil {
		return 0, err
	}
	rebate := roundAffiliateAmount(amount * s.effectiveRate(ctx, inviter) / 100)
	if rebate <= 0 {
		return 0, nil
	}
	credited := 0.0
	err = s.referralRepo.WithLockedInviter(ctx, inviter.UserID, func(txCtx context.Context) error {
		if settings.RebatePerInviteeCap > 0 {
			existing, err := s.referralRepo.GetAccruedRebateFromInvitee(txCtx, inviter.UserID, inviteeUserID)
			if err != nil {
				return err
			}
			if existing >= settings.RebatePerInviteeCap {
				return nil
			}
			rebate = roundAffiliateAmount(math.Min(rebate, settings.RebatePerInviteeCap-existing))
			if rebate <= 0 {
				return nil
			}
		}
		applied, err := s.referralRepo.AccrueQuota(txCtx, inviter.UserID, inviteeUserID, rebate, settings.RebateFreezeHours, &sourceOrderRef)
		if err != nil {
			return err
		}
		if applied {
			credited = rebate
		}
		return nil
	})
	return credited, err
}

func (s *ReferralService) TransferQuotaToBalance(ctx context.Context, userID int64) (*AffiliateLedgerEntry, error) {
	if _, err := s.referralRepo.ThawFrozenQuota(ctx, userID); err != nil {
		return nil, err
	}
	entry, err := s.referralRepo.TransferQuotaToBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.billingCacheService != nil {
		if err := s.billingCacheService.InvalidateUserBalance(ctx, userID); err != nil {
			log.Printf("[Affiliate] invalidate balance for user %d: %v", userID, err)
		}
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	return entry, nil
}

func (s *ReferralService) GetReferralHistory(ctx context.Context, userID int64, params pagination.PaginationParams) ([]AffiliateLedgerEntry, *pagination.PaginationResult, error) {
	return s.referralRepo.ListLedger(ctx, AffiliateRecordFilter{UserID: userID, Page: params.Page, PageSize: params.PageSize})
}

func (s *ReferralService) ListRelations(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateSummary, *pagination.PaginationResult, error) {
	return s.referralRepo.ListRelations(ctx, filter)
}

func (s *ReferralService) ListLedger(ctx context.Context, filter AffiliateRecordFilter) ([]AffiliateLedgerEntry, *pagination.PaginationResult, error) {
	return s.referralRepo.ListLedger(ctx, filter)
}

func (s *ReferralService) UpdateUserAffCode(ctx context.Context, userID int64, code string) error {
	code = normalizeAffiliateCode(code)
	if !IsValidAffiliateCodeFormat(code) {
		return ErrAffiliateCodeInvalid
	}
	return s.referralRepo.UpdateUserAffCode(ctx, userID, code)
}

func (s *ReferralService) ResetUserAffCode(ctx context.Context, userID int64) (string, error) {
	return s.referralRepo.ResetUserAffCode(ctx, userID)
}

func (s *ReferralService) SetUserRebateRate(ctx context.Context, userID int64, rate *float64) error {
	if rate != nil {
		if math.IsNaN(*rate) || math.IsInf(*rate, 0) || *rate < 0 || *rate > 100 {
			return ErrAffiliateRateInvalid
		}
		v := math.Round(*rate*100) / 100
		rate = &v
	}
	return s.referralRepo.SetUserRebateRate(ctx, userID, rate)
}

func parseFloatSetting(raw string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

func parseIntSetting(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func (s *ReferralService) GetReferralSettings(ctx context.Context) *ReferralSettings {
	keys := []string{SettingKeyAffiliateEnabled, SettingKeyAffiliateRebateRate, SettingKeyAffiliateRebateFreezeHours, SettingKeyAffiliateRebateDurationDays, SettingKeyAffiliateRebatePerInviteeCap, SettingKeyAffiliateAdminRechargeEnabled}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		values = map[string]string{}
	}
	result := &ReferralSettings{Enabled: values[SettingKeyAffiliateEnabled] == "true", RebateRatePercent: AffiliateRebateRateDefault, AdminRechargeEnabled: values[SettingKeyAffiliateAdminRechargeEnabled] == "true"}
	if v, err := strconv.ParseFloat(values[SettingKeyAffiliateRebateRate], 64); err == nil {
		result.RebateRatePercent = ClampAffiliateRebateRate(v)
	}
	if v, err := strconv.Atoi(values[SettingKeyAffiliateRebateFreezeHours]); err == nil && v >= 0 {
		result.RebateFreezeHours = min(v, AffiliateRebateFreezeHoursMax)
	}
	if v, err := strconv.Atoi(values[SettingKeyAffiliateRebateDurationDays]); err == nil && v >= 0 {
		result.RebateDurationDays = min(v, AffiliateRebateDurationDaysMax)
	}
	if v, err := strconv.ParseFloat(values[SettingKeyAffiliateRebatePerInviteeCap], 64); err == nil && v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
		result.RebatePerInviteeCap = v
	}
	return result
}

func (s *ReferralService) UpdateReferralSettings(ctx context.Context, settings *ReferralSettings) error {
	if settings == nil {
		return infraerrors.BadRequest("AFFILIATE_SETTINGS_INVALID", "settings are required")
	}
	settings.RebateRatePercent = ClampAffiliateRebateRate(settings.RebateRatePercent)
	settings.RebateFreezeHours = max(0, min(settings.RebateFreezeHours, AffiliateRebateFreezeHoursMax))
	settings.RebateDurationDays = max(0, min(settings.RebateDurationDays, AffiliateRebateDurationDaysMax))
	if settings.RebatePerInviteeCap < 0 || math.IsNaN(settings.RebatePerInviteeCap) || math.IsInf(settings.RebatePerInviteeCap, 0) {
		settings.RebatePerInviteeCap = 0
	}
	updates := map[string]string{
		SettingKeyAffiliateEnabled:              strconv.FormatBool(settings.Enabled),
		SettingKeyAffiliateRebateRate:           strconv.FormatFloat(settings.RebateRatePercent, 'f', 8, 64),
		SettingKeyAffiliateRebateFreezeHours:    strconv.Itoa(settings.RebateFreezeHours),
		SettingKeyAffiliateRebateDurationDays:   strconv.Itoa(settings.RebateDurationDays),
		SettingKeyAffiliateRebatePerInviteeCap:  strconv.FormatFloat(settings.RebatePerInviteeCap, 'f', 8, 64),
		SettingKeyAffiliateAdminRechargeEnabled: strconv.FormatBool(settings.AdminRechargeEnabled),
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return err
	}
	if s.onSettingsUpdate != nil {
		s.onSettingsUpdate()
	}
	return nil
}

func NewAffiliateCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate affiliate code: %w", err)
	}
	return strings.ToUpper(hex.EncodeToString(buf)), nil
}

func IsAffiliateDuplicateError(err error) bool { return errors.Is(err, ErrAffiliateCodeTaken) }
