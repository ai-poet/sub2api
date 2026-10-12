import { pickLocaleText, type Locale } from '@/lib/locale';

/**
 * 下单时支付网关调用失败的对外说明。
 *
 * 用户需要看到具体原因（商户未开通该方式、签名错误、渠道超时……）才能自行换方式或报给客服，
 * 但原始异常里可能带网关地址、IP 或密钥片段，所以只给出归类后的说明，
 * 或者去掉这些内容并截断后的渠道原话。完整异常仍写进服务端日志，用参考号对应。
 */

const MAX_DETAIL_LENGTH = 160;

/** 支付渠道回了无法解析的响应（通常是网关或 PHP 的 HTML 报错页）。 */
export class PaymentGatewayResponseError extends Error {
  readonly status: number;

  constructor(status: number) {
    super(`payment channel returned a non-JSON response (HTTP ${status})`);
    this.name = 'PaymentGatewayResponseError';
    this.status = status;
  }
}

/** 各渠道客户端包在渠道原话前面的固定前缀。 */
const PROVIDER_PREFIXES = [/^EasyPay create payment failed:\s*/i];

const NETWORK_CODES = new Set([
  'ECONNREFUSED',
  'ECONNRESET',
  'ENOTFOUND',
  'EAI_AGAIN',
  'EHOSTUNREACH',
  'ENETUNREACH',
  'UND_ERR_CONNECT_TIMEOUT',
  'UND_ERR_SOCKET',
]);

function errorCode(error: unknown): string {
  if (!error || typeof error !== 'object') return '';
  const own = (error as { code?: unknown }).code;
  if (typeof own === 'string') return own;
  const cause = (error as { cause?: unknown }).cause;
  if (cause && typeof cause === 'object') {
    const code = (cause as { code?: unknown }).code;
    if (typeof code === 'string') return code;
  }
  return '';
}

/** 去掉网址、IP 和像密钥的长串，压缩空白并截断。 */
export function sanitizeGatewayMessage(raw: string): string {
  let text = raw;
  for (const prefix of PROVIDER_PREFIXES) text = text.replace(prefix, '');
  text = text
    .replace(/https?:\/\/\S+/gi, '[url]')
    .replace(/\b\d{1,3}(?:\.\d{1,3}){3}(?::\d+)?\b/g, '[ip]')
    .replace(/[A-Za-z0-9_\-+/=]{24,}/g, '[redacted]')
    .replace(/\s+/g, ' ')
    .trim();
  if (text.length > MAX_DETAIL_LENGTH) text = `${text.slice(0, MAX_DETAIL_LENGTH)}…`;
  return text;
}

/** 把支付网关异常归类成一句对用户可见的原因。 */
export function describePaymentGatewayError(error: unknown, locale: Locale): string {
  const name = error instanceof Error ? error.name : '';
  const raw = error instanceof Error ? error.message : String(error ?? '');
  const code = errorCode(error);

  if (error instanceof PaymentGatewayResponseError) {
    return pickLocaleText(
      locale,
      `支付渠道返回了无法识别的响应（HTTP ${error.status}）`,
      `The payment channel returned an unreadable response (HTTP ${error.status})`,
    );
  }
  if (error instanceof SyntaxError) {
    return pickLocaleText(
      locale,
      '支付渠道返回了无法识别的响应',
      'The payment channel returned an unreadable response',
    );
  }
  if (name === 'TimeoutError' || name === 'AbortError' || /timed? ?out|aborted/i.test(raw)) {
    return pickLocaleText(locale, '支付渠道响应超时', 'The payment channel timed out');
  }
  if (NETWORK_CODES.has(code) || /fetch failed|socket hang up|ECONNREFUSED|ENOTFOUND|ECONNRESET/i.test(raw)) {
    return pickLocaleText(locale, '无法连接支付渠道', 'Could not reach the payment channel');
  }

  const detail = sanitizeGatewayMessage(raw);
  return detail || pickLocaleText(locale, '支付渠道返回未知错误', 'The payment channel returned an unknown error');
}
