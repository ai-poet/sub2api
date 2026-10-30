'use client';

import React from 'react';
import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';
import { formatValidityLabel, formatValiditySuffix, type ValidityUnit } from '@/lib/subscription-utils';
import { PlatformBadge, getPlatformStyle } from '@/lib/platform-style';
import { planWindowLimit, PLAN_WINDOWS, type PlanWindow } from '@/lib/plan-usage';
import { formatUsageUsd } from '@/lib/subscription-utils';
import type { PlanModel } from '@/lib/plan-model-usage';

export interface PlanInfo {
  id: string;
  groupId: number;
  groupName: string | null;
  name: string;
  price: number;
  originalPrice: number | null;
  validityDays: number;
  validityUnit?: ValidityUnit;
  features: string[];
  description: string | null;
  platform: string | null;
  rateMultiplier: number | null;
  limits: {
    daily_limit_usd: number | null;
    weekly_limit_usd: number | null;
    monthly_limit_usd: number | null;
  } | null;
  allowMessagesDispatch: boolean;
  defaultMappedModel: string | null;
  models?: PlanModel[] | null;
}

/** 套餐信息展示（Header + 价格 + 描述 + 倍率/限额 + 特性），不含操作按钮 */
export function PlanInfoDisplay({ plan, isDark, locale }: { plan: PlanInfo; isDark: boolean; locale: Locale }) {
  const unit = plan.validityUnit ?? 'day';
  const periodLabel = formatValidityLabel(plan.validityDays, unit, locale);
  const periodSuffix = formatValiditySuffix(plan.validityDays, unit, locale);

  const windowLabels: Record<PlanWindow, string> = {
    daily: pickLocaleText(locale, '日额度', 'Daily quota'),
    weekly: pickLocaleText(locale, '周额度', 'Weekly quota'),
    monthly: pickLocaleText(locale, '月额度', 'Monthly quota'),
  };

  const isOpenAI = plan.platform?.toLowerCase() === 'openai';
  const ps = getPlatformStyle(plan.platform ?? '');
  const accentCls = isDark ? ps.accent.dark : ps.accent.light;

  return (
    <>
      {/* Header: Platform badge + Name + Period + /v1/messages */}
      <div className="mb-4">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          {plan.platform && <PlatformBadge platform={plan.platform} />}
          <span
            className={[
              'rounded-full px-2.5 py-0.5 text-xs font-medium',
              isDark ? 'bg-emerald-900/40 text-emerald-300' : 'bg-emerald-50 text-emerald-700',
            ].join(' ')}
          >
            {periodLabel}
          </span>
          {isOpenAI && plan.allowMessagesDispatch && (
            <span
              className={[
                'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
                isDark ? 'bg-green-500/20 text-green-300' : 'bg-green-100 text-green-700',
              ].join(' ')}
            >
              /v1/messages
              {plan.defaultMappedModel && (
                <span className={['font-mono', isDark ? 'text-green-400' : 'text-green-800'].join(' ')}>
                  {plan.defaultMappedModel}
                </span>
              )}
            </span>
          )}
        </div>
        <h3 className={['mb-3 text-xl font-semibold tracking-tight', isDark ? 'text-slate-100' : 'text-slate-900'].join(' ')}>{plan.name}</h3>

        {/* Price */}
        <div className="flex items-baseline gap-2">
          {plan.originalPrice !== null && (
            <span className={['text-sm line-through', isDark ? 'text-slate-500' : 'text-slate-400'].join(' ')}>
              ¥{plan.originalPrice}
            </span>
          )}
          <span className={['text-4xl font-semibold tracking-tight', isDark ? 'text-slate-100' : 'text-slate-900'].join(' ')}>¥{plan.price}</span>
          <span className={['text-sm', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>{periodSuffix}</span>
        </div>
      </div>

      {/* Description */}
      {plan.description && (
        <p className={['mb-4 text-sm leading-relaxed', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
          {plan.description}
        </p>
      )}

      <dl className={`mb-5 space-y-3 border-y py-4 text-sm ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
        {PLAN_WINDOWS.map((window) => {
          const limit = planWindowLimit(plan.limits, window);
          return (
            <div key={window} className="flex items-baseline justify-between gap-3">
              <dt className={isDark ? 'text-slate-400' : 'text-slate-500'}>{windowLabels[window]}</dt>
              <dd className="text-right font-medium tabular-nums">
                {limit === undefined ? pickLocaleText(locale, '待确认', 'Unavailable')
                  : limit === null ? pickLocaleText(locale, '该窗口不限额', 'Uncapped window') : `$${formatUsageUsd(limit)}`}
              </dd>
            </div>
          );
        })}
        {plan.rateMultiplier != null && (
          <div className="flex items-baseline justify-between gap-3">
            <dt className={isDark ? 'text-slate-400' : 'text-slate-500'}>{pickLocaleText(locale, '分组倍率', 'Group rate')}</dt>
            <dd className={`font-semibold ${accentCls}`}>×{plan.rateMultiplier}</dd>
          </div>
        )}
      </dl>

      {/* Features */}
      {plan.features.length > 0 && (
        <div className="mb-5">
          <p className={['mb-2 text-xs', isDark ? 'text-slate-500' : 'text-slate-400'].join(' ')}>
            {pickLocaleText(locale, '功能特性', 'Features')}
          </p>
          <ul className="space-y-2">
            {plan.features.map((feature) => (
              <li
                key={feature}
                className={`flex items-start gap-2 text-sm leading-5 ${isDark ? 'text-slate-300' : 'text-slate-600'}`}
              >
                <span aria-hidden="true" className="text-emerald-500">✓</span>
                <span>{feature}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}

interface SubscriptionPlanCardProps {
  plan: PlanInfo;
  onSubscribe: (planId: string) => void;
  isDark: boolean;
  locale: Locale;
}

export default function SubscriptionPlanCard({ plan, onSubscribe, isDark, locale }: SubscriptionPlanCardProps) {
  const ps = getPlatformStyle(plan.platform ?? '');

  return (
    <div
      className={[
        'flex min-w-0 flex-col rounded-2xl border p-6 transition-colors',
        isDark ? 'border-slate-700 bg-slate-800/40 hover:border-slate-500' : 'border-slate-200 bg-white hover:border-slate-400',
      ].join(' ')}
    >
      <PlanInfoDisplay plan={plan} isDark={isDark} locale={locale} />

      {/* Spacer */}
      <div className="flex-1" />

      {/* Subscribe button */}
      <button
        type="button"
        onClick={() => onSubscribe(plan.id)}
        className={[
          'mt-2 inline-flex w-full items-center justify-center gap-2 rounded-xl py-3 text-sm font-semibold text-white transition-colors',
          isDark ? ps.button.dark : ps.button.light,
        ].join(' ')}
      >
        <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
        </svg>
        {pickLocaleText(locale, '立即开通', 'Subscribe Now')}
      </button>
    </div>
  );
}
