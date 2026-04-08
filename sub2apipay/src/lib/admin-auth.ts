import { NextRequest, NextResponse } from 'next/server';
import { getEnv } from '@/lib/config';
import { getInternalPayHeaders } from '@/lib/internal-auth';
import { resolveLocale } from '@/lib/locale';

async function isSub2ApiAdmin(token: string): Promise<boolean> {
  try {
    const env = getEnv();
    const candidate = env.SUB2API_INTERNAL_BASE_URL || env.SUB2API_BASE_URL || 'http://127.0.0.1:8080';
    const baseUrl = candidate.endsWith('/') ? candidate.slice(0, -1) : candidate;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 5000);
    const response = await fetch(`${baseUrl}/api/internal/pay/auth/admin`, {
      headers: {
        ...getInternalPayHeaders(),
        Authorization: `Bearer ${token}`,
      },
      signal: controller.signal,
    });
    clearTimeout(timeout);
    if (!response.ok) return false;
    return true;
  } catch {
    return false;
  }
}

export async function verifyAdminToken(request: NextRequest): Promise<boolean> {
  // 优先从 Authorization: Bearer <token> header 获取
  let token: string | null = null;
  const authHeader = request.headers.get('authorization');
  if (authHeader?.startsWith('Bearer ')) {
    token = authHeader.slice(7).trim();
  }

  // Fallback: query parameter（向后兼容，已弃用）
  if (!token) {
    token = request.nextUrl.searchParams.get('token');
    if (token) {
      console.warn(
        '[DEPRECATED] Admin token passed via query parameter. Use "Authorization: Bearer <token>" header instead.',
      );
    }
  }

  if (!token) return false;

  return isSub2ApiAdmin(token);
}

export function unauthorizedResponse(request?: NextRequest) {
  const locale = resolveLocale(request?.nextUrl.searchParams.get('lang'));
  return NextResponse.json({ error: locale === 'en' ? 'Unauthorized' : '未授权' }, { status: 401 });
}
