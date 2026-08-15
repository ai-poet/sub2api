import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest, NextResponse } from 'next/server';

const mockVerifyAdminToken = vi.fn();
const mockFindMany = vi.fn();

vi.mock('@/lib/admin-auth', () => ({
  verifyAdminToken: (...args: unknown[]) => mockVerifyAdminToken(...args),
  unauthorizedResponse: () => NextResponse.json({ error: '未授权' }, { status: 401 }),
}));

vi.mock('@/lib/db', () => ({
  prisma: {
    paymentProviderInstance: {
      findMany: (...args: unknown[]) => mockFindMany(...args),
      create: vi.fn(),
    },
  },
}));

// 模拟真实 crypto：密钥不匹配时 decrypt 抛错（与 AES-GCM 认证失败一致）。
vi.mock('@/lib/crypto', () => ({
  encrypt: (text: string) => `enc:${text}`,
  decrypt: (text: string) => {
    if (!text.startsWith('enc:')) {
      throw new Error('Unsupported state or unable to authenticate data');
    }
    return text.slice(4);
  },
}));

import { GET } from '@/app/api/admin/provider-instances/route';

function instance(id: string, config: string) {
  return {
    id,
    providerKey: 'alipay',
    name: `实例 ${id}`,
    config,
    supportedTypes: 'alipay',
    enabled: true,
    sortOrder: 0,
    limits: null,
    refundEnabled: false,
    createdAt: new Date(),
    updatedAt: new Date(),
  };
}

const GOOD_CONFIG = `enc:${JSON.stringify({ pid: '123', pkey: 'secret-key-value-1234' })}`;
// 用旧 JWT_SECRET 加密的密文，换密钥后再也解不开。
const UNDECRYPTABLE_CONFIG = 'HqLm:0aB9:Zx1c';

beforeEach(() => {
  vi.clearAllMocks();
  mockVerifyAdminToken.mockResolvedValue(true);
});

describe('GET /api/admin/provider-instances', () => {
  function request() {
    return new NextRequest('https://pay.example.com/api/admin/provider-instances');
  }

  it('masks sensitive fields on a healthy instance', async () => {
    mockFindMany.mockResolvedValue([instance('inst-1', GOOD_CONFIG)]);

    const res = await GET(request());
    expect(res.status).toBe(200);

    const { instances } = await res.json();
    expect(instances[0].config.pid).toBe('123');
    expect(instances[0].config.pkey).toBe('*****************1234');
    expect(instances[0].config_decrypt_failed).toBe(false);
  });

  // 轮换 JWT_SECRET 后旧密文永久解不开。此前一行坏掉会抛穿整个 map，
  // 管理员连页面都打不开，也就无从删除重建——恰好在最需要后台时把后台锁死。
  it('degrades a single undecryptable instance instead of failing the whole list', async () => {
    mockFindMany.mockResolvedValue([
      instance('broken', UNDECRYPTABLE_CONFIG),
      instance('healthy', GOOD_CONFIG),
    ]);

    const res = await GET(request());
    expect(res.status).toBe(200);

    const { instances } = await res.json();
    expect(instances).toHaveLength(2);

    const broken = instances.find((i: { id: string }) => i.id === 'broken');
    expect(broken.config_decrypt_failed).toBe(true);
    expect(broken.config).toEqual({});
    // id 与名称必须保留，否则界面上没法定位要删哪一个。
    expect(broken.name).toBe('实例 broken');

    const healthy = instances.find((i: { id: string }) => i.id === 'healthy');
    expect(healthy.config_decrypt_failed).toBe(false);
    expect(healthy.config.pid).toBe('123');
  });

  it('survives a row whose limits JSON is corrupt', async () => {
    mockFindMany.mockResolvedValue([{ ...instance('inst-1', GOOD_CONFIG), limits: '{not json' }]);

    const res = await GET(request());
    expect(res.status).toBe(200);

    const { instances } = await res.json();
    expect(instances[0].limits).toBeNull();
  });

  it('still requires an admin token', async () => {
    mockVerifyAdminToken.mockResolvedValue(false);
    const res = await GET(request());
    expect(res.status).toBe(401);
    expect(mockFindMany).not.toHaveBeenCalled();
  });
});
