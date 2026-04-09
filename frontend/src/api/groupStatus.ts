import { apiClient } from './client'
import type {
  GroupStatusEvent,
  GroupStatusHistoryBucket,
  GroupStatusListItem,
} from '@/types'

export async function listStatuses(): Promise<GroupStatusListItem[]> {
  const { data } = await apiClient.get<GroupStatusListItem[]>('/group-status')
  return data
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

export const groupStatusAPI = {
  listStatuses,
  getHistory,
  getEvents
}

export default groupStatusAPI
