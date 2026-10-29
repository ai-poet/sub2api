/**
 * Referral API endpoints (user-side)
 */

import { apiClient } from './client'
import type { ReferralInfo, UserReferral, BasePaginationResponse, AffiliateLedgerEntry } from '@/types'

/**
 * Get referral info (code, link, stats)
 */
export async function getReferralInfo(): Promise<ReferralInfo> {
  const { data } = await apiClient.get<ReferralInfo>('/referral/info')
  return data
}

/**
 * Get referral history (paginated)
 */
export async function getReferralHistory(params?: {
  page?: number
  page_size?: number
}): Promise<BasePaginationResponse<UserReferral>> {
  const { data } = await apiClient.get<BasePaginationResponse<UserReferral>>('/referral/history', { params })
  return data
}

/** Transfer all available affiliate rebate to the main balance. */
export async function transferAffiliateQuota(): Promise<AffiliateLedgerEntry> {
  const { data } = await apiClient.post<AffiliateLedgerEntry>('/referral/transfer')
  return data
}
