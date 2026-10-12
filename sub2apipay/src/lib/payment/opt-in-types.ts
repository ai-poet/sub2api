import { OPT_IN_PAYMENT_TYPES } from '@/lib/constants';

/** 是否为必须显式启用的支付方式（见 OPT_IN_PAYMENT_TYPES）。 */
export function isOptInPaymentType(type: string): boolean {
  return OPT_IN_PAYMENT_TYPES.has(type);
}

/** 解析逗号分隔的支付方式列表。 */
export function parsePaymentTypeList(raw: string | null | undefined): string[] {
  return (raw ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

/**
 * 服务商实例是否支持某支付方式：「支持方式」留空表示支持全部，
 * 但必须显式启用的方式只认明确勾选。
 */
export function instanceSupportsPaymentType(instanceSupportedTypes: string | null | undefined, type: string): boolean {
  const types = parsePaymentTypeList(instanceSupportedTypes);
  if (types.length === 0) return !isOptInPaymentType(type);
  return types.includes(type);
}

/** 后台「全选」类默认值：去掉必须显式启用的方式。 */
export function withoutOptInPaymentTypes(types: readonly string[]): string[] {
  return types.filter((type) => !isOptInPaymentType(type));
}
