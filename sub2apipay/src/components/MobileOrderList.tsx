'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import OrderFilterBar from '@/components/OrderFilterBar';
import type { Locale } from '@/lib/locale';
import {
  formatStatus,
  formatCreatedAt,
  formatInvoiceStatus,
  getStatusBadgeClass,
  getInvoiceStatusBadgeClass,
  getPaymentDisplayInfo,
  type MyOrder,
  type OrderStatusFilter,
} from '@/lib/pay-utils';

interface MobileOrderListProps {
  isDark: boolean;
  hasToken: boolean;
  orders: MyOrder[];
  hasMore: boolean;
  loadingMore: boolean;
  onRefresh: () => void;
  onLoadMore: () => void;
  locale?: Locale;
  /** 未提供时不显示开票入口（开票功能关闭或调用方不支持）。 */
  onInvoiceRequest?: (order: MyOrder) => void;
  onInvoiceDownload?: (invoiceId: string) => void;
}

export default function MobileOrderList({
  isDark,
  hasToken,
  orders,
  hasMore,
  loadingMore,
  onRefresh,
  onLoadMore,
  locale = 'zh',
  onInvoiceRequest,
  onInvoiceDownload,
}: MobileOrderListProps) {
  const [activeFilter, setActiveFilter] = useState<OrderStatusFilter>('ALL');
  const sentinelRef = useRef<HTMLDivElement>(null);

  const filteredOrders = useMemo(() => {
    if (activeFilter === 'ALL') return orders;
    return orders.filter((item) => item.status === activeFilter);
  }, [orders, activeFilter]);
  const formatOrderAmount = (order: MyOrder) => `${order.orderType === 'subscription' ? '¥' : '$'}${order.amount.toFixed(2)}`;

  useEffect(() => {
    if (!hasMore || loadingMore) return;
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          onLoadMore();
        }
      },
      { threshold: 0.1 },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore, loadingMore, onLoadMore]);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className={['text-base font-semibold', isDark ? 'text-slate-100' : 'text-slate-900'].join(' ')}>
          {locale === 'en' ? 'My Orders' : '我的订单'}
        </h3>
        <button
          type="button"
          onClick={onRefresh}
          className={[
            'rounded-lg border px-2.5 py-1 text-xs font-medium',
            isDark
              ? 'border-slate-600 text-slate-200 hover:bg-slate-800'
              : 'border-slate-300 text-slate-700 hover:bg-slate-100',
          ].join(' ')}
        >
          {locale === 'en' ? 'Refresh' : '刷新'}
        </button>
      </div>

      <OrderFilterBar isDark={isDark} locale={locale} activeFilter={activeFilter} onChange={setActiveFilter} />

      {!hasToken ? (
        <div
          className={[
            'rounded-xl border border-dashed px-4 py-8 text-center text-sm',
            isDark ? 'border-amber-500/40 text-amber-200' : 'border-amber-300 text-amber-700',
          ].join(' ')}
        >
          {locale === 'en'
            ? 'The current link does not include a login token, so "My Orders" is unavailable.'
            : '当前链接未携带登录 token，无法查询"我的订单"。'}
        </div>
      ) : filteredOrders.length === 0 ? (
        <div
          className={[
            'rounded-xl border border-dashed px-4 py-8 text-center text-sm',
            isDark ? 'border-slate-600 text-slate-400' : 'border-slate-300 text-slate-500',
          ].join(' ')}
        >
          {locale === 'en' ? 'No matching orders found' : '暂无符合条件的订单记录'}
        </div>
      ) : (
        <div className="space-y-2">
          {filteredOrders.map((order) => (
            <div
              key={order.id}
              className={[
                'rounded-xl border px-3 py-3',
                isDark ? 'border-slate-700 bg-slate-900/70' : 'border-slate-200 bg-white',
              ].join(' ')}
            >
              <div className="flex items-center justify-between">
                <span className="text-2xl font-semibold">{formatOrderAmount(order)}</span>
                <span
                  className={['rounded-full px-2 py-0.5 text-xs', getStatusBadgeClass(order.status, isDark)].join(' ')}
                >
                  {formatStatus(order.status, locale)}
                </span>
              </div>
              <div className={['mt-1 text-sm', isDark ? 'text-slate-300' : 'text-slate-600'].join(' ')}>
                {getPaymentDisplayInfo(order.paymentType, locale).channel}
              </div>
              <div className={['mt-0.5 text-xs', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                {formatCreatedAt(order.createdAt, locale)}
              </div>
              {onInvoiceRequest && (
                <div className="mt-2 flex items-center gap-2">
                  {order.invoice && order.invoice.status !== 'CANCELLED' && order.invoice.status !== 'REJECTED' ? (
                    <>
                      <span
                        className={[
                          'rounded-full px-2 py-0.5 text-xs',
                          getInvoiceStatusBadgeClass(order.invoice.status, isDark),
                        ].join(' ')}
                      >
                        {formatInvoiceStatus(order.invoice.status, locale)}
                      </span>
                      {order.invoice.status === 'ISSUED' && order.invoice.hasFile && onInvoiceDownload && (
                        <button
                          type="button"
                          onClick={() => onInvoiceDownload(order.invoice!.id)}
                          className={[
                            'rounded px-2 py-1 text-xs',
                            isDark
                              ? 'bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30'
                              : 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200',
                          ].join(' ')}
                        >
                          {locale === 'en' ? 'Download invoice' : '下载发票'}
                        </button>
                      )}
                    </>
                  ) : order.canRequestInvoice ? (
                    <button
                      type="button"
                      onClick={() => onInvoiceRequest(order)}
                      className={[
                        'rounded px-2 py-1 text-xs',
                        isDark
                          ? 'bg-blue-500/20 text-blue-300 hover:bg-blue-500/30'
                          : 'bg-blue-100 text-blue-700 hover:bg-blue-200',
                      ].join(' ')}
                    >
                      {locale === 'en' ? 'Request invoice' : '申请开票'}
                    </button>
                  ) : null}
                </div>
              )}
            </div>
          ))}

          {hasMore && (
            <div ref={sentinelRef} className="py-3 text-center">
              {loadingMore ? (
                <span className={['text-xs', isDark ? 'text-slate-400' : 'text-slate-500'].join(' ')}>
                  {locale === 'en' ? 'Loading...' : '加载中...'}
                </span>
              ) : (
                <span className={['text-xs', isDark ? 'text-slate-400' : 'text-slate-400'].join(' ')}>
                  {locale === 'en' ? 'Scroll up to load more' : '上滑加载更多'}
                </span>
              )}
            </div>
          )}

          {!hasMore && orders.length > 0 && (
            <div className={['py-2 text-center text-xs', isDark ? 'text-slate-400' : 'text-slate-400'].join(' ')}>
              {locale === 'en' ? 'All orders loaded' : '已显示全部订单'}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
