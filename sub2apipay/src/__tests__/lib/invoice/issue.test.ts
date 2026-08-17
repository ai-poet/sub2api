import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockUploadAttachment = vi.fn();
const mockDeleteAttachmentQuietly = vi.fn();
const mockSendInvoiceReadyEmail = vi.fn();
const mockFetchAttachment = vi.fn();

vi.mock('@/lib/sub2api/attachments', async () => {
  const actual = await vi.importActual<typeof import('@/lib/sub2api/attachments')>('@/lib/sub2api/attachments');
  return {
    AttachmentError: actual.AttachmentError,
    uploadAttachment: (...a: unknown[]) => mockUploadAttachment(...a),
    deleteAttachmentQuietly: (...a: unknown[]) => mockDeleteAttachmentQuietly(...a),
    sendInvoiceReadyEmail: (...a: unknown[]) => mockSendInvoiceReadyEmail(...a),
    fetchAttachment: (...a: unknown[]) => mockFetchAttachment(...a),
  };
});

const mockInvoiceFindUnique = vi.fn();
const mockInvoiceUpdate = vi.fn();
const mockInvoiceUpdateMany = vi.fn();
const mockInvoiceOrderFindUnique = vi.fn();
const mockAuditCreate = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    invoiceRequest: {
      findUnique: (...a: unknown[]) => mockInvoiceFindUnique(...a),
      update: (...a: unknown[]) => mockInvoiceUpdate(...a),
      updateMany: (...a: unknown[]) => mockInvoiceUpdateMany(...a),
    },
    invoiceRequestOrder: {
      findUnique: (...a: unknown[]) => mockInvoiceOrderFindUnique(...a),
    },
    auditLog: { create: (...a: unknown[]) => mockAuditCreate(...a) },
    systemConfig: { findMany: vi.fn().mockResolvedValue([]) },
  },
}));

/** 退款联动按明细表查发票：合并开票时退款订单可能不是主订单。 */
function mockLinkedInvoice(invoice: { id: string; status: string } | null) {
  mockInvoiceOrderFindUnique.mockResolvedValue(invoice ? { invoice } : null);
}

import { AttachmentError } from '@/lib/sub2api/attachments';
import { adminIssueInvoice, applyRefundToInvoice } from '@/lib/invoice/service';
import { InvoiceError } from '@/lib/invoice/types';

const PENDING_INVOICE = {
  id: 'inv-1',
  orderId: 'order-1',
  userId: 7,
  status: 'PENDING',
  titleName: '某某科技有限公司',
  taxNo: '91310000MA1FL1XXXX',
  amount: '128.00',
  fileKey: null,
  issuedAt: null,
};

const UPLOADED = {
  key: 'pay-attachments/invoice/2026/08/inv-1-abcd1234.pdf',
  file_name: '发票.pdf',
  size: 2048,
  content_type: 'application/pdf',
};

function pdfFile() {
  return {
    fileName: '发票.pdf',
    contentType: 'application/pdf',
    data: new TextEncoder().encode('%PDF-1.7\ntrailer\n%%EOF\n').buffer as ArrayBuffer,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  mockAuditCreate.mockResolvedValue({});
  mockInvoiceUpdate.mockResolvedValue({});
  mockUploadAttachment.mockResolvedValue(UPLOADED);
});

describe('adminIssueInvoice', () => {
  it('uploads, flips to ISSUED, and notifies the user', async () => {
    mockInvoiceFindUnique
      .mockResolvedValueOnce(PENDING_INVOICE) // 状态检查
      .mockResolvedValue({ ...PENDING_INVOICE, status: 'ISSUED', fileKey: UPLOADED.key, issuedAt: new Date() });
    mockInvoiceUpdateMany.mockResolvedValue({ count: 1 });
    mockSendInvoiceReadyEmail.mockResolvedValue(undefined);

    const result = await adminIssueInvoice({
      invoiceId: 'inv-1',
      file: pdfFile(),
      operator: 'admin',
      locale: 'zh',
    });

    expect(result.warning).toBeUndefined();
    expect(mockSendInvoiceReadyEmail).toHaveBeenCalledTimes(1);

    const emailArgs = mockSendInvoiceReadyEmail.mock.calls[0][0];
    // fileKey 参与投递去重键，否则「重新发送通知」会被投递标记永久静默。
    expect(emailArgs.reminderKey).toBe(UPLOADED.key);
    // 金额按人民币展示。
    expect(emailArgs.amountDisplay).toBe('¥128.00');
    // 收件邮箱与跳转链接都不由这里决定，避免内部接口变成定向发信中继。
    expect(emailArgs).not.toHaveProperty('email');
    expect(emailArgs).not.toHaveProperty('invoiceUrl');
  });

  // 文件必须先上传才能拿到 fileKey 写库；输掉状态竞争时要把孤儿对象删掉。
  it('deletes the orphaned object when another admin wins the race', async () => {
    mockInvoiceFindUnique.mockResolvedValue(PENDING_INVOICE);
    mockInvoiceUpdateMany.mockResolvedValue({ count: 0 });

    await expect(
      adminIssueInvoice({ invoiceId: 'inv-1', file: pdfFile(), operator: 'admin', locale: 'zh' }),
    ).rejects.toMatchObject({ code: 'CONFLICT', statusCode: 409 });

    expect(mockDeleteAttachmentQuietly).toHaveBeenCalledWith(UPLOADED.key);
    expect(mockSendInvoiceReadyEmail).not.toHaveBeenCalled();
  });

  // SMTP 抖动绝不能回滚一张已经开好的发票。
  it('keeps the invoice ISSUED and returns a warning when the email fails', async () => {
    mockInvoiceFindUnique
      .mockResolvedValueOnce(PENDING_INVOICE)
      .mockResolvedValue({ ...PENDING_INVOICE, status: 'ISSUED', fileKey: UPLOADED.key, issuedAt: new Date() });
    mockInvoiceUpdateMany.mockResolvedValue({ count: 1 });
    mockSendInvoiceReadyEmail.mockRejectedValue(new Error('smtp down'));

    const result = await adminIssueInvoice({
      invoiceId: 'inv-1',
      file: pdfFile(),
      operator: 'admin',
      locale: 'zh',
    });

    expect(result.warning).toContain('通知邮件发送失败');
    // notifiedAt 保持 NULL（没有被写成已通知），并留下审计记录供排查。
    expect(mockInvoiceUpdate).not.toHaveBeenCalled();
    expect(mockAuditCreate.mock.calls.some(([arg]) => arg.data.action === 'INVOICE_NOTIFY_FAILED')).toBe(true);
  });

  it('surfaces unconfigured object storage as an actionable error', async () => {
    mockInvoiceFindUnique.mockResolvedValue(PENDING_INVOICE);
    mockUploadAttachment.mockRejectedValue(
      new AttachmentError('PAY_ATTACHMENT_STORAGE_NOT_CONFIGURED', 'not configured', 400),
    );

    await expect(
      adminIssueInvoice({ invoiceId: 'inv-1', file: pdfFile(), operator: 'admin', locale: 'zh' }),
    ).rejects.toMatchObject({ code: 'INVOICE_STORAGE_NOT_CONFIGURED', statusCode: 503 });
  });

  it('rejects a disallowed file type before uploading anything', async () => {
    mockInvoiceFindUnique.mockResolvedValue(PENDING_INVOICE);

    await expect(
      adminIssueInvoice({
        invoiceId: 'inv-1',
        file: { fileName: 'x.html', contentType: 'text/html', data: new ArrayBuffer(16) },
        operator: 'admin',
        locale: 'zh',
      }),
    ).rejects.toBeInstanceOf(InvoiceError);
    expect(mockUploadAttachment).not.toHaveBeenCalled();
  });

  it('refuses to issue an invoice that is no longer pending', async () => {
    mockInvoiceFindUnique.mockResolvedValue({ ...PENDING_INVOICE, status: 'ISSUED' });

    await expect(
      adminIssueInvoice({ invoiceId: 'inv-1', file: pdfFile(), operator: 'admin', locale: 'zh' }),
    ).rejects.toMatchObject({ statusCode: 409 });
    expect(mockUploadAttachment).not.toHaveBeenCalled();
  });
});

describe('applyRefundToInvoice', () => {
  it('cancels a pending invoice', async () => {
    mockLinkedInvoice({ id: 'inv-1', status: 'PENDING' });
    mockInvoiceUpdateMany.mockResolvedValue({ count: 1 });

    const impact = await applyRefundToInvoice('order-1');
    expect(impact).toEqual({ needsCreditNote: false, cancelledPending: true });
  });

  // 已开具的发票不自动作废：红冲是线下动作，这里只提示。
  it('flags an issued invoice as needing a credit note', async () => {
    mockLinkedInvoice({ id: 'inv-1', status: 'ISSUED' });

    const impact = await applyRefundToInvoice('order-1');
    expect(impact).toEqual({ needsCreditNote: true, cancelledPending: false });
    expect(mockInvoiceUpdateMany).not.toHaveBeenCalled();
    expect(mockAuditCreate.mock.calls.some(([a]) => a.data.action === 'INVOICE_NEEDS_CREDIT_NOTE')).toBe(true);
  });

  // 退款绝不能因为发票副作用而失败。
  it('swallows database errors so a refund never fails because of invoicing', async () => {
    mockInvoiceOrderFindUnique.mockRejectedValue(new Error('db down'));

    await expect(applyRefundToInvoice('order-1')).resolves.toEqual({
      needsCreditNote: false,
      cancelledPending: false,
    });
  });

  it('is a no-op when the order has no invoice', async () => {
    mockLinkedInvoice(null);
    await expect(applyRefundToInvoice('order-1')).resolves.toEqual({
      needsCreditNote: false,
      cancelledPending: false,
    });
  });
});
