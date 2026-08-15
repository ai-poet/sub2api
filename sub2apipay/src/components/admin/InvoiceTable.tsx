'use client';

import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';
import { formatInvoiceStatus, getInvoiceStatusBadgeClass } from '@/lib/pay-utils';

export interface AdminInvoice {
  id: string;
  orderId: string;
  userId: number;
  status: string;
  titleName: string;
  taxNo: string;
  remark: string | null;
  contactEmail: string | null;
  amount: number;
  fileName: string | null;
  hasFile: boolean;
  rejectReason: string | null;
  adminNote: string | null;
  issuedAt: string | null;
  notifiedAt: string | null;
  createdAt: string;
  downloadCount: number;
  order?: {
    id: string;
    payAmount: number | null;
    paymentType: string;
    status: string;
    paidAt: string | null;
    userEmail: string | null;
    userName: string | null;
  } | null;
}

interface InvoiceTableProps {
  isDark: boolean;
  locale: Locale;
  loading: boolean;
  error: string;
  invoices: AdminInvoice[];
  onIssue: (invoice: AdminInvoice) => void;
  onReject: (invoice: AdminInvoice) => void;
  onResendNotify: (invoice: AdminInvoice) => void;
  onDownload: (invoice: AdminInvoice) => void;
}

const GRID = 'md:grid-cols-[1.6fr_1fr_0.7fr_0.8fr_1fr_1.1fr]';

export default function InvoiceTable({
  isDark,
  locale,
  loading,
  error,
  invoices,
  onIssue,
  onReject,
  onResendNotify,
  onDownload,
}: InvoiceTableProps) {
  const text = {
    empty: pickLocaleText(locale, '暂无发票申请', 'No invoice requests'),
    title: pickLocaleText(locale, '抬头 / 税号', 'Title / Tax ID'),
    user: pickLocaleText(locale, '用户', 'User'),
    amount: pickLocaleText(locale, '金额', 'Amount'),
    status: pickLocaleText(locale, '状态', 'Status'),
    createdAt: pickLocaleText(locale, '申请时间', 'Requested at'),
    actions: pickLocaleText(locale, '操作', 'Actions'),
    issue: pickLocaleText(locale, '上传并开票', 'Upload & issue'),
    reject: pickLocaleText(locale, '驳回', 'Reject'),
    download: pickLocaleText(locale, '查看发票', 'View file'),
    resend: pickLocaleText(locale, '重发通知', 'Resend email'),
    notNotified: pickLocaleText(locale, '未通知', 'Not notified'),
    order: pickLocaleText(locale, '订单', 'Order'),
  };

  const formatTime = (value: string | null) =>
    value ? new Date(value).toLocaleString(locale === 'en' ? 'en-US' : 'zh-CN', { hour12: false }) : '-';

  const actionBtn = (tone: 'primary' | 'danger' | 'neutral') =>
    [
      'rounded px-2 py-1 text-xs',
      tone === 'primary'
        ? isDark
          ? 'bg-blue-500/20 text-blue-300 hover:bg-blue-500/30'
          : 'bg-blue-100 text-blue-700 hover:bg-blue-200'
        : tone === 'danger'
          ? isDark
            ? 'bg-red-500/20 text-red-300 hover:bg-red-500/30'
            : 'bg-red-100 text-red-700 hover:bg-red-200'
          : isDark
            ? 'bg-slate-700 text-slate-200 hover:bg-slate-600'
            : 'bg-slate-100 text-slate-700 hover:bg-slate-200',
    ].join(' ');

  return (
    <div
      className={[
        'rounded-2xl border p-3 sm:p-4',
        isDark ? 'border-slate-700 bg-slate-800/60' : 'border-slate-200 bg-slate-50/80',
      ].join(' ')}
    >
      {loading ? (
        <div className="flex items-center justify-center py-10">
          <div
            className={[
              'h-6 w-6 animate-spin rounded-full border-2 border-t-transparent',
              isDark ? 'border-slate-400' : 'border-slate-500',
            ].join(' ')}
          />
        </div>
      ) : error ? (
        <div
          className={[
            'rounded-xl border border-dashed px-4 py-10 text-center text-sm',
            isDark ? 'border-amber-500/40 text-amber-200' : 'border-amber-300 text-amber-700',
          ].join(' ')}
        >
          {error}
        </div>
      ) : invoices.length === 0 ? (
        <div
          className={[
            'rounded-xl border border-dashed px-4 py-10 text-center text-sm',
            isDark ? 'border-slate-600 text-slate-400' : 'border-slate-300 text-slate-500',
          ].join(' ')}
        >
          {text.empty}
        </div>
      ) : (
        <>
          <div
            className={[
              'hidden rounded-xl px-4 py-2 text-xs font-medium md:grid',
              GRID,
              isDark ? 'text-slate-300' : 'text-slate-600',
            ].join(' ')}
          >
            <span>{text.title}</span>
            <span>{text.user}</span>
            <span>{text.amount}</span>
            <span>{text.status}</span>
            <span>{text.createdAt}</span>
            <span>{text.actions}</span>
          </div>

          <div className="space-y-2 md:space-y-0">
            {invoices.map((invoice) => (
              <div
                key={invoice.id}
                className={[
                  'border-t px-4 py-3 first:border-t-0 md:grid md:items-center',
                  GRID,
                  isDark ? 'border-slate-700 text-slate-200' : 'border-slate-200 text-slate-700',
                ].join(' ')}
              >
                <div className="min-w-0">
                  <div className="truncate font-medium">{invoice.titleName}</div>
                  <div className={['truncate font-mono text-xs', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                    {invoice.taxNo}
                  </div>
                  {invoice.remark && (
                    <div className={['truncate text-xs', isDark ? 'text-slate-500' : 'text-slate-400'].join(' ')}>
                      {invoice.remark}
                    </div>
                  )}
                </div>

                <div className="min-w-0 text-xs">
                  <div className="truncate">{invoice.order?.userName || `#${invoice.userId}`}</div>
                  <div className={['truncate', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                    {invoice.contactEmail || invoice.order?.userEmail || '-'}
                  </div>
                  <div className={['truncate font-mono', isDark ? 'text-slate-500' : 'text-slate-400'].join(' ')}>
                    {text.order} #{invoice.orderId.slice(0, 10)}
                  </div>
                </div>

                <div className="font-semibold">¥{invoice.amount.toFixed(2)}</div>

                <div>
                  <span
                    className={['rounded-full px-2 py-0.5 text-xs', getInvoiceStatusBadgeClass(invoice.status, isDark)].join(' ')}
                  >
                    {formatInvoiceStatus(invoice.status, locale)}
                  </span>
                  {invoice.status === 'ISSUED' && !invoice.notifiedAt && (
                    <div className={['mt-1 text-xs', isDark ? 'text-amber-300' : 'text-amber-700'].join(' ')}>
                      {text.notNotified}
                    </div>
                  )}
                  {invoice.rejectReason && (
                    <div
                      title={invoice.rejectReason}
                      className={['mt-1 truncate text-xs', isDark ? 'text-red-300' : 'text-red-600'].join(' ')}
                    >
                      {invoice.rejectReason}
                    </div>
                  )}
                </div>

                <div className={['text-xs', isDark ? 'text-slate-300' : 'text-slate-600'].join(' ')}>
                  {formatTime(invoice.createdAt)}
                </div>

                <div className="mt-2 flex flex-wrap gap-1.5 md:mt-0">
                  {invoice.status === 'PENDING' && (
                    <>
                      <button type="button" onClick={() => onIssue(invoice)} className={actionBtn('primary')}>
                        {text.issue}
                      </button>
                      <button type="button" onClick={() => onReject(invoice)} className={actionBtn('danger')}>
                        {text.reject}
                      </button>
                    </>
                  )}
                  {invoice.hasFile && (
                    <button type="button" onClick={() => onDownload(invoice)} className={actionBtn('neutral')}>
                      {text.download}
                    </button>
                  )}
                  {invoice.status === 'ISSUED' && (
                    <button type="button" onClick={() => onResendNotify(invoice)} className={actionBtn('neutral')}>
                      {text.resend}
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
