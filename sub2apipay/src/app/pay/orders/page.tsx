'use client';

import { useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import PayPageLayout from '@/components/PayPageLayout';
import OrderFilterBar from '@/components/OrderFilterBar';
import OrderSummaryCards from '@/components/OrderSummaryCards';
import OrderTable from '@/components/OrderTable';
import InvoiceRequestDialog, {
  type InvoiceRequestPayload,
  type SavedInvoiceTitleOption,
} from '@/components/InvoiceRequestDialog';
import PaginationBar from '@/components/PaginationBar';
import { getOrdersAccessHint, getOrdersSessionExpiredHint } from '@/lib/branding';
import { applyLocaleToSearchParams, pickLocaleText, resolveLocale } from '@/lib/locale';
import { detectDeviceIsMobile, type UserInfo, type MyOrder, type OrderStatusFilter } from '@/lib/pay-utils';
import { buildAppApiPath } from '@/lib/public-path';

const PAGE_SIZE_OPTIONS = [20, 50, 100];

interface Summary {
  total: number;
  pending: number;
  completed: number;
  failed: number;
}

function OrdersContent() {
  const searchParams = useSearchParams();
  const token = (searchParams.get('token') || '').trim();
  const theme = searchParams.get('theme') === 'dark' ? 'dark' : 'light';
  const uiMode = searchParams.get('ui_mode') || 'standalone';
  const srcHost = searchParams.get('src_host') || '';
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = theme === 'dark';

  const text = {
    missingAuth: pickLocaleText(locale, '缺少认证信息', 'Missing authentication information'),
    visitOrders: getOrdersAccessHint(locale),
    sessionExpired: getOrdersSessionExpiredHint(locale),
    loadFailed: pickLocaleText(locale, '订单加载失败，请稍后重试。', 'Failed to load orders. Please try again later.'),
    networkError: pickLocaleText(locale, '网络错误，请稍后重试。', 'Network error. Please try again later.'),
    switchingMobileTab: pickLocaleText(locale, '正在切换到移动端订单 Tab...', 'Switching to mobile orders tab...'),
    myOrders: pickLocaleText(locale, '我的订单', 'My Orders'),
    refresh: pickLocaleText(locale, '刷新', 'Refresh'),
    selectAllInvoiceable: pickLocaleText(locale, '全选可开票', 'Select all invoiceable'),
    clearSelection: pickLocaleText(locale, '清空选择', 'Clear selection'),
    quickInvoice: pickLocaleText(locale, '一键开票', 'One-click invoice'),
    mergeInvoice: pickLocaleText(locale, '填写抬头开票', 'Enter title & invoice'),
    invoiceSelectionHint: pickLocaleText(
      locale,
      '勾选订单可合并为一张发票',
      'Select orders to merge them into one invoice',
    ),
    invoiceBelowMin: (short: number, min: number) =>
      pickLocaleText(
        locale,
        `还差 ¥${short.toFixed(2)} 满 ¥${min.toFixed(2)} 起开，可多勾几张订单`,
        `¥${short.toFixed(2)} short of the ¥${min.toFixed(2)} minimum — select more orders`,
      ),
    backToPay: pickLocaleText(locale, '返回充值/订单', 'Back to Purchase'),
    loading: pickLocaleText(locale, '加载中...', 'Loading...'),
    userPrefix: pickLocaleText(locale, '用户', 'User'),
    authError: pickLocaleText(
      locale,
      '缺少认证信息，请从主系统正确进入订单页面',
      'Missing authentication information. Please open the orders page from the main system.',
    ),
    refundRequestFailed: pickLocaleText(locale, '退款申请失败，请稍后重试。', 'Refund request failed. Please try again later.'),
    invoiceRequestFailed: pickLocaleText(locale, '开票申请失败，请稍后重试。', 'Invoice request failed. Please try again later.'),
    invoiceDownloadFailed: pickLocaleText(locale, '发票下载失败，请稍后重试。', 'Invoice download failed. Please try again later.'),
  };

  const [isIframeContext, setIsIframeContext] = useState(true);
  const [isMobile, setIsMobile] = useState(false);
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [orders, setOrders] = useState<MyOrder[]>([]);
  const [summary, setSummary] = useState<Summary>({ total: 0, pending: 0, completed: 0, failed: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeFilter, setActiveFilter] = useState<OrderStatusFilter>('ALL');
  const [resolvedUserId, setResolvedUserId] = useState<number | null>(null);

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [totalPages, setTotalPages] = useState(1);

  const [invoiceEnabled, setInvoiceEnabled] = useState(false);
  const [invoiceMinAmount, setInvoiceMinAmount] = useState(0);
  /** 开票对话框覆盖的订单；单张开票时长度为 1，合并开票时为勾选集合。 */
  const [invoiceTargets, setInvoiceTargets] = useState<MyOrder[] | null>(null);
  const [savedTitles, setSavedTitles] = useState<SavedInvoiceTitleOption[]>([]);
  const [invoiceSubmitting, setInvoiceSubmitting] = useState(false);
  const [invoiceError, setInvoiceError] = useState('');
  const [selectedInvoiceIds, setSelectedInvoiceIds] = useState<Set<string>>(new Set());
  const [quickInvoiceError, setQuickInvoiceError] = useState('');

  const isEmbedded = uiMode === 'embedded' && isIframeContext;
  const hasToken = token.length > 0;

  useEffect(() => {
    if (typeof window === 'undefined') return;
    setIsIframeContext(window.self !== window.top);
    setIsMobile(detectDeviceIsMobile());
  }, []);

  useEffect(() => {
    if (!isMobile || isEmbedded || typeof window === 'undefined') return;
    const params = new URLSearchParams();
    if (token) params.set('token', token);
    params.set('theme', theme);
    params.set('ui_mode', uiMode);
    params.set('tab', 'orders');
    applyLocaleToSearchParams(params, locale);
    window.location.replace(`/pay?${params.toString()}`);
  }, [isMobile, isEmbedded, token, theme, uiMode, locale]);

  const loadOrders = async (targetPage = page, targetPageSize = pageSize) => {
    setLoading(true);
    setError('');
    try {
      if (!hasToken) {
        setOrders([]);
        setError(text.authError);
        return;
      }

      const params = new URLSearchParams({
        token,
        page: String(targetPage),
        page_size: String(targetPageSize),
      });
      const res = await fetch(buildAppApiPath(`/api/orders/my?${params}`));
      if (!res.ok) {
        setError(res.status === 401 ? text.sessionExpired : text.loadFailed);
        setOrders([]);
        return;
      }

      const data = await res.json();
      const meUser = data.user || {};
      const meId = Number(meUser.id);
      if (Number.isInteger(meId) && meId > 0) setResolvedUserId(meId);

      setUserInfo({
        id: Number.isInteger(meId) && meId > 0 ? meId : undefined,
        username:
          (typeof meUser.displayName === 'string' && meUser.displayName.trim()) ||
          (typeof meUser.username === 'string' && meUser.username.trim()) ||
          `${text.userPrefix} #${meId}`,
        balance: typeof meUser.balance === 'number' ? meUser.balance : 0,
      });

      setOrders(Array.isArray(data.orders) ? data.orders : []);
      setSummary(data.summary ?? { total: 0, pending: 0, completed: 0, failed: 0 });
      setPage(data.page ?? targetPage);
      setTotalPages(data.total_pages ?? 1);
      setInvoiceEnabled(!!data.invoiceEnabled);
      setInvoiceMinAmount(typeof data.invoiceMinAmount === 'number' ? data.invoiceMinAmount : 0);
    } catch {
      setOrders([]);
      setError(text.networkError);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isMobile && !isEmbedded) return;
    loadOrders(1, pageSize);
  }, [token, isMobile, isEmbedded]);

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
    loadOrders(newPage, pageSize);
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setPage(1);
    loadOrders(1, newSize);
  };

  const handleRefundRequest = async (orderId: string, amount: number, reason: string) => {
    const params = new URLSearchParams({ token });
    applyLocaleToSearchParams(params, locale);
    const res = await fetch(buildAppApiPath(`/api/orders/${orderId}/refund-request?${params.toString()}`), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount, reason }),
    });

    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error || text.refundRequestFailed);
    }

    await loadOrders(page, pageSize);
  };

  const fetchSavedTitles = async (): Promise<SavedInvoiceTitleOption[]> => {
    try {
      const params = new URLSearchParams({ token, page: '1', page_size: '20' });
      applyLocaleToSearchParams(params, locale);
      const res = await fetch(buildAppApiPath(`/api/invoices/my?${params}`));
      if (!res.ok) return [];
      const data = await res.json();
      return Array.isArray(data.titles) ? data.titles : [];
    } catch {
      // 回填失败不阻塞开票，用户手填即可。
      return [];
    }
  };

  // 一键开票要用最近一次抬头直接提交，所以抬头必须在用户点按钮之前就在手上。
  useEffect(() => {
    if (!invoiceEnabled || !hasToken) return;
    let cancelled = false;
    fetchSavedTitles().then((titles) => {
      if (!cancelled) setSavedTitles(titles);
    });
    return () => {
      cancelled = true;
    };
  }, [invoiceEnabled, hasToken, token, locale]);

  const defaultTitle = savedTitles[0] ?? null;

  // 抬头记忆：先取回已存抬头再挂载对话框，让对话框能在初始化时直接回填。
  const openInvoiceDialog = async (targets: MyOrder[]) => {
    if (targets.length === 0) return;
    setInvoiceError('');
    setQuickInvoiceError('');
    const titles = await fetchSavedTitles();
    setSavedTitles(titles);
    setInvoiceTargets(targets);
  };

  /** 提交开票。单张与合并走同一个批量接口，避免两条路径的行为漂移。 */
  const submitInvoiceRequest = async (
    orderIds: string[],
    payload: InvoiceRequestPayload,
  ): Promise<string | null> => {
    const params = new URLSearchParams({ token });
    applyLocaleToSearchParams(params, locale);
    const res = await fetch(buildAppApiPath(`/api/invoices/requests?${params}`), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        order_ids: orderIds,
        title_name: payload.titleName,
        tax_no: payload.taxNo,
        ...(payload.remark && { remark: payload.remark }),
        ...(payload.contactEmail && { contact_email: payload.contactEmail }),
      }),
    });

    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      return data.error || text.invoiceRequestFailed;
    }
    return null;
  };

  const handleInvoiceRequest = async (payload: InvoiceRequestPayload) => {
    if (!invoiceTargets?.length) return;
    setInvoiceSubmitting(true);
    setInvoiceError('');
    try {
      const failure = await submitInvoiceRequest(
        invoiceTargets.map((order) => order.id),
        payload,
      );
      if (failure) {
        setInvoiceError(failure);
        return;
      }

      setInvoiceTargets(null);
      setSelectedInvoiceIds(new Set());
      await loadOrders(page, pageSize);
    } catch {
      setInvoiceError(text.invoiceRequestFailed);
    } finally {
      setInvoiceSubmitting(false);
    }
  };

  /**
   * 一键开票：直接用最近一次使用的抬头提交，不弹对话框。
   * 没有历史抬头时退回到对话框——首次开票总得有人把抬头敲进去。
   */
  const handleQuickInvoice = async (targets: MyOrder[]) => {
    if (targets.length === 0) return;
    if (!defaultTitle) {
      await openInvoiceDialog(targets);
      return;
    }

    setInvoiceSubmitting(true);
    setQuickInvoiceError('');
    try {
      const failure = await submitInvoiceRequest(
        targets.map((order) => order.id),
        {
          titleName: defaultTitle.titleName,
          taxNo: defaultTitle.taxNo,
          remark: defaultTitle.remark ?? '',
          contactEmail: defaultTitle.contactEmail ?? '',
        },
      );
      if (failure) {
        setQuickInvoiceError(failure);
        return;
      }

      setSelectedInvoiceIds(new Set());
      await loadOrders(page, pageSize);
    } catch {
      setQuickInvoiceError(text.invoiceRequestFailed);
    } finally {
      setInvoiceSubmitting(false);
    }
  };

  const toggleInvoiceSelect = (orderId: string) => {
    setQuickInvoiceError('');
    setSelectedInvoiceIds((prev) => {
      const next = new Set(prev);
      if (next.has(orderId)) next.delete(orderId);
      else next.add(orderId);
      return next;
    });
  };

  const handleInvoiceDownload = (invoiceId: string) => {
    const params = new URLSearchParams({ token });
    applyLocaleToSearchParams(params, locale);
    // 后端 302 到预签名链接，且响应头带 attachment disposition，同页导航即触发下载。
    window.location.href = buildAppApiPath(`/api/invoices/${invoiceId}/download?${params}`);
  };

  const filteredOrders = activeFilter === 'ALL' ? orders : orders.filter((o) => o.status === activeFilter);

  // 只在当前筛选结果内做全选/合计：勾选的是用户此刻看得见的行。
  const invoiceableOrders = invoiceEnabled ? filteredOrders.filter((order) => order.canRequestInvoice) : [];
  const selectedOrders = invoiceableOrders.filter((order) => selectedInvoiceIds.has(order.id));
  const selectedTotal = selectedOrders.reduce((sum, order) => sum + (order.payAmount ?? 0), 0);
  const allInvoiceableSelected =
    invoiceableOrders.length > 0 && invoiceableOrders.every((order) => selectedInvoiceIds.has(order.id));
  // 起开金额按合计判定，所以差额提示挂在勾选集合上，而不是单张订单上。
  const belowMinAmount = invoiceMinAmount > 0 && selectedTotal < invoiceMinAmount;
  const canSubmitInvoice = selectedOrders.length > 0 && !belowMinAmount && !invoiceSubmitting;

  const toggleSelectAllInvoiceable = () => {
    setQuickInvoiceError('');
    setSelectedInvoiceIds(allInvoiceableSelected ? new Set() : new Set(invoiceableOrders.map((order) => order.id)));
  };

  const btnClass = [
    'inline-flex items-center rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
    isDark ? 'border-slate-600 text-slate-200 hover:bg-slate-800' : 'border-slate-300 text-slate-700 hover:bg-slate-100',
  ].join(' ');

  if (isMobile) {
    return (
      <div className={`flex min-h-screen items-center justify-center p-4 ${isDark ? 'bg-slate-950 text-slate-100' : 'bg-slate-50 text-slate-900'}`}>
        {text.switchingMobileTab}
      </div>
    );
  }

  if (!hasToken && !resolvedUserId) {
    return (
      <div className={`flex min-h-screen items-center justify-center p-4 ${isDark ? 'bg-slate-950' : 'bg-slate-50'}`}>
        <div className="text-center text-red-500">
          <p className="text-lg font-medium">{text.missingAuth}</p>
          <p className={`mt-2 text-sm ${isDark ? 'text-slate-400' : 'text-gray-500'}`}>{text.visitOrders}</p>
        </div>
      </div>
    );
  }

  const buildScopedUrl = (path: string) => {
    const params = new URLSearchParams();
    if (token) params.set('token', token);
    params.set('theme', theme);
    params.set('ui_mode', uiMode);
    if (srcHost) params.set('src_host', srcHost);
    applyLocaleToSearchParams(params, locale);
    const srcUrl = searchParams.get('src_url') || '';
    if (srcUrl) params.set('src_url', srcUrl);
    return `${path}?${params.toString()}`;
  };

  return (
    <PayPageLayout
      isDark={isDark}
      isEmbedded={isEmbedded}
      title={text.myOrders}
      subtitle={userInfo?.username || text.myOrders}
      leadingAction={
        <a href={buildScopedUrl('/pay')} className={btnClass}>
          {'<'} {text.backToPay}
        </a>
      }
      actions={
        <>
          <button type="button" onClick={() => loadOrders(page, pageSize)} className={btnClass}>
            {text.refresh}
          </button>
        </>
      }
    >
      <OrderSummaryCards isDark={isDark} locale={locale} summary={summary} />

      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <OrderFilterBar isDark={isDark} locale={locale} activeFilter={activeFilter} onChange={setActiveFilter} />
      </div>

      {invoiceEnabled && invoiceableOrders.length > 0 && (
        <div
          className={[
            'mb-3 flex flex-wrap items-center gap-3 rounded-xl border px-3 py-2 text-xs',
            isDark ? 'border-slate-700 bg-slate-800/60 text-slate-300' : 'border-slate-200 bg-slate-50 text-slate-600',
          ].join(' ')}
        >
          <label className="inline-flex cursor-pointer items-center gap-2">
            <input
              type="checkbox"
              className="h-4 w-4 cursor-pointer accent-blue-600"
              checked={allInvoiceableSelected}
              onChange={toggleSelectAllInvoiceable}
            />
            {text.selectAllInvoiceable}
          </label>

          {selectedOrders.length > 0 ? (
            <span className={isDark ? 'text-slate-200' : 'text-slate-800'}>
              {selectedOrders.length} · ¥{selectedTotal.toFixed(2)}
            </span>
          ) : (
            <span>{text.invoiceSelectionHint}</span>
          )}

          {belowMinAmount && selectedOrders.length > 0 && (
            <span className={isDark ? 'text-amber-300' : 'text-amber-700'}>
              {text.invoiceBelowMin(invoiceMinAmount - selectedTotal, invoiceMinAmount)}
            </span>
          )}

          <div className="ml-auto flex flex-wrap items-center gap-2">
            {selectedOrders.length > 0 && (
              <button type="button" onClick={() => setSelectedInvoiceIds(new Set())} className={btnClass}>
                {text.clearSelection}
              </button>
            )}
            <button
              type="button"
              disabled={!canSubmitInvoice}
              onClick={() => openInvoiceDialog(selectedOrders)}
              className={`${btnClass} disabled:cursor-not-allowed disabled:opacity-50`}
            >
              {text.mergeInvoice}
            </button>
            <button
              type="button"
              disabled={!canSubmitInvoice}
              onClick={() => handleQuickInvoice(selectedOrders)}
              // 一键开票是这一排的主操作：用实心样式跟旁边的次要按钮区分开。
              className={[
                'inline-flex items-center rounded-lg border border-blue-500 bg-blue-500 px-3 py-1.5 text-xs font-semibold text-white transition-colors',
                'hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50',
              ].join(' ')}
              title={defaultTitle ? `${defaultTitle.titleName} · ${defaultTitle.taxNo}` : undefined}
            >
              {invoiceSubmitting ? '...' : text.quickInvoice}
            </button>
          </div>

          {quickInvoiceError && (
            <div className={['w-full', isDark ? 'text-red-400' : 'text-red-600'].join(' ')}>{quickInvoiceError}</div>
          )}
        </div>
      )}

      <OrderTable
        isDark={isDark}
        locale={locale}
        loading={loading}
        error={error}
        orders={filteredOrders}
        userBalance={userInfo?.balance ?? 0}
        onRefundRequest={async (orderId, amount, reason) => {
          try {
            await handleRefundRequest(orderId, amount, reason);
          } catch (err) {
            setError(err instanceof Error ? err.message : text.refundRequestFailed);
          }
        }}
        onInvoiceRequest={
          invoiceEnabled
            ? (order) => {
                // 单张不够起开金额时先勾上让用户继续凑单，而不是填完抬头才被拒。
                if (invoiceMinAmount > 0 && (order.payAmount ?? 0) < invoiceMinAmount) {
                  setQuickInvoiceError('');
                  setSelectedInvoiceIds((prev) => new Set(prev).add(order.id));
                  return;
                }
                openInvoiceDialog([order]);
              }
            : undefined
        }
        onInvoiceDownload={invoiceEnabled ? handleInvoiceDownload : undefined}
        selectedInvoiceOrderIds={selectedInvoiceIds}
        onToggleInvoiceSelect={invoiceEnabled ? toggleInvoiceSelect : undefined}
      />

      <PaginationBar
        page={page}
        totalPages={totalPages}
        total={summary.total}
        pageSize={pageSize}
        pageSizeOptions={PAGE_SIZE_OPTIONS}
        locale={locale}
        isDark={isDark}
        loading={loading}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
      />

      {invoiceTargets && invoiceTargets.length > 0 && (
        <InvoiceRequestDialog
          isDark={isDark}
          locale={locale}
          amount={invoiceTargets.reduce((sum, order) => sum + (order.payAmount ?? 0), 0)}
          orderIds={invoiceTargets.map((order) => order.id)}
          savedTitles={savedTitles}
          submitting={invoiceSubmitting}
          error={invoiceError}
          onSubmit={handleInvoiceRequest}
          onClose={() => {
            if (invoiceSubmitting) return;
            setInvoiceTargets(null);
            setInvoiceError('');
          }}
        />
      )}
    </PayPageLayout>
  );
}

function OrdersPageFallback() {
  const searchParams = useSearchParams();
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = searchParams.get('theme') === 'dark';

  return (
    <div className={`flex min-h-screen items-center justify-center ${isDark ? 'bg-slate-950' : 'bg-slate-50'}`}>
      <div className={isDark ? 'text-slate-400' : 'text-gray-500'}>{pickLocaleText(locale, '加载中...', 'Loading...')}</div>
    </div>
  );
}

export default function OrdersPage() {
  return (
    <Suspense fallback={<OrdersPageFallback />}>
      <OrdersContent />
    </Suspense>
  );
}
