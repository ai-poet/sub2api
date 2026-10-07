'use client';

import { useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import PayPageLayout from '@/components/PayPageLayout';
import PaginationBar from '@/components/PaginationBar';
import OrderDetail from '@/components/admin/OrderDetail';
import RiskDailyChart from '@/components/admin/RiskDailyChart';
import RiskRecheckPanel from '@/components/admin/RiskRecheckPanel';
import { getAdminAccessHint } from '@/lib/branding';
import { resolveLocale, type Locale } from '@/lib/locale';
import { formatCreatedAt, formatStatus, getPaymentDisplayInfo, getStatusBadgeClass } from '@/lib/pay-utils';
import { buildAppApiPath } from '@/lib/public-path';
import {
  DUPLICATE_TRADE_NO_META,
  PAID_WITHOUT_TRADE_NO_META,
  PAYMENT_RISK_ACTIONS,
  PAYMENT_RISK_ACTION_META,
  PAYMENT_RISK_REASON_LABELS,
  type PaymentRiskAction,
  type PaymentRiskOrderBrief,
  type PaymentRiskOverview,
  type RiskSeverity,
} from '@/lib/payment-risk/shared';

const DAYS_OPTIONS = [7, 30, 90, 365] as const;

const EVENTS_SECTION_ID = 'risk-events';
const NO_TRADE_NO_SECTION_ID = 'risk-no-trade-no';
const DUPLICATES_SECTION_ID = 'risk-duplicates';

/** 审计 detail 中值得在列表里直接展示的字段，按顺序拼接。 */
const DETAIL_KEYS = [
  'param',
  'notifyTradeNo',
  'storedTradeNo',
  'upstreamStatus',
  'upstreamAmount',
  'upstreamTradeNo',
  'upstreamMessage',
  'notifyAmount',
  'expected',
  'paid',
  'tradeNo',
] as const;

type OrderDetailData = Parameters<typeof OrderDetail>[0]['order'];

function getText(locale: Locale) {
  return locale === 'en'
    ? {
        missingToken: 'Missing admin token',
        missingTokenHint: getAdminAccessHint(locale),
        invalidToken: 'Invalid admin token',
        loadFailed: 'Failed to load risk data',
        loadDetailFailed: 'Failed to load order details',
        title: 'Payment Risk',
        subtitle: 'Suspicious callbacks and orders that need a closer look',
        daySuffix: 'd',
        refresh: 'Refresh',
        loading: 'Loading...',
        events: 'Risk events',
        all: 'All',
        time: 'Time',
        type: 'Type',
        order: 'Order',
        user: 'User',
        amount: 'Paid amount',
        status: 'Status',
        payment: 'Payment',
        details: 'Details',
        source: 'Source',
        noEvents: 'No risk events in this period',
        tradeNo: 'Trade No.',
        orderCount: 'Orders',
        paidAt: 'Paid at',
        topUsers: 'Users with the most risk events',
        topUsersHint: 'Counts every risk event above, grouped by the user who placed the order.',
        email: 'Email',
        eventCount: 'Events',
        lastEvent: 'Last event',
        none: 'Nothing here',
        showingFirst: (n: number) => `Showing the first ${n}`,
        severity: { high: 'High', medium: 'Medium', low: 'Low' } as Record<RiskSeverity, string>,
      }
    : {
        missingToken: '缺少管理员凭证',
        missingTokenHint: getAdminAccessHint(locale),
        invalidToken: '管理员凭证无效',
        loadFailed: '加载支付风控数据失败',
        loadDetailFailed: '加载订单详情失败',
        title: '支付风控',
        subtitle: '可疑回调与需要人工核对的订单',
        daySuffix: '天',
        refresh: '刷新',
        loading: '加载中...',
        events: '风险事件',
        all: '全部',
        time: '时间',
        type: '类型',
        order: '订单',
        user: '用户',
        amount: '支付金额',
        status: '订单状态',
        payment: '支付方式',
        details: '详情',
        source: '来源',
        noEvents: '这段时间没有风险事件',
        tradeNo: '交易号',
        orderCount: '订单数',
        paidAt: '支付时间',
        topUsers: '风险事件最多的用户',
        topUsersHint: '按下单用户汇总上面所有风险事件。',
        email: '邮箱',
        eventCount: '事件数',
        lastEvent: '最近一次',
        none: '暂无',
        showingFirst: (n: number) => `只显示前 ${n} 条`,
        severity: { high: '高危', medium: '关注', low: '一般' } as Record<RiskSeverity, string>,
      };
}

function severityBadgeClass(severity: RiskSeverity, dark: boolean): string {
  if (severity === 'high') return dark ? 'bg-red-500/20 text-red-200' : 'bg-red-100 text-red-700';
  if (severity === 'medium') return dark ? 'bg-amber-500/20 text-amber-200' : 'bg-amber-100 text-amber-800';
  return dark ? 'bg-slate-600 text-slate-200' : 'bg-slate-100 text-slate-700';
}

function severityCountClass(severity: RiskSeverity, count: number, dark: boolean): string {
  if (count === 0) return dark ? 'text-slate-500' : 'text-slate-400';
  if (severity === 'high') return dark ? 'text-red-300' : 'text-red-600';
  if (severity === 'medium') return dark ? 'text-amber-300' : 'text-amber-600';
  return dark ? 'text-slate-200' : 'text-slate-800';
}

function truncate(value: string, max: number): string {
  return value.length > max ? `${value.slice(0, max)}…` : value;
}

/** 把审计 detail 转成一行可读文字：原因 + 关键字段，非 JSON 原样截断。 */
function describeDetail(detail: string | null, locale: Locale): string {
  if (!detail) return '-';
  let parsed: unknown;
  try {
    parsed = JSON.parse(detail);
  } catch {
    return truncate(detail, 160);
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return truncate(detail, 160);
  }
  const record = parsed as Record<string, unknown>;
  const parts: string[] = [];
  if (typeof record.reason === 'string') {
    parts.push(PAYMENT_RISK_REASON_LABELS[record.reason]?.[locale] ?? record.reason);
  }
  for (const key of DETAIL_KEYS) {
    const value = record[key];
    if (value === undefined || value === null || value === '') continue;
    parts.push(`${key}=${typeof value === 'object' ? JSON.stringify(value) : String(value)}`);
  }
  return truncate(parts.length > 0 ? parts.join(' · ') : detail, 160);
}

function SummaryCard({
  label,
  description,
  count,
  severity,
  severityText,
  active,
  onClick,
  dark,
}: {
  label: string;
  description: string;
  count: number;
  severity: RiskSeverity;
  severityText: string;
  active: boolean;
  onClick: () => void;
  dark: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={description}
      className={[
        'rounded-xl border p-4 text-left transition-colors',
        active
          ? dark
            ? 'border-indigo-400/60 bg-indigo-500/10'
            : 'border-blue-400 bg-blue-50'
          : dark
            ? 'border-slate-700 bg-slate-800/60 hover:bg-slate-800'
            : 'border-slate-200 bg-white shadow-sm hover:bg-slate-50',
      ].join(' ')}
    >
      <div className="flex items-start justify-between gap-2">
        <span className={`text-xs ${dark ? 'text-slate-400' : 'text-slate-500'}`}>{label}</span>
        <span
          className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold ${severityBadgeClass(severity, dark)}`}
        >
          {severityText}
        </span>
      </div>
      <div className={`mt-2 text-2xl font-semibold ${severityCountClass(severity, count, dark)}`}>{count}</div>
    </button>
  );
}

function RiskContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token');
  const theme = searchParams.get('theme') === 'dark' ? 'dark' : 'light';
  const uiMode = searchParams.get('ui_mode') || 'standalone';
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = theme === 'dark';
  const isEmbedded = uiMode === 'embedded';
  const currency = locale === 'en' ? '$' : '¥';
  const text = useMemo(() => getText(locale), [locale]);

  const [days, setDays] = useState<number>(30);
  const [action, setAction] = useState<PaymentRiskAction | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [data, setData] = useState<PaymentRiskOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [detailOrder, setDetailOrder] = useState<OrderDetailData | null>(null);

  const fetchData = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const params = new URLSearchParams({ days: String(days), page: String(page), page_size: String(pageSize) });
      if (action) params.set('action', action);
      if (locale === 'en') params.set('lang', 'en');
      const res = await fetch(buildAppApiPath(`/api/admin/risk?${params}`), {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        setError(res.status === 401 ? text.invalidToken : text.loadFailed);
        return;
      }
      setData((await res.json()) as PaymentRiskOverview);
    } catch {
      setError(text.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [token, days, page, pageSize, action, locale, text]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleViewDetail = useCallback(
    async (orderId: string) => {
      if (!token) return;
      try {
        const res = await fetch(buildAppApiPath(`/api/admin/orders/${encodeURIComponent(orderId)}`), {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!res.ok) {
          setError(text.loadDetailFailed);
          return;
        }
        setDetailOrder((await res.json()) as OrderDetailData);
      } catch {
        setError(text.loadDetailFailed);
      }
    },
    [token, text],
  );

  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  const selectAction = (next: PaymentRiskAction | null) => {
    setAction(next);
    setPage(1);
  };

  if (!token) {
    return (
      <div className={`flex min-h-screen items-center justify-center p-4 ${isDark ? 'bg-slate-950' : 'bg-slate-50'}`}>
        <div className="text-center text-red-500">
          <p className="text-lg font-medium">{text.missingToken}</p>
          <p className={`mt-2 text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{text.missingTokenHint}</p>
        </div>
      </div>
    );
  }

  const btnBase = [
    'inline-flex items-center rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
    isDark ? 'border-slate-600 text-slate-200 hover:bg-slate-800' : 'border-slate-300 text-slate-700 hover:bg-slate-100',
  ].join(' ');
  const btnActive = [
    'inline-flex items-center rounded-lg px-3 py-1.5 text-xs font-medium',
    isDark ? 'bg-indigo-500/30 text-indigo-200 ring-1 ring-indigo-400/40' : 'bg-blue-600 text-white',
  ].join(' ');
  const sectionCls = [
    'rounded-xl border',
    isDark ? 'border-slate-700 bg-slate-800/70' : 'border-slate-200 bg-white shadow-sm',
  ].join(' ');
  const sectionTitleCls = `text-sm font-semibold ${isDark ? 'text-slate-200' : 'text-slate-800'}`;
  const hintCls = `mt-1 text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`;
  const thCls = `whitespace-nowrap px-4 py-3 text-left text-xs font-medium uppercase ${isDark ? 'text-slate-400' : 'text-gray-500'}`;
  const tdCls = `whitespace-nowrap px-4 py-3 text-sm ${isDark ? 'text-slate-200' : 'text-slate-900'}`;
  const tdMuted = `whitespace-nowrap px-4 py-3 text-sm ${isDark ? 'text-slate-400' : 'text-gray-500'}`;
  const tableCls = `min-w-full divide-y ${isDark ? 'divide-slate-700' : 'divide-gray-200'}`;
  const theadCls = isDark ? 'bg-slate-800/50' : 'bg-gray-50';
  const tbodyCls = `divide-y ${isDark ? 'divide-slate-700/60 bg-slate-900' : 'divide-gray-200 bg-white'}`;
  const rowCls = isDark ? 'hover:bg-slate-700/40' : 'hover:bg-gray-50';
  const linkCls = isDark ? 'text-indigo-400 hover:underline' : 'text-blue-600 hover:underline';
  const emptyCls = `py-10 text-center text-sm ${isDark ? 'text-slate-500' : 'text-gray-500'}`;

  const orderLink = (orderId: string) => (
    <button type="button" onClick={() => handleViewDetail(orderId)} className={linkCls} title={orderId}>
      {orderId.slice(0, 12)}…
    </button>
  );
  const userCell = (order: PaymentRiskOrderBrief) => (
    <div>
      <div>{order.userName || `#${order.userId}`}</div>
      <div className={`text-xs ${isDark ? 'text-slate-500' : 'text-gray-400'}`}>{order.userEmail || `#${order.userId}`}</div>
    </div>
  );
  const amountText = (order: PaymentRiskOrderBrief) => `${currency}${(order.payAmount ?? order.amount).toFixed(2)}`;
  const statusBadge = (status: string) => (
    <span className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${getStatusBadgeClass(status, isDark)}`}>
      {formatStatus(status, locale)}
    </span>
  );
  const paymentText = (paymentType: string) => {
    const { channel, provider } = getPaymentDisplayInfo(paymentType, locale);
    return provider ? `${channel} · ${provider}` : channel;
  };

  return (
    <PayPageLayout
      isDark={isDark}
      isEmbedded={isEmbedded}
      maxWidth="full"
      title={text.title}
      subtitle={text.subtitle}
      locale={locale}
      actions={
        <>
          {DAYS_OPTIONS.map((d) => (
            <button
              key={d}
              type="button"
              onClick={() => {
                setDays(d);
                setPage(1);
              }}
              className={days === d ? btnActive : btnBase}
            >
              {d}
              {text.daySuffix}
            </button>
          ))}
          <button type="button" onClick={fetchData} className={btnBase}>
            {text.refresh}
          </button>
        </>
      }
    >
      {error && (
        <div
          className={`mb-4 rounded-lg border p-3 text-sm ${isDark ? 'border-red-800 bg-red-950/50 text-red-400' : 'border-red-200 bg-red-50 text-red-600'}`}
        >
          {error}
          <button onClick={() => setError('')} className="ml-2 opacity-60 hover:opacity-100">
            ✕
          </button>
        </div>
      )}

      {!data ? (
        <div className={`py-24 text-center ${isDark ? 'text-slate-400' : 'text-gray-500'}`}>
          {loading ? text.loading : null}
        </div>
      ) : (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            {PAYMENT_RISK_ACTIONS.map((riskAction) => {
              const meta = PAYMENT_RISK_ACTION_META[riskAction];
              return (
                <SummaryCard
                  key={riskAction}
                  label={meta.label[locale]}
                  description={meta.description[locale]}
                  count={data.summary.events[riskAction]}
                  severity={meta.severity}
                  severityText={text.severity[meta.severity]}
                  active={action === riskAction}
                  onClick={() => {
                    selectAction(action === riskAction ? null : riskAction);
                    scrollTo(EVENTS_SECTION_ID);
                  }}
                  dark={isDark}
                />
              );
            })}
            <SummaryCard
              label={PAID_WITHOUT_TRADE_NO_META.label[locale]}
              description={PAID_WITHOUT_TRADE_NO_META.description[locale]}
              count={data.summary.paidWithoutTradeNo}
              severity={PAID_WITHOUT_TRADE_NO_META.severity}
              severityText={text.severity[PAID_WITHOUT_TRADE_NO_META.severity]}
              active={false}
              onClick={() => scrollTo(NO_TRADE_NO_SECTION_ID)}
              dark={isDark}
            />
            <SummaryCard
              label={DUPLICATE_TRADE_NO_META.label[locale]}
              description={DUPLICATE_TRADE_NO_META.description[locale]}
              count={data.summary.duplicateTradeNoGroups}
              severity={DUPLICATE_TRADE_NO_META.severity}
              severityText={text.severity[DUPLICATE_TRADE_NO_META.severity]}
              active={false}
              onClick={() => scrollTo(DUPLICATES_SECTION_ID)}
              dark={isDark}
            />
          </div>

          <RiskDailyChart data={data.daily} dark={isDark} locale={locale} />

          <RiskRecheckPanel
            token={token}
            days={days}
            locale={locale}
            dark={isDark}
            onViewDetail={handleViewDetail}
            onFinished={fetchData}
          />

          <section id={EVENTS_SECTION_ID} className={sectionCls}>
            <div className="p-4 pb-2">
              <div className="flex items-center justify-between gap-2">
                <h3 className={sectionTitleCls}>{text.events}</h3>
                {loading && <span className={hintCls}>{text.loading}</span>}
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => selectAction(null)}
                  className={[
                    'rounded-full px-3 py-1 text-sm transition-colors',
                    action === null
                      ? isDark
                        ? 'bg-indigo-500/30 text-indigo-200 ring-1 ring-indigo-400/40'
                        : 'bg-blue-600 text-white'
                      : isDark
                        ? 'bg-slate-800 text-slate-400 hover:bg-slate-700'
                        : 'bg-gray-100 text-gray-600 hover:bg-gray-200',
                  ].join(' ')}
                >
                  {text.all}
                </button>
                {PAYMENT_RISK_ACTIONS.map((riskAction) => (
                  <button
                    key={riskAction}
                    type="button"
                    onClick={() => selectAction(riskAction)}
                    title={PAYMENT_RISK_ACTION_META[riskAction].description[locale]}
                    className={[
                      'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm transition-colors',
                      action === riskAction
                        ? isDark
                          ? 'bg-indigo-500/30 text-indigo-200 ring-1 ring-indigo-400/40'
                          : 'bg-blue-600 text-white'
                        : isDark
                          ? 'bg-slate-800 text-slate-400 hover:bg-slate-700'
                          : 'bg-gray-100 text-gray-600 hover:bg-gray-200',
                    ].join(' ')}
                  >
                    <span
                      className="inline-block h-2 w-2 rounded-full"
                      style={{ backgroundColor: PAYMENT_RISK_ACTION_META[riskAction].color }}
                    />
                    {PAYMENT_RISK_ACTION_META[riskAction].label[locale]}
                    <span className="opacity-70">{data.summary.events[riskAction]}</span>
                  </button>
                ))}
              </div>
              {action && <p className={hintCls}>{PAYMENT_RISK_ACTION_META[action].description[locale]}</p>}
            </div>
            <div className="overflow-x-auto">
              <table className={tableCls}>
                <thead className={theadCls}>
                  <tr>
                    <th className={thCls}>{text.time}</th>
                    <th className={thCls}>{text.type}</th>
                    <th className={thCls}>{text.order}</th>
                    <th className={thCls}>{text.user}</th>
                    <th className={thCls}>{text.amount}</th>
                    <th className={thCls}>{text.status}</th>
                    <th className={thCls}>{text.payment}</th>
                    <th className={thCls}>{text.details}</th>
                    <th className={thCls}>{text.source}</th>
                  </tr>
                </thead>
                <tbody className={tbodyCls}>
                  {data.events.items.map((event) => {
                    const meta = PAYMENT_RISK_ACTION_META[event.action];
                    return (
                      <tr key={event.id} className={rowCls}>
                        <td className={tdMuted}>{formatCreatedAt(event.createdAt, locale)}</td>
                        <td className="whitespace-nowrap px-4 py-3 text-sm">
                          <span
                            className={`inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-xs font-semibold ${severityBadgeClass(meta.severity, isDark)}`}
                            title={meta.description[locale]}
                          >
                            <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: meta.color }} />
                            {meta.label[locale]}
                          </span>
                        </td>
                        <td className={tdCls}>{orderLink(event.order.id)}</td>
                        <td className={tdCls}>{userCell(event.order)}</td>
                        <td className={tdCls}>{amountText(event.order)}</td>
                        <td className="whitespace-nowrap px-4 py-3 text-sm">{statusBadge(event.order.status)}</td>
                        <td className={tdMuted}>{paymentText(event.order.paymentType)}</td>
                        <td
                          className={`max-w-md px-4 py-3 text-xs break-all ${isDark ? 'text-slate-300' : 'text-slate-600'}`}
                          title={event.detail ?? undefined}
                        >
                          {describeDetail(event.detail, locale)}
                        </td>
                        <td className={tdMuted}>{event.operator || '-'}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              {data.events.items.length === 0 && <div className={emptyCls}>{text.noEvents}</div>}
            </div>
            <div className="px-4 pb-2">
              <PaginationBar
                page={data.events.page}
                totalPages={data.events.totalPages}
                total={data.events.total}
                pageSize={pageSize}
                loading={loading}
                onPageChange={(p) => setPage(p)}
                onPageSizeChange={(s) => {
                  setPageSize(s);
                  setPage(1);
                }}
                locale={locale}
                isDark={isDark}
              />
            </div>
          </section>

          <section id={NO_TRADE_NO_SECTION_ID} className={sectionCls}>
            <div className="p-4 pb-2">
              <h3 className={sectionTitleCls}>
                {PAID_WITHOUT_TRADE_NO_META.label[locale]}
                <span className={`ml-2 font-normal ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                  {data.summary.paidWithoutTradeNo}
                </span>
              </h3>
              <p className={hintCls}>{PAID_WITHOUT_TRADE_NO_META.description[locale]}</p>
              {data.summary.paidWithoutTradeNo > data.paidWithoutTradeNo.length && (
                <p className={hintCls}>{text.showingFirst(data.paidWithoutTradeNo.length)}</p>
              )}
            </div>
            <div className="overflow-x-auto">
              <table className={tableCls}>
                <thead className={theadCls}>
                  <tr>
                    <th className={thCls}>{text.order}</th>
                    <th className={thCls}>{text.user}</th>
                    <th className={thCls}>{text.amount}</th>
                    <th className={thCls}>{text.status}</th>
                    <th className={thCls}>{text.payment}</th>
                    <th className={thCls}>{text.paidAt}</th>
                  </tr>
                </thead>
                <tbody className={tbodyCls}>
                  {data.paidWithoutTradeNo.map((order) => (
                    <tr key={order.id} className={rowCls}>
                      <td className={tdCls}>{orderLink(order.id)}</td>
                      <td className={tdCls}>{userCell(order)}</td>
                      <td className={tdCls}>{amountText(order)}</td>
                      <td className="whitespace-nowrap px-4 py-3 text-sm">{statusBadge(order.status)}</td>
                      <td className={tdMuted}>{paymentText(order.paymentType)}</td>
                      <td className={tdMuted}>{order.paidAt ? formatCreatedAt(order.paidAt, locale) : '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {data.paidWithoutTradeNo.length === 0 && <div className={emptyCls}>{text.none}</div>}
            </div>
          </section>

          <div className="grid gap-6 lg:grid-cols-2">
            <section id={DUPLICATES_SECTION_ID} className={sectionCls}>
              <div className="p-4 pb-2">
                <h3 className={sectionTitleCls}>{DUPLICATE_TRADE_NO_META.label[locale]}</h3>
                <p className={hintCls}>{DUPLICATE_TRADE_NO_META.description[locale]}</p>
                {data.duplicateTradeNos.length >= data.meta.limits.duplicateTradeNoGroups && (
                  <p className={hintCls}>{text.showingFirst(data.duplicateTradeNos.length)}</p>
                )}
              </div>
              <div className="overflow-x-auto">
                <table className={tableCls}>
                  <thead className={theadCls}>
                    <tr>
                      <th className={thCls}>{text.tradeNo}</th>
                      <th className={thCls}>{text.orderCount}</th>
                      <th className={thCls}>{text.order}</th>
                    </tr>
                  </thead>
                  <tbody className={tbodyCls}>
                    {data.duplicateTradeNos.map((group) => (
                      <tr key={group.tradeNo} className={rowCls}>
                        <td className={`px-4 py-3 text-sm break-all ${isDark ? 'text-slate-200' : 'text-slate-900'}`}>
                          {group.tradeNo}
                        </td>
                        <td className={tdCls}>{group.count}</td>
                        <td className="px-4 py-3 text-sm">
                          <div className="flex flex-wrap gap-x-3 gap-y-1">
                            {group.orders.map((order) => (
                              <span key={order.id}>{orderLink(order.id)}</span>
                            ))}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {data.duplicateTradeNos.length === 0 && <div className={emptyCls}>{text.none}</div>}
              </div>
            </section>

            <section className={sectionCls}>
              <div className="p-4 pb-2">
                <h3 className={sectionTitleCls}>{text.topUsers}</h3>
                <p className={hintCls}>{text.topUsersHint}</p>
              </div>
              <div className="overflow-x-auto">
                <table className={tableCls}>
                  <thead className={theadCls}>
                    <tr>
                      <th className={thCls}>{text.user}</th>
                      <th className={thCls}>{text.email}</th>
                      <th className={thCls}>{text.eventCount}</th>
                      <th className={thCls}>{text.orderCount}</th>
                      <th className={thCls}>{text.lastEvent}</th>
                    </tr>
                  </thead>
                  <tbody className={tbodyCls}>
                    {data.topUsers.map((user) => (
                      <tr key={user.userId} className={rowCls}>
                        <td className={tdCls}>{user.userName || `#${user.userId}`}</td>
                        <td className={tdMuted}>{user.userEmail || '-'}</td>
                        <td className={tdCls}>{user.eventCount}</td>
                        <td className={tdCls}>{user.orderCount}</td>
                        <td className={tdMuted}>{formatCreatedAt(user.lastEventAt, locale)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {data.topUsers.length === 0 && <div className={emptyCls}>{text.none}</div>}
              </div>
            </section>
          </div>
        </div>
      )}

      {detailOrder && (
        <OrderDetail order={detailOrder} onClose={() => setDetailOrder(null)} dark={isDark} locale={locale} />
      )}
    </PayPageLayout>
  );
}

function RiskPageFallback() {
  const searchParams = useSearchParams();
  const locale = resolveLocale(searchParams.get('lang'));

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-slate-500">{locale === 'en' ? 'Loading...' : '加载中...'}</div>
    </div>
  );
}

export default function RiskPage() {
  return (
    <Suspense fallback={<RiskPageFallback />}>
      <RiskContent />
    </Suspense>
  );
}
