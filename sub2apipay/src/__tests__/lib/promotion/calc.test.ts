import { describe, it, expect } from 'vitest';
import {
  MAX_CREDIT_AMOUNT,
  computeBonus,
  describePromotion,
  formatAmount,
  isPromotionInWindow,
  mergeQuickAmounts,
  nextPromotionHint,
  pickBestPromotion,
  type PromotionRule,
} from '@/lib/promotion/calc';

function rule(overrides: Partial<PromotionRule> = {}): PromotionRule {
  return {
    id: 'p1',
    name: '充100送10',
    description: null,
    minAmount: 100,
    bonusType: 'fixed',
    bonusValue: 10,
    maxBonus: null,
    sortOrder: 0,
    ...overrides,
  };
}

describe('computeBonus', () => {
  it('未达门槛返回 0', () => {
    expect(computeBonus(99.99, rule())).toBe(0);
    expect(computeBonus(0, rule())).toBe(0);
    expect(computeBonus(-5, rule())).toBe(0);
    expect(computeBonus(Number.NaN, rule())).toBe(0);
  });

  it('fixed：达到门槛返回固定赠送额', () => {
    expect(computeBonus(100, rule())).toBe(10);
    expect(computeBonus(999, rule())).toBe(10);
  });

  it('percent：按到账金额比例向下取整到分', () => {
    const p = rule({ bonusType: 'percent', bonusValue: 10 });
    expect(computeBonus(100, p)).toBe(10);
    expect(computeBonus(333.33, p)).toBe(33.33);
    expect(computeBonus(100.05, p)).toBe(10); // 10.005 → 10.00
  });

  it('percent：支持小数百分比', () => {
    const p = rule({ bonusType: 'percent', bonusValue: 12.5 });
    expect(computeBonus(200, p)).toBe(25);
  });

  it('percent：按 maxBonus 封顶', () => {
    const p = rule({ bonusType: 'percent', bonusValue: 10, maxBonus: 50 });
    expect(computeBonus(400, p)).toBe(40);
    expect(computeBonus(1000, p)).toBe(50);
  });

  it('无效规则返回 0', () => {
    expect(computeBonus(100, rule({ bonusValue: 0 }))).toBe(0);
    expect(computeBonus(100, rule({ bonusValue: -1 }))).toBe(0);
  });

  it('到账总额不超过 Decimal(10,2) 上限', () => {
    const p = rule({ bonusType: 'percent', bonusValue: 100 });
    const amount = MAX_CREDIT_AMOUNT - 1;
    expect(computeBonus(amount, p)).toBe(1);
  });
});

describe('pickBestPromotion', () => {
  const fixed100 = rule({ id: 'a', name: 'A', minAmount: 100, bonusValue: 10, sortOrder: 1 });
  const fixed500 = rule({ id: 'b', name: 'B', minAmount: 500, bonusValue: 80, sortOrder: 2 });
  const percent = rule({ id: 'c', name: 'C', minAmount: 200, bonusType: 'percent', bonusValue: 10, sortOrder: 3 });

  it('无匹配返回 null', () => {
    expect(pickBestPromotion(50, [fixed100, fixed500, percent])).toBeNull();
    expect(pickBestPromotion(100, [])).toBeNull();
  });

  it('多个活动同时满足时取赠送最高者，不叠加', () => {
    const best = pickBestPromotion(600, [fixed100, fixed500, percent]);
    expect(best?.rule.id).toBe('b');
    expect(best?.bonus).toBe(80);

    const mid = pickBestPromotion(300, [fixed100, fixed500, percent]);
    expect(mid?.rule.id).toBe('c');
    expect(mid?.bonus).toBe(30);
  });

  it('赠送额并列时取 sortOrder 小者，再并列取 id 小者', () => {
    const x = rule({ id: 'x', sortOrder: 5, bonusValue: 10 });
    const y = rule({ id: 'y', sortOrder: 2, bonusValue: 10 });
    const z = rule({ id: 'a', sortOrder: 2, bonusValue: 10 });
    expect(pickBestPromotion(100, [x, y, z])?.rule.id).toBe('a');
    expect(pickBestPromotion(100, [x, y])?.rule.id).toBe('y');
  });
});

describe('nextPromotionHint', () => {
  const fixed100 = rule({ id: 'a', name: 'A', minAmount: 100, bonusValue: 10 });
  const fixed500 = rule({ id: 'b', name: 'B', minAmount: 500, bonusValue: 80 });

  it('返回最近的更高门槛与差额', () => {
    const hint = nextPromotionHint(90, [fixed100, fixed500]);
    expect(hint).toEqual({ rule: fixed100, threshold: 100, needMore: 10, bonus: 10 });
  });

  it('当前已有赠送时提示下一档', () => {
    const hint = nextPromotionHint(120, [fixed100, fixed500]);
    expect(hint?.rule.id).toBe('b');
    expect(hint?.needMore).toBe(380);
  });

  it('更高门槛不能带来更高赠送时跳过', () => {
    const lower = rule({ id: 'c', name: 'C', minAmount: 300, bonusValue: 5 });
    const hint = nextPromotionHint(120, [fixed100, lower]);
    expect(hint).toBeNull();
  });

  it('没有更高门槛返回 null', () => {
    expect(nextPromotionHint(600, [fixed100, fixed500])).toBeNull();
    expect(nextPromotionHint(0, [])).toBeNull();
  });
});

describe('mergeQuickAmounts', () => {
  it('合并门槛、去重、升序并限定范围', () => {
    expect(mergeQuickAmounts([10, 50, 100], [100, 300, 5000, 0.5], 1, 1000)).toEqual([10, 50, 100, 300]);
  });

  it('忽略非法值', () => {
    expect(mergeQuickAmounts([Number.NaN, -1, 20], [], 1, 100)).toEqual([20]);
  });
});

describe('isPromotionInWindow', () => {
  const now = new Date('2026-09-02T00:00:00Z');

  it('无时间限制视为进行中', () => {
    expect(isPromotionInWindow({ startsAt: null, endsAt: null }, now)).toBe(true);
  });

  it('未开始 / 已结束返回 false', () => {
    expect(isPromotionInWindow({ startsAt: new Date('2026-09-03T00:00:00Z'), endsAt: null }, now)).toBe(false);
    expect(isPromotionInWindow({ startsAt: null, endsAt: '2026-09-02T00:00:00Z' }, now)).toBe(false);
    expect(isPromotionInWindow({ startsAt: '2026-09-01T00:00:00Z', endsAt: '2026-09-03T00:00:00Z' }, now)).toBe(true);
  });
});

describe('describePromotion / formatAmount', () => {
  it('整数金额不带小数', () => {
    expect(formatAmount(100)).toBe('100');
    expect(formatAmount(12.5)).toBe('12.50');
  });

  it('生成中英文描述', () => {
    expect(describePromotion(rule(), 'zh')).toBe('充 $100 送 $10');
    expect(describePromotion(rule(), 'en')).toBe('Top up $100+ and get $10 bonus');
    const p = rule({ bonusType: 'percent', bonusValue: 10, maxBonus: 50 });
    expect(describePromotion(p, 'zh')).toBe('充 $100 送 10%（最高 $50）');
    expect(describePromotion(p, 'en')).toBe('Top up $100+ and get 10% bonus (up to $50)');
  });
});
