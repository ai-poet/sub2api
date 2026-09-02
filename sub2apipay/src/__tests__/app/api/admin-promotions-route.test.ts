import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest, NextResponse } from 'next/server';
import { Prisma } from '@prisma/client';

const mockVerifyAdminToken = vi.fn();
const mockPromotionFindMany = vi.fn();
const mockPromotionFindUnique = vi.fn();
const mockPromotionCreate = vi.fn();
const mockPromotionUpdate = vi.fn();
const mockPromotionDelete = vi.fn();
const mockOrderCount = vi.fn();
const mockGetPromotionUsageStats = vi.fn();

vi.mock('@/lib/admin-auth', () => ({
  verifyAdminToken: (...args: unknown[]) => mockVerifyAdminToken(...args),
  unauthorizedResponse: () => NextResponse.json({ error: '未授权' }, { status: 401 }),
}));

vi.mock('@/lib/db', () => ({
  prisma: {
    rechargePromotion: {
      findMany: (...args: unknown[]) => mockPromotionFindMany(...args),
      findUnique: (...args: unknown[]) => mockPromotionFindUnique(...args),
      create: (...args: unknown[]) => mockPromotionCreate(...args),
      update: (...args: unknown[]) => mockPromotionUpdate(...args),
      delete: (...args: unknown[]) => mockPromotionDelete(...args),
    },
    order: {
      count: (...args: unknown[]) => mockOrderCount(...args),
    },
  },
}));

vi.mock('@/lib/promotion/service', () => ({
  getPromotionUsageStats: (...args: unknown[]) => mockGetPromotionUsageStats(...args),
}));

import { GET, POST } from '@/app/api/admin/promotions/route';
import { PUT, DELETE } from '@/app/api/admin/promotions/[id]/route';

function createRequest(method = 'GET', body?: object, path = '/api/admin/promotions') {
  const headers: Record<string, string> = { Authorization: 'Bearer test-admin-token' };
  if (body) headers['Content-Type'] = 'application/json';
  return new NextRequest(`https://pay.example.com${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
}

function makeRecord(overrides: Record<string, unknown> = {}) {
  return {
    id: 'promo-1',
    name: '充100送10',
    description: null,
    minAmount: new Prisma.Decimal('100.00'),
    bonusType: 'fixed',
    bonusValue: new Prisma.Decimal('10.00'),
    maxBonus: null,
    startsAt: null,
    endsAt: null,
    perUserLimit: 0,
    totalLimit: 0,
    enabled: true,
    sortOrder: 0,
    createdAt: new Date('2026-09-01T00:00:00Z'),
    updatedAt: new Date('2026-09-01T00:00:00Z'),
    ...overrides,
  };
}

const validBody = {
  name: '充100送10',
  min_amount: 100,
  bonus_type: 'fixed',
  bonus_value: 10,
};

beforeEach(() => {
  vi.clearAllMocks();
  mockVerifyAdminToken.mockResolvedValue(true);
  mockGetPromotionUsageStats.mockResolvedValue(new Map());
  mockOrderCount.mockResolvedValue(0);
});

describe('GET /api/admin/promotions', () => {
  it('returns 401 when unauthenticated', async () => {
    mockVerifyAdminToken.mockResolvedValue(false);
    const res = await GET(createRequest());
    expect(res.status).toBe(401);
  });

  it('returns promotions with usage stats and status', async () => {
    mockPromotionFindMany.mockResolvedValue([
      makeRecord(),
      makeRecord({ id: 'promo-2', enabled: false }),
      makeRecord({ id: 'promo-3', totalLimit: 2 }),
    ]);
    mockGetPromotionUsageStats.mockResolvedValue(
      new Map([
        ['promo-1', { usedCount: 3, bonusTotal: 30 }],
        ['promo-3', { usedCount: 2, bonusTotal: 20 }],
      ]),
    );

    const res = await GET(createRequest());
    const data = await res.json();

    expect(res.status).toBe(200);
    expect(data.promotions).toHaveLength(3);
    expect(data.promotions[0]).toMatchObject({
      id: 'promo-1',
      minAmount: 100,
      bonusValue: 10,
      usedCount: 3,
      bonusTotal: 30,
      status: 'active',
    });
    expect(data.promotions[1].status).toBe('disabled');
    expect(data.promotions[2].status).toBe('exhausted');
  });
});

describe('POST /api/admin/promotions', () => {
  it('returns 401 when unauthenticated', async () => {
    mockVerifyAdminToken.mockResolvedValue(false);
    const res = await POST(createRequest('POST', validBody));
    expect(res.status).toBe(401);
  });

  it('rejects invalid body', async () => {
    const res = await POST(createRequest('POST', { ...validBody, min_amount: -1 }));
    expect(res.status).toBe(400);
    const data = await res.json();
    expect(data.details.min_amount).toBeTruthy();
    expect(mockPromotionCreate).not.toHaveBeenCalled();
  });

  it('rejects ends_at earlier than starts_at', async () => {
    const res = await POST(
      createRequest('POST', {
        ...validBody,
        starts_at: '2026-09-10T00:00:00.000Z',
        ends_at: '2026-09-01T00:00:00.000Z',
      }),
    );
    expect(res.status).toBe(400);
    const data = await res.json();
    expect(data.details.ends_at).toBeTruthy();
  });

  it('rejects percent bonus over 1000', async () => {
    const res = await POST(createRequest('POST', { ...validBody, bonus_type: 'percent', bonus_value: 2000 }));
    expect(res.status).toBe(400);
  });

  it('creates a promotion and returns 201', async () => {
    mockPromotionCreate.mockResolvedValue(
      makeRecord({
        bonusType: 'percent',
        bonusValue: new Prisma.Decimal('10.00'),
        maxBonus: new Prisma.Decimal('50.00'),
        endsAt: new Date('2026-09-30T00:00:00Z'),
        perUserLimit: 1,
      }),
    );

    const res = await POST(
      createRequest('POST', {
        ...validBody,
        bonus_type: 'percent',
        bonus_value: 10,
        max_bonus: 50,
        ends_at: '2026-09-30T00:00:00.000Z',
        per_user_limit: 1,
      }),
    );

    expect(res.status).toBe(201);
    const createArgs = mockPromotionCreate.mock.calls[0][0];
    expect(createArgs.data).toMatchObject({
      name: '充100送10',
      bonusType: 'percent',
      perUserLimit: 1,
      totalLimit: 0,
      enabled: true,
    });
    expect(createArgs.data.minAmount.toString()).toBe('100');
    expect(createArgs.data.maxBonus.toString()).toBe('50');
    expect(createArgs.data.endsAt).toEqual(new Date('2026-09-30T00:00:00.000Z'));

    const data = await res.json();
    expect(data).toMatchObject({ id: 'promo-1', bonusType: 'percent', maxBonus: 50, status: 'active' });
  });

  it('drops max_bonus for fixed promotions', async () => {
    mockPromotionCreate.mockResolvedValue(makeRecord());
    await POST(createRequest('POST', { ...validBody, max_bonus: 50 }));
    expect(mockPromotionCreate.mock.calls[0][0].data.maxBonus).toBeNull();
  });
});

describe('PUT /api/admin/promotions/[id]', () => {
  const params = Promise.resolve({ id: 'promo-1' });

  it('returns 404 when promotion does not exist', async () => {
    mockPromotionFindUnique.mockResolvedValue(null);
    const res = await PUT(createRequest('PUT', { enabled: false }, '/api/admin/promotions/promo-1'), { params });
    expect(res.status).toBe(404);
  });

  it('merges patch with existing record and validates as a whole', async () => {
    mockPromotionFindUnique.mockResolvedValue(makeRecord({ startsAt: new Date('2026-09-10T00:00:00Z') }));
    const res = await PUT(
      createRequest('PUT', { ends_at: '2026-09-01T00:00:00.000Z' }, '/api/admin/promotions/promo-1'),
      { params },
    );
    expect(res.status).toBe(400);
    expect(mockPromotionUpdate).not.toHaveBeenCalled();
  });

  it('updates partial fields', async () => {
    mockPromotionFindUnique.mockResolvedValue(makeRecord());
    mockPromotionUpdate.mockResolvedValue(makeRecord({ enabled: false }));

    const res = await PUT(createRequest('PUT', { enabled: false }, '/api/admin/promotions/promo-1'), { params });

    expect(res.status).toBe(200);
    expect(mockPromotionUpdate.mock.calls[0][0]).toMatchObject({
      where: { id: 'promo-1' },
      data: { enabled: false, name: '充100送10' },
    });
    const data = await res.json();
    expect(data.status).toBe('disabled');
  });
});

describe('DELETE /api/admin/promotions/[id]', () => {
  const params = Promise.resolve({ id: 'promo-1' });

  it('returns 404 when promotion does not exist', async () => {
    mockPromotionFindUnique.mockResolvedValue(null);
    const res = await DELETE(createRequest('DELETE', undefined, '/api/admin/promotions/promo-1'), { params });
    expect(res.status).toBe(404);
  });

  it('returns 409 when active orders reference the promotion', async () => {
    mockPromotionFindUnique.mockResolvedValue(makeRecord());
    mockOrderCount.mockResolvedValue(2);
    const res = await DELETE(createRequest('DELETE', undefined, '/api/admin/promotions/promo-1'), { params });
    expect(res.status).toBe(409);
    expect(mockPromotionDelete).not.toHaveBeenCalled();
  });

  it('deletes the promotion', async () => {
    mockPromotionFindUnique.mockResolvedValue(makeRecord());
    mockPromotionDelete.mockResolvedValue({});
    const res = await DELETE(createRequest('DELETE', undefined, '/api/admin/promotions/promo-1'), { params });
    expect(res.status).toBe(200);
    expect(mockPromotionDelete).toHaveBeenCalledWith({ where: { id: 'promo-1' } });
  });
});
