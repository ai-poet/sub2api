/**
 * Admin Referral API endpoints
 */

import { apiClient } from '../client'
import type { ReferralSettings, UserReferral, BasePaginationResponse } from '@/types'

/**
 * Get referral settings
 */
export async function getReferralSettings(): Promise<ReferralSettings> {
  const { data } = await apiClient.get<ReferralSettings>('/admin/referral/settings')
  return data
}

/**
 * Update referral settings
 */
export async function updateReferralSettings(settings: ReferralSettings): Promise<void> {
  await apiClient.put('/admin/referral/settings', settings)
}

/**
 * List all referral records (paginated)
 */
export async function listReferrals(params?: {
  page?: number
  page_size?: number
}): Promise<BasePaginationResponse<UserReferral>> {
  const { data } = await apiClient.get<BasePaginationResponse<UserReferral>>('/admin/referral/list', { params })
  return data
}
