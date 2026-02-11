/**
 * Admin Referral API endpoints
 */

import { apiClient } from '../client'
import type { ReferralSettings, UserReferral, BasePaginationResponse } from '@/types'

/**
 * Get referral settings
 */
export function getReferralSettings(): Promise<ReferralSettings> {
  return apiClient.get('/admin/referral/settings')
}

/**
 * Update referral settings
 */
export function updateReferralSettings(settings: ReferralSettings): Promise<void> {
  return apiClient.put('/admin/referral/settings', settings)
}

/**
 * List all referral records (paginated)
 */
export function listReferrals(params?: {
  page?: number
  page_size?: number
}): Promise<BasePaginationResponse<UserReferral>> {
  return apiClient.get('/admin/referral/list', { params })
}
