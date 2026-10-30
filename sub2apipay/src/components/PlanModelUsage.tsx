'use client';

import React, { useId, useState } from 'react';
import type { PlanInfo } from './SubscriptionPlanCard';
import { pickLocaleText, type Locale } from '@/lib/locale';
import { modelRequestCost, modelWindowEstimate } from '@/lib/plan-model-usage';
import { PLAN_WINDOWS, type PlanWindow, type RequestEstimate } from '@/lib/plan-usage';

const COLORS = ['bg-amber-400', 'bg-orange-400', 'bg-sky-400', 'bg-violet-400', 'bg-emerald-400'];

export default function PlanModelUsage({ plans, isDark, locale }: {
  plans: PlanInfo[]; isDark: boolean; locale: Locale;
}) {
  const [selectedIds, setSelectedIds] = useState<string[]>(() => plans.slice(0, 2).map((plan) => plan.id));
  const [window, setWindow] = useState<PlanWindow>('monthly');
  const [input, setInput] = useState('10000');
  const [output, setOutput] = useState('2000');
  const id = useId();
  const text = (zh: string, en: string) => pickLocaleText(locale, zh, en);
  const muted = isDark ? 'text-slate-400' : 'text-slate-500';
  const border = isDark ? 'border-slate-700' : 'border-slate-200';
  const selected = plans.filter((plan) => selectedIds.includes(plan.id));
  const visible = selected.length ? selected : plans.slice(0, 1);
  const assumptions = {
    inputTokens: input.trim() === '' ? NaN : Number(input),
    outputTokens: output.trim() === '' ? NaN : Number(output),
  };
  const labels: Record<PlanWindow, string> = {
    daily: text('日', 'day'), weekly: text('周', 'week'), monthly: text('月', 'month'),
  };
  const number = new Intl.NumberFormat(locale === 'zh' ? 'zh-CN' : 'en-US');
  const estimateLabel = (estimate: RequestEstimate) => estimate.kind === 'count'
    ? `≈ ${number.format(estimate.count)}`
    : estimate.kind === 'free' ? text('不扣额度', 'No quota cost')
      : estimate.kind === 'unlimited' ? text('该窗口不限额', 'Uncapped window')
        : text('暂无法估算', 'Unavailable');
  const models = Array.from(new Set(visible.flatMap((plan) => plan.models?.map((model) => model.model) ?? [])));
  const rows = models.map((modelId) => ({
    modelId,
    offers: visible.flatMap((plan) => {
      const model = plan.models?.find((entry) => entry.model === modelId);
      if (!model) return [];
      return [{
        plan, model,
        cost: modelRequestCost(model, assumptions),
        estimate: modelWindowEstimate(model, plan.limits, window, assumptions),
        color: COLORS[plans.findIndex((entry) => entry.id === plan.id) % COLORS.length],
      }];
    }),
  }));
  const unitKey = (mode: string) => mode === 'video' ? 'seconds' : mode === 'image' ? 'images' : 'requests';
  const maxCounts: Record<string, number> = {};
  for (const row of rows) {
    for (const offer of row.offers) {
      const key = unitKey(offer.model.billing_mode);
      maxCounts[key] = Math.max(maxCounts[key] ?? 1, offer.estimate.kind === 'count' ? offer.estimate.count : 0);
    }
  }

  return (
    <section aria-labelledby={`${id}-title`} className={`overflow-hidden rounded-2xl border ${border}`}>
      <div className={`space-y-5 border-b p-5 sm:p-6 ${border} ${isDark ? 'bg-slate-800/40' : 'bg-slate-50/80'}`}>
        <div>
          <p className={`mb-1 text-xs uppercase tracking-widest ${muted}`}>Limits</p>
          <h3 id={`${id}-title`} className="text-xl font-semibold">{text('每个模型，能用多少？', 'How much can you use each model?')}</h3>
          <p className={`mt-2 text-sm leading-6 ${muted}`}>
            {text('选中套餐，逐个模型对比预计用量。不同模型共享套餐额度，下列次数不能相加。', 'Select plans to compare usage model by model. Models share each plan’s quota; these estimates do not add up.')}
          </p>
        </div>
        <div role="group" aria-label={text('对比套餐', 'Compare plans')} className="flex flex-wrap gap-2">
          {plans.map((plan, index) => {
            const active = visible.some((entry) => entry.id === plan.id);
            return (
              <button key={plan.id} type="button" aria-pressed={active} onClick={() => {
                setSelectedIds(active
                  ? visible.length > 1 ? visible.filter((entry) => entry.id !== plan.id).map((entry) => entry.id) : [plan.id]
                  : [...visible.map((entry) => entry.id), plan.id]);
              }} className={`flex items-center gap-2 rounded-lg border px-3 py-2 text-xs font-medium focus-visible:outline-2 focus-visible:outline-emerald-500 ${
                active ? isDark ? 'border-slate-400 bg-slate-800' : 'border-slate-400 bg-white' : `${border} ${muted}`
              }`}>
                <span aria-hidden="true" className={`h-2 w-2 rounded-sm ${active ? COLORS[index % COLORS.length] : isDark ? 'bg-slate-600' : 'bg-slate-300'}`} />
                {plan.name}
              </button>
            );
          })}
        </div>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className={`flex flex-wrap items-center gap-3 text-xs ${muted}`}>
            <span>{text('每次请求假设', 'Per-request assumption')}</span>
            {([
              { key: 'input', label: text('输入 tokens', 'Input tokens'), value: input, set: setInput },
              { key: 'output', label: text('输出 tokens', 'Output tokens'), value: output, set: setOutput },
            ]).map((field) => (
              <label key={field.key} htmlFor={`${id}-${field.key}`} className="flex items-center gap-2">
                {field.label}
                <input id={`${id}-${field.key}`} type="number" min="0" step="1" inputMode="numeric"
                  value={field.value} onChange={(event) => field.set(event.target.value)}
                  className={`w-24 rounded-md border px-2 py-1.5 font-mono tabular-nums focus-visible:outline-2 focus-visible:outline-emerald-500 ${border} ${isDark ? 'bg-slate-900 text-slate-100' : 'bg-white text-slate-900'}`} />
              </label>
            ))}
          </div>
          <div role="group" aria-label={text('用量窗口', 'Usage window')} className={`inline-flex rounded-lg border p-1 ${border}`}>
            {PLAN_WINDOWS.map((key) => (
              <button key={key} type="button" aria-pressed={window === key} onClick={() => setWindow(key)}
                className={`rounded-md px-3 py-1.5 text-xs font-medium focus-visible:outline-2 focus-visible:outline-emerald-500 ${
                  window === key ? isDark ? 'bg-slate-100 text-slate-900' : 'bg-slate-900 text-white' : muted
                }`}>{labels[key]}</button>
            ))}
          </div>
        </div>
        <p className={`text-xs leading-5 ${muted}`}>
          {text('文本按无缓存命中计算，价格已含分组倍率；图片按默认档位每张、视频按每秒计算。可展开每个模型查看费用与各窗口用量。', 'Text estimates assume no cache hits and include group rates. Images use the default tier per image; video uses cost per second. Expand a model for costs and all windows.')}
        </p>
      </div>

      <div className="p-5 sm:p-6">
        {visible.some((plan) => plan.models == null) && (
          <p role="status" className={`mb-4 rounded-lg border p-3 text-sm ${border} ${muted}`}>
            {text('部分套餐的模型定价暂未加载，可使用页面顶部「刷新」重试。', 'Model prices for some plans are unavailable. Use Refresh at the top of the page to retry.')}
          </p>
        )}
        {rows.length === 0 ? (
          <p className={`py-8 text-center text-sm ${muted}`}>{text('暂无可展示的模型用量。', 'No model usage data available.')}</p>
        ) : (
          <>
            <div aria-hidden="true" className={`mb-5 hidden grid-cols-[minmax(160px,1fr)_2fr_110px_110px] gap-4 text-xs sm:grid ${muted}`}>
              <span>{text('模型', 'Model')}</span><span>{text('预计用量', 'Estimated usage')} / {labels[window]}</span>
              <span className="text-right">{text('数量', 'Amount')}</span><span className="text-right">{text('单次计费', 'Unit cost')}</span>
            </div>
            <div className={`divide-y ${isDark ? 'divide-slate-700' : 'divide-slate-200'}`}>
              {rows.map((row) => (
                <details key={row.modelId} className="py-4 first:pt-0">
                  <summary className="cursor-pointer list-none rounded-sm focus-visible:outline-2 focus-visible:outline-emerald-500">
                    <div className="grid items-start gap-x-4 gap-y-3 sm:grid-cols-[minmax(160px,1fr)_2fr_110px_110px]">
                      <div className="min-w-0 sm:row-span-1">
                        <p className="break-words text-sm font-medium">{row.offers[0].model.display_name || row.modelId}</p>
                        <span className={`mt-1 block text-xs ${muted}`}>{text('定价与用量详情', 'Pricing & usage details')} <span aria-hidden="true">⌄</span></span>
                      </div>
                      <div className="space-y-3 sm:col-span-3">
                        {row.offers.map((offer) => {
                          const unit = offer.model.billing_mode === 'image' ? text('张', 'images')
                            : offer.model.billing_mode === 'video' ? text('秒', 'seconds') : text('次', 'requests');
                          return (
                            <div key={offer.plan.id} className="grid grid-cols-[minmax(0,1fr)_100px] items-center gap-x-3 gap-y-1 sm:grid-cols-[minmax(0,1fr)_110px_110px] sm:gap-x-4">
                              <div className="min-w-0">
                                <p className={`mb-1.5 truncate text-[11px] ${muted}`}>{offer.plan.name}</p>
                                <div aria-hidden="true" className={`h-2 overflow-hidden rounded-sm ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}>
                                  {offer.estimate.kind === 'count' && <div className={`h-full rounded-sm transition-[width] motion-reduce:transition-none ${offer.color}`}
                                    style={{ width: `${offer.estimate.count / maxCounts[unitKey(offer.model.billing_mode)] * 100}%` }} />}
                                </div>
                              </div>
                              <span className="text-right text-xs tabular-nums">
                                {estimateLabel(offer.estimate)}
                                {offer.estimate.kind === 'count' && <span className={`ml-1 ${muted}`}>{unit}</span>}
                              </span>
                              <span className={`col-span-2 text-right font-mono text-xs tabular-nums sm:col-span-1 ${muted}`}>
                                {offer.cost === null ? '—' : `$${offer.cost.toLocaleString('en-US', { maximumSignificantDigits: 4 })}`}
                                <span className="ml-1 font-sans">/ {offer.model.billing_mode === 'video' ? text('秒', 's') : offer.model.billing_mode === 'image' ? text('张', 'image') : text('次', 'req')}</span>
                              </span>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  </summary>
                  <div className={`mt-4 space-y-4 rounded-xl p-4 text-xs leading-6 ${isDark ? 'bg-slate-900/60' : 'bg-slate-50'}`}>
                    <p className={`break-all font-mono ${muted}`}>{row.modelId}</p>
                    {row.offers.map((offer) => (
                      <div key={offer.plan.id}>
                        <p className="font-semibold">{offer.plan.name}</p>
                        <p className={muted}>{text('当前假设单次费用', 'Unit cost with current assumptions')}: {offer.cost === null ? text('暂无法估算', 'Unavailable') : `$${offer.cost.toLocaleString('en-US', { maximumSignificantDigits: 6 })}`}</p>
                        {offer.model.billing_mode === 'token' && (
                          <div className={muted}>
                            <p>{text('费用 = 输入 tokens × 对应区间输入单价 + 输出 tokens × 对应区间输出单价，单价以百万 tokens 为单位，已含分组倍率。', 'Cost = input tokens × applicable input price + output tokens × applicable output price, with prices per million tokens and group rates already included.')}</p>
                            {(offer.model.pricing_details.intervals.length
                              ? offer.model.pricing_details.intervals
                              : [{ ...offer.model.effective_pricing_usd, min_tokens: 0, max_tokens: null }]
                            ).map((tier) => (
                              <p key={tier.min_tokens}>
                                {tier.min_tokens > 0 || tier.max_tokens !== null
                                  ? `${number.format(tier.min_tokens)} < tokens${tier.max_tokens === null ? '' : ` ≤ ${number.format(tier.max_tokens)}`} · `
                                  : ''}
                                {text('输入', 'Input')} ${tier.input_per_mtok_usd ?? '—'} / 1M · {text('输出', 'Output')} ${tier.output_per_mtok_usd ?? '—'} / 1M
                              </p>
                            ))}
                          </div>
                        )}
                        <div className="mt-2 flex flex-wrap gap-x-5 gap-y-1">
                          {PLAN_WINDOWS.map((key) => <span key={key}>{labels[key]}: {estimateLabel(modelWindowEstimate(offer.model, offer.plan.limits, key, assumptions))}</span>)}
                        </div>
                      </div>
                    ))}
                  </div>
                </details>
              ))}
            </div>
          </>
        )}
        <p className={`mt-5 border-t pt-4 text-xs leading-6 ${border} ${muted}`}>
          {text(
            '每行表示将所选窗口的共享额度全部用于该模型时的估算，不是每个模型各有一份独立额度。各窗口单独估算，其他窗口限额、套餐到期、缓存、实际上下文与高峰计费仍会影响结果。相同单位的条形使用统一线性比例，未知或无上限时不绘制。',
            'Each row estimates spending the window’s entire shared quota on that model, not a separate allowance per model. Windows are estimated separately; other limits, expiry, caching, actual context and peak pricing still affect results. Bars of the same unit share a linear scale; unknown or uncapped values have no bar.',
          )}
        </p>
      </div>
    </section>
  );
}
