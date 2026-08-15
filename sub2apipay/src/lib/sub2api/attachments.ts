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

/** 签发短期下载链接。网关会拒绝不在发票前缀下的 key。 */
export async function presignAttachment(params: {
  key: string;
  fileName?: string | null;
  expiresInSeconds?: number;
}): Promise<{ url: string; expiresAt: string }> {
  const response = await fetch(buildInternalUrl('/api/internal/pay/attachments/presign'), {
    method: 'POST',
    headers: getInternalPayHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      key: params.key,
      file_name: params.fileName ?? '',
      expires_in_seconds: params.expiresInSeconds ?? 300,
    }),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw await readError(response, `Presign failed (${response.status})`);
  }

  const body = (await response.json()) as InternalEnvelope<{ url: string; expires_at: string }>;
  if (!body.data?.url) {
    throw new AttachmentError('ATTACHMENT_PRESIGN_INVALID_RESPONSE', 'Presign returned no URL', 502);
  }
  return { url: body.data.url, expiresAt: body.data.expires_at };
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
