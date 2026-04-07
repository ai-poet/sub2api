export type Locale = 'zh' | 'en';

export function resolveLocale(lang: string | null | undefined): Locale {
  const normalized = lang
    ?.split(',')[0]
    ?.split(';')[0]
    ?.trim()
    ?.toLowerCase();

  if (!normalized) return 'zh';
  if (normalized.startsWith('zh')) return 'zh';
  if (normalized.startsWith('en')) return 'en';
  return 'en';
}

export function isEnglish(locale: Locale): boolean {
  return locale === 'en';
}

export function pickLocaleText<T>(locale: Locale, zh: T, en: T): T {
  return locale === 'en' ? en : zh;
}

export function applyLocaleToSearchParams(params: URLSearchParams, locale: Locale): URLSearchParams {
  if (locale === 'en') {
    params.set('lang', 'en');
  }
  return params;
}
