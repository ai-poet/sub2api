'use client';

import React from 'react';
import SubscriptionPlanCard, { type PlanInfo } from '@/components/SubscriptionPlanCard';
import PlanModelUsage from '@/components/PlanModelUsage';
import { pickLocaleText, type Locale } from '@/lib/locale';

interface Props {
  plans: PlanInfo[];
  onSubscribe: (planId: string) => void;
  isDark: boolean;
  locale: Locale;
}

export default function SubscriptionPlanCatalog({ plans, onSubscribe, isDark, locale }: Props) {
  const text = (zh: string, en: string) => pickLocaleText(locale, zh, en);
  const muted = isDark ? 'text-slate-400' : 'text-slate-500';
  const border = isDark ? 'border-slate-700' : 'border-slate-200';
  const rules = [
    {
      title: text('这些额度能用多少次？', 'How many requests can I make?'),
      body: text(
        '上方按套餐内每个模型的实际定价，结合输入、输出 tokens 假设，分别估算可用次数。每行都假设将共享额度全部用于该模型，各模型次数不能相加。缓存命中、专属倍率、高峰规则和一次任务中的多次模型调用，都会影响实际用量。',
        'The table uses each model’s actual plan price and your input/output token assumptions. Each row assumes the shared quota is spent entirely on that model; the counts cannot be added together. Cache hits, personal rates, peak pricing and multiple calls within one task affect actual usage.',
      ),
    },
    {
      title: text('日、周、月额度可以相加吗？', 'Do daily, weekly and monthly allowances add up?'),
      body: text(
        '不能相加。同一次请求会同时计入日、周、月用量，任一已设置的限额用尽后，需要等待该窗口重置。图表每次只比较一个窗口；「该窗口不限额」不代表其他窗口不限额，也不代表套餐永久有效。',
        'No. Each request counts toward daily, weekly and monthly usage at the same time. Reaching any configured limit requires waiting for that window to reset. The chart compares one window at a time. An uncapped window does not remove other limits or the plan expiry.',
      ),
    },
    {
      title: text('额度何时恢复？有效期和月额度有什么区别？', 'When does quota reset, and how does validity differ?'),
      body: text(
        '有效期决定订阅可以使用到哪一天；日、周、月额度限制各自窗口内的消耗。周窗口为 7 天，月窗口为 30 天，并非自然周或自然月；具体重置时间以订阅详情为准。不要将日额度 × 有效天数当作保证可用的总额度，周、月限额也会约束用量。',
        'Validity determines when the subscription expires; each quota limits spending within its own window. Weekly windows are 7 days and monthly windows are 30 days, rather than calendar weeks or months. See subscription details for reset times. Daily quota × validity is not a guaranteed total, because weekly and monthly limits also apply.',
      ),
    },
    {
      title: text('购买价格、美元额度和倍率分别是什么？', 'How do the purchase price, USD quota and rate relate?'),
      body: text(
        '卡片上的 ¥ 是购买套餐的人民币价格；$ 是用于记录 API 消耗的美元计费额度，不是可提现余额。分组倍率 ×0.5 表示倍率前 $1 的请求通常扣减 $0.50 额度；×2 则扣减 $2。估算按当前分组倍率计算，实际以请求的计费结果为准。',
        '¥ is the purchase price in CNY. $ is the USD billing quota for API usage, not a withdrawable balance. At ×0.5, a request costing $1 before multiplier normally consumes $0.50 of quota; at ×2 it consumes $2. Estimates use the current group rate. Actual billing is authoritative.',
      ),
    },
  ];

  if (plans.length === 0) return null;

  return (
    <div className="space-y-10">
      <header className="max-w-2xl">
        <p className={`mb-2 text-xs font-semibold uppercase tracking-[0.18em] ${muted}`}>
          {text('订阅套餐 · 用量一目了然', 'SUBSCRIPTIONS · KNOW YOUR LIMITS')}
        </p>
        <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
          {text('为你的使用节奏，选一份套餐', 'Find a plan for the way you work')}
        </h2>
        <p className={`mt-3 text-sm leading-6 ${muted}`}>
          {text(
            '比较套餐价格，再逐个模型看看能用多少次。购买价格为人民币，API 额度按美元计费金额记录。',
            'Compare plan prices, then explore how much you can use each model. Purchase prices are in CNY; API allowances are measured in USD billed cost.',
          )}
        </p>
      </header>

      <div className={`grid grid-cols-1 gap-4 ${plans.length === 2 ? 'md:grid-cols-2' : 'md:grid-cols-2 xl:grid-cols-3'}`}>
        {plans.map((plan) => (
          <SubscriptionPlanCard key={plan.id} plan={plan} onSubscribe={onSubscribe} isDark={isDark} locale={locale} />
        ))}
      </div>

      <PlanModelUsage plans={plans} isDark={isDark} locale={locale} />

      <section>
        <h3 className="mb-4 text-lg font-semibold">{text('买之前，把用量弄明白', 'Understand your quota before subscribing')}</h3>
        <div className={`divide-y rounded-2xl border px-5 sm:px-6 ${border} ${isDark ? 'divide-slate-700' : 'divide-slate-200'}`}>
          {rules.map((rule, index) => (
            <details key={rule.title} className="group py-4" open={index === 0}>
              <summary className="cursor-pointer text-sm font-medium focus-visible:outline-2 focus-visible:outline-emerald-500">{rule.title}</summary>
              <p className={`mt-3 text-sm leading-7 ${muted}`}>{rule.body}</p>
            </details>
          ))}
        </div>
      </section>
    </div>
  );
}
