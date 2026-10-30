import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';

const mocks = vi.hoisted(() => ({
  auth: vi.fn(), group: vi.fn(), models: vi.fn(), plans: vi.fn(),
}));
vi.mock('@/lib/db', () => ({ prisma: { subscriptionPlan: { findMany: mocks.plans } } }));
vi.mock('@/lib/sub2api/client', () => ({
  getCurrentUserByToken: mocks.auth, getGroup: mocks.group, getGroupModels: mocks.models,
}));
vi.mock('@/lib/sub2api/content-translations', () => ({ syncPayTranslationSourcesIfStale: vi.fn() }));
import { GET } from '@/app/api/subscription-plans/route';

const request = () => new NextRequest('http://localhost/api/subscription-plans', { headers: { Authorization: 'Bearer fixture' } });
beforeEach(() => {
  vi.resetAllMocks();
  mocks.auth.mockResolvedValue({ id: 1 });
  mocks.group.mockResolvedValue({ id: 10, name: 'Group', status: 'active', daily_limit_usd: 10 });
  mocks.plans.mockResolvedValue([
    { id: 'monthly', groupId: 10, price: 99 }, { id: 'weekly', groupId: 10, price: 29 },
  ]);
});

describe('for-sale plan model catalog', () => {
  it('authenticates first and loads each offered group once even before purchase', async () => {
    mocks.models.mockResolvedValue([{ model: 'actual-model', effective_pricing_usd: { input_per_mtok_usd: 3 } }]);
    const res = await GET(request());
    const data = await res.json();
    expect(res.status).toBe(200);
    expect(data.plans).toHaveLength(2);
    expect(data.plans[0].models[0].model).toBe('actual-model');
    expect(mocks.models).toHaveBeenCalledTimes(1);
    expect(mocks.models).toHaveBeenCalledWith(10);
  });
  it('keeps models unavailable on bridge failure without removing purchasable plans', async () => {
    mocks.models.mockRejectedValue(new Error('Bridge unavailable'));
    const data = await (await GET(request())).json();
    expect(data.plans).toHaveLength(2);
    expect(data.plans[0].models).toBeNull();
  });
  it('never loads prices without a valid user session', async () => {
    mocks.auth.mockRejectedValue(new Error('Invalid token'));
    expect((await GET(request())).status).toBe(401);
    expect(mocks.models).not.toHaveBeenCalled();
    expect(mocks.plans).not.toHaveBeenCalled();
  });
});
