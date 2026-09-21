export type ValidityUnit = 'day' | 'week' | 'month';

/**
 * 根据数值和单位计算实际有效天数。
 * - day: 直接返回
 * - week: value * 7
 * - month: 从 fromDate 到 value 个月后同一天的天数差
 */
export function computeValidityDays(value: number, unit: ValidityUnit, fromDate?: Date): number {
  if (unit === 'day') return value;
  if (unit === 'week') return value * 7;

  // month: 计算到 value 个月后同一天的天数差
  const from = fromDate ?? new Date();
  const target = new Date(from);
  target.setMonth(target.getMonth() + value);
  return Math.round((target.getTime() - from.getTime()) / (1000 * 60 * 60 * 24));
}

/**
 * 格式化有效期显示文本（配置什么就显示什么，不做转换）。
 * - unit=month, value=1 → 1月 / 1 Month
 * - unit=week, value=2 → 2周 / 2 Weeks
 * - unit=day, value=30 → 30天 / 30 Days
 */
export function formatValidityLabel(value: number, unit: ValidityUnit, locale: 'zh' | 'en'): string {
  const unitLabels: Record<ValidityUnit, { zh: string; en: string; enPlural: string }> = {
    day: { zh: '天', en: 'Day', enPlural: 'Days' },
    week: { zh: '周', en: 'Week', enPlural: 'Weeks' },
    month: { zh: '月', en: 'Month', enPlural: 'Months' },
  };
  const u = unitLabels[unit];
  if (locale === 'zh') return `${value}${u.zh}`;
  return `${value} ${value === 1 ? u.en : u.enPlural}`;
}

/**
 * 格式化有效期后缀（用于价格展示，配置什么就显示什么）。
 * - unit=month, value=1 → /1月 / /1mo
 * - unit=week, value=2 → /2周 / /2wk
 * - unit=day, value=30 → /30天 / /30d
 */
export function formatValiditySuffix(value: number, unit: ValidityUnit, locale: 'zh' | 'en'): string {
  const unitLabels: Record<ValidityUnit, { zh: string; en: string }> = {
    day: { zh: '天', en: 'd' },
    week: { zh: '周', en: 'wk' },
    month: { zh: '月', en: 'mo' },
  };
  const u = unitLabels[unit];
  if (locale === 'zh') return `/${value}${u.zh}`;
  return `/${value}${u.en}`;
}

/**
 * 格式化有效期列表展示文本（管理后台表格用）。
 * - unit=day → "30 天"
 * - unit=week → "2 周"
 * - unit=month → "1 月"
 */
export function formatValidityDisplay(value: number, unit: ValidityUnit, locale: 'zh' | 'en'): string {
  const unitLabels: Record<ValidityUnit, { zh: string; en: string }> = {
    day: { zh: '天', en: 'day(s)' },
    week: { zh: '周', en: 'week(s)' },
    month: { zh: '月', en: 'month(s)' },
  };
  const label = locale === 'zh' ? unitLabels[unit].zh : unitLabels[unit].en;
  return `${value} ${label}`;
}

/**
 * 用量金额展示：常规两位小数；小于 1 分但不为 0 时给四位，
 * 免得几次小模型的测试请求显示成 $0.00 让人以为没记上。
 */
export function formatUsageUsd(value: number | null | undefined): string {
  const v = typeof value === 'number' && Number.isFinite(value) ? value : 0;
  if (v > 0 && v < 0.01) return v.toFixed(4);
  return v.toFixed(2);
}

/** 已用 / 上限 的比例，截到 [0, 1]；没有上限（null / 0 / 非法）返回 null。 */
export function usageWindowRatio(used: number | null | undefined, limit: number | null | undefined): number | null {
  if (typeof limit !== 'number' || !Number.isFinite(limit) || limit <= 0) return null;
  const u = typeof used === 'number' && Number.isFinite(used) ? used : 0;
  return Math.min(1, Math.max(0, u / limit));
}

export type UsageWindowTone = 'ok' | 'warn' | 'over';

/** 进度条配色：≥100% 红，≥80% 黄，其余绿。 */
export function usageWindowTone(ratio: number): UsageWindowTone {
  if (ratio >= 1) return 'over';
  if (ratio >= 0.8) return 'warn';
  return 'ok';
}
