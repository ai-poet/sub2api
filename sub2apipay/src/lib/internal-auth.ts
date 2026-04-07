import crypto from 'crypto';
import { getEnv } from '@/lib/config';

export const INTERNAL_PAY_TOKEN_HEADER = 'x-sub2api-pay-token';
const INTERNAL_PAY_TOKEN_PURPOSE = 'sub2api-pay-internal-bridge:v1';

export function deriveInternalPayToken(): string {
  return crypto.createHmac('sha256', getEnv().JWT_SECRET).update(INTERNAL_PAY_TOKEN_PURPOSE).digest('hex');
}

export function getInternalPayHeaders(headers?: HeadersInit): Record<string, string> {
  return {
    [INTERNAL_PAY_TOKEN_HEADER]: deriveInternalPayToken(),
    ...(headers instanceof Headers ? Object.fromEntries(headers.entries()) : (headers as Record<string, string> | undefined)),
  };
}
