import { apiClient } from './client'
import type {
  Group,
  GroupStatusEvent,
  GroupStatusHistoryBucket,
  GroupStatusListItem,
  GroupStatusRecord,
} from '@/types'

function normalizeGroup(raw: any): Group {
  return {
    id: raw?.id ?? raw?.ID ?? 0,
    name: raw?.name ?? raw?.Name ?? '',
    description: raw?.description ?? raw?.Description ?? '',
    platform: raw?.platform ?? raw?.Platform ?? 'anthropic',
    rate_multiplier: raw?.rate_multiplier ?? raw?.RateMultiplier ?? 1,
    is_exclusive: raw?.is_exclusive ?? raw?.IsExclusive ?? false,
    status: raw?.status ?? raw?.Status ?? 'inactive',
    subscription_type: raw?.subscription_type ?? raw?.SubscriptionType ?? 'standard',
    daily_limit_usd: raw?.daily_limit_usd ?? raw?.DailyLimitUSD ?? null,
    weekly_limit_usd: raw?.weekly_limit_usd ?? raw?.WeeklyLimitUSD ?? null,
    monthly_limit_usd: raw?.monthly_limit_usd ?? raw?.MonthlyLimitUSD ?? null,
    image_price_1k: raw?.image_price_1k ?? raw?.ImagePrice1K ?? null,
    image_price_2k: raw?.image_price_2k ?? raw?.ImagePrice2K ?? null,
    image_price_4k: raw?.image_price_4k ?? raw?.ImagePrice4K ?? null,
    claude_code_only: raw?.claude_code_only ?? raw?.ClaudeCodeOnly ?? false,
    fallback_group_id: raw?.fallback_group_id ?? raw?.FallbackGroupID ?? null,
    fallback_group_id_on_invalid_request:
      raw?.fallback_group_id_on_invalid_request ?? raw?.FallbackGroupIDOnInvalidRequest ?? null,
    allow_messages_dispatch: raw?.allow_messages_dispatch ?? raw?.AllowMessagesDispatch ?? false,
    require_oauth_only: raw?.require_oauth_only ?? raw?.RequireOAuthOnly ?? false,
    require_privacy_set: raw?.require_privacy_set ?? raw?.RequirePrivacySet ?? false,
    created_at: raw?.created_at ?? raw?.CreatedAt ?? '',
    updated_at: raw?.updated_at ?? raw?.UpdatedAt ?? '',
  }
}

export async function listStatuses(): Promise<GroupStatusListItem[]> {
  const { data } = await apiClient.get<GroupStatusListItem[]>('/group-status')
  return data.map((item: any) => ({
    ...item,
    group: normalizeGroup(item.group)
  }))
}

export async function getHistory(
  groupId: number,
  period: '24h' | '7d' = '24h'
): Promise<GroupStatusHistoryBucket[]> {
  const { data } = await apiClient.get<GroupStatusHistoryBucket[]>(`/group-status/${groupId}/history`, {
    params: { period }
  })
  return data
}

export async function getEvents(
  groupId: number,
  limit: number = 20
): Promise<GroupStatusEvent[]> {
  const { data } = await apiClient.get<GroupStatusEvent[]>(`/group-status/${groupId}/events`, {
    params: { limit }
  })
  return data
}

export async function getRecentRecords(
  groupId: number,
  limit: number = 24
): Promise<GroupStatusRecord[]> {
  const { data } = await apiClient.get<GroupStatusRecord[]>(`/group-status/${groupId}/records`, {
    params: { limit }
  })
  return data
}

export const groupStatusAPI = {
  listStatuses,
  getHistory,
  getEvents,
  getRecentRecords
}

export default groupStatusAPI
