import type { PublicSettings } from '@/types'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

/**
 * 用户端购买入口的文案模式，只由订阅功能开关派生。
 * 余额充值方式由支付服务独立配置，主站不再读取或写入 payment_balance_disabled。
 */
export type SiteBillingMode = 'recharge_and_subscription' | 'recharge_only'

export interface BillingModeSettings {
  subscription_enabled?: boolean
}

export function resolveSiteBillingMode(settings: BillingModeSettings | null | undefined): SiteBillingMode {
  const subscriptionEnabled = resolveFeatureFlag(
    { subscription_enabled: settings?.subscription_enabled } as Partial<PublicSettings>,
    FeatureFlags.subscription,
  )
  return subscriptionEnabled ? 'recharge_and_subscription' : 'recharge_only'
}
