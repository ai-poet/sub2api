'use client';

import { useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useState } from 'react';
import PaginationBar from '@/components/PaginationBar';
import PayPageLayout from '@/components/PayPageLayout';
import InvoiceIssueDialog from '@/components/admin/InvoiceIssueDialog';
import InvoiceTable, { type AdminInvoice } from '@/components/admin/InvoiceTable';
import { getAdminAccessHint } from '@/lib/branding';
import { pickLocaleText, resolveLocale } from '@/lib/locale';
import { buildAppApiPath } from '@/lib/public-path';

const STATUS_FILTERS = ['', 'PENDING', 'ISSUED', 'REJECTED', 'CANCELLED'] as const;

function AdminInvoicesContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token');
  const theme = searchParams.get('theme') === 'dark' ? 'dark' : 'light';
  const uiMode = searchParams.get('ui_mode') || 'standalone';
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = theme === 'dark';
  const isEmbedded = uiMode === 'embedded';

  const text = {
    title: pickLocaleText(locale, '发票管理', 'Invoices'),
    subtitle: pickLocaleText(locale, '增值税普通发票开具', 'VAT general invoices'),
    missingToken: pickLocaleText(locale, '缺少认证信息', 'Missing authentication'),
    missingTokenHint: getAdminAccessHint(locale),
    refresh: pickLocaleText(locale, '刷新', 'Refresh'),
    search: pickLocaleText(locale, '搜索抬头 / 税号 / 订单号', 'Search title, tax ID or order ID'),
    all: pickLocaleText(locale, '全部', 'All'),
    pending: pickLocaleText(locale, '待处理', 'Pending'),
    issued: pickLocaleText(locale, '已开具', 'Issued'),
    rejected: pickLocaleText(locale, '已驳回', 'Rejected'),
    cancelled: pickLocaleText(locale, '已取消', 'Cancelled'),
    rejectPrompt: pickLocaleText(locale, '请输入驳回原因', 'Enter the rejection reason'),
    actionFailed: pickLocaleText(locale, '操作失败', 'Action failed'),
    notifyResent: pickLocaleText(locale, '通知已重新发送', 'Notification resent'),
    settings: pickLocaleText(locale, '开票设置', 'Invoice settings'),
    enableInvoicing: pickLocaleText(locale, '开放在线开票', 'Enable online invoicing'),
    maxAgeDays: pickLocaleText(locale, '可开票期限（天，0 = 不限）', 'Invoicing window (days, 0 = unlimited)'),
    dailyLimit: pickLocaleText(locale, '每用户每日申请上限（0 = 不限）', 'Daily requests per user (0 = unlimited)'),
    minAmount: pickLocaleText(locale, '起开金额（元，0 = 不限）', 'Minimum amount (CNY, 0 = unlimited)'),
    minAmountHint: pickLocaleText(
      locale,
      '按开票合计金额判定，用户可勾选多张订单合并到这个金额。',
      'Checked against the invoice total; users can merge several orders to reach it.',
    ),
    save: pickLocaleText(locale, '保存设置', 'Save settings'),
    saved: pickLocaleText(locale, '设置已保存', 'Settings saved'),
    storageHint: pickLocaleText(
      locale,
      '发票文件存放于对象存储，需先在主站「系统设置 → 备份 → S3 配置」完成配置。',
      'Invoice files live in object storage; configure the backup S3 settings on the main site first.',
    ),
  };

  const statusLabel = (status: string) => {
    switch (status) {
      case 'PENDING':
        return text.pending;
      case 'ISSUED':
        return text.issued;
      case 'REJECTED':
        return text.rejected;
      case 'CANCELLED':
        return text.cancelled;
      default:
        return text.all;
    }
  };

  const [invoices, setInvoices] = useState<AdminInvoice[]>([]);
  const [statusCounts, setStatusCounts] = useState<Record<string, number>>({});
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  // 默认停在待处理：管理员打开这个页面就是来处理积压的。
  const [statusFilter, setStatusFilter] = useState<string>('PENDING');
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [issueTarget, setIssueTarget] = useState<AdminInvoice | null>(null);
  const [issuing, setIssuing] = useState(false);
  const [issueError, setIssueError] = useState('');

  const [showSettings, setShowSettings] = useState(false);
  const [cfgEnabled, setCfgEnabled] = useState(false);
  const [cfgMaxAgeDays, setCfgMaxAgeDays] = useState('180');
  const [cfgDailyLimit, setCfgDailyLimit] = useState('20');
  const [cfgMinAmount, setCfgMinAmount] = useState('100');
  const [savingSettings, setSavingSettings] = useState(false);

  const authHeaders = useCallback(
    (extra?: Record<string, string>) => ({ Authorization: `Bearer ${token ?? ''}`, ...extra }),
    [token],
  );

  const fetchInvoices = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
      if (statusFilter) params.set('status', statusFilter);
      if (keyword.trim()) params.set('q', keyword.trim());
      if (locale === 'en') params.set('lang', 'en');

      const res = await fetch(buildAppApiPath(`/api/admin/invoices?${params}`), { headers: authHeaders() });
      if (!res.ok) {
        setError(
          res.status === 401
            ? pickLocaleText(locale, '登录已过期，请重新进入', 'Session expired, please re-enter')
            : pickLocaleText(locale, '加载发票列表失败', 'Failed to load invoices'),
        );
        setInvoices([]);
        return;
      }
      const data = await res.json();
      setInvoices(data.invoices ?? []);
      setStatusCounts(data.status_counts ?? {});
      setTotal(data.total ?? 0);
      setTotalPages(data.total_pages ?? 1);
    } catch {
      setError(pickLocaleText(locale, '加载发票列表失败', 'Failed to load invoices'));
    } finally {
      setLoading(false);
    }
  }, [token, page, pageSize, statusFilter, keyword, locale, authHeaders]);

  useEffect(() => {
    fetchInvoices();
  }, [fetchInvoices]);

  // 开票开关等设置存在 SystemConfig 里，和支付配置走同一套接口。
  useEffect(() => {
    if (!token) return;
    fetch(buildAppApiPath('/api/admin/config'), { headers: { Authorization: `Bearer ${token}` } })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        const configs: { key: string; value: string }[] = data?.configs ?? [];
        const find = (key: string) => configs.find((c) => c.key === key)?.value;
        setCfgEnabled(find('invoice_enabled') === 'true');
        setCfgMaxAgeDays(find('invoice_max_age_days') ?? '180');
        setCfgDailyLimit(find('invoice_daily_request_limit') ?? '20');
        setCfgMinAmount(find('invoice_min_amount') ?? '100');
      })
      .catch(() => {});
  }, [token]);

  const saveSettings = async () => {
    setSavingSettings(true);
    try {
      const res = await fetch(buildAppApiPath('/api/admin/config'), {
        method: 'PUT',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          configs: [
            { key: 'invoice_enabled', value: cfgEnabled ? 'true' : 'false', group: 'invoice', label: '开放在线开票' },
            { key: 'invoice_max_age_days', value: cfgMaxAgeDays || '0', group: 'invoice', label: '可开票期限（天）' },
            {
              key: 'invoice_daily_request_limit',
              value: cfgDailyLimit || '0',
              group: 'invoice',
              label: '每用户每日申请上限',
            },
            { key: 'invoice_min_amount', value: cfgMinAmount || '0', group: 'invoice', label: '起开金额（元）' },
          ],
        }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || text.actionFailed);
        return;
      }
      setNotice(text.saved);
      await fetchInvoices();
    } catch {
      setError(text.actionFailed);
    } finally {
      setSavingSettings(false);
    }
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

  const langQuery = locale === 'en' ? '?lang=en' : '';

  const handleIssue = async (file: File, adminNote: string) => {
    if (!issueTarget) return;
    setIssuing(true);
    setIssueError('');
    try {
      const form = new FormData();
      form.append('file', file);
      if (adminNote) form.append('admin_note', adminNote);

      const res = await fetch(buildAppApiPath(`/api/admin/invoices/${issueTarget.id}/issue${langQuery}`), {
        method: 'POST',
        headers: authHeaders(),
        body: form,
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setIssueError(data.error || text.actionFailed);
        return;
      }
      setIssueTarget(null);
      // 邮件失败不影响开票成功，但要让管理员看见，以便手动重发。
      setNotice(data.warning || '');
      await fetchInvoices();
    } catch {
      setIssueError(text.actionFailed);
    } finally {
      setIssuing(false);
    }
  };

  const handleReject = async (invoice: AdminInvoice) => {
    const reason = window.prompt(text.rejectPrompt);
    if (!reason?.trim()) return;
    try {
      const res = await fetch(buildAppApiPath(`/api/admin/invoices/${invoice.id}/reject${langQuery}`), {
        method: 'POST',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify({ reason: reason.trim() }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || text.actionFailed);
        return;
      }
      await fetchInvoices();
    } catch {
      setError(text.actionFailed);
    }
  };

  const handleResendNotify = async (invoice: AdminInvoice) => {
    try {
      const res = await fetch(buildAppApiPath(`/api/admin/invoices/${invoice.id}/notify${langQuery}`), {
        method: 'POST',
        headers: authHeaders(),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error || text.actionFailed);
        return;
      }
      setNotice(data.warning || text.notifyResent);
      await fetchInvoices();
    } catch {
      setError(text.actionFailed);
    }
  };

  const handleDownload = (invoice: AdminInvoice) => {
    // 管理员预览：后端 302 到预签名链接。token 走 query，因为这是整页导航。
    const params = new URLSearchParams({ token });
    if (locale === 'en') params.set('lang', 'en');
    window.open(buildAppApiPath(`/api/admin/invoices/${invoice.id}/download?${params}`), '_blank');
  };

  const filterBtnClass = (active: boolean) =>
    [
      'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
      active
        ? isDark
          ? 'bg-indigo-500/30 text-indigo-100'
          : 'bg-indigo-100 text-indigo-700'
        : isDark
          ? 'text-slate-400 hover:bg-slate-800'
          : 'text-slate-500 hover:bg-slate-100',
    ].join(' ');

  return (
    <PayPageLayout isDark={isDark} isEmbedded={isEmbedded} title={text.title} subtitle={text.subtitle}>
      <div className="mb-4 flex flex-wrap items-center gap-2">
        {STATUS_FILTERS.map((status) => (
          <button
            key={status || 'ALL'}
            type="button"
            onClick={() => {
              setStatusFilter(status);
              setPage(1);
            }}
            className={filterBtnClass(statusFilter === status)}
          >
            {statusLabel(status)}
            {status && statusCounts[status] ? ` (${statusCounts[status]})` : ''}
          </button>
        ))}

        <input
          type="search"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              setPage(1);
              fetchInvoices();
            }
          }}
          placeholder={text.search}
          className={[
            'ml-auto w-56 rounded-lg border px-3 py-1.5 text-xs focus:border-blue-500 focus:outline-none',
            isDark ? 'border-slate-600 bg-slate-800 text-slate-100' : 'border-slate-300 bg-white text-slate-900',
          ].join(' ')}
        />
        <button
          type="button"
          onClick={() => fetchInvoices()}
          className={[
            'rounded-lg border px-3 py-1.5 text-xs font-medium',
            isDark ? 'border-slate-600 text-slate-200 hover:bg-slate-800' : 'border-slate-300 text-slate-700 hover:bg-slate-100',
          ].join(' ')}
        >
          {text.refresh}
        </button>
        <button
          type="button"
          onClick={() => setShowSettings((v) => !v)}
          className={[
            'rounded-lg border px-3 py-1.5 text-xs font-medium',
            isDark ? 'border-slate-600 text-slate-200 hover:bg-slate-800' : 'border-slate-300 text-slate-700 hover:bg-slate-100',
          ].join(' ')}
        >
          {text.settings}
        </button>
      </div>

      {showSettings && (
        <div
          className={[
            'mb-4 rounded-xl border p-4',
            isDark ? 'border-slate-700 bg-slate-800/60' : 'border-slate-200 bg-slate-50/80',
          ].join(' ')}
        >
          <div className="flex flex-wrap items-end gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={cfgEnabled} onChange={(e) => setCfgEnabled(e.target.checked)} />
              <span className={isDark ? 'text-slate-200' : 'text-slate-700'}>{text.enableInvoicing}</span>
            </label>

            <label className="text-xs">
              <span className={['mb-1 block', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                {text.maxAgeDays}
              </span>
              <input
                type="number"
                min="0"
                value={cfgMaxAgeDays}
                onChange={(e) => setCfgMaxAgeDays(e.target.value)}
                className={[
                  'w-28 rounded-lg border px-2 py-1 focus:border-blue-500 focus:outline-none',
                  isDark ? 'border-slate-600 bg-slate-800 text-slate-100' : 'border-slate-300 bg-white text-slate-900',
                ].join(' ')}
              />
            </label>

            <label className="text-xs">
              <span className={['mb-1 block', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                {text.dailyLimit}
              </span>
              <input
                type="number"
                min="0"
                value={cfgDailyLimit}
                onChange={(e) => setCfgDailyLimit(e.target.value)}
                className={[
                  'w-28 rounded-lg border px-2 py-1 focus:border-blue-500 focus:outline-none',
                  isDark ? 'border-slate-600 bg-slate-800 text-slate-100' : 'border-slate-300 bg-white text-slate-900',
                ].join(' ')}
              />
            </label>

            <label className="text-xs">
              <span className={['mb-1 block', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                {text.minAmount}
              </span>
              <input
                type="number"
                min="0"
                step="0.01"
                value={cfgMinAmount}
                onChange={(e) => setCfgMinAmount(e.target.value)}
                title={text.minAmountHint}
                className={[
                  'w-28 rounded-lg border px-2 py-1 focus:border-blue-500 focus:outline-none',
                  isDark ? 'border-slate-600 bg-slate-800 text-slate-100' : 'border-slate-300 bg-white text-slate-900',
                ].join(' ')}
              />
            </label>

            <button
              type="button"
              onClick={saveSettings}
              disabled={savingSettings}
              className={[
                'rounded-lg px-3 py-1.5 text-xs font-medium text-white disabled:cursor-not-allowed',
                isDark ? 'bg-blue-600/90 hover:bg-blue-700 disabled:bg-slate-700' : 'bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300',
              ].join(' ')}
            >
              {text.save}
            </button>
          </div>
          <p className={['mt-3 text-xs', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>{text.storageHint}</p>
        </div>
      )}

      {notice && (
        <div
          className={[
            'mb-3 rounded-lg border px-3 py-2 text-xs',
            isDark ? 'border-amber-500/40 bg-amber-500/10 text-amber-200' : 'border-amber-300 bg-amber-50 text-amber-700',
          ].join(' ')}
        >
          {notice}
        </div>
      )}

      <InvoiceTable
        isDark={isDark}
        locale={locale}
        loading={loading}
        error={error}
        invoices={invoices}
        onIssue={(invoice) => {
          setIssueError('');
          setIssueTarget(invoice);
        }}
        onReject={handleReject}
        onResendNotify={handleResendNotify}
        onDownload={handleDownload}
      />

      <PaginationBar
        page={page}
        totalPages={totalPages}
        total={total}
        pageSize={pageSize}
        pageSizeOptions={[20, 50, 100]}
        locale={locale}
        isDark={isDark}
        loading={loading}
        onPageChange={setPage}
        onPageSizeChange={(size) => {
          setPageSize(size);
          setPage(1);
        }}
      />

      {issueTarget && (
        <InvoiceIssueDialog
          isDark={isDark}
          locale={locale}
          invoice={issueTarget}
          submitting={issuing}
          error={issueError}
          onSubmit={handleIssue}
          onClose={() => {
            if (issuing) return;
            setIssueTarget(null);
            setIssueError('');
          }}
        />
      )}
    </PayPageLayout>
  );
}

export default function AdminInvoicesPage() {
  return (
    <Suspense fallback={null}>
      <AdminInvoicesContent />
    </Suspense>
  );
}
