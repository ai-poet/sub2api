'use client';

import { useEffect, useRef, useState } from 'react';
import type { Locale } from '@/lib/locale';
import { formatCreatedAt, formatStatus, getPaymentDisplayInfo, getStatusBadgeClass } from '@/lib/pay-utils';
import { buildAppApiPath } from '@/lib/public-path';
import {
  PAYMENT_RECHECK_OUTCOMES,
  PAYMENT_RECHECK_OUTCOME_LABELS,
  PAYMENT_RISK_REASON_LABELS,
  type PaymentRecheckBatch,
  type PaymentRecheckItem,
  type PaymentRecheckOutcome,
} from '@/lib/payment-risk/shared';

interface RiskRecheckPanelProps {
  token: string;
  days: number;
  locale: Locale;
  dark: boolean;
  onViewDetail: (orderId: string) => void;
  /** 一轮复核结束（完成、停止或出错）后调用，用来刷新风控面板 */
  onFinished: () => void;
}

type OutcomeCounts = Record<PaymentRecheckOutcome, number>;

const EMPTY_COUNTS: OutcomeCounts = { ok: 0, mismatch: 0, failed: 0, skipped: 0 };

function getText(locale: Locale) {
  return locale === 'en'
    ? {
        title: 'Historical order re-check',
        description:
          'Queries the payment platform order by order for every order credited by an EasyPay callback in this period. Read-only: order states are not changed. Orders that do not match are recorded as "Platform check mismatch" and appear in the risk events. Queries run one at a time in small batches to keep the load on the platform low.',
        start: (days: number) => `Re-check the last ${days} days`,
        stop: 'Stop',
        progress: (done: number, total: number) => `Checked ${done} / ${total}`,
        window: (date: string) => `Since ${date}`,
        noCandidates: 'No EasyPay callback-credited orders in this period.',
        allMatched: 'Every order in this period matches the platform records.',
        stopped: 'Stopped. Start again to re-check from the beginning.',
        failed: 'The re-check request failed and was stopped.',
        invalidToken: 'Invalid admin token',
        order: 'Order',
        user: 'User',
        amount: 'Paid amount',
        status: 'Status',
        payment: 'Payment',
        paidAt: 'Paid at',
        outcome: 'Result',
        details: 'Details',
        alreadyRecorded: 'already recorded',
      }
    : {
        title: '历史订单复核',
        description:
          '逐笔向支付平台查单，核对这段时间里由易支付回调入账的订单。只读，不改订单状态；对不上的订单会记为"平台复核不符"，显示在上面的风险事件里。查单按顺序小批量进行，避免给平台造成压力。',
        start: (days: number) => `复核最近 ${days} 天`,
        stop: '停止',
        progress: (done: number, total: number) => `已复核 ${done} / ${total}`,
        window: (date: string) => `范围：${date} 起`,
        noCandidates: '这段时间没有由易支付回调入账的订单。',
        allMatched: '这段时间的订单与平台记录全部一致。',
        stopped: '已停止。重新开始会从头复核。',
        failed: '复核请求失败，已停止。',
        invalidToken: '管理员凭证无效',
        order: '订单',
        user: '用户',
        amount: '支付金额',
        status: '订单状态',
        payment: '支付方式',
        paidAt: '支付时间',
        outcome: '结果',
        details: '说明',
        alreadyRecorded: '之前已记录',
      };
}

function outcomeBadgeClass(outcome: PaymentRecheckOutcome, dark: boolean): string {
  if (outcome === 'mismatch') return dark ? 'bg-red-500/20 text-red-200' : 'bg-red-100 text-red-700';
  if (outcome === 'failed') return dark ? 'bg-amber-500/20 text-amber-200' : 'bg-amber-100 text-amber-800';
  if (outcome === 'ok') return dark ? 'bg-emerald-500/20 text-emerald-200' : 'bg-emerald-100 text-emerald-700';
  return dark ? 'bg-slate-600 text-slate-200' : 'bg-slate-100 text-slate-700';
}

export default function RiskRecheckPanel({ token, days, locale, dark, onViewDetail, onFinished }: RiskRecheckPanelProps) {
  const text = getText(locale);
  const currency = locale === 'en' ? '$' : '¥';

  const [running, setRunning] = useState(false);
  const [state, setState] = useState<'idle' | 'done' | 'stopped' | 'failed'>('idle');
  const [since, setSince] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [counts, setCounts] = useState<OutcomeCounts>(EMPTY_COUNTS);
  const [items, setItems] = useState<PaymentRecheckItem[]>([]);
  const [error, setError] = useState('');
  const stopRequested = useRef(false);
  const mounted = useRef(true);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      stopRequested.current = true;
    };
  }, []);

  const processed = PAYMENT_RECHECK_OUTCOMES.reduce((sum, outcome) => sum + counts[outcome], 0);
  const percent = total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0;

  const run = async () => {
    stopRequested.current = false;
    setRunning(true);
    setState('idle');
    setError('');
    setSince(null);
    setTotal(0);
    setCounts(EMPTY_COUNTS);
    setItems([]);

    let windowStart: string | null = null;
    let cursor: string | null = null;
    let outcome: 'done' | 'stopped' | 'failed' = 'stopped';
    try {
      while (!stopRequested.current) {
        const body = windowStart ? { since: windowStart, cursor } : { days };
        const res = await fetch(buildAppApiPath(`/api/admin/risk/recheck${locale === 'en' ? '?lang=en' : ''}`), {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        if (!mounted.current) return;
        if (!res.ok) {
          setError(res.status === 401 ? text.invalidToken : text.failed);
          outcome = 'failed';
          break;
        }

        const batch = (await res.json()) as PaymentRecheckBatch;
        if (!mounted.current) return;
        windowStart = batch.since;
        cursor = batch.nextCursor;
        setSince(batch.since);
        setTotal(batch.total);
        setCounts((prev) => {
          const next = { ...prev };
          for (const item of batch.results) next[item.outcome] += 1;
          return next;
        });
        const notable = batch.results.filter((item) => item.outcome !== 'ok');
        if (notable.length > 0) setItems((prev) => [...prev, ...notable]);

        if (batch.done) {
          outcome = 'done';
          break;
        }
      }
    } catch {
      if (!mounted.current) return;
      setError(text.failed);
      outcome = 'failed';
    }

    setRunning(false);
    setState(outcome);
    onFinished();
  };

  const sectionCls = [
    'rounded-xl border',
    dark ? 'border-slate-700 bg-slate-800/70' : 'border-slate-200 bg-white shadow-sm',
  ].join(' ');
  const hintCls = `text-xs ${dark ? 'text-slate-400' : 'text-slate-500'}`;
  const thCls = `whitespace-nowrap px-4 py-3 text-left text-xs font-medium uppercase ${dark ? 'text-slate-400' : 'text-gray-500'}`;
  const tdCls = `whitespace-nowrap px-4 py-3 text-sm ${dark ? 'text-slate-200' : 'text-slate-900'}`;
  const tdMuted = `whitespace-nowrap px-4 py-3 text-sm ${dark ? 'text-slate-400' : 'text-gray-500'}`;
  const primaryBtn = [
    'inline-flex items-center rounded-lg px-3 py-1.5 text-xs font-medium transition-colors disabled:opacity-50',
    dark ? 'bg-indigo-500/80 text-white hover:bg-indigo-500' : 'bg-blue-600 text-white hover:bg-blue-700',
  ].join(' ');
  const secondaryBtn = [
    'inline-flex items-center rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
    dark ? 'border-slate-600 text-slate-200 hover:bg-slate-800' : 'border-slate-300 text-slate-700 hover:bg-slate-100',
  ].join(' ');

  const describeItem = (item: PaymentRecheckItem) => {
    const parts: string[] = [];
    if (item.reason) parts.push(PAYMENT_RISK_REASON_LABELS[item.reason]?.[locale] ?? item.reason);
    if (item.message) parts.push(item.message);
    if (item.outcome === 'mismatch' && item.recorded === false) parts.push(text.alreadyRecorded);
    return parts.join(' · ') || '-';
  };

  const started = running || state !== 'idle';

  return (
    <section className={sectionCls}>
      <div className="p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <h3 className={`text-sm font-semibold ${dark ? 'text-slate-200' : 'text-slate-800'}`}>{text.title}</h3>
            <p className={`mt-1 max-w-3xl ${hintCls}`}>{text.description}</p>
          </div>
          {running ? (
            <button
              type="button"
              className={secondaryBtn}
              onClick={() => {
                stopRequested.current = true;
              }}
            >
              {text.stop}
            </button>
          ) : (
            <button type="button" className={primaryBtn} onClick={run}>
              {text.start(days)}
            </button>
          )}
        </div>

        {started && (
          <div className="mt-4 space-y-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className={`text-sm ${dark ? 'text-slate-200' : 'text-slate-800'}`}>{text.progress(processed, total)}</span>
              {since && <span className={hintCls}>{text.window(formatCreatedAt(since, locale))}</span>}
            </div>
            <div className={`h-2 w-full overflow-hidden rounded-full ${dark ? 'bg-slate-700' : 'bg-slate-200'}`}>
              <div
                className={`h-full rounded-full transition-all ${counts.mismatch > 0 ? 'bg-red-500' : dark ? 'bg-indigo-400' : 'bg-blue-600'}`}
                style={{ width: `${state === 'done' ? 100 : percent}%` }}
              />
            </div>
            <div className="flex flex-wrap gap-2">
              {PAYMENT_RECHECK_OUTCOMES.map((outcome) => (
                <span
                  key={outcome}
                  className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${outcomeBadgeClass(outcome, dark)}`}
                >
                  {PAYMENT_RECHECK_OUTCOME_LABELS[outcome][locale]} {counts[outcome]}
                </span>
              ))}
            </div>
            {error && <p className={`text-sm ${dark ? 'text-red-400' : 'text-red-600'}`}>{error}</p>}
            {state === 'stopped' && <p className={hintCls}>{text.stopped}</p>}
            {state === 'done' && total === 0 && <p className={hintCls}>{text.noCandidates}</p>}
            {state === 'done' && total > 0 && items.length === 0 && (
              <p className={`text-sm ${dark ? 'text-emerald-300' : 'text-emerald-700'}`}>{text.allMatched}</p>
            )}
          </div>
        )}
      </div>

      {items.length > 0 && (
        <div className="overflow-x-auto">
          <table className={`min-w-full divide-y ${dark ? 'divide-slate-700' : 'divide-gray-200'}`}>
            <thead className={dark ? 'bg-slate-800/50' : 'bg-gray-50'}>
              <tr>
                <th className={thCls}>{text.order}</th>
                <th className={thCls}>{text.user}</th>
                <th className={thCls}>{text.amount}</th>
                <th className={thCls}>{text.status}</th>
                <th className={thCls}>{text.payment}</th>
                <th className={thCls}>{text.paidAt}</th>
                <th className={thCls}>{text.outcome}</th>
                <th className={thCls}>{text.details}</th>
              </tr>
            </thead>
            <tbody className={`divide-y ${dark ? 'divide-slate-700/60 bg-slate-900' : 'divide-gray-200 bg-white'}`}>
              {items.map((item) => {
                const { channel, provider } = getPaymentDisplayInfo(item.paymentType, locale);
                return (
                  <tr key={item.orderId} className={dark ? 'hover:bg-slate-700/40' : 'hover:bg-gray-50'}>
                    <td className={tdCls}>
                      <button
                        type="button"
                        onClick={() => onViewDetail(item.orderId)}
                        className={dark ? 'text-indigo-400 hover:underline' : 'text-blue-600 hover:underline'}
                        title={item.orderId}
                      >
                        {item.orderId.slice(0, 12)}…
                      </button>
                    </td>
                    <td className={tdCls}>
                      <div>{item.userName || `#${item.userId}`}</div>
                      <div className={`text-xs ${dark ? 'text-slate-500' : 'text-gray-400'}`}>
                        {item.userEmail || `#${item.userId}`}
                      </div>
                    </td>
                    <td className={tdCls}>
                      {currency}
                      {item.amount.toFixed(2)}
                    </td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm">
                      <span
                        className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${getStatusBadgeClass(item.orderStatus, dark)}`}
                      >
                        {formatStatus(item.orderStatus, locale)}
                      </span>
                    </td>
                    <td className={tdMuted}>{provider ? `${channel} · ${provider}` : channel}</td>
                    <td className={tdMuted}>{item.paidAt ? formatCreatedAt(item.paidAt, locale) : '-'}</td>
                    <td className="whitespace-nowrap px-4 py-3 text-sm">
                      <span
                        className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${outcomeBadgeClass(item.outcome, dark)}`}
                      >
                        {PAYMENT_RECHECK_OUTCOME_LABELS[item.outcome][locale]}
                      </span>
                    </td>
                    <td className={`max-w-md px-4 py-3 text-xs break-all ${dark ? 'text-slate-300' : 'text-slate-600'}`}>
                      {describeItem(item)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
