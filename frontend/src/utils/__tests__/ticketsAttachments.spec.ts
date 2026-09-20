import { describe, expect, it } from 'vitest'
import {
  TICKET_ATTACHMENT_MAX_SIZE,
  rewriteTicketAttachmentUrls,
  ticketAttachmentContentUrl,
  ticketAttachmentMarkdown
} from '@/utils/tickets'

describe('ticket attachment helpers', () => {
  it('builds the markdown image for an attachment key', () => {
    expect(ticketAttachmentMarkdown('tickets/1/a-b_c.png')).toBe('![image](ticket-attachment://tickets/1/a-b_c.png)')
  })

  it('builds the side-specific content URL with an encoded key', () => {
    expect(ticketAttachmentContentUrl('tickets/1/a.png', 'user')).toBe(
      '/api/v1/tickets/attachments/content?key=tickets%2F1%2Fa.png'
    )
    expect(ticketAttachmentContentUrl('tickets/1/a.png', 'admin')).toBe(
      '/api/v1/admin/tickets/attachments/content?key=tickets%2F1%2Fa.png'
    )
  })

  it('rewrites ticket-attachment urls to the user-side content endpoint', () => {
    const body = '看这张图 ![image](ticket-attachment://tickets/1/a-b_c.png) 和 ![image](ticket-attachment://x/y.jpg) 结尾'
    expect(rewriteTicketAttachmentUrls(body, 'user')).toBe(
      '看这张图 ![image](/api/v1/tickets/attachments/content?key=tickets%2F1%2Fa-b_c.png) 和 ' +
        '![image](/api/v1/tickets/attachments/content?key=x%2Fy.jpg) 结尾'
    )
  })

  it('rewrites ticket-attachment urls to the admin-side content endpoint', () => {
    expect(rewriteTicketAttachmentUrls('![image](ticket-attachment://k)', 'admin')).toBe(
      '![image](/api/v1/admin/tickets/attachments/content?key=k)'
    )
  })

  it('leaves keys with characters outside [A-Za-z0-9/._-] untouched', () => {
    const body = '![image](ticket-attachment://bad?key=1) ![image](ticket-attachment://ok_1.png)'
    expect(rewriteTicketAttachmentUrls(body, 'user')).toBe(
      '![image](ticket-attachment://bad?key=1) ![image](/api/v1/tickets/attachments/content?key=ok_1.png)'
    )
  })

  it('returns empty input unchanged', () => {
    expect(rewriteTicketAttachmentUrls('', 'user')).toBe('')
  })

  it('caps attachments at 5MiB', () => {
    expect(TICKET_ATTACHMENT_MAX_SIZE).toBe(5 * 1024 * 1024)
  })
})
