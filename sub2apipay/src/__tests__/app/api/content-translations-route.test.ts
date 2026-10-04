import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NextRequest } from 'next/server';

const mockLookupContentTranslations = vi.fn();

vi.mock('@/lib/sub2api/content-translations', () => ({
  MAX_LOOKUP_TEXTS: 200,
  MAX_TRANSLATION_TEXT_LENGTH: 20_000,
  lookupContentTranslations: (...args: unknown[]) => mockLookupContentTranslations(...args),
}));

import { POST } from '@/app/api/content-translations/route';

function createRequest(body: unknown, init?: { headers?: Record<string, string>; query?: string }) {
  return new NextRequest(`https://pay.example.com/api/content-translations${init?.query ?? ''}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    body: typeof body === 'string' ? body : JSON.stringify(body),
  });
}

describe('POST /api/content-translations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLookupContentTranslations.mockResolvedValue({ translations: { 月卡: 'Monthly plan' }, pending: true });
  });

  it('forwards the resolved locale and deduped texts, returning translations and pending', async () => {
    const res = await POST(createRequest({ lang: 'en-US', texts: ['月卡', '月卡', '  ', '', '稳定'] }));

    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ translations: { 月卡: 'Monthly plan' }, pending: true });
    expect(mockLookupContentTranslations).toHaveBeenCalledWith('en', ['月卡', '稳定']);
  });

  it('defaults to zh when lang is missing', async () => {
    await POST(createRequest({ texts: ['Pro'] }));
    expect(mockLookupContentTranslations).toHaveBeenCalledWith('zh', ['Pro']);
  });

  it('drops texts the backend would ignore anyway', async () => {
    await POST(createRequest({ lang: 'en', texts: ['月卡', 'x'.repeat(20_001)] }));
    expect(mockLookupContentTranslations).toHaveBeenCalledWith('en', ['月卡']);
  });

  it('answers an empty list without calling the backend', async () => {
    const res = await POST(createRequest({ lang: 'en', texts: [] }));
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ translations: {}, pending: false });
    expect(mockLookupContentTranslations).not.toHaveBeenCalled();
  });

  it.each([
    ['texts missing', { lang: 'en' }],
    ['texts not an array', { lang: 'en', texts: '月卡' }],
    ['non-string item', { lang: 'en', texts: ['月卡', 1] }],
    ['object item', { lang: 'en', texts: [{ text: '月卡' }] }],
    ['non-string lang', { lang: 1, texts: ['月卡'] }],
    ['array body', ['月卡']],
  ])('rejects %s with 400', async (_name, body) => {
    const res = await POST(createRequest(body));
    expect(res.status).toBe(400);
    expect((await res.json()).error).toBeTruthy();
    expect(mockLookupContentTranslations).not.toHaveBeenCalled();
  });

  it('rejects invalid JSON with 400', async () => {
    const res = await POST(createRequest('{not json'));
    expect(res.status).toBe(400);
  });

  it('rejects more than 200 texts with 400', async () => {
    const texts = Array.from({ length: 201 }, (_, i) => `text ${i}`);
    const res = await POST(createRequest({ lang: 'en', texts }, { query: '?lang=en' }));
    expect(res.status).toBe(400);
    expect((await res.json()).error).toContain('200');
    expect(mockLookupContentTranslations).not.toHaveBeenCalled();
  });

  it('rejects bodies over 256 KiB with 413', async () => {
    const res = await POST(createRequest({ lang: 'en', texts: ['x'.repeat(300 * 1024)] }));
    expect(res.status).toBe(413);
    expect(mockLookupContentTranslations).not.toHaveBeenCalled();
  });

  it('rejects a declared Content-Length over the limit before reading the body', async () => {
    const res = await POST(
      createRequest({ lang: 'en', texts: ['月卡'] }, { headers: { 'Content-Length': String(512 * 1024) } }),
    );
    expect(res.status).toBe(413);
    expect(mockLookupContentTranslations).not.toHaveBeenCalled();
  });

  it('localizes errors by the body lang once it is known', async () => {
    const res = await POST(createRequest({ lang: 'en', texts: 'nope' }));
    expect((await res.json()).error).toBe('texts must be an array of strings');

    const zh = await POST(createRequest({ lang: 'zh', texts: 'nope' }));
    expect((await zh.json()).error).toBe('texts 必须是字符串数组');
  });
});
