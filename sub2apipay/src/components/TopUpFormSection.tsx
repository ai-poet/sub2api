'use client';

import React from 'react';
import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';

interface TopUpFormSectionProps {
  isDark: boolean;
  locale: Locale;
  /** config.maxDailyAmount，仅在 > 0 时展示上限说明 */
  maxDailyAmount: number;
  /** 侧栏里跟在「支付说明」后面的内容（帮助/客服区块） */
  aside?: React.ReactNode;
  /** 充值表单本体 */
  children: React.ReactNode;
}

/**
 * 余额充值表单的双栏外壳：左侧表单，右侧「支付说明」+ 可选帮助区块。
 * 无论站点是否配置了套餐，充值表单都走这里，避免两套布局各自漂移。
 */
export default function TopUpFormSection({ isDark, locale, maxDailyAmount, aside, children }: TopUpFormSectionProps) {
  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1.45fr)_minmax(300px,0.8fr)]">
      <div className="min-w-0">{children}</div>
      <div className="space-y-4">
        <div
          className={[
            'rounded-2xl border p-4',
            isDark ? 'border-slate-700 bg-slate-800/70' : 'border-slate-200 bg-slate-50',
          ].join(' ')}
        >
          <div className={['text-xs', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
            {pickLocaleText(locale, '支付说明', 'Payment Notes')}
          </div>
          <ul className={['mt-2 space-y-1 text-sm', isDark ? 'text-slate-300' : 'text-slate-600'].join(' ')}>
            <li>{pickLocaleText(locale, '订单完成后会自动到账', 'Balance will be credited automatically')}</li>
            <li>
              {pickLocaleText(
                locale,
                '如需历史记录和开票请查看「我的订单」',
                'Check "My Orders" for history and invoices',
              )}
            </li>
            {maxDailyAmount > 0 && (
              <li>
                {pickLocaleText(locale, '每日累计到账上限', 'Max daily credited balance')} ${maxDailyAmount.toFixed(2)}
              </li>
            )}
          </ul>
        </div>
        {aside}
      </div>
    </div>
  );
}
