import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const env: Record<string, unknown> = {
  SUB2API_INTERNAL_BASE_URL: 'https://test.sub2api.com',
  SUB2API_BASE_URL: 'https://test.sub2api.com',
  JWT_SECRET: 'test-jwt-secret-123456',
  PAY_HELP_TEXT: undefined,
};

vi.mock('@/lib/config', () => ({
  getEnv: () => env,
}));

const mockPlanFindMany = vi.fn();
const mockPromotionFindMany = vi.fn();
const mockChannelFindMany = vi.fn();

vi.mock('@/lib/db', () => ({
  prisma: {
    subscriptionPlan: { findMany: (...args: unknown[]) => mockPlanFindMany(...args) },
    rechargePromotion: { findMany: (...args: unknown[]) => mockPromotionFindMany(...args) },
    channel: { findMany: (...args: unknown[]) => mockChannelFindMany(...args) },
  },
}));

import {
  collectPayTranslationSources,
  lookupContentTranslations,
  parseFeatureTexts,
  resetPayTranslationSyncStateForTests,
  schedulePayTranslationSync,
  syncPayTranslationSources,
  syncPayTranslationSourcesIfStale,
  MAX_PAY_TRANSLATION_SOURCES,
} from '@/lib/sub2api/content-translations';
import { deriveInternalPayToken } from '@/lib/internal-auth';

function okResponse(body: unknown = { code: 0, data: { count: 0 } }) {
  return { ok: true, status: 200, json: () => Promise.resolve(body) };
}

function fetchMock() {
  return fetch as unknown as ReturnType<typeof vi.fn>;
}

beforeEach(() => {
  vi.restoreAllMocks();
  vi.clearAllMocks();
  resetPayTranslationSyncStateForTests();
  env.PAY_HELP_TEXT = undefined;
  mockPlanFindMany.mockResolvedValue([]);
  mockPromotionFindMany.mockResolvedValue([]);
  mockChannelFindMany.mockResolvedValue([]);
  global.fetch = vi.fn().mockResolvedValue(okResponse()) as typeof fetch;
  vi.spyOn(console, 'warn').mockImplementation(() => undefined);
});

afterEach(() => {
  vi.useRealTimers();
  resetPayTranslationSyncStateForTests();
});

describe('parseFeatureTexts', () => {
  it('accepts string items and { text } items, ignores the rest', () => {
    expect(parseFeatureTexts(JSON.stringify(['a', { text: 'b' }, { label: 'c' }, 3, null]))).toEqual(['a', 'b']);
  });

  it('returns [] for empty, malformed or non-array values', () => {
    expect(parseFeatureTexts(null)).toEqual([]);
    expect(parseFeatureTexts('')).toEqual([]);
    expect(parseFeatureTexts('not json')).toEqual([]);
    expect(parseFeatureTexts('{"a":1}')).toEqual([]);
  });
});

describe('collectPayTranslationSources', () => {
  it('collects for-sale plans, live promotions, enabled channels and the help text, trimmed and deduped', async () => {
    env.PAY_HELP_TEXT = '联系客服\n微信：abc';
    mockPlanFindMany.mockResolvedValue([
      {
        name: ' 国模月卡 ',
        description: '适合日常使用',
        features: JSON.stringify(['不限速', { text: '优先调度' }, '  ']),
        productName: '国模月卡',
      },
      { name: 'Pro', description: null, features: 'oops', productName: null },
    ]);
    mockPromotionFindMany.mockResolvedValue([
      { name: '充100送10', description: '' },
      { name: '不限速', description: '限时活动' },
    ]);
    mockChannelFindMany.mockResolvedValue([
      { name: 'Claude 渠道', description: '稳定', features: JSON.stringify(['限时活动', '高并发']) },
    ]);

    const texts = await collectPayTranslationSources(new Date('2026-10-04T00:00:00Z'));

    expect(texts).toEqual([
      '联系客服\n微信：abc',
      '国模月卡',
      '适合日常使用',
      '不限速',
      '优先调度',
      'Pro',
      '充100送10',
      '限时活动',
      'Claude 渠道',
      '稳定',
      '高并发',
    ]);

    expect(mockPlanFindMany.mock.calls[0][0]).toMatchObject({ where: { forSale: true } });
    expect(mockChannelFindMany.mock.calls[0][0]).toMatchObject({ where: { enabled: true } });
    expect(mockPromotionFindMany.mock.calls[0][0]).toMatchObject({
      where: {
        enabled: true,
        OR: [{ endsAt: null }, { endsAt: { gt: new Date('2026-10-04T00:00:00Z') } }],
      },
    });
  });

  it('caps the list at 1000 texts', async () => {
    mockPlanFindMany.mockResolvedValue(
      Array.from({ length: 600 }, (_, i) => ({
        name: `plan ${i}`,
        description: `desc ${i}`,
        features: null,
        productName: null,
      })),
    );

    const texts = await collectPayTranslationSources();
    expect(texts).toHaveLength(MAX_PAY_TRANSLATION_SOURCES);
    expect(texts[0]).toBe('plan 0');
  });
});

describe('syncPayTranslationSources', () => {
  it('PUTs the collected texts to the internal route with the internal pay token', async () => {
    mockPlanFindMany.mockResolvedValue([{ name: '月卡', description: null, features: null, productName: null }]);

    await expect(syncPayTranslationSources()).resolves.toBe(true);

    expect(fetchMock()).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock().mock.calls[0];
    expect(url).toBe('https://test.sub2api.com/api/internal/pay/content-translations/sources');
    expect(init.method).toBe('PUT');
    expect(init.headers['x-sub2api-pay-token']).toBe(deriveInternalPayToken());
    expect(init.headers['Content-Type']).toBe('application/json');
    expect(JSON.parse(init.body)).toEqual({ texts: ['月卡'] });
    expect(init.signal).toBeInstanceOf(AbortSignal);
  });

  it('never throws: HTTP errors, network errors and DB errors resolve to false', async () => {
    fetchMock().mockResolvedValueOnce({ ok: false, status: 404, json: () => Promise.resolve({}) });
    await expect(syncPayTranslationSources()).resolves.toBe(false);

    fetchMock().mockRejectedValueOnce(new TypeError('fetch failed'));
    await expect(syncPayTranslationSources()).resolves.toBe(false);

    mockPlanFindMany.mockRejectedValueOnce(new Error('db down'));
    await expect(syncPayTranslationSources()).resolves.toBe(false);
  });
});

describe('syncPayTranslationSourcesIfStale', () => {
  it('syncs on the first call, then at most once per 10 minutes', async () => {
    const now = vi.spyOn(Date, 'now').mockReturnValue(1_000_000);

    syncPayTranslationSourcesIfStale();
    await vi.waitFor(() => expect(fetchMock()).toHaveBeenCalledTimes(1));

    now.mockReturnValue(1_000_000 + 9 * 60 * 1000);
    syncPayTranslationSourcesIfStale();
    syncPayTranslationSourcesIfStale();
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(fetchMock()).toHaveBeenCalledTimes(1);

    now.mockReturnValue(1_000_000 + 10 * 60 * 1000);
    syncPayTranslationSourcesIfStale();
    await vi.waitFor(() => expect(fetchMock()).toHaveBeenCalledTimes(2));
  });

  it('does not throw even when the sync fails', async () => {
    fetchMock().mockRejectedValue(new TypeError('fetch failed'));
    expect(() => syncPayTranslationSourcesIfStale()).not.toThrow();
    await vi.waitFor(() => expect(console.warn).toHaveBeenCalled());
  });
});

describe('schedulePayTranslationSync', () => {
  it('debounces a burst of admin writes into one sync', async () => {
    vi.useFakeTimers();

    schedulePayTranslationSync();
    schedulePayTranslationSync();
    await vi.advanceTimersByTimeAsync(1500);
    schedulePayTranslationSync();

    await vi.advanceTimersByTimeAsync(1999);
    expect(fetchMock()).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1);
    await vi.waitFor(() => expect(fetchMock()).toHaveBeenCalledTimes(1));
    expect(fetchMock().mock.calls[0][1].method).toBe('PUT');

    // the debounced sync counts as fresh for the stale guard
    syncPayTranslationSourcesIfStale();
    await vi.advanceTimersByTimeAsync(10);
    expect(fetchMock()).toHaveBeenCalledTimes(1);
  });
});

describe('lookupContentTranslations', () => {
  it('POSTs to the public lookup route and returns only requested string translations', async () => {
    fetchMock().mockResolvedValueOnce(
      okResponse({
        code: 0,
        message: 'success',
        data: {
          lang: 'en',
          translations: { 月卡: 'Monthly plan', 未请求: 'Not requested', 稳定: 42, 空: '  ' },
          pending: true,
        },
      }),
    );

    const result = await lookupContentTranslations('en', ['月卡', '稳定', '空']);

    expect(result).toEqual({ translations: { 月卡: 'Monthly plan' }, pending: true });
    const [url, init] = fetchMock().mock.calls[0];
    expect(url).toBe('https://test.sub2api.com/api/v1/content-translations/lookup');
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ lang: 'en', texts: ['月卡', '稳定', '空'] });
  });

  it('keeps a "__proto__" key as plain data', async () => {
    fetchMock().mockResolvedValueOnce(
      okResponse({ code: 0, data: JSON.parse('{"translations":{"__proto__":"proto"},"pending":false}') }),
    );
    const result = await lookupContentTranslations('en', ['__proto__']);
    expect(Object.getPrototypeOf(result.translations)).toBe(Object.prototype);
    expect(Object.prototype.hasOwnProperty.call(result.translations, '__proto__')).toBe(true);
  });

  it('returns empty translations on failure', async () => {
    fetchMock().mockResolvedValueOnce({ ok: false, status: 500, json: () => Promise.resolve({}) });
    await expect(lookupContentTranslations('en', ['月卡'])).resolves.toEqual({ translations: {}, pending: false });

    fetchMock().mockRejectedValueOnce(new TypeError('fetch failed'));
    await expect(lookupContentTranslations('en', ['月卡'])).resolves.toEqual({ translations: {}, pending: false });

    fetchMock().mockResolvedValueOnce(okResponse({ code: 0, data: null }));
    await expect(lookupContentTranslations('en', ['月卡'])).resolves.toEqual({ translations: {}, pending: false });
  });

  it('skips the request when there is nothing to look up', async () => {
    await expect(lookupContentTranslations('en', [])).resolves.toEqual({ translations: {}, pending: false });
    expect(fetchMock()).not.toHaveBeenCalled();
  });
});
