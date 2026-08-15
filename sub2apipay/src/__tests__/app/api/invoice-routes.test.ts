import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';

const mockGetCurrentUserByToken = vi.fn();
vi.mock('@/lib/sub2api/client', () => ({
  getCurrentUserByToken: (...args: unknown[]) => mockGetCurrentUserByToken(...args),
}));

const mockPresignAttachment = vi.fn();
const mockUploadAttachment = vi.fn();
const mockDeleteAttachmentQuietly = vi.fn();
const mockSendInvoiceReadyEmail = vi.fn();
vi.mock('@/lib/sub2api/attachments', async () => {
  const actual = await vi.importActual<typeof import('@/lib/sub2api/attachments')>('@/lib/sub2api/attachments');
  return {
    AttachmentError: actual.AttachmentError,
    presignAttachment: (...args: unknown[]) => mockPresignAttachment(...args),
    uploadAttachment: (...args: unknown[]) => mockUploadAttachment(...args),
    deleteAttachmentQuietly: (...args: unknown[]) => mockDeleteAttachmentQuietly(...args),
    sendInvoiceReadyEmail: (...args: unknown[]) => mockSendInvoiceReadyEmail(...args),
  };
});

const mockOrderFindUnique = vi.fn();
const mockInvoiceFindUnique = vi.fn();
const mockInvoiceFindUniqueOrThrow = vi.fn();
const mockInvoiceFindFirst = vi.fn();
const mockInvoiceCreate = vi.fn();
const mockInvoiceCount = vi.fn();
const mockInvoiceUpdate = vi.fn();
const mockInvoiceUpdateMany = vi.fn();
const mockTitleUpsert = vi.fn();
const mockAuditCreate = vi.fn();
const mockSystemConfigFindMany = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    order: { findUnique: (...a: unknown[]) => mockOrderFindUnique(...a) },
    invoiceRequest: {
      findUnique: (...a: unknown[]) => mockInvoiceFindUnique(...a),
      findUniqueOrThrow: (...a: unknown[]) => mockInvoiceFindUniqueOrThrow(...a),
      findFirst: (...a: unknown[]) => mockInvoiceFindFirst(...a),
      create: (...a: unknown[]) => mockInvoiceCreate(...a),
      count: (...a: unknown[]) => mockInvoiceCount(...a),
      update: (...a: unknown[]) => mockInvoiceUpdate(...a),
      updateMany: (...a: unknown[]) => mockInvoiceUpdateMany(...a),
    },
    invoiceTitle: { upsert: (...a: unknown[]) => mockTitleUpsert(...a) },
    auditLog: { create: (...a: unknown[]) => mockAuditCreate(...a) },
    systemConfig: { findMany: (...a: unknown[]) => mockSystemConfigFindMany(...a) },
  },
}));

import { POST as requestInvoiceRoute } from '@/app/api/orders/[id]/invoice-request/route';
import { GET as downloadInvoiceRoute } from '@/app/api/invoices/[id]/download/route';
import { invalidateConfigCache } from '@/lib/system-config';

const USER = { id: 7, username: 'alice', email: 'alice@example.com', balance: 10 };

function enableInvoicing() {
  mockSystemConfigFindMany.mockResolvedValue([
    { key: 'invoice_enabled', value: 'true' },
    { key: 'invoice_max_age_days', value: '180' },
    { key: 'invoice_daily_request_limit', value: '20' },
  ]);
}

function eligibleOrder(overrides: Record<string, unknown> = {}) {
  return {
    id: 'order-1',
    userId: USER.id,
    status: 'COMPLETED',
    paymentType: 'alipay',
    paidAt: new Date(),
    payAmount: '128.00',
    refundAmount: null,
    ...overrides,
  };
}

function postRequest(body: unknown, params?: Record<string, string>) {
  const qs = new URLSearchParams({ token: 'tok', ...params });
  return new NextRequest(`https://pay.example.com/api/orders/order-1/invoice-request?${qs}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

const VALID_BODY = { title_name: '某某科技有限公司', tax_no: '91310000MA1FL1XXXX' };

beforeEach(() => {
  vi.clearAllMocks();
  invalidateConfigCache();
  mockGetCurrentUserByToken.mockResolvedValue(USER);
  mockInvoiceCount.mockResolvedValue(0);
  mockInvoiceFindUnique.mockResolvedValue(null);
  mockTitleUpsert.mockResolvedValue({});
  mockAuditCreate.mockResolvedValue({});
  enableInvoicing();
});

describe('POST /api/orders/[id]/invoice-request', () => {
  const params = Promise.resolve({ id: 'order-1' });

  it('requires a token', async () => {
    const req = new NextRequest('https://pay.example.com/api/orders/order-1/invoice-request', {
      method: 'POST',
      body: JSON.stringify(VALID_BODY),
    });
    const res = await requestInvoiceRoute(req, { params });
    expect(res.status).toBe(401);
  });

  it('creates a pending invoice for an eligible order', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    mockInvoiceCreate.mockResolvedValue({
      id: 'inv-1',
      orderId: 'order-1',
      userId: USER.id,
      status: 'PENDING',
      titleName: VALID_BODY.title_name,
      taxNo: VALID_BODY.tax_no,
      remark: null,
      contactEmail: null,
      amount: '128.00',
      fileKey: null,
      fileName: null,
      rejectReason: null,
      issuedAt: null,
      createdAt: new Date(),
    });

    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(200);
    const data = await res.json();
    expect(data.invoice.status).toBe('PENDING');
    // 开票金额必须取实付人民币 payAmount，而不是到账美元 amount。
    expect(data.invoice.amount).toBe(128);
    // 抬头被记住，供下次自动回填。
    expect(mockTitleUpsert).toHaveBeenCalledTimes(1);
  });

  // 归属检查必须先于资格检查，否则错误码会泄露「这张订单存在」。
  it("refuses another user's order with the same 404 as a missing one", async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder({ userId: 999 }));
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(404);
    expect(mockInvoiceCreate).not.toHaveBeenCalled();

    mockOrderFindUnique.mockResolvedValue(null);
    const missing = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(missing.status).toBe(404);
    expect(await missing.json()).toEqual(await res.json());
  });

  it('rejects a stablecoin order', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder({ paymentType: 'usdt.polygon' }));
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(400);
    expect((await res.json()).code).toBe('STABLECOIN_NOT_SUPPORTED');
  });

  it('returns 409 when an invoice already exists in a non-re-requestable state', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    mockInvoiceFindUnique.mockResolvedValue({ id: 'inv-1', status: 'ISSUED' });
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(409);
    expect((await res.json()).code).toBe('ALREADY_REQUESTED');
  });

  it('reuses the existing row when re-requesting after a rejection', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    mockInvoiceFindUnique.mockResolvedValue({ id: 'inv-1', status: 'REJECTED' });
    mockInvoiceUpdateMany.mockResolvedValue({ count: 1 });
    mockInvoiceFindUniqueOrThrow.mockResolvedValue({
      id: 'inv-1',
      orderId: 'order-1',
      userId: USER.id,
      status: 'PENDING',
      titleName: VALID_BODY.title_name,
      taxNo: VALID_BODY.tax_no,
      remark: null,
      contactEmail: null,
      amount: '128.00',
      fileKey: null,
      fileName: null,
      rejectReason: null,
      issuedAt: null,
      createdAt: new Date(),
    });

    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(200);
    // 复用同一行而不是新建：「一单一票」由 orderId 唯一约束兜底。
    expect(mockInvoiceCreate).not.toHaveBeenCalled();
    expect(mockInvoiceUpdateMany).toHaveBeenCalledTimes(1);
    // 上一轮的驳回原因与文件必须被清空，否则会串到新申请上。
    const data = mockInvoiceUpdateMany.mock.calls[0][0].data;
    expect(data.status).toBe('PENDING');
    expect(data.rejectReason).toBeNull();
    expect(data.fileKey).toBeNull();
  });

  it('returns 409 when the re-request loses the race with an admin', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    mockInvoiceFindUnique.mockResolvedValue({ id: 'inv-1', status: 'REJECTED' });
    mockInvoiceUpdateMany.mockResolvedValue({ count: 0 });

    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(409);
  });

  it('rejects a malformed tax number before touching the database', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    const res = await requestInvoiceRoute(postRequest({ ...VALID_BODY, tax_no: 'too-short' }), { params });
    expect(res.status).toBe(400);
    expect(mockOrderFindUnique).not.toHaveBeenCalled();
  });

  it('is disabled entirely when invoice_enabled is not true', async () => {
    mockSystemConfigFindMany.mockResolvedValue([]);
    invalidateConfigCache();
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(400);
    expect((await res.json()).code).toBe('FEATURE_DISABLED');
  });

  it('enforces the daily request limit', async () => {
    mockOrderFindUnique.mockResolvedValue(eligibleOrder());
    mockInvoiceCount.mockResolvedValue(20);
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(429);
  });
});

describe('GET /api/invoices/[id]/download', () => {
  const params = Promise.resolve({ id: 'inv-1' });

  function downloadRequest() {
    return new NextRequest('https://pay.example.com/api/invoices/inv-1/download?token=tok');
  }

  it('redirects to a presigned URL for the owner', async () => {
    mockInvoiceFindFirst.mockResolvedValue({
      id: 'inv-1',
      orderId: 'order-1',
      fileKey: 'pay-attachments/invoice/2026/08/inv-1-abcd.pdf',
      fileName: '发票.pdf',
    });
    mockPresignAttachment.mockResolvedValue({ url: 'https://s3.example/signed', expiresAt: '2026-08-15T00:05:00Z' });
    mockInvoiceUpdate.mockResolvedValue({});

    const res = await downloadInvoiceRoute(downloadRequest(), { params });
    expect(res.status).toBe(302);
    expect(res.headers.get('location')).toBe('https://s3.example/signed');
    // 预签名链接是短期凭证，不能被任何一层缓存留存。
    expect(res.headers.get('cache-control')).toContain('no-store');
  });

  // 这是整个功能里最关键的一条越权防线：归属与状态都写在查询条件里。
  it("does not redirect for another user's invoice", async () => {
    mockInvoiceFindFirst.mockResolvedValue(null);
    const res = await downloadInvoiceRoute(downloadRequest(), { params });
    expect(res.status).toBe(404);
    expect(res.headers.get('location')).toBeNull();
    expect(mockPresignAttachment).not.toHaveBeenCalled();

    const where = mockInvoiceFindFirst.mock.calls[0][0].where;
    expect(where.userId).toBe(USER.id);
    expect(where.status).toBe('ISSUED');
  });

  it('requires a token', async () => {
    const res = await downloadInvoiceRoute(
      new NextRequest('https://pay.example.com/api/invoices/inv-1/download'),
      { params },
    );
    expect(res.status).toBe(401);
    expect(mockPresignAttachment).not.toHaveBeenCalled();
  });
});
