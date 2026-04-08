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
