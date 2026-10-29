package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

// These aliases keep the HTTP package stable while the implementation moves
// from one-time rewards to proportional affiliate rebates.
type UserReferral = service.AffiliateLedgerEntry
type ReferralInfo = service.ReferralInfo
type ReferralStats = service.ReferralStats
type ReferralSettings = service.ReferralSettings

type AffiliateRelation = service.AffiliateSummary

func ReferralInfoFromService(info *service.ReferralInfo) *ReferralInfo { return info }
func ReferralSettingsFromService(settings *service.ReferralSettings) *ReferralSettings {
	return settings
}
func ReferralSettingsToService(settings *ReferralSettings) *service.ReferralSettings { return settings }
