import type { NextRequest } from 'next/server';
import { inferPublicBasePathFromPathname, normalizeBasePath } from '@/lib/public-path';

function firstHeaderValue(value: string | null): string {
  return (value || '')
    .split(',')
    .map((item) => item.trim())
    .find(Boolean) || '';
}

export function resolveRequestOrigin(request: NextRequest): string {
  const forwardedProto = firstHeaderValue(request.headers.get('x-forwarded-proto'));
  const forwardedHost = firstHeaderValue(request.headers.get('x-forwarded-host'));
  const forwardedPrefix = normalizeBasePath(firstHeaderValue(request.headers.get('x-forwarded-prefix')));
  const inferredPrefix = inferPublicBasePathFromPathname(request.nextUrl.pathname);
  const basePath = forwardedPrefix || inferredPrefix;
  const proto = forwardedProto || request.nextUrl.protocol.replace(/:$/, '') || 'http';
  const host = forwardedHost || request.headers.get('host') || request.nextUrl.host;

  if (host) {
    return `${proto}://${host}${basePath}`;
  }

  return `${request.nextUrl.origin}${basePath}`;
}

/**
 * Resolve a root-relative path a provider returned (the Alipay short link
 * `/pay/{orderId}`) against the public app URL, so it still works once it is
 * encoded into a QR code or opened by the desktop client. Absolute URLs, app
 * schemes (`weixin://…`) and protocol-relative values pass through unchanged.
 */
export function resolveAbsoluteUrl<T extends string | null | undefined>(value: T, appUrl: string): T | string {
  if (!value || !value.startsWith('/') || value.startsWith('//')) {
    return value;
  }
  try {
    return new URL(value, appUrl).toString();
  } catch {
    return value;
  }
}
