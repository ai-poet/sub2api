import { getInternalPayHeaders } from '@/lib/internal-auth';
import { buildInternalUrl } from './client';

const DEFAULT_TIMEOUT_MS = 30_000;

export interface UploadedAttachment {
  key: string;
  file_name: string;
  size: number;
  content_type: string;
}

interface InternalEnvelope<T> {
  data?: T;
  message?: string;
  reason?: string;
}

/**
 * 网关返回的错误码。调用方据此把「对象存储没配」这类可修复问题呈现给管理员，
 * 而不是笼统报 500。
 */
export class AttachmentError extends Error {
  code: string;
  statusCode: number;

  constructor(code: string, message: string, statusCode: number) {
    super(message);
    this.name = 'AttachmentError';
    this.code = code;
    this.statusCode = statusCode;
  }
}

async function readError(response: Response, fallback: string): Promise<AttachmentError> {
  const body = (await response.json().catch(() => ({}))) as InternalEnvelope<unknown>;
  return new AttachmentError(body.reason || 'ATTACHMENT_ERROR', body.message || fallback, response.status);
}

/**
 * 以裸二进制上传到网关，由网关写入对象存储并返回服务端生成的 key。
 *
 * 这一跳不用 multipart：元数据放 query 即可，网关侧因此不需要引入 multipart 解析
 * （Gin 超过内存阈值会静默落临时文件）。浏览器那一跳仍然是 multipart。
 */
export async function uploadAttachment(params: {
  scope: 'invoice';
  ref: string;
  fileName: string;
  contentType: string;
  data: ArrayBuffer | Uint8Array;
}): Promise<UploadedAttachment> {
  const query = new URLSearchParams({
    scope: params.scope,
    ref: params.ref,
    filename: params.fileName,
  });

  const response = await fetch(buildInternalUrl(`/api/internal/pay/attachments?${query}`), {
    method: 'POST',
    headers: getInternalPayHeaders({ 'Content-Type': params.contentType }),
    body: params.data as BodyInit,
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw await readError(response, `Attachment upload failed (${response.status})`);
  }

  const body = (await response.json()) as InternalEnvelope<UploadedAttachment>;
  if (!body.data?.key) {
    throw new AttachmentError('ATTACHMENT_UPLOAD_INVALID_RESPONSE', 'Attachment upload returned no key', 502);
  }
  return body.data;
}

export interface AttachmentStream {
  body: ReadableStream<Uint8Array>;
  contentType: string;
  contentDisposition: string;
  contentLength: string | null;
}

/**
 * 取回附件内容，由本服务同源回传给浏览器。
 *
 * 不用预签名链接直接跳转：下载页跑在 iframe 里，浏览器直接访问对象存储会被父页面
 * CSP 的 frame-src 拦下（Chrome 提示 "This content is blocked"），HTTPS 页面里跳
 * http:// 还会再撞一次混合内容。同源回传也让对象存储不必对公网暴露。
 *
 * 返回的是流，不落地成 Buffer——发票虽小，但没必要为每次下载占一份完整内存。
 */
export async function fetchAttachment(params: {
  key: string;
  fileName?: string | null;
}): Promise<AttachmentStream> {
  const query = new URLSearchParams({ key: params.key, filename: params.fileName ?? '' });

  const response = await fetch(buildInternalUrl(`/api/internal/pay/attachments/content?${query}`), {
    headers: getInternalPayHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw await readError(response, `Attachment download failed (${response.status})`);
  }
  if (!response.body) {
    throw new AttachmentError('ATTACHMENT_EMPTY_BODY', 'Attachment response had no body', 502);
  }

  return {
    body: response.body,
    contentType: response.headers.get('content-type') || 'application/octet-stream',
    contentDisposition: response.headers.get('content-disposition') || 'attachment',
    contentLength: response.headers.get('content-length'),
  };
}

/**
 * 删除附件对象。用于回滚孤儿对象与替换文件后清理旧对象——两者都不该让主流程失败，
 * 因此这里吞掉错误只记日志。
 */
export async function deleteAttachmentQuietly(key: string): Promise<void> {
  try {
    const response = await fetch(buildInternalUrl('/api/internal/pay/attachments'), {
      method: 'DELETE',
      headers: getInternalPayHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ key }),
      signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
    });
    if (!response.ok) {
      console.warn(`[invoice] failed to delete orphaned attachment ${key}: ${response.status}`);
    }
  } catch (error) {
    console.warn(`[invoice] failed to delete orphaned attachment ${key}:`, error);
  }
}

/**
 * 触发「发票已开具」通知邮件。
 *
 * 收件邮箱与站点链接都不在参数里：两者由网关按 user_id 与站点设置解析，避免这个
 * 内部接口被当成可定向发信的中继。
 */
export async function sendInvoiceReadyEmail(params: {
  userId: number;
  invoiceId: string;
  orderId: string;
  amountDisplay: string;
  titleName: string;
  taxNo: string;
  issuedAt: string;
  /** 传对象存储 fileKey：同一份文件去重，换文件后重发才会真正发出。 */
  reminderKey: string;
}): Promise<void> {
  const response = await fetch(buildInternalUrl('/api/internal/pay/notifications/invoice-ready'), {
    method: 'POST',
    headers: getInternalPayHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      user_id: params.userId,
      invoice_id: params.invoiceId,
      order_id: params.orderId,
      amount_display: params.amountDisplay,
      title_name: params.titleName,
      tax_no: params.taxNo,
      issued_at: params.issuedAt,
      reminder_key: params.reminderKey,
    }),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw await readError(response, `Invoice notification failed (${response.status})`);
  }
}
