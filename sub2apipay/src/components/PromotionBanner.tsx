'use client';

import React from 'react';
import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';
import { describePromotion, type PublicPromotion } from '@/lib/promotion/calc';

interface PromotionBannerProps {
  promotions: PublicPromotion[];
  isDark: boolean;
  locale: Locale;
  className?: string;
}

function formatEndsAt(iso: string, locale: Locale): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString(locale === 'en' ? 'en-US' : 'zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/** 充值页的活动列表：进行中的活动一行一条，用户已达上限的置灰展示 */
export default function PromotionBanner({ promotions, isDark, locale, className }: PromotionBannerProps) {
  if (!promotions || promotions.length === 0) return null;

  return (
    <div
      className={[
        'rounded-2xl border p-4',
        isDark
          ? 'border-amber-500/30 bg-gradient-to-r from-amber-500/10 to-rose-500/10'
          : 'border-amber-200 bg-gradient-to-r from-amber-50 to-rose-50',
        className ?? '',
      ].join(' ')}
    >
      <div className="flex items-center gap-2">
        <span
          className={[
            'flex h-7 w-7 items-center justify-center rounded-lg',
            isDark ? 'bg-amber-500/20 text-amber-300' : 'bg-amber-100 text-amber-600',
          ].join(' ')}
        >
          <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
            <polyline points="20 12 20 22 4 22 4 12" />
            <rect x="2" y="7" width="20" height="5" />
            <line x1="12" y1="22" x2="12" y2="7" />
            <path d="M12 7H7.5a2.5 2.5 0 0 1 0-5C11 2 12 7 12 7z" />
            <path d="M12 7h4.5a2.5 2.5 0 0 0 0-5C13 2 12 7 12 7z" />
          </svg>
        </span>
        <span className={['text-sm font-semibold', isDark ? 'text-amber-300' : 'text-amber-700'].join(' ')}>
          {pickLocaleText(locale, '充值活动', 'Top-up Promotions')}
        </span>
        <span className={['text-xs', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
          {pickLocaleText(locale, '赠送金额直接计入余额', 'Bonus is credited to your balance')}
        </span>
      </div>

      <ul className="mt-3 space-y-2">
        {promotions.map((promo) => (
          <li
            key={promo.id}
            className={[
              'flex flex-wrap items-center gap-x-3 gap-y-1 text-sm',
              promo.available ? '' : 'opacity-60',
            ].join(' ')}
          >
            <span
              className={[
                'rounded-md px-2 py-0.5 text-xs font-semibold',
                promo.available
                  ? isDark
                    ? 'bg-emerald-500/20 text-emerald-300'
                    : 'bg-emerald-100 text-emerald-700'
                  : isDark
                    ? 'bg-slate-700 text-slate-300'
                    : 'bg-slate-200 text-slate-600',
              ].join(' ')}
            >
              {describePromotion(promo, locale)}
            </span>
            <span className={['font-medium', isDark ? 'text-slate-200' : 'text-slate-800'].join(' ')}>
              {promo.name}
            </span>
            {promo.description && (
              <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>{promo.description}</span>
            )}
            {promo.endsAt && (
              <span className={['text-xs', isDark ? 'text-slate-500' : 'text-slate-400'].join(' ')}>
                {pickLocaleText(
                  locale,
                  `截止 ${formatEndsAt(promo.endsAt, locale)}`,
                  `Ends ${formatEndsAt(promo.endsAt, locale)}`,
                )}
              </span>
            )}
            {!promo.available && (
              <span className={['text-xs font-medium', isDark ? 'text-amber-400' : 'text-amber-600'].join(' ')}>
                {pickLocaleText(locale, '已达参与上限', 'Limit reached')}
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
