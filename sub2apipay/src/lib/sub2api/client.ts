import { getEnv } from '@/lib/config';
import { getInternalPayHeaders } from '@/lib/internal-auth';
import type { Sub2ApiUser, Sub2ApiRedeemCode, Sub2ApiGroup, Sub2ApiSubscription } from './types';

const DEFAULT_TIMEOUT_MS = 10_000;
const RECHARGE_TIMEOUT_MS = 30_000;
const RECHARGE_MAX_ATTEMPTS = 2;

function buildInternalUrl(pathname: string): string {
  const env = getEnv();
  const baseUrl = env.SUB2API_INTERNAL_BASE_URL || env.SUB2API_BASE_URL || 'http://127.0.0.1:8080';
  const normalized = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
  return `${normalized}${pathname}`;
}

function getHeaders(idempotencyKey?: string): Record<string, string> {
  const headers = getInternalPayHeaders({
    'Content-Type': 'application/json',
  });
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey;
  }
  return headers;
}

function isRetryableFetchError(error: unknown): boolean {
  if (!(error instanceof Error)) return false;
  return error.name === 'TimeoutError' || error.name === 'AbortError' || error.name === 'TypeError';
}

export async function getCurrentUserByToken(token: string): Promise<Sub2ApiUser> {
  const response = await fetch(buildInternalUrl('/api/internal/pay/auth/me'), {
    headers: {
      ...getInternalPayHeaders(),
      Authorization: `Bearer ${token}`,
    },
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw new Error(`Failed to get current user: ${response.status}`);
  }

  const data = await response.json();
  return data.data as Sub2ApiUser;
}

export async function getUser(userId: number): Promise<Sub2ApiUser> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/users/${userId}`), {
    headers: getHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    if (response.status === 404) throw new Error('USER_NOT_FOUND');
    throw new Error(`Failed to get user: ${response.status}`);
  }

  const data = await response.json();
  return data.data as Sub2ApiUser;
}

export async function createAndRedeem(
  code: string,
  value: number,
  userId: number,
  notes: string,
  options?: { type?: 'balance' | 'subscription'; groupId?: number; validityDays?: number },
): Promise<Sub2ApiRedeemCode> {
  const url = buildInternalUrl('/api/internal/pay/redeem-codes/create-and-redeem');
  const body = JSON.stringify({
    code,
    type: options?.type ?? 'balance',
    value,
    user_id: userId,
    notes,
    ...(options?.type === 'subscription' && {
      group_id: options.groupId,
      validity_days: options.validityDays,
    }),
  });

  let lastError: unknown;

  for (let attempt = 1; attempt <= RECHARGE_MAX_ATTEMPTS; attempt += 1) {
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: getHeaders(`sub2apipay:recharge:${code}`),
        body,
        signal: AbortSignal.timeout(RECHARGE_TIMEOUT_MS),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(`Recharge failed (${response.status}): ${JSON.stringify(errorData)}`);
      }

      const data = await response.json();
      return data.redeem_code as Sub2ApiRedeemCode;
    } catch (error) {
      lastError = error;
      if (attempt >= RECHARGE_MAX_ATTEMPTS || !isRetryableFetchError(error)) {
        throw error;
      }
      console.warn(`Internal recharge attempt ${attempt} timed out, retrying...`);
    }
  }

  throw lastError instanceof Error ? lastError : new Error('Recharge failed');
}

// ── 分组 API ──

export async function getAllGroups(): Promise<Sub2ApiGroup[]> {
  const response = await fetch(buildInternalUrl('/api/internal/pay/groups/all'), {
    headers: getHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw new Error(`Failed to get groups: ${response.status}`);
  }

  const data = await response.json();
  return (data.data ?? []) as Sub2ApiGroup[];
}

export async function getGroup(groupId: number): Promise<Sub2ApiGroup | null> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/groups/${groupId}`), {
    headers: getHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    if (response.status === 404) return null;
    throw new Error(`Failed to get group ${groupId}: ${response.status}`);
  }

  const data = await response.json();
  return data.data as Sub2ApiGroup;
}

// ── 订阅 API ──

export async function assignSubscription(
  userId: number,
  groupId: number,
  validityDays: number,
  notes?: string,
  idempotencyKey?: string,
): Promise<Sub2ApiSubscription> {
  const response = await fetch(buildInternalUrl('/api/internal/pay/subscriptions/assign'), {
    method: 'POST',
    headers: getHeaders(idempotencyKey),
    body: JSON.stringify({
      user_id: userId,
      group_id: groupId,
      validity_days: validityDays,
      notes: notes || `payment center subscription order`,
    }),
    signal: AbortSignal.timeout(RECHARGE_TIMEOUT_MS),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(`Assign subscription failed (${response.status}): ${JSON.stringify(errorData)}`);
  }

  const data = await response.json();
  return data.data as Sub2ApiSubscription;
}

export async function getUserSubscriptions(userId: number): Promise<Sub2ApiSubscription[]> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/users/${userId}/subscriptions`), {
    headers: getHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    if (response.status === 404) return [];
    throw new Error(`Failed to get user subscriptions: ${response.status}`);
  }

  const data = await response.json();
  return (data.data ?? []) as Sub2ApiSubscription[];
}

export async function extendSubscription(subscriptionId: number, days: number, idempotencyKey?: string): Promise<void> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/subscriptions/${subscriptionId}/extend`), {
    method: 'POST',
    headers: getHeaders(idempotencyKey),
    body: JSON.stringify({ days }),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(`Extend subscription failed (${response.status}): ${JSON.stringify(errorData)}`);
  }
}

// ── 余额 API ──

export async function subtractBalance(
  userId: number,
  amount: number,
  notes: string,
  idempotencyKey: string,
): Promise<void> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/users/${userId}/balance`), {
    method: 'POST',
    headers: getHeaders(idempotencyKey),
    body: JSON.stringify({
      operation: 'subtract',
      balance: amount,
      notes,
    }),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(`Subtract balance failed (${response.status}): ${JSON.stringify(errorData)}`);
  }
}

// ── 用户搜索 API ──

export async function searchUsers(
  keyword: string,
): Promise<{ id: number; email: string; username: string; notes?: string }[]> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/users?search=${encodeURIComponent(keyword)}&page=1&page_size=30`), {
    headers: getHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw new Error(`Failed to search users: ${response.status}`);
  }

  const data = await response.json();
  const paginated = data.data ?? {};
  return (paginated.items ?? []) as { id: number; email: string; username: string; notes?: string }[];
}

export async function listSubscriptions(params?: {
  user_id?: number;
  group_id?: number;
  status?: string;
  page?: number;
  page_size?: number;
}): Promise<{ subscriptions: Sub2ApiSubscription[]; total: number; page: number; page_size: number }> {
  const qs = new URLSearchParams();
  if (params?.user_id != null) qs.set('user_id', String(params.user_id));
  if (params?.group_id != null) qs.set('group_id', String(params.group_id));
  if (params?.status) qs.set('status', params.status);
  if (params?.page != null) qs.set('page', String(params.page));
  if (params?.page_size != null) qs.set('page_size', String(params.page_size));

  const response = await fetch(buildInternalUrl(`/api/internal/pay/subscriptions?${qs}`), {
    headers: getHeaders(),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw new Error(`Failed to list subscriptions: ${response.status}`);
  }

  const data = await response.json();
  const paginated = data.data ?? {};
  return {
    subscriptions: (paginated.items ?? []) as Sub2ApiSubscription[],
    total: paginated.total ?? 0,
    page: paginated.page ?? 1,
    page_size: paginated.page_size ?? 50,
  };
}

export async function addBalance(userId: number, amount: number, notes: string, idempotencyKey: string): Promise<void> {
  const response = await fetch(buildInternalUrl(`/api/internal/pay/users/${userId}/balance`), {
    method: 'POST',
    headers: getHeaders(idempotencyKey),
    body: JSON.stringify({
      operation: 'add',
      balance: amount,
      notes,
    }),
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(`Add balance failed (${response.status}): ${JSON.stringify(errorData)}`);
  }
}
