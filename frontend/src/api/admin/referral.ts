import { apiClient } from '../client'
import type {
  ReferralSettings,
  AffiliateLedgerEntry,
  AffiliateRelation,
  AffiliateDetail,
  BasePaginationResponse
} from '@/types'

export async function getReferralSettings(): Promise<ReferralSettings> {
  const { data } = await apiClient.get<ReferralSettings>('/admin/referral/settings')
  return data
}

export async function updateReferralSettings(settings: ReferralSettings): Promise<void> {
  await apiClient.put('/admin/referral/settings', settings)
}

export interface AffiliateRecordParams {
  page?: number
  page_size?: number
  search?: string
  user_id?: number
  start_at?: string
  end_at?: string
}

async function list<T>(path: string, params?: AffiliateRecordParams): Promise<BasePaginationResponse<T>> {
  const { data } = await apiClient.get<BasePaginationResponse<T>>(path, { params })
  return data
}

export const listReferrals = (params?: AffiliateRecordParams) => list<AffiliateRelation>('/admin/affiliates/invites', params)
export const listAffiliateRebates = (params?: AffiliateRecordParams) => list<AffiliateLedgerEntry>('/admin/affiliates/rebates', params)
export const listAffiliateTransfers = (params?: AffiliateRecordParams) => list<AffiliateLedgerEntry>('/admin/affiliates/transfers', params)

export async function getAffiliateUserOverview(userId: number): Promise<AffiliateDetail> {
  const { data } = await apiClient.get<AffiliateDetail>(`/admin/affiliates/users/${userId}/overview`)
  return data
}

export async function updateAffiliateUser(
  userId: number,
  input: { aff_code?: string; aff_rebate_rate_percent?: number; clear_rebate_rate?: boolean }
): Promise<AffiliateDetail> {
  const { data } = await apiClient.put<AffiliateDetail>(`/admin/affiliates/users/${userId}`, input)
  return data
}

export async function resetAffiliateUserCode(userId: number): Promise<string> {
  const { data } = await apiClient.delete<{ aff_code: string }>(`/admin/affiliates/users/${userId}/code`)
  return data.aff_code
}

export async function batchSetAffiliateRate(input: {
  user_ids: number[]
  aff_rebate_rate_percent?: number
  clear?: boolean
}): Promise<void> {
  await apiClient.post('/admin/affiliates/users/batch-rate', input)
}
