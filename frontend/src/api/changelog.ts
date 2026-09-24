/**
 * 官网更新日志 —— 桌面客户端 GitHub Releases 的缓存副本（fork 本地）。
 *
 * 后端按后台设置的仓库拉取 release，缓存 15 分钟；这个接口不需要登录且永不返回 401，
 * GitHub 不可用时返回空列表。
 */

import { apiClient } from './client'

export interface ClientChangelogEntry {
  /** 去掉 v 前缀的版本号，如 0.2.1 */
  version: string
  /** RFC 3339 发布时间；缺失时为空串 */
  published_at: string
  /** release 名称不只是版本号时才有值 */
  title: string
  /** release 说明拆出的条目，每条都是 Markdown */
  items: string[]
}

export interface ClientChangelogResponse {
  entries: ClientChangelogEntry[]
}

/** 拉取客户端更新日志（无需认证）。 */
export async function getClientChangelog(options?: {
  signal?: AbortSignal
}): Promise<ClientChangelogEntry[]> {
  const { data } = await apiClient.get<ClientChangelogResponse>('/changelog', {
    signal: options?.signal
  })
  return data?.entries ?? []
}
