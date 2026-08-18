import { z } from 'zod';

/**
 * 开票抬头字段的共享 schema：单订单、合并、一键三个路由必须收同一套字段规则，
 * 各写一份迟早漂移（收票邮箱的空串宽容就曾只修了一处）。
 */
export const invoiceTitleFieldsSchema = z.object({
  title_name: z.string().trim().min(2).max(100),
  // 统一社会信用代码 18 位；兼容旧的 15 位纳税人识别号及部分 20 位号段。
  tax_no: z
    .string()
    .trim()
    .regex(/^[0-9A-Za-z]{15,20}$/),
  remark: z.string().trim().max(200).optional(),
  // 选填字段对空串宽容：调用方漏掉「空则省略」的逻辑不该变成 400。
  contact_email: z
    .union([z.literal(''), z.email().max(200)])
    .optional()
    .transform((value) => (value === '' ? undefined : value)),
});
