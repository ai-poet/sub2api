'use client';

import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend } from 'recharts';
import type { Locale } from '@/lib/locale';
import {
  PAYMENT_RISK_ACTIONS,
  PAYMENT_RISK_ACTION_META,
  type PaymentRiskDailyPoint,
} from '@/lib/payment-risk/shared';

interface RiskDailyChartProps {
  data: PaymentRiskDailyPoint[];
  dark?: boolean;
  locale?: Locale;
}

interface TooltipEntry {
  dataKey?: string | number;
  name?: string;
  value?: number;
  color?: string;
}

function formatDate(dateStr: string) {
  const [, m, d] = dateStr.split('-');
  return `${m}/${d}`;
}

function RiskTooltip({
  active,
  payload,
  label,
  dark,
  totalLabel,
}: {
  active?: boolean;
  payload?: TooltipEntry[];
  label?: string;
  dark?: boolean;
  totalLabel: string;
}) {
  if (!active || !payload?.length) return null;
  const nonZero = payload.filter((entry) => (entry.value ?? 0) > 0);
  const total = nonZero.reduce((sum, entry) => sum + (entry.value ?? 0), 0);
  return (
    <div
      className={[
        'rounded-lg border px-3 py-2 text-sm shadow-lg',
        dark ? 'border-slate-600 bg-slate-800 text-slate-200' : 'border-slate-200 bg-white text-slate-800',
      ].join(' ')}
    >
      <p className={['mb-1 text-xs', dark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>{label}</p>
      {nonZero.map((entry) => (
        <p key={String(entry.dataKey)} className="flex items-center gap-2">
          <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: entry.color }} />
          {entry.name}: {entry.value}
        </p>
      ))}
      <p className={['mt-1 text-xs', dark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
        {totalLabel}: {total}
      </p>
    </div>
  );
}

export default function RiskDailyChart({ data, dark, locale = 'zh' }: RiskDailyChartProps) {
  const title = locale === 'en' ? 'Daily Risk Events' : '每日风险事件';
  const emptyText = locale === 'en' ? 'No risk events in this period' : '这段时间没有风险事件';
  const totalLabel = locale === 'en' ? 'Total' : '合计';
  const hasEvents = data.some((point) => PAYMENT_RISK_ACTIONS.some((action) => point[action] > 0));
  const axisColor = dark ? '#64748b' : '#94a3b8';
  const gridColor = dark ? '#334155' : '#e2e8f0';
  const tickInterval = data.length > 30 ? Math.ceil(data.length / 12) - 1 : 0;

  return (
    <div
      className={[
        'rounded-xl border p-6',
        dark ? 'border-slate-700 bg-slate-800/60' : 'border-slate-200 bg-white shadow-sm',
      ].join(' ')}
    >
      <h3 className={['mb-4 text-sm font-semibold', dark ? 'text-slate-200' : 'text-slate-800'].join(' ')}>{title}</h3>
      {hasEvents ? (
        <ResponsiveContainer width="100%" height={280}>
          <BarChart data={data} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
            <CartesianGrid stroke={gridColor} strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="date"
              tickFormatter={formatDate}
              tick={{ fill: axisColor, fontSize: 12 }}
              axisLine={{ stroke: gridColor }}
              tickLine={false}
              interval={tickInterval}
            />
            <YAxis
              allowDecimals={false}
              tick={{ fill: axisColor, fontSize: 12 }}
              axisLine={{ stroke: gridColor }}
              tickLine={false}
              width={40}
            />
            <Tooltip
              cursor={{ fill: dark ? 'rgba(148, 163, 184, 0.08)' : 'rgba(148, 163, 184, 0.15)' }}
              content={<RiskTooltip dark={dark} totalLabel={totalLabel} />}
            />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            {PAYMENT_RISK_ACTIONS.map((action) => (
              <Bar
                key={action}
                dataKey={action}
                name={PAYMENT_RISK_ACTION_META[action].label[locale]}
                stackId="risk"
                fill={PAYMENT_RISK_ACTION_META[action].color}
                maxBarSize={28}
              />
            ))}
          </BarChart>
        </ResponsiveContainer>
      ) : (
        <p className={['py-16 text-center text-sm', dark ? 'text-slate-500' : 'text-gray-400'].join(' ')}>{emptyText}</p>
      )}
    </div>
  );
}
