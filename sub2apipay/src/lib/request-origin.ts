import type { NextRequest } from 'next/server';

function firstHeaderValue(value: string | null): string {
  return (value || '')
    .split(',')
    .map((item) => item.trim())
    .find(Boolean) || '';
}

export function resolveRequestOrigin(request: NextRequest): string {
  const forwardedProto = firstHeaderValue(request.headers.get('x-forwarded-proto'));
  const forwardedHost = firstHeaderValue(request.headers.get('x-forwarded-host'));
  const proto = forwardedProto || request.nextUrl.protocol.replace(/:$/, '') || 'http';
  const host = forwardedHost || request.headers.get('host') || request.nextUrl.host;

  if (host) {
    return `${proto}://${host}`;
  }

  return request.nextUrl.origin;
}
