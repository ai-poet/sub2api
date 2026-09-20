import { describe, expect, it } from 'vitest'
import {
  TICKET_ATTACHMENT_MAX_SIZE,
  TICKET_ATTACHMENT_PLACEHOLDER,
  rewriteTicketAttachmentUrls,
  ticketAttachmentContentPath,
  ticketAttachmentKeys,
  ticketAttachmentMarkdown
} from '@/utils/tickets'

/** 测试用的解析器：按 key → URL 的映射回答，没命中就当作取不回来。 */
function resolver(map: Record<string, string>) {
  return (key: string) => map[key]
}

describe('ticket attachment helpers', () => {
  it('builds the markdown image for an attachment key', () => {
    expect(ticketAttachmentMarkdown('tickets/1/a-b_c.png')).toBe('![image](ticket-attachment://tickets/1/a-b_c.png)')
  })

  it('exposes the side-specific content path for the api client', () => {
    expect(ticketAttachmentContentPath('user')).toBe('/tickets/attachments/content')
    expect(ticketAttachmentContentPath('admin')).toBe('/admin/tickets/attachments/content')
  })

  it('collects the distinct attachment keys referenced by a body', () => {
    const body = '![image](ticket-attachment://a.png) 再来一张 ![image](ticket-attachment://b/c.jpg) 和重复的 ![image](ticket-attachment://a.png)'
    expect(ticketAttachmentKeys(body)).toEqual(['a.png', 'b/c.jpg'])
    expect(ticketAttachmentKeys('')).toEqual([])
    expect(ticketAttachmentKeys('没有图')).toEqual([])
  })

  it('rewrites ticket-attachment urls to the resolved object urls', () => {
    const body = '看这张图 ![image](ticket-attachment://tickets/1/a-b_c.png) 和 ![image](ticket-attachment://x/y.jpg) 结尾'
    const rewritten = rewriteTicketAttachmentUrls(
      body,
      resolver({ 'tickets/1/a-b_c.png': 'blob:http://localhost/one', 'x/y.jpg': 'blob:http://localhost/two' })
    )
    expect(rewritten).toBe('看这张图 ![image](blob:http://localhost/one) 和 ![image](blob:http://localhost/two) 结尾')
  })

  it('leaves the original url in place when the attachment cannot be resolved', () => {
    expect(rewriteTicketAttachmentUrls('![image](ticket-attachment://k)', resolver({}))).toBe(
      '![image](ticket-attachment://k)'
    )
  })

  it('renders a transparent placeholder while an attachment is still loading', () => {
    const rewritten = rewriteTicketAttachmentUrls(
      '![image](ticket-attachment://k)',
      resolver({ k: TICKET_ATTACHMENT_PLACEHOLDER })
    )
    expect(rewritten).toBe(`![image](${TICKET_ATTACHMENT_PLACEHOLDER})`)
    expect(TICKET_ATTACHMENT_PLACEHOLDER.startsWith('data:image/png;base64,')).toBe(true)
  })

  it('leaves keys with characters outside [A-Za-z0-9/._-] untouched', () => {
    const body = '![image](ticket-attachment://bad?key=1) ![image](ticket-attachment://ok_1.png)'
    expect(ticketAttachmentKeys(body)).toEqual(['ok_1.png'])
    expect(rewriteTicketAttachmentUrls(body, resolver({ 'ok_1.png': 'blob:http://localhost/ok' }))).toBe(
      '![image](ticket-attachment://bad?key=1) ![image](blob:http://localhost/ok)'
    )
  })

  it('returns empty input unchanged', () => {
    expect(rewriteTicketAttachmentUrls('', resolver({}))).toBe('')
  })

  it('caps attachments at 5MiB', () => {
    expect(TICKET_ATTACHMENT_MAX_SIZE).toBe(5 * 1024 * 1024)
  })
})
