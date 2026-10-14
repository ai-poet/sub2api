/**
 * 支付表单的到账金额上限，0 = 不限。
 *
 * 渠道单笔上限（含当天剩余日额度换算出的上限）只能收紧后台「最大到账金额」，不能放宽：
 * 下单接口只按全局最小 / 最大到账金额校验，表单若直接采用更大的渠道上限，用户填了也会被拒。
 * 任一侧为 0 或缺省时该侧不限。
 */
export function resolveAmountMax(globalMax: number, methodSingleMax?: number): number {
  const methodMax = methodSingleMax !== undefined && methodSingleMax > 0 ? methodSingleMax : 0;
  if (globalMax <= 0) return methodMax;
  return methodMax > 0 ? Math.min(methodMax, globalMax) : globalMax;
}

/** 金额输入框的范围提示：只列出设置了的那一侧，都没设置时返回空串。 */
export function amountRangePlaceholder(min: number, max: number): string {
  if (min > 0 && max > 0) return `${min} - ${max}`;
  if (min > 0) return `≥ ${min}`;
  if (max > 0) return `≤ ${max}`;
  return '';
}
