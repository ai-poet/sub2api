/**
 * 支付页拿 sub2api 登录 token 的地方。
 *
 * 集成部署里支付服务由网关反代在主站同源的 /pay 下，页面直接读主站存在 localStorage 的登录态，
 * 请求一律带 `Authorization: Bearer`。主站在同源时不再把 token 拼进地址：地址会进反代访问日志、
 * 浏览器历史和截图，而管理员的 token 就是完整的后台会话。
 *
 * 地址里的 `token` 参数照旧认：跨域部署、旧版主站拼的链接、手写的管理链接都靠它。
 * 从 localStorage 读到的 token 绝不能写回任何地址。
 */

/** 主站前端（Vue）保存 access token 的键 */
export const MAIN_SITE_TOKEN_KEY = 'auth_token';

/** 主站当前的 access token；非浏览器环境、存储被禁用或未登录时为空串。 */
export function readMainSiteToken(): string {
  if (typeof window === 'undefined') return '';
  try {
    return window.localStorage.getItem(MAIN_SITE_TOKEN_KEY)?.trim() ?? '';
  } catch {
    return '';
  }
}

/** JWT 载荷里的 user_id。不验签，只用来判断两个 token 是不是同一个账号。 */
export function tokenUserId(token: string): number | null {
  const payload = token.split('.')[1];
  if (!payload) return null;
  try {
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
    const id = JSON.parse(atob(padded))?.user_id;
    return typeof id === 'number' && Number.isInteger(id) && id > 0 ? id : null;
  } catch {
    return null;
  }
}

/**
 * 发请求时用的 token。主站会轮换 access token，页面开久了要跟上新的；但只接受同一账号的新 token。
 * 别的标签页换了账号登录时，本页继续用原来的 token（过期就 401），绝不悄悄换成另一个账号——
 * 否则页面显示的是 A，订单却下到 B 名下。
 */
export function freshSessionToken(pinned: string, latest: string): string {
  if (!pinned) return '';
  if (!latest || latest === pinned) return pinned;
  const pinnedUser = tokenUserId(pinned);
  return pinnedUser !== null && pinnedUser === tokenUserId(latest) ? latest : pinned;
}

/**
 * 请求实际带的 token。地址里带了 token（桌面客户端的支付中心、跨域部署、旧链接）就原样用它，
 * 与改动前完全一致：同一浏览器里可能还存着这个账号过期的网页登录态，不能拿它顶替。
 * 只有 token 本来就取自主站登录态时，才跟随同账号的轮换。
 */
export function requestToken(urlToken: string, pinned: string, latest: string): string {
  return urlToken || freshSessionToken(pinned, latest);
}

export function bearerHeaders(token: string, extra?: Record<string, string>): Record<string, string> {
  return token ? { ...extra, Authorization: `Bearer ${token}` } : { ...extra };
}
