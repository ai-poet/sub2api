'use client';

import { useCallback, useSyncExternalStore } from 'react';
import { bearerHeaders, readMainSiteToken, requestToken } from '@/lib/session-token';

export interface PaySession {
  /** 页面确定下来的 token：地址里的，或者主站的登录态；都没有时为空串 */
  token: string;
  /** 地址栏 `token` 参数（跨域部署、旧链接）。页面内的链接只能继续带这一个，绝不能带 localStorage 里的 */
  urlToken: string;
  /** 浏览器端已经读过登录态。未就绪前不要渲染「缺少认证信息」 */
  ready: boolean;
  /** 带上当前 token 的请求头：地址里的 token 原样用；取自主站登录态的，同账号轮换过就用新的 */
  authHeaders: (extra?: Record<string, string>) => Record<string, string>;
}

// 每个页面（文档）只读一次主站登录态，定下账号；之后的轮换交给 freshSessionToken，且只认同一账号。
let pinnedMainSiteToken: string | null = null;

function getPinnedMainSiteToken(): string {
  if (pinnedMainSiteToken === null) pinnedMainSiteToken = readMainSiteToken();
  return pinnedMainSiteToken;
}

const subscribeNever = () => () => {};
// 服务端渲染与水合首帧拿不到 localStorage：返回 null 表示「还没读」。
const getServerSnapshot = () => null;

export function usePaySession(searchParams: { get(name: string): string | null }): PaySession {
  const urlToken = (searchParams.get('token') || '').trim();
  const storedToken = useSyncExternalStore<string | null>(subscribeNever, getPinnedMainSiteToken, getServerSnapshot);

  const token = urlToken || storedToken || '';
  const ready = urlToken !== '' || storedToken !== null;

  const authHeaders = useCallback(
    (extra?: Record<string, string>) => bearerHeaders(requestToken(urlToken, token, readMainSiteToken()), extra),
    [urlToken, token],
  );

  return { token, urlToken, ready, authHeaders };
}
