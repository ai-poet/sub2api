import { buildOrderResultUrl } from '@/lib/order/status-access';

/**
 * 易支付下单时的异步通知地址与同步跳转地址。
 *
 * 默认按当前请求的 origin 拼（同源模式）。请求经 CDN / 反代时 HTTPS 常在途中丢失，
 * 拼出来的就是 http://——自建易支付接受，但要求 HTTPS 的渠道（如启润 PayNow）会拒绝下单。
 *
 * 只有实例配置了 https 的「异步通知地址」时才改用它，其余实例和原来完全一样：
 * - 只填了域名（`https://example.com`）时补上回调路径；路径不是易支付回调路径时视为填错、忽略；
 * - 填写的查询参数一律丢弃，只放路由用的 `inst`：回调验签会把多出来的参数算进签名；
 * - 跳回地址用「同步跳转地址」的域名，没填就用通知地址的域名，路径仍是 /pay/result。
 */

export const EASY_PAY_NOTIFY_PATH = '/api/easy-pay/notify';

export interface EasyPayCallbackUrls {
  notifyUrl: string;
  returnUrl: string;
  /** 是否来自实例配置（否则是按请求 origin 拼的）。 */
  fromInstanceConfig: boolean;
}

/** 解析一个 https 的绝对地址；不是 https 或解析失败返回 null。 */
export function parseHttpsUrl(raw: string | null | undefined): URL | null {
  const value = raw?.trim();
  if (!value) return null;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' && url.hostname ? url : null;
  } catch {
    return null;
  }
}

export function resolveEasyPayCallbackUrls(options: {
  appUrl: string;
  instanceConfig?: Record<string, string> | null;
  instanceId?: string;
  orderId: string;
  userId?: number;
}): EasyPayCallbackUrls {
  const { appUrl, instanceConfig, instanceId, orderId, userId } = options;
  const derived: EasyPayCallbackUrls = {
    notifyUrl: `${appUrl}${EASY_PAY_NOTIFY_PATH}${instanceId ? `?inst=${instanceId}` : ''}`,
    returnUrl: buildOrderResultUrl(appUrl, orderId, userId),
    fromInstanceConfig: false,
  };

  const configured = parseHttpsUrl(instanceConfig?.notifyUrl);
  if (!configured) return derived;

  const configuredPath = configured.pathname.replace(/\/+$/, '');
  let notify: URL;
  if (configuredPath === '') {
    const basePath = new URL(appUrl).pathname.replace(/\/+$/, '');
    notify = new URL(`${basePath}${EASY_PAY_NOTIFY_PATH}`, configured.origin);
  } else if (configuredPath.endsWith(EASY_PAY_NOTIFY_PATH)) {
    notify = new URL(`${configuredPath}`, configured.origin);
  } else {
    console.warn(
      `[easy-pay] instance ${instanceId ?? '(env)'} notifyUrl path "${configured.pathname}" is not ${EASY_PAY_NOTIFY_PATH}; using the request origin instead`,
    );
    return derived;
  }
  if (instanceId) notify.searchParams.set('inst', instanceId);

  const returnOrigin = (parseHttpsUrl(instanceConfig?.returnUrl) ?? configured).origin;
  return {
    notifyUrl: notify.toString(),
    returnUrl: buildOrderResultUrl(returnOrigin, orderId, userId),
    fromInstanceConfig: true,
  };
}
