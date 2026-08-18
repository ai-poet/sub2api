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
import { INVOICE_MAX_MERGED_ORDERS } from '@/lib/invoice/types';
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
    clearSelection: pickLocaleText(locale, '清空选择', 'Clear selection'),
    quickInvoice: pickLocaleText(locale, '一键开票', 'One-click invoice'),
    selectedInvoice: pickLocaleText(locale, '开票已选', 'Invoice selected'),
    invoiceableLabel: (count: number, amount: number) =>
      pickLocaleText(locale, `可开票 ${count} 单 · ¥${amount.toFixed(2)}`, `Invoiceable: ${count} orders · ¥${amount.toFixed(2)}`),
    selectedLabel: (count: number, amount: number) =>
      pickLocaleText(locale, `已选 ${count} 单 · ¥${amount.toFixed(2)}`, `Selected: ${count} orders · ¥${amount.toFixed(2)}`),
    invoiceBelowMin: (short: number, min: number) =>
      pickLocaleText(
        locale,
        `还差 ¥${short.toFixed(2)} 满 ¥${min.toFixed(2)} 起开，可多勾几张订单`,
        `¥${short.toFixed(2)} short of the ¥${min.toFixed(2)} minimum — select more orders`,
      ),
    invoiceOverLimit: pickLocaleText(
      locale,
      `单次最多合并 ${INVOICE_MAX_MERGED_ORDERS} 张订单，请减少勾选`,
      `At most ${INVOICE_MAX_MERGED_ORDERS} orders per invoice — deselect some`,
    ),
    quickBelowMin: (amount: number, min: number) =>
      pickLocaleText(
        locale,
        `可开票金额 ¥${amount.toFixed(2)}，未满 ¥${min.toFixed(2)} 起开——累计满 ¥${min.toFixed(2)} 后即可一键开票`,
        `Invoiceable total is ¥${amount.toFixed(2)} — invoicing opens at ¥${min.toFixed(2)}`,
      ),
    quickCappedHint: (limit: number) =>
      pickLocaleText(
        locale,
        `本次将开最早的 ${limit} 单，其余可在完成后再次开票`,
        `The earliest ${limit} orders will be invoiced; run it again for the rest`,
      ),
    invoiceSubmitted: (count: number, amount: number) =>
      pickLocaleText(
        locale,
        `开票申请已提交（${count} 单 · ¥${amount.toFixed(2)}），开具完成后将邮件通知`,
        `Invoice requested (${count} orders · ¥${amount.toFixed(2)}). You will be notified by email.`,
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
  /**
   * 开票对话框上下文。selected = 勾选开票（前端给出订单号清单）；
   * quick = 一键开票（订单由服务端挑选，前端只知道单数与金额）。
   */
  const [invoiceDialog, setInvoiceDialog] = useState<{
    mode: 'quick' | 'selected';
    orderIds: string[];
    count: number;
    amount: number;
  } | null>(null);
  const [savedTitles, setSavedTitles] = useState<SavedInvoiceTitleOption[]>([]);
  const [invoiceSubmitting, setInvoiceSubmitting] = useState(false);
  const [invoiceError, setInvoiceError] = useState('');
  /** 跨页勾选：id → 实付金额。跨页时当前页外的订单只剩这两样可用。 */
  const [selectedInvoiceOrders, setSelectedInvoiceOrders] = useState<Map<string, number>>(new Map());
  const [quickInvoiceError, setQuickInvoiceError] = useState('');
  const [invoiceNotice, setInvoiceNotice] = useState('');
  /** 一键开票预览：服务端跨全部分页算出的可开票单数与合计。 */
  const [quickPreview, setQuickPreview] = useState<{
    count: number;
    amount: number;
    eligibleCount: number;
    capped: boolean;
  } | null>(null);

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

      const loadedOrders: MyOrder[] = Array.isArray(data.orders) ? data.orders : [];
      setOrders(loadedOrders);
      setSummary(data.summary ?? { total: 0, pending: 0, completed: 0, failed: 0 });
      setPage(data.page ?? targetPage);
      setTotalPages(data.total_pages ?? 1);
      setInvoiceEnabled(!!data.invoiceEnabled);
      setInvoiceMinAmount(typeof data.invoiceMinAmount === 'number' ? data.invoiceMinAmount : 0);

      // 勾选是跨页保留的；本页里已不可开票的订单（刚开完票/退款了）从勾选里剔除。
      setSelectedInvoiceOrders((prev) => {
        if (prev.size === 0) return prev;
        const next = new Map(prev);
        for (const order of loadedOrders) {
          if (!order.canRequestInvoice && next.has(order.id)) next.delete(order.id);
        }
        return next.size === prev.size ? prev : next;
      });

      if (data.invoiceEnabled) {
        // 可开票概览独立请求，失败不影响订单列表。
        void loadQuickPreview();
      } else {
        setQuickPreview(null);
      }
    } catch {
      setOrders([]);
      setError(text.networkError);
    } finally {
      setLoading(false);
    }
  };

  const loadQuickPreview = async () => {
    try {
      const params = new URLSearchParams({ token });
      applyLocaleToSearchParams(params, locale);
      const res = await fetch(buildAppApiPath(`/api/invoices/quick?${params}`));
      if (!res.ok) return;
      const data = await res.json();
      setQuickPreview({
        count: typeof data.count === 'number' ? data.count : 0,
        amount: typeof data.amount === 'number' ? data.amount : 0,
        eligibleCount: typeof data.eligible_count === 'number' ? data.eligible_count : 0,
        capped: !!data.capped,
      });
      if (typeof data.min_amount === 'number') setInvoiceMinAmount(data.min_amount);
    } catch {
      // 概览失败保持旧值即可，开票提交端仍会校验。
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

  /** 组装抬头请求体：勾选开票带 order_ids，一键开票由服务端挑单不带。 */
  const buildTitleBody = (payload: InvoiceRequestPayload) => ({
    title_name: payload.titleName,
    tax_no: payload.taxNo,
    ...(payload.remark && { remark: payload.remark }),
    ...(payload.contactEmail && { contact_email: payload.contactEmail }),
  });

  const postInvoice = async (
    path: string,
    body: Record<string, unknown>,
  ): Promise<{ error: string | null; data: { count?: number; amount?: number } }> => {
    const params = new URLSearchParams({ token });
    applyLocaleToSearchParams(params, locale);
    const res = await fetch(buildAppApiPath(`${path}?${params}`), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      return { error: data.error || text.invoiceRequestFailed, data: {} };
    }
    return { error: null, data };
  };

  const afterInvoiceSubmitted = async (count: number, amount: number) => {
    setInvoiceDialog(null);
    setSelectedInvoiceOrders(new Map());
    setInvoiceNotice(text.invoiceSubmitted(count, amount));
    await loadOrders(page, pageSize);
  };

  // 抬头记忆：先取回已存抬头再挂载对话框，让对话框能在初始化时直接回填。
  const openSelectedInvoiceDialog = async () => {
    if (selectedInvoiceOrders.size === 0) return;
    setInvoiceError('');
    setQuickInvoiceError('');
    setInvoiceNotice('');
    const titles = await fetchSavedTitles();
    setSavedTitles(titles);
    setInvoiceDialog({
      mode: 'selected',
      orderIds: [...selectedInvoiceOrders.keys()],
      count: selectedInvoiceOrders.size,
      amount: selectedTotal,
    });
  };

  /** 对话框提交：按模式分流到勾选开票 / 一键开票接口。 */
  const handleInvoiceRequest = async (payload: InvoiceRequestPayload) => {
    if (!invoiceDialog) return;
    setInvoiceSubmitting(true);
    setInvoiceError('');
    try {
      const { error: failure, data } =
        invoiceDialog.mode === 'quick'
          ? await postInvoice('/api/invoices/quick', buildTitleBody(payload))
          : await postInvoice('/api/invoices/requests', {
              ...buildTitleBody(payload),
              order_ids: invoiceDialog.orderIds,
            });
      if (failure) {
        setInvoiceError(failure);
        return;
      }
      await afterInvoiceSubmitted(data.count ?? invoiceDialog.count, data.amount ?? invoiceDialog.amount);
    } catch {
      setInvoiceError(text.invoiceRequestFailed);
    } finally {
      setInvoiceSubmitting(false);
    }
  };

  /**
   * 一键开票：订单由服务端按可开票规则全量挑选，前端只展示金额。
   * 有历史抬头直接提交；首次开票弹对话框把抬头敲进去。
   */
  const handleQuickInvoice = async () => {
    if (!quickPreview || quickPreview.count === 0) return;
    setQuickInvoiceError('');
    setInvoiceNotice('');

    if (!defaultTitle) {
      const titles = await fetchSavedTitles();
      setSavedTitles(titles);
      setInvoiceError('');
      setInvoiceDialog({ mode: 'quick', orderIds: [], count: quickPreview.count, amount: quickPreview.amount });
      return;
    }

    setInvoiceSubmitting(true);
    try {
      const { error: failure, data } = await postInvoice('/api/invoices/quick', {
        title_name: defaultTitle.titleName,
        tax_no: defaultTitle.taxNo,
        ...(defaultTitle.remark && { remark: defaultTitle.remark }),
        ...(defaultTitle.contactEmail && { contact_email: defaultTitle.contactEmail }),
      });
      if (failure) {
        setQuickInvoiceError(failure);
        return;
      }
      await afterInvoiceSubmitted(data.count ?? quickPreview.count, data.amount ?? quickPreview.amount);
    } catch {
      setQuickInvoiceError(text.invoiceRequestFailed);
    } finally {
      setInvoiceSubmitting(false);
    }
  };

  const toggleInvoiceSelect = (order: MyOrder) => {
    setQuickInvoiceError('');
    setInvoiceNotice('');
    setSelectedInvoiceOrders((prev) => {
      const next = new Map(prev);
      if (next.has(order.id)) next.delete(order.id);
      else next.set(order.id, order.payAmount ?? 0);
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

  // 勾选跨页保留：合计与单数看全部勾选（Map），表头全选只作用于当前页可见的可开票行。
  const invoiceableOrders = invoiceEnabled ? filteredOrders.filter((order) => order.canRequestInvoice) : [];
  const selectedIdSet = new Set(selectedInvoiceOrders.keys());
  const selectedCount = selectedInvoiceOrders.size;
  const selectedTotal = [...selectedInvoiceOrders.values()].reduce((sum, amount) => sum + amount, 0);
  const pageAllSelected =
    invoiceableOrders.length > 0 && invoiceableOrders.every((order) => selectedInvoiceOrders.has(order.id));
  // 起开金额按合计判定，所以差额提示挂在勾选集合上，而不是单张订单上。
  const belowMinAmount = invoiceMinAmount > 0 && selectedTotal < invoiceMinAmount;
  // 服务端有同样的上限；这里先拦住并给出人话，别让用户填完抬头才被拒。
  const overMergeLimit = selectedCount > INVOICE_MAX_MERGED_ORDERS;
  const canSubmitSelected = selectedCount > 0 && !belowMinAmount && !overMergeLimit && !invoiceSubmitting;
  // count > 0 才提示：一单都没有时不存在「凑满起开金额」的问题，操作栏也不显示。
  const quickBelowMin =
    quickPreview != null && quickPreview.count > 0 && invoiceMinAmount > 0 && quickPreview.amount < invoiceMinAmount;
  const canSubmitQuick = !!quickPreview && quickPreview.count > 0 && !quickBelowMin && !invoiceSubmitting;

  const toggleSelectAllPage = () => {
    setQuickInvoiceError('');
    setInvoiceNotice('');
    setSelectedInvoiceOrders((prev) => {
      const next = new Map(prev);
      if (pageAllSelected) {
        for (const order of invoiceableOrders) next.delete(order.id);
      } else {
        for (const order of invoiceableOrders) next.set(order.id, order.payAmount ?? 0);
      }
      return next;
    });
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

      {/* 表格顶部操作栏：筛选在左，开票入口在右。一键开票（服务端挑单）与勾选开票分离。 */}
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <OrderFilterBar isDark={isDark} locale={locale} activeFilter={activeFilter} onChange={setActiveFilter} />

        {invoiceEnabled && (quickPreview?.count ?? 0) + selectedCount > 0 && (
          <div className="flex flex-wrap items-center gap-2 text-xs">
            {quickPreview && quickPreview.count > 0 && (
              <span className={isDark ? 'text-slate-300' : 'text-slate-600'}>
                {text.invoiceableLabel(quickPreview.count, quickPreview.amount)}
              </span>
            )}
            <button
              type="button"
              disabled={!canSubmitQuick}
              onClick={handleQuickInvoice}
              // 一键开票是主操作：实心样式；订单由服务端挑选，这里只展示金额。
              className={[
                'inline-flex items-center rounded-lg border border-blue-500 bg-blue-500 px-3 py-1.5 text-xs font-semibold text-white transition-colors',
                'hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50',
              ].join(' ')}
              // 按钮禁用时 hover 也能看到原因；正文提示在下方消息带里常显。
              title={
                quickBelowMin && quickPreview
                  ? text.quickBelowMin(quickPreview.amount, invoiceMinAmount)
                  : defaultTitle
                    ? `${defaultTitle.titleName} · ${defaultTitle.taxNo}`
                    : undefined
              }
            >
              {invoiceSubmitting ? '...' : text.quickInvoice}
            </button>

            {selectedCount > 0 && (
              <>
                <span className={['border-l pl-2', isDark ? 'border-slate-700 text-slate-200' : 'border-slate-300 text-slate-800'].join(' ')}>
                  {text.selectedLabel(selectedCount, selectedTotal)}
                </span>
                <button
                  type="button"
                  disabled={!canSubmitSelected}
                  onClick={openSelectedInvoiceDialog}
                  className={`${btnClass} disabled:cursor-not-allowed disabled:opacity-50`}
                >
                  {text.selectedInvoice}
                </button>
                <button type="button" onClick={() => setSelectedInvoiceOrders(new Map())} className={btnClass}>
                  {text.clearSelection}
                </button>
              </>
            )}
          </div>
        )}
      </div>

      {invoiceEnabled &&
        (invoiceNotice ||
          quickInvoiceError ||
          quickBelowMin ||
          (belowMinAmount && selectedCount > 0) ||
          overMergeLimit ||
          (quickPreview?.capped && quickPreview.count > 0)) && (
          <div
            className={[
              'mb-3 space-y-1 rounded-xl border px-3 py-2 text-xs',
              isDark ? 'border-slate-700 bg-slate-800/60' : 'border-slate-200 bg-slate-50',
            ].join(' ')}
          >
            {invoiceNotice && (
              <div className={isDark ? 'text-emerald-300' : 'text-emerald-700'}>{invoiceNotice}</div>
            )}
            {quickInvoiceError && (
              <div className={isDark ? 'text-red-400' : 'text-red-600'}>{quickInvoiceError}</div>
            )}
            {quickBelowMin && quickPreview && (
              <div className={isDark ? 'text-amber-300' : 'text-amber-700'}>
                {text.quickBelowMin(quickPreview.amount, invoiceMinAmount)}
              </div>
            )}
            {belowMinAmount && selectedCount > 0 && (
              <div className={isDark ? 'text-amber-300' : 'text-amber-700'}>
                {text.invoiceBelowMin(invoiceMinAmount - selectedTotal, invoiceMinAmount)}
              </div>
            )}
            {overMergeLimit && (
              <div className={isDark ? 'text-amber-300' : 'text-amber-700'}>{text.invoiceOverLimit}</div>
            )}
            {quickPreview?.capped && quickPreview.count > 0 && (
              <div className={isDark ? 'text-slate-400' : 'text-slate-500'}>
                {text.quickCappedHint(INVOICE_MAX_MERGED_ORDERS)}
              </div>
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
        invoiceEnabled={invoiceEnabled}
        onInvoiceDownload={invoiceEnabled ? handleInvoiceDownload : undefined}
        selectedInvoiceOrderIds={selectedIdSet}
        onToggleInvoiceSelect={invoiceEnabled ? toggleInvoiceSelect : undefined}
        pageAllSelected={pageAllSelected}
        onToggleSelectAllPage={toggleSelectAllPage}
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

      {invoiceDialog && (
        <InvoiceRequestDialog
          isDark={isDark}
          locale={locale}
          amount={invoiceDialog.amount}
          orderIds={invoiceDialog.orderIds}
          orderCount={invoiceDialog.count}
          savedTitles={savedTitles}
          submitting={invoiceSubmitting}
          error={invoiceError}
          onSubmit={handleInvoiceRequest}
          onClose={() => {
            if (invoiceSubmitting) return;
            setInvoiceDialog(null);
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
