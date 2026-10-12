import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockFindMany = vi.fn();
const mockGetSystemConfig = vi.fn();
const supportedTypes = ['alipay', 'wxpay', 'bank', 'paynow', 'stripe'];

vi.mock('@/lib/db', () => ({
  prisma: {
    paymentProviderInstance: {
      findMany: (...args: unknown[]) => mockFindMany(...args),
    },
  },
}));

vi.mock('@/lib/system-config', () => ({
  getSystemConfig: (...args: unknown[]) => mockGetSystemConfig(...args),
}));

vi.mock('@/lib/payment', () => ({
  ensureDBProviders: vi.fn().mockResolvedValue(undefined),
  paymentRegistry: {
    getSupportedTypes: () => supportedTypes,
    getProviderKey: (type: string) => (type === 'stripe' ? 'stripe' : 'easypay'),
  },
}));

import { getVisiblePaymentTypes } from '@/lib/payment/visibility';

function instance(providerKey: string, types: string) {
  return { providerKey, supportedTypes: types };
}

describe('getVisiblePaymentTypes — opt-in PayNow', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keeps today's visibility when nothing lists PayNow (empty global list, wildcard instance)", async () => {
    mockGetSystemConfig.mockResolvedValue('');
    mockFindMany.mockResolvedValue([instance('easypay', ''), instance('stripe', '')]);

    expect(await getVisiblePaymentTypes()).toEqual(['alipay', 'wxpay', 'bank', 'stripe']);
  });

  it("keeps today's visibility for an instance with an explicit list without PayNow", async () => {
    mockGetSystemConfig.mockResolvedValue(undefined);
    mockFindMany.mockResolvedValue([instance('easypay', 'alipay,wxpay')]);

    expect(await getVisiblePaymentTypes()).toEqual(['alipay', 'wxpay', 'stripe']);
  });

  it('shows PayNow once an active instance lists it', async () => {
    mockGetSystemConfig.mockResolvedValue('');
    mockFindMany.mockResolvedValue([instance('easypay', 'alipay,paynow')]);

    expect(await getVisiblePaymentTypes()).toEqual(['alipay', 'paynow', 'stripe']);
  });

  it('hides PayNow when the global list leaves it out, even if an instance lists it', async () => {
    mockGetSystemConfig.mockResolvedValue('alipay,wxpay');
    mockFindMany.mockResolvedValue([instance('easypay', 'alipay,wxpay,paynow')]);

    expect(await getVisiblePaymentTypes()).toEqual(['alipay', 'wxpay']);
  });

  it('without database instances, shows PayNow only when the global list names it', async () => {
    mockFindMany.mockResolvedValue([]);

    mockGetSystemConfig.mockResolvedValue('');
    expect(await getVisiblePaymentTypes()).toEqual(['alipay', 'wxpay', 'bank', 'stripe']);

    mockGetSystemConfig.mockResolvedValue('alipay,paynow');
    expect(await getVisiblePaymentTypes()).toEqual(['alipay', 'paynow']);
  });
});
