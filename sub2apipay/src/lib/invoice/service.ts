import { Prisma } from '@prisma/client';

import { prisma } from '@/lib/db';
import { ORDER_STATUS } from '@/lib/constants';
import { getSystemConfigs } from '@/lib/system-config';
import { getBizDayStartUTC } from '@/lib/time/biz-day';
import type { Locale } from '@/lib/locale';
import {
  AttachmentError,
  deleteAttachmentQuietly,
  fetchAttachment,
  sendInvoiceReadyEmail,
  uploadAttachment,
  type AttachmentStream,
} from '@/lib/sub2api/attachments';
import { evaluateInvoiceEligibility } from './eligibility';
import {
  INVOICE_MAX_MERGED_ORDERS,
  INVOICE_REREQUESTABLE_STATUSES,
  INVOICE_STATUS,
  InvoiceError,
  invoiceIneligibleMessage,
  invoiceMessage,
  type AdminInvoiceRequestView,
  type InvoiceRequestView,
  type InvoiceStatus,
  type SavedInvoiceTitle,
} from './types';

/** 浏览器上传时允许的发票文件类型，与网关侧白名单保持一致。 */
export const INVOICE_ALLOWED_CONTENT_TYPES = [
  'application/pdf',
  'application/ofd',
  'application/zip',
  'image/jpeg',
  'image/png',
] as const;

export const INVOICE_MAX_FILE_BYTES = 10 * 1024 * 1024;

const CONFIG_KEYS = [
  'invoice_enabled',
  'invoice_max_age_days',
  'invoice_daily_request_limit',
  'invoice_min_amount',
] as const;

/** 起开金额默认值（人民币）：小额订单需要攒够或合并后才能开票。 */
export const INVOICE_DEFAULT_MIN_AMOUNT = 100;

export interface InvoiceSettings {
  enabled: boolean;
  maxAgeDays: number;
  dailyRequestLimit: number;
  /** 单张发票的最低金额；0 表示不限。合并开票按合计金额判定。 */
  minAmount: number;
}

export async function getInvoiceSettings(): Promise<InvoiceSettings> {
  const configs = await getSystemConfigs([...CONFIG_KEYS]);
  const maxAgeDays = Number.parseInt(configs.invoice_max_age_days ?? '', 10);
  const dailyLimit = Number.parseInt(configs.invoice_daily_request_limit ?? '', 10);
  const minAmount = Number.parseFloat(configs.invoice_min_amount ?? '');
  return {
    enabled: configs.invoice_enabled === 'true',
    maxAgeDays: Number.isFinite(maxAgeDays) && maxAgeDays >= 0 ? maxAgeDays : 180,
    dailyRequestLimit: Number.isFinite(dailyLimit) && dailyLimit >= 0 ? dailyLimit : 20,
    minAmount: Number.isFinite(minAmount) && minAmount >= 0 ? minAmount : INVOICE_DEFAULT_MIN_AMOUNT,
  };
}

/** 人民币金额展示，用于邮件与前端。 */
export function formatInvoiceAmount(amount: Prisma.Decimal | number | string): string {
  return `¥${Number(amount).toFixed(2)}`;
}

function formatIssuedAt(date: Date): string {
  // 与订单页展示一致：本地时区、分钟精度。
  return date.toLocaleString('zh-CN', { hour12: false }).replace(/\//g, '-');
}

// ─── 视图映射 ───

type InvoiceRow = Prisma.InvoiceRequestGetPayload<object>;
type InvoiceRowWithOrders = InvoiceRow & { orders?: { orderId: string }[] };

function toInvoiceView(row: InvoiceRowWithOrders): InvoiceRequestView {
  return {
    id: row.id,
    orderId: row.orderId,
    // 合并开票时把全部订单号带给前端；未 include 关联时退回主订单，保持字段恒非空。
    orderIds: row.orders?.length ? row.orders.map((item) => item.orderId) : [row.orderId],
    status: row.status as InvoiceStatus,
    titleName: row.titleName,
    taxNo: row.taxNo,
    remark: row.remark,
    contactEmail: row.contactEmail,
    amount: Number(row.amount),
    fileName: row.fileName,
    hasFile: !!row.fileKey,
    rejectReason: row.rejectReason,
    issuedAt: row.issuedAt,
    createdAt: row.createdAt,
  };
}

function toAdminInvoiceView(
  row: InvoiceRowWithOrders & {
    order?: {
      id: string;
      payAmount: Prisma.Decimal | null;
      paymentType: string;
      orderType: string;
      paidAt: Date | null;
      status: string;
      userEmail: string | null;
      userName: string | null;
    } | null;
  },
): AdminInvoiceRequestView {
  return {
    ...toInvoiceView(row),
    userId: row.userId,
    adminNote: row.adminNote,
    notifiedAt: row.notifiedAt,
    issuedBy: row.issuedBy,
    downloadCount: row.downloadCount,
    order: row.order
      ? {
          id: row.order.id,
          payAmount: row.order.payAmount == null ? null : Number(row.order.payAmount),
          paymentType: row.order.paymentType,
          orderType: row.order.orderType,
          paidAt: row.order.paidAt,
          status: row.order.status,
          userEmail: row.order.userEmail,
          userName: row.order.userName,
        }
      : null,
  };
}

// ─── 用户侧 ───

export async function listSavedTitles(userId: number): Promise<SavedInvoiceTitle[]> {
  const rows = await prisma.invoiceTitle.findMany({
    where: { userId },
    orderBy: { lastUsedAt: 'desc' },
    take: 10,
    select: { id: true, titleName: true, taxNo: true, remark: true, contactEmail: true, lastUsedAt: true },
  });
  return rows;
}

export async function listMyInvoices(
  userId: number,
  page: number,
  pageSize: number,
): Promise<{ invoices: InvoiceRequestView[]; total: number }> {
  const [rows, total] = await Promise.all([
    prisma.invoiceRequest.findMany({
      where: { userId },
      orderBy: { createdAt: 'desc' },
      skip: (page - 1) * pageSize,
      take: pageSize,
      include: { orders: { select: { orderId: true } } },
    }),
    prisma.invoiceRequest.count({ where: { userId } }),
  ]);
  return { invoices: rows.map(toInvoiceView), total };
}

/**
 * 批量取订单对应的开票状态，供订单列表一次性回填（避免 N+1）。
 *
 * 走明细表而不是 invoice_requests.order_id：合并开票时只有主订单落在那一列，
 * 其余订单必须同样显示为「已申请」，否则用户会看到可以重复开票的按钮。
 */
export async function loadInvoicesForOrders(orderIds: string[]): Promise<Map<string, InvoiceRow>> {
  if (orderIds.length === 0) return new Map();
  const rows = await prisma.invoiceRequestOrder.findMany({
    where: { orderId: { in: orderIds } },
    include: { invoice: true },
  });
  return new Map(rows.map((row) => [row.orderId, row.invoice]));
}

export { INVOICE_MAX_MERGED_ORDERS };

export interface RequestInvoiceInput {
  /** 一张或多张订单。多张时合并为一张发票，金额为各单实付之和。 */
  orderIds: string[];
  userId: number;
  titleName: string;
  taxNo: string;
  remark?: string | null;
  contactEmail?: string | null;
  locale: Locale;
}

/**
 * 申请开票。支持把多张订单合并成一张发票。
 *
 * 合并的判定是「逐单资格 + 金额求和」：任何一张不合格就整体拒绝，并回报是哪一张、
 * 为什么——部分成功会让用户搞不清最终开了哪些单。
 */
export async function requestInvoice(input: RequestInvoiceInput): Promise<InvoiceRequestView> {
  const { locale } = input;
  // 去重后保持稳定顺序：同一张单被勾两次不该撑大金额。
  const orderIds = [...new Set(input.orderIds.map((id) => id.trim()).filter(Boolean))];

  if (orderIds.length === 0) {
    throw new InvoiceError('ORDER_NOT_FOUND', invoiceMessage(locale, '请选择要开票的订单', 'Select at least one order'), 400);
  }
  if (orderIds.length > INVOICE_MAX_MERGED_ORDERS) {
    throw new InvoiceError(
      'INVOICE_TOO_MANY_ORDERS',
      invoiceMessage(
        locale,
        `单次最多合并 ${INVOICE_MAX_MERGED_ORDERS} 张订单`,
        `At most ${INVOICE_MAX_MERGED_ORDERS} orders can be merged into one invoice`,
      ),
      400,
    );
  }

  const settings = await getInvoiceSettings();

  const orders = await prisma.order.findMany({
    where: { id: { in: orderIds } },
    select: {
      id: true,
      userId: true,
      status: true,
      paymentType: true,
      paidAt: true,
      payAmount: true,
      refundAmount: true,
    },
  });

  // 归属检查先于资格检查：不能让他人通过错误码区分「订单不存在」与「订单不属于我」。
  if (orders.length !== orderIds.length || orders.some((order) => order.userId !== input.userId)) {
    throw new InvoiceError('ORDER_NOT_FOUND', invoiceMessage(locale, '订单不存在', 'Order not found'), 404);
  }

  const existingLinks = await prisma.invoiceRequestOrder.findMany({
    where: { orderId: { in: orderIds } },
    include: { invoice: { select: { id: true, status: true } } },
  });
  const existingByOrder = new Map(existingLinks.map((link) => [link.orderId, link.invoice]));

  for (const order of orders) {
    const eligibility = evaluateInvoiceEligibility(order, {
      featureEnabled: settings.enabled,
      maxAgeDays: settings.maxAgeDays,
      existingStatus: (existingByOrder.get(order.id)?.status as InvoiceStatus | undefined) ?? null,
    });
    if (!eligibility.eligible) {
      const reason = eligibility.reason!;
      const message =
        orders.length > 1
          ? `${invoiceMessage(locale, '订单', 'Order')} ${order.id}: ${invoiceIneligibleMessage(locale, reason)}`
          : invoiceIneligibleMessage(locale, reason);
      throw new InvoiceError(reason, message, reason === 'ALREADY_REQUESTED' ? 409 : 400, { orderId: order.id });
    }
  }

  if (settings.dailyRequestLimit > 0) {
    const since = getBizDayStartUTC();
    const todayCount = await prisma.invoiceRequest.count({
      where: { userId: input.userId, createdAt: { gte: since } },
    });
    if (todayCount >= settings.dailyRequestLimit) {
      throw new InvoiceError(
        'INVOICE_RATE_LIMITED',
        invoiceMessage(locale, '今日开票申请次数已达上限，请明天再试', 'Daily invoice request limit reached'),
        429,
      );
    }
  }

  // 主订单取最早付款的那张：发票覆盖的账期从它算起，审计日志与邮件也按它归档。
  const sortedOrders = [...orders].sort(
    (a, b) => (a.paidAt?.getTime() ?? 0) - (b.paidAt?.getTime() ?? 0) || a.id.localeCompare(b.id),
  );
  const primaryOrder = sortedOrders[0];
  const lineAmounts = new Map(
    sortedOrders.map((order) => [order.id, new Prisma.Decimal(Number(order.payAmount).toFixed(2))]),
  );
  const amount = sortedOrders.reduce(
    (sum, order) => sum.add(lineAmounts.get(order.id)!),
    new Prisma.Decimal(0),
  );

  // 起开金额按合计判定，而不是逐单：小额订单本来就该攒够或合并后再开，
  // 逐单判定会把它们永远挡在门外。
  if (settings.minAmount > 0 && amount.lessThan(settings.minAmount)) {
    throw new InvoiceError(
      'INVOICE_BELOW_MIN_AMOUNT',
      invoiceMessage(
        locale,
        `开票金额需满 ${formatInvoiceAmount(settings.minAmount)}，当前 ${formatInvoiceAmount(amount)}，可勾选多张订单合并开票`,
        `The invoice amount must reach ${formatInvoiceAmount(settings.minAmount)} (currently ${formatInvoiceAmount(amount)}). Select more orders to merge them into one invoice.`,
      ),
      400,
      { minAmount: settings.minAmount, amount: Number(amount) },
    );
  }

  const titleName = input.titleName.trim();
  const taxNo = input.taxNo.trim().toUpperCase();
  const remark = input.remark?.trim() || null;
  const contactEmail = input.contactEmail?.trim() || null;

  // 被驳回/取消的旧申请整行删除（级联清掉明细行），让订单重新变为可开票。
  // 这些行本来就是死申请：原实现也是就地覆写、清空文件与驳回原因，信息量等同。
  const supersededInvoiceIds = [
    ...new Set(
      existingLinks
        .filter((link) => INVOICE_REREQUESTABLE_STATUSES.includes(link.invoice.status as InvoiceStatus))
        .map((link) => link.invoice.id),
    ),
  ];

  let row: InvoiceRow;
  try {
    row = await prisma.$transaction(async (tx) => {
      if (supersededInvoiceIds.length > 0) {
        // 守卫 status：与管理员的处理动作并发时，这里删不掉，随后的唯一约束会给出 409。
        await tx.invoiceRequest.deleteMany({
          where: {
            id: { in: supersededInvoiceIds },
            userId: input.userId,
            status: { in: INVOICE_REREQUESTABLE_STATUSES },
          },
        });
      }

      return tx.invoiceRequest.create({
        data: {
          orderId: primaryOrder.id,
          userId: input.userId,
          titleName,
          taxNo,
          remark,
          contactEmail,
          amount,
          status: INVOICE_STATUS.PENDING,
          orders: {
            create: sortedOrders.map((order) => ({
              orderId: order.id,
              amount: lineAmounts.get(order.id)!,
            })),
          },
        },
      });
    });
  } catch (error) {
    // 并发申请：唯一约束是最终裁决者，先到者胜出。
    if (error instanceof Prisma.PrismaClientKnownRequestError && error.code === 'P2002') {
      throw new InvoiceError('ALREADY_REQUESTED', invoiceIneligibleMessage(locale, 'ALREADY_REQUESTED'), 409);
    }
    throw error;
  }

  await rememberInvoiceTitle(input.userId, { titleName, taxNo, remark, contactEmail });
  // 每张被覆盖的订单各记一条：按订单号查审计时不会漏掉合并进来的那些。
  for (const order of sortedOrders) {
    await writeInvoiceAuditLog(order.id, 'INVOICE_REQUESTED', `user:${input.userId}`, {
      invoiceId: row.id,
      amount: Number(lineAmounts.get(order.id)!),
      totalAmount: Number(amount),
      orderCount: sortedOrders.length,
    });
  }

  return toInvoiceView({ ...row, orders: sortedOrders.map((order) => ({ orderId: order.id })) });
}

// ─── 一键开票 ───

export interface QuickInvoicePreview {
  enabled: boolean;
  /** 本次一键开票会覆盖的订单（最早付款优先，封顶 INVOICE_MAX_MERGED_ORDERS）。 */
  orderIds: string[];
  count: number;
  /** 覆盖订单的实付合计（人民币）。 */
  amount: number;
  /** 全部可开票订单数（未截断前）。 */
  eligibleCount: number;
  capped: boolean;
  minAmount: number;
}

/**
 * 计算「一键开票」会覆盖哪些订单。
 *
 * 订单由服务端挑选而不是前端勾选：前端只有当前页的数据，而可开票订单散落在所有
 * 分页里。挑选规则与订单列表展示的 canRequestInvoice 完全一致（同一个
 * evaluateInvoiceEligibility），最早付款的排前面——开票窗口先过期的先开掉。
 */
export async function previewQuickInvoice(userId: number): Promise<QuickInvoicePreview> {
  const settings = await getInvoiceSettings();
  const empty: QuickInvoicePreview = {
    enabled: settings.enabled,
    orderIds: [],
    count: 0,
    amount: 0,
    eligibleCount: 0,
    capped: false,
    minAmount: settings.minAmount,
  };
  if (!settings.enabled) return empty;

  const windowStart =
    settings.maxAgeDays > 0 ? new Date(Date.now() - settings.maxAgeDays * 24 * 60 * 60 * 1000) : null;
  const orders = await prisma.order.findMany({
    where: {
      userId,
      status: ORDER_STATUS.COMPLETED,
      paidAt: windowStart ? { gte: windowStart } : { not: null },
    },
    select: { id: true, status: true, paymentType: true, paidAt: true, payAmount: true, refundAmount: true },
    orderBy: { paidAt: 'asc' },
  });
  if (orders.length === 0) return empty;

  const links = await loadInvoicesForOrders(orders.map((order) => order.id));
  const eligible = orders.filter(
    (order) =>
      evaluateInvoiceEligibility(order, {
        featureEnabled: true,
        maxAgeDays: settings.maxAgeDays,
        existingStatus: (links.get(order.id)?.status as InvoiceStatus | undefined) ?? null,
      }).eligible,
  );

  const candidates = eligible.slice(0, INVOICE_MAX_MERGED_ORDERS);
  const amount = candidates.reduce((sum, order) => sum + Number(order.payAmount), 0);
  return {
    enabled: true,
    orderIds: candidates.map((order) => order.id),
    count: candidates.length,
    amount: Number(amount.toFixed(2)),
    eligibleCount: eligible.length,
    capped: eligible.length > candidates.length,
    minAmount: settings.minAmount,
  };
}

export interface RequestQuickInvoiceInput {
  userId: number;
  titleName: string;
  taxNo: string;
  remark?: string | null;
  contactEmail?: string | null;
  locale: Locale;
}

/**
 * 一键开票：服务端挑单后走 requestInvoice——资格、起开金额、并发唯一约束全部复用，
 * 预览与提交之间订单状态变了也由那边的逐单校验兜住。
 */
export async function requestQuickInvoice(
  input: RequestQuickInvoiceInput,
): Promise<{ invoice: InvoiceRequestView; count: number; amount: number }> {
  const preview = await previewQuickInvoice(input.userId);
  if (!preview.enabled) {
    throw new InvoiceError('FEATURE_DISABLED', invoiceIneligibleMessage(input.locale, 'FEATURE_DISABLED'), 400);
  }
  if (preview.count === 0) {
    throw new InvoiceError(
      'NO_INVOICEABLE_ORDERS',
      invoiceMessage(input.locale, '当前没有可开票的订单', 'There are no invoiceable orders right now'),
      400,
    );
  }

  const invoice = await requestInvoice({
    orderIds: preview.orderIds,
    userId: input.userId,
    titleName: input.titleName,
    taxNo: input.taxNo,
    remark: input.remark,
    contactEmail: input.contactEmail,
    locale: input.locale,
  });
  return { invoice, count: preview.count, amount: preview.amount };
}

async function rememberInvoiceTitle(
  userId: number,
  title: { titleName: string; taxNo: string; remark: string | null; contactEmail: string | null },
): Promise<void> {
  try {
    await prisma.invoiceTitle.upsert({
      where: { userId_taxNo: { userId, taxNo: title.taxNo } },
      update: { titleName: title.titleName, remark: title.remark, contactEmail: title.contactEmail, lastUsedAt: new Date() },
      create: { userId, ...title },
    });
  } catch (error) {
    // 抬头记忆是便利功能，失败不应让开票申请回滚。
    console.warn('[invoice] failed to remember invoice title:', error);
  }
}

export async function cancelInvoice(input: { invoiceId: string; userId: number; locale: Locale }): Promise<void> {
  const invoice = await prisma.invoiceRequest.findFirst({
    where: { id: input.invoiceId, userId: input.userId },
    select: { id: true, orderId: true },
  });
  if (!invoice) {
    throw new InvoiceError('INVOICE_NOT_FOUND', invoiceMessage(input.locale, '发票申请不存在', 'Invoice not found'), 404);
  }

  // 归属与状态都写在 WHERE 里，而不是先读后写。
  const updated = await prisma.invoiceRequest.updateMany({
    where: { id: input.invoiceId, userId: input.userId, status: INVOICE_STATUS.PENDING },
    data: { status: INVOICE_STATUS.CANCELLED },
  });
  if (updated.count === 0) {
    throw new InvoiceError(
      'CONFLICT',
      invoiceMessage(input.locale, '该申请已被处理，无法取消', 'This request has already been processed'),
      409,
    );
  }
  await writeInvoiceAuditLog(invoice.orderId, 'INVOICE_CANCELLED', `user:${input.userId}`, {
    invoiceId: input.invoiceId,
  });
}

export async function openUserInvoiceFile(input: {
  invoiceId: string;
  userId: number;
  locale: Locale;
}): Promise<AttachmentStream> {
  // 归属 + 状态一起写进查询条件：这是整个功能里最关键的一条越权防线。
  const invoice = await prisma.invoiceRequest.findFirst({
    where: { id: input.invoiceId, userId: input.userId, status: INVOICE_STATUS.ISSUED },
    select: { id: true, orderId: true, fileKey: true, fileName: true },
  });
  if (!invoice?.fileKey) {
    throw new InvoiceError(
      'INVOICE_FILE_NOT_FOUND',
      invoiceMessage(input.locale, '发票文件不存在', 'Invoice file not found'),
      404,
    );
  }

  const stream = await fetchAttachment({ key: invoice.fileKey, fileName: invoice.fileName });

  await prisma.invoiceRequest
    .update({
      where: { id: invoice.id },
      data: { downloadCount: { increment: 1 }, lastDownloadedAt: new Date() },
    })
    .catch(() => {});
  await writeInvoiceAuditLog(invoice.orderId, 'INVOICE_DOWNLOADED', `user:${input.userId}`, {
    invoiceId: invoice.id,
  });

  return stream;
}

// ─── 管理侧 ───

export interface AdminInvoiceFilters {
  status?: InvoiceStatus;
  userId?: number;
  dateFrom?: Date;
  dateTo?: Date;
  keyword?: string;
}

export async function adminListInvoices(
  filters: AdminInvoiceFilters,
  page: number,
  pageSize: number,
): Promise<{ invoices: AdminInvoiceRequestView[]; total: number; statusCounts: Record<string, number> }> {
  const where: Prisma.InvoiceRequestWhereInput = {};
  if (filters.status) where.status = filters.status;
  if (filters.userId != null) where.userId = filters.userId;
  if (filters.dateFrom || filters.dateTo) {
    where.createdAt = {
      ...(filters.dateFrom && { gte: filters.dateFrom }),
      ...(filters.dateTo && { lte: filters.dateTo }),
    };
  }
  if (filters.keyword) {
    where.OR = [
      { titleName: { contains: filters.keyword, mode: 'insensitive' } },
      { taxNo: { contains: filters.keyword, mode: 'insensitive' } },
      { orderId: { contains: filters.keyword } },
      // 合并开票时被并进来的订单号不在主列上，按订单号搜索必须也能命中。
      { orders: { some: { orderId: { contains: filters.keyword } } } },
    ];
  }

  const [rows, total, grouped] = await Promise.all([
    prisma.invoiceRequest.findMany({
      where,
      orderBy: [{ status: 'asc' }, { createdAt: 'desc' }],
      skip: (page - 1) * pageSize,
      take: pageSize,
      include: {
        order: {
          select: {
            id: true,
            payAmount: true,
            paymentType: true,
            orderType: true,
            paidAt: true,
            status: true,
            userEmail: true,
            userName: true,
          },
        },
        orders: { select: { orderId: true } },
      },
    }),
    prisma.invoiceRequest.count({ where }),
    prisma.invoiceRequest.groupBy({ by: ['status'], _count: true }),
  ]);

  const statusCounts: Record<string, number> = {};
  for (const group of grouped) {
    statusCounts[group.status] = group._count;
  }

  return { invoices: rows.map(toAdminInvoiceView), total, statusCounts };
}

export async function adminGetInvoice(invoiceId: string): Promise<AdminInvoiceRequestView | null> {
  const row = await prisma.invoiceRequest.findUnique({
    where: { id: invoiceId },
    include: {
      order: {
        select: {
          id: true,
          payAmount: true,
          paymentType: true,
          orderType: true,
          paidAt: true,
          status: true,
          userEmail: true,
          userName: true,
        },
      },
      orders: { select: { orderId: true } },
    },
  });
  return row ? toAdminInvoiceView(row) : null;
}

export interface InvoiceFileInput {
  fileName: string;
  contentType: string;
  data: ArrayBuffer;
}

function assertInvoiceFile(file: InvoiceFileInput, locale: Locale): void {
  const contentType = file.contentType.split(';')[0].trim().toLowerCase();
  if (!(INVOICE_ALLOWED_CONTENT_TYPES as readonly string[]).includes(contentType)) {
    throw new InvoiceError(
      'INVOICE_FILE_TYPE_UNSUPPORTED',
      invoiceMessage(locale, '仅支持 PDF / OFD / ZIP / JPG / PNG 格式', 'Only PDF, OFD, ZIP, JPG and PNG are supported'),
      400,
    );
  }
  if (file.data.byteLength === 0) {
    throw new InvoiceError('INVOICE_FILE_EMPTY', invoiceMessage(locale, '发票文件为空', 'Invoice file is empty'), 400);
  }
  if (file.data.byteLength > INVOICE_MAX_FILE_BYTES) {
    throw new InvoiceError(
      'INVOICE_FILE_TOO_LARGE',
      invoiceMessage(locale, '发票文件不能超过 10MB', 'Invoice file must not exceed 10MB'),
      400,
    );
  }
}

function mapAttachmentError(error: unknown, locale: Locale): never {
  if (error instanceof AttachmentError) {
    if (error.code === 'PAY_ATTACHMENT_STORAGE_NOT_CONFIGURED') {
      throw new InvoiceError(
        'INVOICE_STORAGE_NOT_CONFIGURED',
        invoiceMessage(
          locale,
          '对象存储未配置，请先在「系统设置 → 备份 → S3 配置」中完成配置',
          'Object storage is not configured; set up the backup S3 settings first',
        ),
        503,
      );
    }
    throw new InvoiceError(error.code, error.message, error.statusCode >= 500 ? 502 : error.statusCode);
  }
  throw error;
}

export interface AdminIssueInvoiceInput {
  invoiceId: string;
  file: InvoiceFileInput;
  adminNote?: string | null;
  operator: string;
  locale: Locale;
}

export interface AdminIssueInvoiceResult {
  invoice: AdminInvoiceRequestView;
  /** 邮件发送失败时给管理员的提示；发票本身已开具成功。 */
  warning?: string;
}

/**
 * 上传文件并把申请置为已开具，两步合成一个动作。
 *
 * 刻意不拆成「先传文件」「再点开票」两个接口：那会产生「有文件但仍是 PENDING」的
 * 半状态，以及一整类「传了却忘了开」的问题。
 */
export async function adminIssueInvoice(input: AdminIssueInvoiceInput): Promise<AdminIssueInvoiceResult> {
  const { locale } = input;
  assertInvoiceFile(input.file, locale);

  const invoice = await prisma.invoiceRequest.findUnique({ where: { id: input.invoiceId } });
  if (!invoice) {
    throw new InvoiceError('INVOICE_NOT_FOUND', invoiceMessage(locale, '发票申请不存在', 'Invoice not found'), 404);
  }
  if (invoice.status !== INVOICE_STATUS.PENDING) {
    throw new InvoiceError(
      'CONFLICT',
      invoiceMessage(locale, '该申请已被处理', 'This request has already been processed'),
      409,
    );
  }

  // 必须先上传才能拿到 fileKey 写库；若随后的状态流转输掉竞争，再把孤儿对象删掉。
  let uploaded;
  try {
    uploaded = await uploadAttachment({
      scope: 'invoice',
      ref: invoice.id,
      fileName: input.file.fileName,
      contentType: input.file.contentType.split(';')[0].trim().toLowerCase(),
      data: input.file.data,
    });
  } catch (error) {
    mapAttachmentError(error, locale);
  }

  const issuedAt = new Date();
  const updated = await prisma.invoiceRequest.updateMany({
    where: { id: invoice.id, status: INVOICE_STATUS.PENDING },
    data: {
      status: INVOICE_STATUS.ISSUED,
      fileKey: uploaded.key,
      fileName: uploaded.file_name,
      fileSize: uploaded.size,
      fileContentType: uploaded.content_type,
      adminNote: input.adminNote?.trim() || null,
      issuedAt,
      issuedBy: input.operator,
      notifiedAt: null,
    },
  });

  if (updated.count === 0) {
    // 另一个管理员抢先开票了：删掉本次上传的孤儿对象再报冲突。
    await deleteAttachmentQuietly(uploaded.key);
    throw new InvoiceError(
      'CONFLICT',
      invoiceMessage(locale, '该申请已被其他管理员处理', 'This request was just processed by another admin'),
      409,
    );
  }

  await writeInvoiceAuditLog(invoice.orderId, 'INVOICE_ISSUED', input.operator, {
    invoiceId: invoice.id,
    fileKey: uploaded.key,
  });

  const warning = await notifyInvoiceReady(invoice.id, locale);
  const view = await adminGetInvoice(invoice.id);
  return { invoice: view!, warning };
}

/**
 * 发通知邮件。失败只记审计日志并返回提示——绝不因 SMTP 抖动回滚一张已开具的发票。
 */
async function notifyInvoiceReady(invoiceId: string, locale: Locale): Promise<string | undefined> {
  const invoice = await prisma.invoiceRequest.findUnique({ where: { id: invoiceId } });
  if (!invoice || !invoice.fileKey || !invoice.issuedAt) return undefined;

  try {
    await sendInvoiceReadyEmail({
      userId: invoice.userId,
      invoiceId: invoice.id,
      orderId: invoice.orderId,
      amountDisplay: formatInvoiceAmount(invoice.amount),
      titleName: invoice.titleName,
      taxNo: invoice.taxNo,
      issuedAt: formatIssuedAt(invoice.issuedAt),
      // fileKey 参与去重键：换了文件后重发才会真正发出。
      reminderKey: invoice.fileKey,
    });
    await prisma.invoiceRequest.update({ where: { id: invoice.id }, data: { notifiedAt: new Date() } });
    return undefined;
  } catch (error) {
    console.error('[invoice] failed to send invoice-ready notification:', error);
    await writeInvoiceAuditLog(invoice.orderId, 'INVOICE_NOTIFY_FAILED', 'system', {
      invoiceId: invoice.id,
      error: error instanceof Error ? error.message : String(error),
    });
    return invoiceMessage(
      locale,
      '发票已开具，但通知邮件发送失败，请稍后重新发送通知',
      'The invoice was issued, but the notification email failed to send. Please resend it later.',
    );
  }
}

export async function adminResendNotification(input: {
  invoiceId: string;
  locale: Locale;
}): Promise<{ sent: boolean; warning?: string }> {
  const invoice = await prisma.invoiceRequest.findUnique({
    where: { id: input.invoiceId },
    select: { id: true, status: true },
  });
  if (!invoice) {
    throw new InvoiceError('INVOICE_NOT_FOUND', invoiceMessage(input.locale, '发票申请不存在', 'Invoice not found'), 404);
  }
  if (invoice.status !== INVOICE_STATUS.ISSUED) {
    throw new InvoiceError(
      'INVOICE_NOT_ISSUED',
      invoiceMessage(input.locale, '仅已开具的发票可以重新发送通知', 'Only issued invoices can be re-notified'),
      400,
    );
  }
  const warning = await notifyInvoiceReady(invoice.id, input.locale);
  return { sent: !warning, warning };
}

export async function adminRejectInvoice(input: {
  invoiceId: string;
  reason: string;
  operator: string;
  locale: Locale;
}): Promise<AdminInvoiceRequestView> {
  const invoice = await prisma.invoiceRequest.findUnique({
    where: { id: input.invoiceId },
    select: { id: true, orderId: true },
  });
  if (!invoice) {
    throw new InvoiceError('INVOICE_NOT_FOUND', invoiceMessage(input.locale, '发票申请不存在', 'Invoice not found'), 404);
  }

  const updated = await prisma.invoiceRequest.updateMany({
    where: { id: input.invoiceId, status: INVOICE_STATUS.PENDING },
    data: { status: INVOICE_STATUS.REJECTED, rejectReason: input.reason.trim(), rejectedAt: new Date() },
  });
  if (updated.count === 0) {
    throw new InvoiceError(
      'CONFLICT',
      invoiceMessage(input.locale, '该申请已被处理', 'This request has already been processed'),
      409,
    );
  }

  await writeInvoiceAuditLog(invoice.orderId, 'INVOICE_REJECTED', input.operator, {
    invoiceId: invoice.id,
    reason: input.reason.trim(),
  });
  return (await adminGetInvoice(invoice.id))!;
}

/** 已开具后替换发票文件（如开错抬头重开）。状态保持 ISSUED，并清空通知时间以便重发。 */
export async function adminReplaceInvoiceFile(input: {
  invoiceId: string;
  file: InvoiceFileInput;
  operator: string;
  locale: Locale;
}): Promise<AdminInvoiceRequestView> {
  const { locale } = input;
  assertInvoiceFile(input.file, locale);

  const invoice = await prisma.invoiceRequest.findUnique({ where: { id: input.invoiceId } });
  if (!invoice) {
    throw new InvoiceError('INVOICE_NOT_FOUND', invoiceMessage(locale, '发票申请不存在', 'Invoice not found'), 404);
  }
  if (invoice.status !== INVOICE_STATUS.ISSUED) {
    throw new InvoiceError(
      'INVOICE_NOT_ISSUED',
      invoiceMessage(locale, '仅已开具的发票可以替换文件', 'Only issued invoices can have their file replaced'),
      400,
    );
  }

  let uploaded;
  try {
    uploaded = await uploadAttachment({
      scope: 'invoice',
      ref: invoice.id,
      fileName: input.file.fileName,
      contentType: input.file.contentType.split(';')[0].trim().toLowerCase(),
      data: input.file.data,
    });
  } catch (error) {
    mapAttachmentError(error, locale);
  }

  const previousKey = invoice.fileKey;
  const updated = await prisma.invoiceRequest.updateMany({
    where: { id: invoice.id, status: INVOICE_STATUS.ISSUED, fileKey: previousKey },
    data: {
      fileKey: uploaded.key,
      fileName: uploaded.file_name,
      fileSize: uploaded.size,
      fileContentType: uploaded.content_type,
      notifiedAt: null,
    },
  });
  if (updated.count === 0) {
    await deleteAttachmentQuietly(uploaded.key);
    throw new InvoiceError(
      'CONFLICT',
      invoiceMessage(locale, '发票已被其他管理员更新，请刷新后重试', 'The invoice was just updated by another admin'),
      409,
    );
  }

  // 旧对象只在数据库提交成功之后才删，避免回滚后文件已经没了。
  if (previousKey && previousKey !== uploaded.key) {
    await deleteAttachmentQuietly(previousKey);
  }

  await writeInvoiceAuditLog(invoice.orderId, 'INVOICE_FILE_REPLACED', input.operator, {
    invoiceId: invoice.id,
    fileKey: uploaded.key,
  });
  return (await adminGetInvoice(invoice.id))!;
}

export async function adminOpenInvoiceFile(input: { invoiceId: string; locale: Locale }): Promise<AttachmentStream> {
  const invoice = await prisma.invoiceRequest.findUnique({
    where: { id: input.invoiceId },
    select: { fileKey: true, fileName: true },
  });
  if (!invoice?.fileKey) {
    throw new InvoiceError(
      'INVOICE_FILE_NOT_FOUND',
      invoiceMessage(input.locale, '发票文件不存在', 'Invoice file not found'),
      404,
    );
  }
  return fetchAttachment({ key: invoice.fileKey, fileName: invoice.fileName });
}

// ─── 退款联动 ───

export interface InvoiceRefundImpact {
  /** 已开具的发票需要人工红冲，这里只提示，不阻断退款。 */
  needsCreditNote: boolean;
  cancelledPending: boolean;
}

/**
 * 订单退款后调整发票状态。任何异常都被吞掉：退款绝不能因为发票副作用而失败。
 */
export async function applyRefundToInvoice(orderId: string): Promise<InvoiceRefundImpact> {
  const impact: InvoiceRefundImpact = { needsCreditNote: false, cancelledPending: false };
  try {
    // 按明细表查：合并开票时退款的那张订单可能不是主订单。
    const link = await prisma.invoiceRequestOrder.findUnique({
      where: { orderId },
      select: { invoice: { select: { id: true, status: true } } },
    });
    const invoice = link?.invoice;
    if (!invoice) return impact;

    if (invoice.status === INVOICE_STATUS.PENDING) {
      const updated = await prisma.invoiceRequest.updateMany({
        where: { id: invoice.id, status: INVOICE_STATUS.PENDING },
        data: { status: INVOICE_STATUS.CANCELLED, rejectReason: '订单已退款', rejectedAt: new Date() },
      });
      impact.cancelledPending = updated.count > 0;
      if (impact.cancelledPending) {
        await writeInvoiceAuditLog(orderId, 'INVOICE_CANCELLED', 'system', {
          invoiceId: invoice.id,
          reason: 'order refunded',
        });
      }
    } else if (invoice.status === INVOICE_STATUS.ISSUED) {
      impact.needsCreditNote = true;
      await writeInvoiceAuditLog(orderId, 'INVOICE_NEEDS_CREDIT_NOTE', 'system', { invoiceId: invoice.id });
    }
  } catch (error) {
    console.error('[invoice] failed to apply refund to invoice:', error);
  }
  return impact;
}

// ─── 审计 ───

async function writeInvoiceAuditLog(
  orderId: string,
  action: string,
  operator: string,
  detail: Record<string, unknown>,
): Promise<void> {
  try {
    // 只记 id 与状态，不记抬头/税号——那是客户的商业信息。
    await prisma.auditLog.create({
      data: { orderId, action, detail: JSON.stringify(detail), operator },
    });
  } catch (error) {
    console.warn(`[invoice] failed to write audit log ${action}:`, error);
  }
}
