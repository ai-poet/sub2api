/**
 * Client-safe utility for building order status API URLs.
 * This module must NOT import any server-only modules (config, fs, crypto, etc.).
 */

import { buildAppApiPath } from '@/lib/public-path';

const ACCESS_TOKEN_KEY = 'access_token';

export function buildOrderStatusUrl(orderId: string, accessToken?: string | null): string {
  const query = new URLSearchParams();
  if (accessToken) {
    query.set(ACCESS_TOKEN_KEY, accessToken);
  }
  const suffix = query.toString();
  const path = `/api/orders/${orderId}`;
  return buildAppApiPath(suffix ? `${path}?${suffix}` : path);
}
