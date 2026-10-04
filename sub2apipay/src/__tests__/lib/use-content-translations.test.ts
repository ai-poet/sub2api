import { describe, expect, it } from 'vitest';
import { looksLikeLocale, pickTextsToTranslate } from '@/lib/use-content-translations';

describe('looksLikeLocale', () => {
  it('treats Han without kana as Chinese', () => {
    expect(looksLikeLocale('国模月卡', 'zh')).toBe(true);
    expect(looksLikeLocale('Claude 渠道', 'zh')).toBe(true);
    expect(looksLikeLocale('月額プラン', 'zh')).toBe(false);
    expect(looksLikeLocale('Monthly plan', 'zh')).toBe(false);
  });

  it('treats text without CJK as English', () => {
    expect(looksLikeLocale('Monthly plan', 'en')).toBe(true);
    expect(looksLikeLocale('Claude 渠道', 'en')).toBe(false);
    expect(looksLikeLocale('プラン', 'en')).toBe(false);
    expect(looksLikeLocale('요금제', 'en')).toBe(false);
  });
});

describe('pickTextsToTranslate', () => {
  it('trims, dedupes and skips empties, non-strings, text already in the locale and text without letters', () => {
    expect(
      pickTextsToTranslate([' 月卡 ', '月卡', '', '   ', null, undefined, 42, 'Pro plan', '100', '$9.9'], 'en'),
    ).toEqual(['月卡']);
    expect(pickTextsToTranslate(['月卡', 'Pro plan', 'Pro plan ', 'GPT-5'], 'zh')).toEqual(['Pro plan', 'GPT-5']);
  });

  it('skips texts the backend would ignore', () => {
    expect(pickTextsToTranslate(['月'.repeat(20_001)], 'en')).toEqual([]);
  });
});
