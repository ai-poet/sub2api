import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';

const mockGetCurrentUserByToken = vi.fn();
vi.mock('@/lib/sub2api/client', () => ({
  getCurrentUserByToken: (...args: unknown[]) => mockGetCurrentUserByToken(...args),
}));

const mockFetchAttachment = vi.fn();
const mockUploadAttachment = vi.fn();
const mockDeleteAttachmentQuietly = vi.fn();
const mockSendInvoiceReadyEmail = vi.fn();
vi.mock('@/lib/sub2api/attachments', async () => {
  const actual = await vi.importActual<typeof import('@/lib/sub2api/attachments')>('@/lib/sub2api/attachments');
  return {
    AttachmentError: actual.AttachmentError,
    fetchAttachment: (...args: unknown[]) => mockFetchAttachment(...args),
    uploadAttachment: (...args: unknown[]) => mockUploadAttachment(...args),
    deleteAttachmentQuietly: (...args: unknown[]) => mockDeleteAttachmentQuietly(...args),
    sendInvoiceReadyEmail: (...args: unknown[]) => mockSendInvoiceReadyEmail(...args),
  };
});

const mockOrderFindMany = vi.fn();
const mockInvoiceFindUnique = vi.fn();
const mockInvoiceFindFirst = vi.fn();
const mockInvoiceCreate = vi.fn();
const mockInvoiceCount = vi.fn();
const mockInvoiceUpdate = vi.fn();
const mockInvoiceUpdateMany = vi.fn();
const mockInvoiceDeleteMany = vi.fn();
const mockInvoiceOrderFindMany = vi.fn();
const mockTitleUpsert = vi.fn();
const mockAuditCreate = vi.fn();
const mockSystemConfigFindMany = vi.fn();

vi.mock('@/lib/db', () => {
  const invoiceRequest = {
    findUnique: (...a: unknown[]) => mockInvoiceFindUnique(...a),
    findFirst: (...a: unknown[]) => mockInvoiceFindFirst(...a),
    create: (...a: unknown[]) => mockInvoiceCreate(...a),
    count: (...a: unknown[]) => mockInvoiceCount(...a),
    update: (...a: unknown[]) => mockInvoiceUpdate(...a),
    updateMany: (...a: unknown[]) => mockInvoiceUpdateMany(...a),
    deleteMany: (...a: unknown[]) => mockInvoiceDeleteMany(...a),
  };
  const client = {
    order: { findMany: (...a: unknown[]) => mockOrderFindMany(...a) },
    invoiceRequest,
    invoiceRequestOrder: { findMany: (...a: unknown[]) => mockInvoiceOrderFindMany(...a) },
    invoiceTitle: { upsert: (...a: unknown[]) => mockTitleUpsert(...a) },
    auditLog: { create: (...a: unknown[]) => mockAuditCreate(...a) },
    systemConfig: { findMany: (...a: unknown[]) => mockSystemConfigFindMany(...a) },
    // 申请开票在事务里删旧申请 + 建新申请；测试直接把同一个 client 当作 tx。
    $transaction: (fn: (tx: unknown) => unknown) => fn(client),
  };
  return { prisma: client };
});

import { POST as requestInvoiceRoute } from '@/app/api/orders/[id]/invoice-request/route';
import { POST as mergedInvoiceRoute } from '@/app/api/invoices/requests/route';
import { GET as downloadInvoiceRoute } from '@/app/api/invoices/[id]/download/route';
import { invalidateConfigCache } from '@/lib/system-config';

const USER = { id: 7, username: 'alice', email: 'alice@example.com', balance: 10 };

function enableInvoicing(overrides: Record<string, string> = {}) {
  const configs: Record<string, string> = {
    invoice_enabled: 'true',
    invoice_max_age_days: '180',
    invoice_daily_request_limit: '20',
    invoice_min_amount: '100',
    ...overrides,
  };
  mockSystemConfigFindMany.mockResolvedValue(
    Object.entries(configs).map(([key, value]) => ({ key, value })),
  );
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

function mergedPostRequest(body: unknown, params?: Record<string, string>) {
  const qs = new URLSearchParams({ token: 'tok', ...params });
  return new NextRequest(`https://pay.example.com/api/invoices/requests?${qs}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

/** 明细表里已存在的开票记录（决定订单能否再次开票）。 */
function linkedInvoice(orderId: string, invoice: { id: string; status: string }) {
  return { orderId, invoice };
}

const VALID_BODY = { title_name: '某某科技有限公司', tax_no: '91310000MA1FL1XXXX' };

function createdInvoice(overrides: Record<string, unknown> = {}) {
  return {
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
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  invalidateConfigCache();
  mockGetCurrentUserByToken.mockResolvedValue(USER);
  mockInvoiceCount.mockResolvedValue(0);
  mockInvoiceFindUnique.mockResolvedValue(null);
  mockInvoiceOrderFindMany.mockResolvedValue([]);
  mockInvoiceDeleteMany.mockResolvedValue({ count: 0 });
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
    mockOrderFindMany.mockResolvedValue([eligibleOrder()]);
    mockInvoiceCreate.mockResolvedValue(createdInvoice());

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
    mockOrderFindMany.mockResolvedValue([eligibleOrder({ userId: 999 })]);
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(404);
    expect(mockInvoiceCreate).not.toHaveBeenCalled();

    mockOrderFindMany.mockResolvedValue([]);
    const missing = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(missing.status).toBe(404);
    expect(await missing.json()).toEqual(await res.json());
  });

  it('rejects a stablecoin order', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder({ paymentType: 'usdt.polygon' })]);
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(400);
    expect((await res.json()).code).toBe('STABLECOIN_NOT_SUPPORTED');
  });

  it('returns 409 when an invoice already exists in a non-re-requestable state', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder()]);
    mockInvoiceOrderFindMany.mockResolvedValue([linkedInvoice('order-1', { id: 'inv-1', status: 'ISSUED' })]);
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(409);
    expect((await res.json()).code).toBe('ALREADY_REQUESTED');
  });

  it('supersedes the dead request when re-requesting after a rejection', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder()]);
    mockInvoiceOrderFindMany.mockResolvedValue([linkedInvoice('order-1', { id: 'inv-1', status: 'REJECTED' })]);
    mockInvoiceDeleteMany.mockResolvedValue({ count: 1 });
    mockInvoiceCreate.mockResolvedValue(createdInvoice({ id: 'inv-2' }));

    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(200);
    // 旧的驳回申请整行删掉，明细行随之级联清理，订单才能重新进新发票。
    expect(mockInvoiceDeleteMany).toHaveBeenCalledTimes(1);
    const deleteWhere = mockInvoiceDeleteMany.mock.calls[0][0].where;
    expect(deleteWhere.id.in).toEqual(['inv-1']);
    expect(deleteWhere.status.in).toContain('REJECTED');
    expect(mockInvoiceCreate).toHaveBeenCalledTimes(1);
  });

  it('rejects a malformed tax number before touching the database', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder()]);
    const res = await requestInvoiceRoute(postRequest({ ...VALID_BODY, tax_no: 'too-short' }), { params });
    expect(res.status).toBe(400);
    expect(mockOrderFindMany).not.toHaveBeenCalled();
  });

  it('is disabled entirely when invoice_enabled is not true', async () => {
    mockSystemConfigFindMany.mockResolvedValue([]);
    invalidateConfigCache();
    mockOrderFindMany.mockResolvedValue([eligibleOrder()]);
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(400);
    expect((await res.json()).code).toBe('FEATURE_DISABLED');
  });

  it('enforces the daily request limit', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder()]);
    mockInvoiceCount.mockResolvedValue(20);
    const res = await requestInvoiceRoute(postRequest(VALID_BODY), { params });
    expect(res.status).toBe(429);
  });
});

describe('POST /api/invoices/requests (合并开票)', () => {
  it('merges several orders into one invoice and sums their paid amounts', async () => {
    const orders = [
      eligibleOrder({ id: 'order-2', paidAt: new Date('2026-08-02'), payAmount: '30.00' }),
      eligibleOrder({ id: 'order-1', paidAt: new Date('2026-08-01'), payAmount: '128.00' }),
    ];
    mockOrderFindMany.mockResolvedValue(orders);
    mockInvoiceCreate.mockResolvedValue(createdInvoice({ amount: '158.00' }));

    const res = await mergedInvoiceRoute(
      mergedPostRequest({ ...VALID_BODY, order_ids: ['order-2', 'order-1'] }),
    );

    expect(res.status).toBe(200);
    const created = mockInvoiceCreate.mock.calls[0][0].data;
    // 主订单取最早付款的那张，明细覆盖全部订单，金额为各单实付之和。
    expect(created.orderId).toBe('order-1');
    expect(created.orders.create.map((line: { orderId: string }) => line.orderId)).toEqual(['order-1', 'order-2']);
    expect(Number(created.amount)).toBe(158);
    // 每张订单各记一条审计，按订单号回溯时不会漏掉合并进来的那些。
    expect(mockAuditCreate.mock.calls.filter(([a]) => a.data.action === 'INVOICE_REQUESTED')).toHaveLength(2);
  });

  it('rejects the whole batch when one order is not invoiceable', async () => {
    mockOrderFindMany.mockResolvedValue([
      eligibleOrder({ id: 'order-1' }),
      eligibleOrder({ id: 'order-2', paymentType: 'usdt.polygon' }),
    ]);

    const res = await mergedInvoiceRoute(
      mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1', 'order-2'] }),
    );

    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.code).toBe('STABLECOIN_NOT_SUPPORTED');
    // 报出是哪一张不合格，否则用户无从下手。
    expect(body.error).toContain('order-2');
    expect(mockInvoiceCreate).not.toHaveBeenCalled();
  });

  // 起开金额按合计判定：小额订单单开不了，攒够就能开。
  it('rejects a total below the configured minimum amount', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder({ id: 'order-1', payAmount: '30.00' })]);

    const res = await mergedInvoiceRoute(mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1'] }));

    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.code).toBe('INVOICE_BELOW_MIN_AMOUNT');
    // 提示里要带上还差多少，否则用户不知道该再勾几张。
    expect(body.error).toContain('100.00');
    expect(body.error).toContain('30.00');
    expect(mockInvoiceCreate).not.toHaveBeenCalled();
  });

  it('accepts small orders once merging reaches the minimum amount', async () => {
    mockOrderFindMany.mockResolvedValue([
      eligibleOrder({ id: 'order-1', paidAt: new Date('2026-08-01'), payAmount: '60.00' }),
      eligibleOrder({ id: 'order-2', paidAt: new Date('2026-08-02'), payAmount: '40.00' }),
    ]);
    mockInvoiceCreate.mockResolvedValue(createdInvoice({ amount: '100.00' }));

    const res = await mergedInvoiceRoute(
      mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1', 'order-2'] }),
    );

    expect(res.status).toBe(200);
    expect(Number(mockInvoiceCreate.mock.calls[0][0].data.amount)).toBe(100);
  });

  it('skips the minimum-amount check when it is configured as 0', async () => {
    enableInvoicing({ invoice_min_amount: '0' });
    invalidateConfigCache();
    mockOrderFindMany.mockResolvedValue([eligibleOrder({ id: 'order-1', payAmount: '5.00' })]);
    mockInvoiceCreate.mockResolvedValue(createdInvoice({ amount: '5.00' }));

    const res = await mergedInvoiceRoute(mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1'] }));

    expect(res.status).toBe(200);
  });

  // 收票邮箱是唯一没有客户端强校验的字段，服务端报错必须点名它，而不是一句裸的「参数错误」。
  it('names the contact email field when it is not a valid address', async () => {
    const res = await mergedInvoiceRoute(
      mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1'], contact_email: '13800138000' }),
    );

    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.error).toContain('收票邮箱');
    expect(body.details.contact_email).toBeDefined();
    expect(mockOrderFindMany).not.toHaveBeenCalled();
  });

  it('treats an empty contact email string as absent', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder({ id: 'order-1', payAmount: '128.00' })]);
    mockInvoiceCreate.mockResolvedValue(createdInvoice());

    const res = await mergedInvoiceRoute(
      mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1'], contact_email: '' }),
    );

    expect(res.status).toBe(200);
  });

  it('deduplicates repeated order ids instead of double counting', async () => {
    mockOrderFindMany.mockResolvedValue([eligibleOrder({ id: 'order-1', payAmount: '128.00' })]);
    mockInvoiceCreate.mockResolvedValue(createdInvoice());

    const res = await mergedInvoiceRoute(
      mergedPostRequest({ ...VALID_BODY, order_ids: ['order-1', 'order-1'] }),
    );

    expect(res.status).toBe(200);
    expect(Number(mockInvoiceCreate.mock.calls[0][0].data.amount)).toBe(128);
  });
});

describe('GET /api/invoices/[id]/download', () => {
  const params = Promise.resolve({ id: 'inv-1' });

  function downloadRequest() {
    return new NextRequest('https://pay.example.com/api/invoices/inv-1/download?token=tok');
  }

  // 同源流式回传，不再 302 到对象存储：下载页在 iframe 里，跳外部源会被父页面
  // CSP 的 frame-src 拦下（Chrome 提示 "This content is blocked"），
  // HTTPS 页面跳 http:// 还会再撞一次混合内容。
  it('streams the file back from the same origin for the owner', async () => {
    mockInvoiceFindFirst.mockResolvedValue({
      id: 'inv-1',
      orderId: 'order-1',
      fileKey: 'invoices/2026/08/inv-1-abcd.pdf',
      fileName: '发票.pdf',
    });
    mockFetchAttachment.mockResolvedValue({
      body: new ReadableStream({
        start(controller) {
          controller.enqueue(new TextEncoder().encode('%PDF-1.7 bytes'));
          controller.close();
        },
      }),
      contentType: 'application/pdf',
      contentDisposition: "attachment; filename=\"_.pdf\"; filename*=UTF-8''%E5%8F%91%E7%A5%A8.pdf",
      contentLength: '14',
    });
    mockInvoiceUpdate.mockResolvedValue({});

    const res = await downloadInvoiceRoute(downloadRequest(), { params });

    expect(res.status).toBe(200);
    // 绝不能是跳转——那正是被 CSP 拦掉的形态。
    expect(res.headers.get('location')).toBeNull();
    expect(res.headers.get('content-type')).toBe('application/pdf');
    expect(res.headers.get('content-disposition')).toContain("filename*=UTF-8''");
    expect(res.headers.get('cache-control')).toContain('no-store');
    expect(res.headers.get('x-content-type-options')).toBe('nosniff');
    expect(await res.text()).toContain('%PDF');
  });

  // 这是整个功能里最关键的一条越权防线：归属与状态都写在查询条件里。
  it("returns nothing for another user's invoice", async () => {
    mockInvoiceFindFirst.mockResolvedValue(null);
    const res = await downloadInvoiceRoute(downloadRequest(), { params });
    expect(res.status).toBe(404);
    expect(mockFetchAttachment).not.toHaveBeenCalled();

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
    expect(mockFetchAttachment).not.toHaveBeenCalled();
  });
});
