import { prisma } from '@/lib/db';
import { ORDER_STATUS } from '@/lib/constants';
import { getEnv } from '@/lib/config';
import { getMethodFeeConfigs, getMaxSettlementAmountForPayLimit, getMinSettlementAmountForPayLimit } from './fee';
import { getBizDayStartUTC } from '@/lib/time/biz-day';
import { getSystemConfig } from '@/lib/system-config';
import { convertCnySettlementToUsdBalance, resolveBalanceCreditCnyPerUsd } from '@/lib/currency';
import { instanceSupportsPaymentType } from '@/lib/payment/opt-in-types';

/**
 * 渠道级限额只认显式配置：getSystemConfig（DB → process.env）里的 `<前缀>_<渠道>`，
 * 没配置或配置无效时返回 0（不限）。
 *
 * 代码里不写死各渠道的限额：以前各 provider 自带 defaultLimits（如易支付单笔 1000、每日 10000），
 * 后台既看不到也改不了，「最大到账金额」调到 5000 仍被单笔 1000 盖住。网关确有上限时，
 * 在支付实例的「每日限额 / 单笔最大」里配置。
 */
async function getConfiguredMethodLimit(prefix: string, paymentType: string): Promise<number> {
  const configVal = await getSystemConfig(`${prefix}_${paymentType.toUpperCase()}`);
  if (configVal === undefined) return 0;
  const num = Number(configVal);
  return Number.isFinite(num) && num >= 0 ? num : 0;
}

/** 指定支付渠道的每日全平台限额（0 = 不限制）。 */
export function getMethodDailyLimit(paymentType: string): Promise<number> {
  return getConfiguredMethodLimit('MAX_DAILY_AMOUNT', paymentType);
}

/** 指定支付渠道的单笔限额（0 = 使用全局 MAX_RECHARGE_AMOUNT）。 */
export function getMethodSingleLimit(paymentType: string): Promise<number> {
  return getConfiguredMethodLimit('MAX_SINGLE_AMOUNT', paymentType);
}

export interface MethodLimitStatus {
  dailyLimit: number;
  used: number;
  remaining: number | null;
  available: boolean;
  singleMin: number;
  singleMax: number;
  feeRate: number;
  fixedFee: number;
  feeEnabled: boolean;
}

interface InstanceChannelLimits {
  dailyLimit?: number;
  singleMin?: number;
  singleMax?: number;
}

function roundDownToCents(value: number): number {
  return Math.floor(value * 100) / 100;
}

function roundUpToCents(value: number): number {
  return Math.ceil(value * 100) / 100;
}

function convertSettlementCnyToCreditedUsd(
  amountCny: number,
  balanceCreditCnyPerUsd: number,
  mode: 'floor' | 'ceil',
): number {
  const converted = convertCnySettlementToUsdBalance(amountCny, balanceCreditCnyPerUsd) ?? 0;
  return mode === 'ceil' ? roundUpToCents(converted) : roundDownToCents(converted);
}

/**
 * 聚合实例级限额：对每个支付类型，取所有实例中最宽松的单笔范围 + 检查日限额可用性。
 * 当剩余日额度 < 该实例的 singleMin 时，视为该实例不可用。
 */
async function aggregateInstanceLimits(paymentTypes: string[]): Promise<
  Record<
    string,
    {
      singleMin: number;
      singleMax: number;
      allInstancesDailyBlocked: boolean;
      maxRemainingCapacity: number | null;
      hasInstances: boolean;
    }
  >
> {
  const result: Record<
    string,
    {
      singleMin: number;
      singleMax: number;
      allInstancesDailyBlocked: boolean;
      maxRemainingCapacity: number | null;
      hasInstances: boolean;
    }
  > = {};

  const allInstances = await prisma.paymentProviderInstance.findMany({
    where: { enabled: true },
    select: { id: true, limits: true, supportedTypes: true },
  });

  if (allInstances.length === 0) {
    for (const type of paymentTypes) {
      result[type] = {
        singleMin: 0,
        singleMax: 0,
        allInstancesDailyBlocked: false,
        maxRemainingCapacity: null,
        hasInstances: false,
      };
    }
    return result;
  }

  const todayStart = getBizDayStartUTC();

  const usageRows = await prisma.order.groupBy({
    by: ['providerInstanceId'],
    where: {
      providerInstanceId: { in: allInstances.map((i) => i.id) },
      status: { in: [ORDER_STATUS.PAID, ORDER_STATUS.RECHARGING, ORDER_STATUS.COMPLETED] },
      paidAt: { gte: todayStart },
    },
    _sum: { payAmount: true },
  });
  const usageMap = new Map(usageRows.map((r) => [r.providerInstanceId, Number(r._sum.payAmount ?? 0)]));

  for (const type of paymentTypes) {
    // 留空 = 支持全部；必须显式启用的方式（PayNow）只算明确勾选的实例，与下单时的实例选择一致
    const supporting = allInstances.filter((inst) => instanceSupportsPaymentType(inst.supportedTypes, type));

    if (supporting.length === 0) {
      result[type] = {
        singleMin: 0,
        singleMax: 0,
        allInstancesDailyBlocked: false,
        maxRemainingCapacity: null,
        hasInstances: false,
      };
      continue;
    }

    let aggSingleMin = Infinity;
    let aggSingleMax = 0;
    let allBlocked = true;
    let maxRemaining: number | null = null; // 所有可用实例中最大的剩余日额度

    for (const inst of supporting) {
      let channelLimits: InstanceChannelLimits | undefined;
      if (inst.limits) {
        try {
          const parsed = JSON.parse(inst.limits) as Record<string, InstanceChannelLimits>;
          channelLimits = parsed[type];
        } catch {
          /* ignore */
        }
      }

      // 单笔范围：取所有实例中最宽松的范围
      const instMin = channelLimits?.singleMin ?? 0;
      const instMax = channelLimits?.singleMax ?? 0;
      if (instMin > 0 && instMin < aggSingleMin) aggSingleMin = instMin;
      if (instMin === 0) aggSingleMin = 0;
      if (instMax > aggSingleMax) aggSingleMax = instMax;
      if (instMax === 0) aggSingleMax = 0;

      // 日限额：计算剩余容量，判断是否可用
      const instDailyLimit = channelLimits?.dailyLimit;
      if (!instDailyLimit || instDailyLimit <= 0) {
        // 无日限额限制
        allBlocked = false;
        maxRemaining = null; // null 表示至少有一个实例无限额
      } else {
        const used = usageMap.get(inst.id) ?? 0;
        const remaining = Math.max(0, instDailyLimit - used);
        const effectiveMin = instMin > 0 ? instMin : 0;

        if (remaining > effectiveMin) {
          // 剩余额度足够下一单（大于最小单笔）
          allBlocked = false;
          if (maxRemaining !== null) {
            maxRemaining = Math.max(maxRemaining, remaining);
          }
          // maxRemaining === null 时说明已有无限额实例，保持 null
        }
        // remaining <= effectiveMin: 该实例实质不可用，不影响 allBlocked
      }
    }

    if (aggSingleMin === Infinity) aggSingleMin = 0;

    result[type] = {
      singleMin: aggSingleMin,
      singleMax: aggSingleMax,
      allInstancesDailyBlocked: allBlocked,
      maxRemainingCapacity: maxRemaining,
      hasInstances: true,
    };
  }

  return result;
}

/**
 * 批量查询多个支付渠道的今日使用情况。
 * 聚合全局限额 + 实例级限额，一次性返回前端所需的可用性信息。
 */
export async function queryMethodLimits(paymentTypes: string[]): Promise<Record<string, MethodLimitStatus>> {
  const todayStart = getBizDayStartUTC();
  const env = getEnv();
  const [balanceCreditCnyPerUsdConfig, legacyBalanceCreditUsdPerCnyConfig] = await Promise.all([
    getSystemConfig('BALANCE_CREDIT_CNY_PER_USD'),
    getSystemConfig('BALANCE_CREDIT_USD_PER_CNY'),
  ]);
  const balanceCreditCnyPerUsd = resolveBalanceCreditCnyPerUsd(
    balanceCreditCnyPerUsdConfig ?? env.BALANCE_CREDIT_CNY_PER_USD,
    legacyBalanceCreditUsdPerCnyConfig ?? env.BALANCE_CREDIT_USD_PER_CNY,
  );

  const [usageRows, instanceAgg] = await Promise.all([
    prisma.order.groupBy({
      by: ['paymentType'],
      where: {
        paymentType: { in: paymentTypes },
        status: { in: [ORDER_STATUS.PAID, ORDER_STATUS.RECHARGING, ORDER_STATUS.COMPLETED] },
        paidAt: { gte: todayStart },
      },
      _sum: { payAmount: true },
    }),
    aggregateInstanceLimits(paymentTypes),
  ]);

  const usageMap = Object.fromEntries(usageRows.map((row) => [row.paymentType, Number(row._sum.payAmount ?? 0)]));
  const feeConfigs = await getMethodFeeConfigs(paymentTypes);

  const result: Record<string, MethodLimitStatus> = {};
  for (const type of paymentTypes) {
    const globalDailyLimit = await getMethodDailyLimit(type);
    const globalSingleMax = await getMethodSingleLimit(type);
    const feeConfig = feeConfigs[type];
    const usedPayAmount = usageMap[type] ?? 0;
    const remainingPayAmount = globalDailyLimit > 0 ? Math.max(0, globalDailyLimit - usedPayAmount) : null;

    const inst = instanceAgg[type];
    // 全局可用：全局日限额未超
    const globalAvailable = globalDailyLimit === 0 || usedPayAmount < globalDailyLimit;
    // 实例可用：无实例(走环境变量provider) 或 不是所有实例都被日限额阻塞
    const instanceAvailable = !inst?.hasInstances || !inst.allInstancesDailyBlocked;

    // 聚合单笔范围：实例级限额与全局取交集（这里仍使用网关实付金额，单位 CNY）
    const singleMinPayAmount = inst?.singleMin ?? 0;
    let singleMaxPayAmount = globalSingleMax;
    if (inst?.hasInstances && inst.singleMax > 0) {
      singleMaxPayAmount = singleMaxPayAmount > 0 ? Math.min(singleMaxPayAmount, inst.singleMax) : inst.singleMax;
    }

    // 实例剩余日容量约束：singleMax 不能超过最大剩余容量
    if (inst?.hasInstances && inst.maxRemainingCapacity !== null && inst.maxRemainingCapacity >= 0) {
      singleMaxPayAmount =
        singleMaxPayAmount > 0 ? Math.min(singleMaxPayAmount, inst.maxRemainingCapacity) : inst.maxRemainingCapacity;
    }

    // 全局剩余日容量约束
    if (remainingPayAmount !== null && remainingPayAmount >= 0) {
      singleMaxPayAmount =
        singleMaxPayAmount > 0 ? Math.min(singleMaxPayAmount, remainingPayAmount) : remainingPayAmount;
    }

    const usedSettlementAmount = getMaxSettlementAmountForPayLimit(usedPayAmount, feeConfig);
    const remainingSettlementAmount =
      remainingPayAmount !== null ? getMaxSettlementAmountForPayLimit(remainingPayAmount, feeConfig) : null;
    const singleMinSettlementAmount =
      singleMinPayAmount > 0 ? getMinSettlementAmountForPayLimit(singleMinPayAmount, feeConfig) : 0;
    const singleMaxSettlementAmount =
      singleMaxPayAmount > 0 ? getMaxSettlementAmountForPayLimit(singleMaxPayAmount, feeConfig) : 0;

    const used = convertSettlementCnyToCreditedUsd(usedSettlementAmount, balanceCreditCnyPerUsd, 'floor');
    const remaining =
      remainingSettlementAmount !== null
        ? convertSettlementCnyToCreditedUsd(remainingSettlementAmount, balanceCreditCnyPerUsd, 'floor')
        : null;
    const dailyLimit =
      globalDailyLimit > 0
        ? convertSettlementCnyToCreditedUsd(
            getMaxSettlementAmountForPayLimit(globalDailyLimit, feeConfig),
            balanceCreditCnyPerUsd,
            'floor',
          )
        : 0;
    const singleMin =
      singleMinSettlementAmount > 0
        ? convertSettlementCnyToCreditedUsd(singleMinSettlementAmount, balanceCreditCnyPerUsd, 'ceil')
        : 0;
    const singleMax =
      singleMaxSettlementAmount > 0
        ? convertSettlementCnyToCreditedUsd(singleMaxSettlementAmount, balanceCreditCnyPerUsd, 'floor')
        : 0;

    // 最终可用性：如果换算后的到账区间为空，该渠道实质不可用
    const effectivelyAvailable =
      globalAvailable &&
      instanceAvailable &&
      !(singleMaxPayAmount > 0 && singleMaxSettlementAmount < 0.01) &&
      (singleMin === 0 || singleMax >= singleMin);

    result[type] = {
      dailyLimit,
      used,
      remaining,
      available: effectivelyAvailable,
      singleMin,
      singleMax,
      feeRate: feeConfig.feeRate,
      fixedFee: feeConfig.fixedFee,
      feeEnabled: feeConfig.enabled,
    };
  }
  return result;
}
