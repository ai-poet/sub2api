/**
 * 工单图片附件的显示（fork 本地功能）。
 *
 * 附件内容端点和其它接口一样只认 Authorization 头，而浏览器给 <img src> 发请求时不会带
 * 这个头——所以不能把同源地址直接塞进 <img>（那样一律 401、渲染成裂图）。这里先用
 * apiClient 带鉴权取回字节，再 URL.createObjectURL 换成 blob: URL 交给 Markdown 渲染，
 * 组件卸载时逐个 revoke。
 *
 * 渲染侧放行 blob: 的口子见 utils/markdown.ts 的 allowBlobImages。
 */

import { onBeforeUnmount, reactive } from 'vue'
import { fetchAttachment as fetchUserAttachment } from '@/api/tickets'
import { fetchAttachment as fetchAdminAttachment } from '@/api/admin/tickets'
import { TICKET_ATTACHMENT_PLACEHOLDER, type TicketSide } from '@/utils/tickets'

export function useTicketAttachmentImages(side: TicketSide) {
  // reactive 的 Map / Set：get / has 会被模板收集为依赖，取回后自动重渲染。
  const urls = reactive(new Map<string, string>())
  const pending = reactive(new Set<string>())
  const failed = reactive(new Set<string>())

  const fetchAttachment = side === 'admin' ? fetchAdminAttachment : fetchUserAttachment

  async function load(key: string) {
    if (urls.has(key) || pending.has(key) || failed.has(key)) return
    pending.add(key)
    try {
      urls.set(key, URL.createObjectURL(await fetchAttachment(key)))
    } catch {
      // 取不到就记下来别重试。这里刻意不 toast：一条工单可能引用多张图，
      // 逐张报错只会刷屏；渲染侧会退回成不带 src 的 img，alt 仍然可见。
      failed.add(key)
    } finally {
      pending.delete(key)
    }
  }

  /** 预取正文里引用到的附件。重复 key 会被 load 自己挡掉。 */
  function ensure(keys: string[]) {
    for (const key of keys) void load(key)
  }

  /**
   * 给 rewriteTicketAttachmentUrls 用：已取回 → blob: URL；在取 → 透明占位（免得闪裂图）；
   * 取失败 → undefined，保持原样。
   */
  function resolve(key: string): string | undefined {
    const url = urls.get(key)
    if (url) return url
    return failed.has(key) ? undefined : TICKET_ATTACHMENT_PLACEHOLDER
  }

  onBeforeUnmount(() => {
    for (const url of urls.values()) URL.revokeObjectURL(url)
    urls.clear()
  })

  return { ensure, resolve }
}
