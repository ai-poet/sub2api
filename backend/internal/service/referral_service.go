package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrReferralNotFound     = infraerrors.NotFound("REFERRAL_NOT_FOUND", "referral record not found")
	ErrReferralDisabled     = infraerrors.Forbidden("REFERRAL_DISABLED", "referral system is disabled")
	ErrReferralSelf         = infraerrors.BadRequest("REFERRAL_SELF", "cannot refer yourself")
	ErrReferralAlreadyExist = infraerrors.Conflict("REFERRAL_ALREADY_EXIST", "user already has a referrer")
	ErrReferralMaxReached   = infraerrors.Forbidden("REFERRAL_MAX_REACHED", "referrer has reached maximum referral limit")
	ErrReferralCodeInvalid  = infraerrors.BadRequest("REFERRAL_CODE_INVALID", "invalid referral code")
)

// Referral status constants
const (
	ReferralStatusPending  = "pending"
	ReferralStatusRewarded = "rewarded"
)

// UserReferral 推荐关系模型
type UserReferral struct {
	ID                       int64
	ReferrerID               int64
	RefereeID                int64
	Status                   string
	ReferrerBalanceReward    float64
	ReferrerGroupID          *int64
	ReferrerSubscriptionDays int
	ReferrerRewardedAt       *time.Time
	RefereeBalanceReward     float64
	RefereeGroupID           *int64
	RefereeSubscriptionDays  int
	RefereeRewardedAt        *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time

	// 关联数据（可选加载）
	ReferrerEmail string
	RefereeEmail  string
}

// ReferralRewardSnapshot 奖励快照
type ReferralRewardSnapshot struct {
	ReferrerBalanceReward    float64
	ReferrerGroupID          *int64
	ReferrerSubscriptionDays int
	RefereeBalanceReward     float64
	RefereeGroupID           *int64
	RefereeSubscriptionDays  int
}

// ReferralRepository 推荐数据访问接口
type ReferralRepository interface {
	Create(ctx context.Context, ref *UserReferral) error
	GetByRefereeID(ctx context.Context, refereeID int64) (*UserReferral, error)
	GetByID(ctx context.Context, id int64) (*UserReferral, error)
	UpdateStatus(ctx context.Context, id int64, status string, rewards *ReferralRewardSnapshot) error
	CountByReferrerID(ctx context.Context, referrerID int64) (int, error)
	CountByReferrerIDAndStatus(ctx context.Context, referrerID int64, status string) (int, error)
	SumReferrerBalanceReward(ctx context.Context, referrerID int64) (float64, error)
	ListByReferrerID(ctx context.Context, referrerID int64, params pagination.PaginationParams) ([]UserReferral, *pagination.PaginationResult, error)
	ListAll(ctx context.Context, params pagination.PaginationParams) ([]UserReferral, *pagination.PaginationResult, error)
}

// ReferralCache 推荐缓存接口
type ReferralCache interface {
	AcquireRewardLock(ctx context.Context, refereeID int64) (bool, error)
	ReleaseRewardLock(ctx context.Context, refereeID int64) error
}

// ReferralInfo 推荐信息（用户侧）
type ReferralInfo struct {
	ReferralCode string        `json:"referral_code"`
	ReferralLink string        `json:"referral_link"`
	Stats        ReferralStats `json:"stats"`
}

// ReferralStats 推荐统计
type ReferralStats struct {
	TotalCount       int     `json:"total_count"`
	RewardedCount    int     `json:"rewarded_count"`
	PendingCount     int     `json:"pending_count"`
	TotalBalanceEarn float64 `json:"total_balance_earn"`
}

// ReferralSettings 推荐系统配置
type ReferralSettings struct {
	Enabled                  bool    `json:"enabled"`
	ReferrerBalanceReward    float64 `json:"referrer_balance_reward"`
	ReferrerGroupID          int64   `json:"referrer_group_id"`
	ReferrerSubscriptionDays int     `json:"referrer_subscription_days"`
	RefereeBalanceReward     float64 `json:"referee_balance_reward"`
	RefereeGroupID           int64   `json:"referee_group_id"`
	RefereeSubscriptionDays  int     `json:"referee_subscription_days"`
	MaxPerUser               int     `json:"max_per_user"`
}

// ReferralService 推荐服务
type ReferralService struct {
	referralRepo        ReferralRepository
	referralCache       ReferralCache
	userRepo            UserRepository
	settingRepo         SettingRepository
	subscriptionService *SubscriptionService
}

// NewReferralService 创建推荐服务
func NewReferralService(
	referralRepo ReferralRepository,
	referralCache ReferralCache,
	userRepo UserRepository,
	settingRepo SettingRepository,
	subscriptionService *SubscriptionService,
) *ReferralService {
	return &ReferralService{
		referralRepo:        referralRepo,
		referralCache:       referralCache,
		userRepo:            userRepo,
		settingRepo:         settingRepo,
		subscriptionService: subscriptionService,
	}
}

// IsReferralEnabled 检查推荐功能是否启用
func (s *ReferralService) IsReferralEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyReferralEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

// GenerateReferralCode 生成 8 字符 URL-safe 唯一推荐码
func (s *ReferralService) GenerateReferralCode(ctx context.Context, userID int64) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	// 如果已有推荐码，直接返回
	if user.ReferralCode != "" {
		return user.ReferralCode, nil
	}

	// 生成 6 字节随机数据 → base64url 编码 → 8 字符
	for i := 0; i < 10; i++ {
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate random bytes: %w", err)
		}
		code := base64.RawURLEncoding.EncodeToString(bytes)

		// 保存到用户记录
		user.ReferralCode = code
		if err := s.userRepo.Update(ctx, user); err != nil {
			// 唯一约束冲突，重试
			continue
		}
		return code, nil
	}

	return "", fmt.Errorf("failed to generate unique referral code after retries")
}

// GetReferralInfo 获取推荐信息（不存在则自动生成推荐码）
func (s *ReferralService) GetReferralInfo(ctx context.Context, userID int64) (*ReferralInfo, error) {
	code, err := s.GenerateReferralCode(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取统计数据
	totalCount, err := s.referralRepo.CountByReferrerID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count referrals: %w", err)
	}

	rewardedCount, err := s.referralRepo.CountByReferrerIDAndStatus(ctx, userID, ReferralStatusRewarded)
	if err != nil {
		return nil, fmt.Errorf("count rewarded referrals: %w", err)
	}

	totalBalanceEarn, err := s.referralRepo.SumReferrerBalanceReward(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("sum balance reward: %w", err)
	}

	// 获取站点 URL 用于生成推荐链接
	siteURL, _ := s.settingRepo.GetValue(ctx, SettingKeyAPIBaseURL)
	referralLink := ""
	if siteURL != "" {
		referralLink = siteURL + "/register?ref=" + code
	}

	return &ReferralInfo{
		ReferralCode: code,
		ReferralLink: referralLink,
		Stats: ReferralStats{
			TotalCount:       totalCount,
			RewardedCount:    rewardedCount,
			PendingCount:     totalCount - rewardedCount,
			TotalBalanceEarn: totalBalanceEarn,
		},
	}, nil
}

// RegisterReferral 注册时记录推荐关系
func (s *ReferralService) RegisterReferral(ctx context.Context, referrerCode string, refereeUserID int64) error {
	if referrerCode == "" {
		return nil
	}

	// 检查推荐功能是否启用
	if !s.IsReferralEnabled(ctx) {
		return nil // 静默忽略
	}

	// 查找推荐人
	referrer, err := s.findUserByReferralCode(ctx, referrerCode)
	if err != nil {
		log.Printf("[Referral] Invalid referral code %s: %v", referrerCode, err)
		return nil // 推荐码无效不阻止注册
	}

	// 不能推荐自己
	if referrer.ID == refereeUserID {
		return nil
	}

	// 检查被推荐人是否已有推荐记录
	_, err = s.referralRepo.GetByRefereeID(ctx, refereeUserID)
	if err == nil {
		return nil // 已有推荐记录，静默忽略
	}
	if !errors.Is(err, ErrReferralNotFound) {
		return fmt.Errorf("check existing referral: %w", err)
	}

	// 检查推荐人是否达到上限
	maxPerUser := s.getMaxPerUser(ctx)
	if maxPerUser > 0 {
		count, err := s.referralRepo.CountByReferrerID(ctx, referrer.ID)
		if err != nil {
			return fmt.Errorf("count referrals: %w", err)
		}
		if count >= maxPerUser {
			log.Printf("[Referral] Referrer %d reached max referral limit %d", referrer.ID, maxPerUser)
			return nil // 达到上限，静默忽略
		}
	}

	// 创建推荐关系
	ref := &UserReferral{
		ReferrerID: referrer.ID,
		RefereeID:  refereeUserID,
		Status:     ReferralStatusPending,
	}
	if err := s.referralRepo.Create(ctx, ref); err != nil {
		log.Printf("[Referral] Failed to create referral record: %v", err)
		return nil // 创建失败不阻止注册
	}

	log.Printf("[Referral] Referral recorded: referrer=%d, referee=%d", referrer.ID, refereeUserID)
	return nil
}

// TriggerReferralReward 触发推荐奖励（被推荐人首次余额兑换后调用）
func (s *ReferralService) TriggerReferralReward(ctx context.Context, refereeUserID int64) {
	// 1. 检查推荐功能是否启用
	if !s.IsReferralEnabled(ctx) {
		return
	}

	// 2. 查询推荐关系
	ref, err := s.referralRepo.GetByRefereeID(ctx, refereeUserID)
	if err != nil {
		if errors.Is(err, ErrReferralNotFound) {
			return // 没有推荐关系
		}
		log.Printf("[Referral] Failed to get referral for referee %d: %v", refereeUserID, err)
		return
	}

	// 3. 已经发放过奖励则跳过
	if ref.Status == ReferralStatusRewarded {
		return
	}

	// 4. 获取分布式锁
	locked, err := s.referralCache.AcquireRewardLock(ctx, refereeUserID)
	if err != nil || !locked {
		log.Printf("[Referral] Failed to acquire reward lock for referee %d: %v", refereeUserID, err)
		return
	}
	defer func() {
		if err := s.referralCache.ReleaseRewardLock(ctx, refereeUserID); err != nil {
			log.Printf("[Referral] Failed to release reward lock for referee %d: %v", refereeUserID, err)
		}
	}()

	// 5. 双重检查（获取锁后再次确认状态）
	ref, err = s.referralRepo.GetByRefereeID(ctx, refereeUserID)
	if err != nil || ref.Status == ReferralStatusRewarded {
		return
	}

	// 6. 读取当前奖励配置
	settings := s.GetReferralSettings(ctx)
	snapshot := &ReferralRewardSnapshot{
		ReferrerBalanceReward:    settings.ReferrerBalanceReward,
		ReferrerSubscriptionDays: settings.ReferrerSubscriptionDays,
		RefereeBalanceReward:     settings.RefereeBalanceReward,
		RefereeSubscriptionDays:  settings.RefereeSubscriptionDays,
	}
	if settings.ReferrerGroupID > 0 {
		gid := settings.ReferrerGroupID
		snapshot.ReferrerGroupID = &gid
	}
	if settings.RefereeGroupID > 0 {
		gid := settings.RefereeGroupID
		snapshot.RefereeGroupID = &gid
	}

	// 7. 发放奖励
	if err := s.distributeRewards(ctx, ref, snapshot); err != nil {
		log.Printf("[Referral] Failed to distribute rewards for referral %d: %v", ref.ID, err)
		return
	}

	// 8. 更新状态为 rewarded
	if err := s.referralRepo.UpdateStatus(ctx, ref.ID, ReferralStatusRewarded, snapshot); err != nil {
		log.Printf("[Referral] Failed to update referral status %d: %v", ref.ID, err)
		return
	}

	log.Printf("[Referral] Rewards distributed: referral=%d, referrer=%d, referee=%d", ref.ID, ref.ReferrerID, ref.RefereeID)
}

// distributeRewards 发放推荐奖励
func (s *ReferralService) distributeRewards(ctx context.Context, ref *UserReferral, snapshot *ReferralRewardSnapshot) error {
	// 推荐人余额奖励
	if snapshot.ReferrerBalanceReward > 0 {
		if err := s.userRepo.UpdateBalance(ctx, ref.ReferrerID, snapshot.ReferrerBalanceReward); err != nil {
			return fmt.Errorf("update referrer balance: %w", err)
		}
	}

	// 推荐人订阅奖励
	if snapshot.ReferrerGroupID != nil && snapshot.ReferrerSubscriptionDays > 0 {
		_, _, err := s.subscriptionService.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
			UserID:       ref.ReferrerID,
			GroupID:      *snapshot.ReferrerGroupID,
			ValidityDays: snapshot.ReferrerSubscriptionDays,
			AssignedBy:   0,
			Notes:        fmt.Sprintf("推荐奖励：推荐用户 %d 注册并充值", ref.RefereeID),
		})
		if err != nil {
			log.Printf("[Referral] Failed to assign referrer subscription: %v", err)
			// 订阅分配失败不阻止其他奖励
		}
	}

	// 被推荐人余额奖励
	if snapshot.RefereeBalanceReward > 0 {
		if err := s.userRepo.UpdateBalance(ctx, ref.RefereeID, snapshot.RefereeBalanceReward); err != nil {
			return fmt.Errorf("update referee balance: %w", err)
		}
	}

	// 被推荐人订阅奖励
	if snapshot.RefereeGroupID != nil && snapshot.RefereeSubscriptionDays > 0 {
		_, _, err := s.subscriptionService.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
			UserID:       ref.RefereeID,
			GroupID:      *snapshot.RefereeGroupID,
			ValidityDays: snapshot.RefereeSubscriptionDays,
			AssignedBy:   0,
			Notes:        fmt.Sprintf("推荐奖励：通过推荐链接注册并充值"),
		})
		if err != nil {
			log.Printf("[Referral] Failed to assign referee subscription: %v", err)
		}
	}

	return nil
}

// GetReferralHistory 获取推荐历史
func (s *ReferralService) GetReferralHistory(ctx context.Context, userID int64, params pagination.PaginationParams) ([]UserReferral, *pagination.PaginationResult, error) {
	refs, pag, err := s.referralRepo.ListByReferrerID(ctx, userID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list referrals: %w", err)
	}

	// 加载被推荐人邮箱（脱敏）
	for i := range refs {
		user, err := s.userRepo.GetByID(ctx, refs[i].RefereeID)
		if err == nil {
			refs[i].RefereeEmail = maskEmail(user.Email)
		}
	}

	return refs, pag, nil
}

// GetAllReferrals 管理员获取所有推荐记录
func (s *ReferralService) GetAllReferrals(ctx context.Context, params pagination.PaginationParams) ([]UserReferral, *pagination.PaginationResult, error) {
	refs, pag, err := s.referralRepo.ListAll(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list all referrals: %w", err)
	}

	// 加载推荐人和被推荐人邮箱
	for i := range refs {
		if referrer, err := s.userRepo.GetByID(ctx, refs[i].ReferrerID); err == nil {
			refs[i].ReferrerEmail = referrer.Email
		}
		if referee, err := s.userRepo.GetByID(ctx, refs[i].RefereeID); err == nil {
			refs[i].RefereeEmail = referee.Email
		}
	}

	return refs, pag, nil
}

// GetReferralSettings 获取推荐系统配置
func (s *ReferralService) GetReferralSettings(ctx context.Context) *ReferralSettings {
	keys := []string{
		SettingKeyReferralEnabled,
		SettingKeyReferralReferrerBalanceReward,
		SettingKeyReferralReferrerGroupID,
		SettingKeyReferralReferrerSubscriptionDays,
		SettingKeyReferralRefereeBalanceReward,
		SettingKeyReferralRefereeGroupID,
		SettingKeyReferralRefereeSubscriptionDays,
		SettingKeyReferralMaxPerUser,
	}

	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return &ReferralSettings{}
	}

	result := &ReferralSettings{
		Enabled: settings[SettingKeyReferralEnabled] == "true",
	}

	if v, err := strconv.ParseFloat(settings[SettingKeyReferralReferrerBalanceReward], 64); err == nil {
		result.ReferrerBalanceReward = v
	}
	if v, err := strconv.ParseInt(settings[SettingKeyReferralReferrerGroupID], 10, 64); err == nil {
		result.ReferrerGroupID = v
	}
	if v, err := strconv.Atoi(settings[SettingKeyReferralReferrerSubscriptionDays]); err == nil {
		result.ReferrerSubscriptionDays = v
	}
	if v, err := strconv.ParseFloat(settings[SettingKeyReferralRefereeBalanceReward], 64); err == nil {
		result.RefereeBalanceReward = v
	}
	if v, err := strconv.ParseInt(settings[SettingKeyReferralRefereeGroupID], 10, 64); err == nil {
		result.RefereeGroupID = v
	}
	if v, err := strconv.Atoi(settings[SettingKeyReferralRefereeSubscriptionDays]); err == nil {
		result.RefereeSubscriptionDays = v
	}
	if v, err := strconv.Atoi(settings[SettingKeyReferralMaxPerUser]); err == nil {
		result.MaxPerUser = v
	}

	return result
}

// UpdateReferralSettings 更新推荐系统配置
func (s *ReferralService) UpdateReferralSettings(ctx context.Context, settings *ReferralSettings) error {
	updates := map[string]string{
		SettingKeyReferralEnabled:                  strconv.FormatBool(settings.Enabled),
		SettingKeyReferralReferrerBalanceReward:    strconv.FormatFloat(settings.ReferrerBalanceReward, 'f', 8, 64),
		SettingKeyReferralReferrerGroupID:          strconv.FormatInt(settings.ReferrerGroupID, 10),
		SettingKeyReferralReferrerSubscriptionDays: strconv.Itoa(settings.ReferrerSubscriptionDays),
		SettingKeyReferralRefereeBalanceReward:     strconv.FormatFloat(settings.RefereeBalanceReward, 'f', 8, 64),
		SettingKeyReferralRefereeGroupID:           strconv.FormatInt(settings.RefereeGroupID, 10),
		SettingKeyReferralRefereeSubscriptionDays:  strconv.Itoa(settings.RefereeSubscriptionDays),
		SettingKeyReferralMaxPerUser:               strconv.Itoa(settings.MaxPerUser),
	}

	return s.settingRepo.SetMultiple(ctx, updates)
}

// findUserByReferralCode 通过推荐码查找用户
func (s *ReferralService) findUserByReferralCode(ctx context.Context, code string) (*User, error) {
	user, err := s.userRepo.GetByReferralCode(ctx, code)
	if err != nil {
		return nil, ErrReferralCodeInvalid
	}
	return user, nil
}

// getMaxPerUser 获取每用户最大推荐数
func (s *ReferralService) getMaxPerUser(ctx context.Context) int {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyReferralMaxPerUser)
	if err != nil {
		return 0 // 默认无限制
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return v
}

// maskEmail 邮箱脱敏
func maskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "***"
	}
	if at <= 2 {
		return email[:1] + "***" + email[at:]
	}
	return email[:2] + "***" + email[at:]
}
