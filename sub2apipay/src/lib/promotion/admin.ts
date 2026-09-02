/**
 * 管理端充值活动接口共用的校验与序列化。
 */
import { z } from 'zod';
import { Prisma } from '@prisma/client';
import type { Locale } from '@/lib/locale';
import { MAX_CREDIT_AMOUNT } from './calc';
import type { PromotionUsageStats } from './service';

const MAX_PERCENT = 1000;

const isoDateTime = z.iso.datetime({ offset: true }).nullable().optional();

export const promotionBaseSchema = z.object({
  name: z.string().trim().min(1).max(100),
  description: z.string().trim().max(2000).nullable().optional(),
  min_amount: z.number().positive().max(MAX_CREDIT_AMOUNT),
  bonus_type: z.enum(['fixed', 'percent']),
  bonus_value: z.number().positive().max(MAX_CREDIT_AMOUNT),
  max_bonus: z.number().positive().max(MAX_CREDIT_AMOUNT).nullable().optional(),
  starts_at: isoDateTime,
  ends_at: isoDateTime,
  per_user_limit: z.number().int().min(0).max(1_000_000).optional(),
  total_limit: z.number().int().min(0).max(100_000_000).optional(),
  sort_order: z.number().int().min(0).max(1_000_000).optional(),
  enabled: z.boolean().optional(),
});

export type PromotionInput = z.infer<typeof promotionBaseSchema>;

function refinePromotion(data: PromotionInput, ctx: z.RefinementCtx): void {
  if (data.starts_at && data.ends_at && new Date(data.ends_at).getTime() <= new Date(data.starts_at).getTime()) {
    ctx.addIssue({ code: 'custom', path: ['ends_at'], message: 'ends_at must be after starts_at' });
  }
  if (data.bonus_type === 'percent' && data.bonus_value > MAX_PERCENT) {
    ctx.addIssue({ code: 'custom', path: ['bonus_value'], message: `percent bonus must be <= ${MAX_PERCENT}` });
  }
}

/** 创建：完整对象 */
export const promotionCreateSchema = promotionBaseSchema.superRefine(refinePromotion);
/** 更新：先按字段形状校验补丁，再与现有记录合并后用 promotionCreateSchema 整体校验 */
export const promotionPatchSchema = promotionBaseSchema.partial();

export function validationErrorBody(locale: Locale, error: z.ZodError) {
  return {
    error: locale === 'en' ? 'Invalid parameters' : '参数错误',
    details: z.flattenError(error).fieldErrors,
  };
}

/** 把校验后的输入转换成 Prisma 写入数据 */
export function toPromotionData(input: PromotionInput) {
  const isPercent = input.bonus_type === 'percent';
  return {
    name: input.name,
    description: input.description?.trim() ? input.description.trim() : null,
    minAmount: new Prisma.Decimal(input.min_amount.toFixed(2)),
    bonusType: input.bonus_type,
    bonusValue: new Prisma.Decimal(input.bonus_value.toFixed(2)),
    maxBonus: isPercent && input.max_bonus != null ? new Prisma.Decimal(input.max_bonus.toFixed(2)) : null,
    startsAt: input.starts_at ? new Date(input.starts_at) : null,
    endsAt: input.ends_at ? new Date(input.ends_at) : null,
    perUserLimit: input.per_user_limit ?? 0,
    totalLimit: input.total_limit ?? 0,
    sortOrder: input.sort_order ?? 0,
    enabled: input.enabled ?? true,
  };
}

export interface PromotionRecord {
  id: string;
  name: string;
  description: string | null;
  minAmount: Prisma.Decimal | number;
  bonusType: string;
  bonusValue: Prisma.Decimal | number;
  maxBonus: Prisma.Decimal | number | null;
  startsAt: Date | null;
  endsAt: Date | null;
  perUserLimit: number;
  totalLimit: number;
  enabled: boolean;
  sortOrder: number;
  createdAt: Date;
  updatedAt: Date;
}

/** 现有记录转回 snake_case 输入，供 PUT 与补丁合并后整体校验 */
export function recordToInput(record: PromotionRecord): PromotionInput {
  return {
    name: record.name,
    description: record.description,
    min_amount: Number(record.minAmount),
    bonus_type: record.bonusType === 'percent' ? 'percent' : 'fixed',
    bonus_value: Number(record.bonusValue),
    max_bonus: record.maxBonus != null ? Number(record.maxBonus) : null,
    starts_at: record.startsAt ? record.startsAt.toISOString() : null,
    ends_at: record.endsAt ? record.endsAt.toISOString() : null,
    per_user_limit: record.perUserLimit,
    total_limit: record.totalLimit,
    sort_order: record.sortOrder,
    enabled: record.enabled,
  };
}

export type PromotionAdminStatus = 'active' | 'scheduled' | 'ended' | 'disabled' | 'exhausted';

export function computePromotionStatus(
  record: Pick<PromotionRecord, 'enabled' | 'startsAt' | 'endsAt' | 'totalLimit'>,
  usedCount: number,
  now: Date = new Date(),
): PromotionAdminStatus {
  if (!record.enabled) return 'disabled';
  if (record.startsAt && now.getTime() < record.startsAt.getTime()) return 'scheduled';
  if (record.endsAt && now.getTime() >= record.endsAt.getTime()) return 'ended';
  if (record.totalLimit > 0 && usedCount >= record.totalLimit) return 'exhausted';
  return 'active';
}

export function serializePromotion(record: PromotionRecord, stats?: PromotionUsageStats, now: Date = new Date()) {
  const usedCount = stats?.usedCount ?? 0;
  return {
    id: record.id,
    name: record.name,
    description: record.description,
    minAmount: Number(record.minAmount),
    bonusType: record.bonusType === 'percent' ? 'percent' : 'fixed',
    bonusValue: Number(record.bonusValue),
    maxBonus: record.maxBonus != null ? Number(record.maxBonus) : null,
    startsAt: record.startsAt ? record.startsAt.toISOString() : null,
    endsAt: record.endsAt ? record.endsAt.toISOString() : null,
    perUserLimit: record.perUserLimit,
    totalLimit: record.totalLimit,
    enabled: record.enabled,
    sortOrder: record.sortOrder,
    usedCount,
    bonusTotal: stats?.bonusTotal ?? 0,
    status: computePromotionStatus(record, usedCount, now),
    createdAt: record.createdAt,
    updatedAt: record.updatedAt,
  };
}
