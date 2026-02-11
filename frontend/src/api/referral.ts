/**
 * Referral API endpoints (user-side)
 */

import { apiClient } from './client'
import type { ReferralInfo, UserReferral, BasePaginationResponse } from '@/types'

/**
 * Get referral info (code, link, stats)
 */
export function getReferralInfo(): Promise<ReferralInfo> {
  return apiClient.get('/referral/info')
}

/**
 * Get referral history (paginated)
 */
export function getReferralHistory(params?: {
  page?: number
  page_size?: number
}): Promise<BasePaginationResponse<UserReferral>> {
  return apiClient.get('/referral/history', { params })
}
