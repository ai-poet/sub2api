/**
 * 充值活动（充 X 送 Y）的纯计算逻辑。
 *
 * 前后端共用：服务端在下单事务里用它决定赠送额并快照到订单，客户端只用它做预览。
 * 这里不能引入 prisma / 订单服务等仅服务端可用的模块。
 */
import type { Locale } from '@/lib/locale';

export type BonusType = 'fixed' | 'percent';

export interface PromotionRule {
  id: string;
  name: string;
  description: string | null;
  /** 到账门槛（USD），amount >= minAmount 时生效 */
  minAmount: number;
  bonusType: BonusType;
  /** fixed：赠送 USD；percent：按到账金额的百分比（如 10 = 10%） */
  bonusValue: number;
  /** percent 模式的赠送封顶（USD），null = 不封顶 */
  maxBonus: number | null;
  sortOrder: number;
}

/** `/api/user` 返回给充值页的活动信息（只含进行中的活动） */
export interface PublicPromotion extends PromotionRule {
  startsAt: string | null;
  endsAt: string | null;
  /** false = 当前用户已达每人次数上限或总名额已满，仅展示不参与计算 */
  available: boolean;
}

export interface PromotionMatch {
  rule: PromotionRule;
  bonus: number;
}

export interface PromotionHint {
  rule: PromotionRule;
  /** 触发该活动的到账门槛 */
  threshold: number;
  /** 还差多少到账金额 */
  needMore: number;
  /** 达到门槛后的赠送额 */
  bonus: number;
}

/** Decimal(10,2) 可表示的最大金额；赠送后到账总额不得超过它 */
export const MAX_CREDIT_AMOUNT = 99999999.99;

function toCents(value: number): number {
  return Math.round(value * 100);
}

function fromCents(cents: number): number {
  return cents / 100;
}

/** 达到门槛时的赠送额（USD，向下取整到分）；未达门槛或规则无效返回 0 */
export function computeBonus(amount: number, rule: PromotionRule): number {
  if (!Number.isFinite(amount) || amount <= 0) return 0;
  if (!Number.isFinite(rule.minAmount) || !Number.isFinite(rule.bonusValue) || rule.bonusValue <= 0) return 0;

  const amountCents = toCents(amount);
  if (amountCents < toCents(rule.minAmount)) return 0;

  let bonusCents: number;
  if (rule.bonusType === 'percent') {
    bonusCents = Math.floor((amountCents * rule.bonusValue) / 100 + 1e-9);
    if (rule.maxBonus != null && Number.isFinite(rule.maxBonus) && rule.maxBonus > 0) {
      bonusCents = Math.min(bonusCents, toCents(rule.maxBonus));
    }
  } else {
    bonusCents = toCents(rule.bonusValue);
  }

  // 到账总额（amount + bonus）不能超过 Decimal(10,2) 上限
  const headroomCents = toCents(MAX_CREDIT_AMOUNT) - amountCents;
  bonusCents = Math.max(0, Math.min(bonusCents, headroomCents));
  return fromCents(bonusCents);
}

/**
 * 多个活动同时满足时不叠加：取赠送最高者，并列时取 sortOrder 小者，再并列取 id 小者。
 * 没有任何活动产生赠送时返回 null。
 */
export function pickBestPromotion(amount: number, rules: PromotionRule[]): PromotionMatch | null {
  let best: PromotionMatch | null = null;
  for (const rule of rules) {
    const bonus = computeBonus(amount, rule);
    if (bonus <= 0) continue;
    if (
      !best ||
      bonus > best.bonus ||
      (bonus === best.bonus &&
        (rule.sortOrder < best.rule.sortOrder || (rule.sortOrder === best.rule.sortOrder && rule.id < best.rule.id)))
    ) {
      best = { rule, bonus };
    }
  }
  return best;
}

/**
 * “再充 $Z 可享…”提示：在高于当前金额的门槛中，找到最近的一个能带来更高赠送的门槛。
 * 当前金额已经是最优或没有更高门槛时返回 null。
 */
export function nextPromotionHint(amount: number, rules: PromotionRule[]): PromotionHint | null {
  const safeAmount = Number.isFinite(amount) && amount > 0 ? amount : 0;
  const currentBonus = pickBestPromotion(safeAmount, rules)?.bonus ?? 0;
  const amountCents = toCents(safeAmount);

  const thresholds = [...new Set(rules.map((r) => toCents(r.minAmount)))]
    .filter((cents) => cents > amountCents)
    .sort((a, b) => a - b);

  for (const cents of thresholds) {
    const threshold = fromCents(cents);
    const match = pickBestPromotion(threshold, rules);
    if (match && match.bonus > currentBonus) {
      return {
        rule: match.rule,
        threshold,
        needMore: fromCents(cents - amountCents),
        bonus: match.bonus,
      };
    }
  }
  return null;
}

/** 把活动门槛并入快捷金额：去重、升序、限定在 [min, max] 内 */
export function mergeQuickAmounts(base: number[], thresholds: number[], min: number, max: number): number[] {
  const set = new Set<number>();
  for (const value of [...base, ...thresholds]) {
    if (!Number.isFinite(value) || value <= 0) continue;
    const rounded = fromCents(toCents(value));
    if (rounded < min || rounded > max) continue;
    set.add(rounded);
  }
  return [...set].sort((a, b) => a - b);
}

export function isPromotionInWindow(
  promo: { startsAt: Date | string | null; endsAt: Date | string | null },
  now: Date = new Date(),
): boolean {
  const ts = now.getTime();
  if (promo.startsAt) {
    const start = new Date(promo.startsAt).getTime();
    if (Number.isFinite(start) && ts < start) return false;
  }
  if (promo.endsAt) {
    const end = new Date(promo.endsAt).getTime();
    if (Number.isFinite(end) && ts >= end) return false;
  }
  return true;
}

/** 整数不带小数位，否则保留两位：100 → "100"，12.5 → "12.50" */
export function formatAmount(value: number): string {
  if (!Number.isFinite(value)) return '0';
  const rounded = fromCents(toCents(value));
  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(2);
}

function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '0';
  return Number.isInteger(value) ? String(value) : String(Math.round(value * 100) / 100);
}

/** “充 $100 送 $10” / “充 $100 送 10%（最高 $50）” */
export function describePromotion(rule: PromotionRule, locale: Locale): string {
  const min = formatAmount(rule.minAmount);
  if (rule.bonusType === 'percent') {
    const pct = formatPercent(rule.bonusValue);
    const cap =
      rule.maxBonus != null && rule.maxBonus > 0
        ? locale === 'en'
          ? ` (up to $${formatAmount(rule.maxBonus)})`
          : `（最高 $${formatAmount(rule.maxBonus)}）`
        : '';
    return locale === 'en' ? `Top up $${min}+ and get ${pct}% bonus${cap}` : `充 $${min} 送 ${pct}%${cap}`;
  }
  const bonus = formatAmount(rule.bonusValue);
  return locale === 'en' ? `Top up $${min}+ and get $${bonus} bonus` : `充 $${min} 送 $${bonus}`;
}
