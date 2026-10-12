import { pickLocaleText, type Locale } from '@/lib/locale';

/**
 * 读接口响应体：是 JSON 对象就返回它，否则返回 null。
 *
 * 失败的请求经过 CDN / 反向代理时，5xx 的响应体常被换成代理自己的 HTML 错误页，
 * 直接 `res.json()` 会抛异常，前端只能笼统地报「网络错误」。这里不抛，交给调用方按状态码说明。
 */
export async function readJsonBody(res: Response): Promise<Record<string, unknown> | null> {
  let text = '';
  try {
    text = await res.text();
  } catch {
    return null;
  }
  if (!text.trim()) return null;
  try {
    const parsed: unknown = JSON.parse(text);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}

/** 响应体不是 JSON 时的说明：带上 HTTP 状态，网关类错误单独说。 */
export function describeHttpFailure(status: number, locale: Locale): string {
  if (status === 502 || status === 503 || status === 504 || (status >= 520 && status <= 530)) {
    return pickLocaleText(
      locale,
      `支付服务暂时无法访问（HTTP ${status}），请稍后重试`,
      `The payment service is temporarily unreachable (HTTP ${status}). Please try again later`,
    );
  }
  if (status === 429) {
    return pickLocaleText(
      locale,
      '请求过于频繁（HTTP 429），请稍后再试',
      'Too many requests (HTTP 429). Please try again later',
    );
  }
  return pickLocaleText(
    locale,
    `请求失败（HTTP ${status}），请稍后重试`,
    `Request failed (HTTP ${status}). Please try again later`,
  );
}

/** fetch 本身失败（断网、请求被拦截）时的说明，附上浏览器给出的原因。 */
export function describeNetworkFailure(error: unknown, locale: Locale): string {
  const detail = error instanceof Error && error.message ? error.message : '';
  return pickLocaleText(
    locale,
    `网络错误，无法连接到支付服务${detail ? `（${detail}）` : ''}，请检查网络后重试`,
    `Network error: could not reach the payment service${detail ? ` (${detail})` : ''}. Please check your connection and try again`,
  );
}
