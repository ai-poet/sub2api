'use client';

import { useSearchParams } from 'next/navigation';
import { Suspense } from 'react';
import { pickLocaleText, resolveLocale } from '@/lib/locale';

function SuccessContent() {
  const searchParams = useSearchParams();
  const outTradeNo = searchParams.get('out_trade_no') || searchParams.get('order_id');
  const theme = searchParams.get('theme') === 'dark' ? 'dark' : 'light';
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = theme === 'dark';

  const text = {
    label: pickLocaleText(locale, '支付成功', 'Payment Successful'),
    message: pickLocaleText(locale, '您的订单已完成支付。', 'Your order has been paid.'),
    orderId: pickLocaleText(locale, '订单号', 'Order ID'),
    unknown: pickLocaleText(locale, '未知', 'Unknown'),
  };

  const accentColor = isDark ? 'text-green-400' : 'text-green-600';

  return (
    <div className={`flex min-h-screen items-center justify-center p-4 ${isDark ? 'bg-slate-950' : 'bg-slate-50'}`}>
      <div
        className={[
          'w-full max-w-md rounded-xl p-8 text-center shadow-lg',
          isDark ? 'bg-slate-900 text-slate-100' : 'bg-white',
        ].join(' ')}
      >
        <div className={`text-6xl ${accentColor}`}>✓</div>
        <h1 className={`mt-4 text-xl font-bold ${accentColor}`}>{text.label}</h1>
        <p className={isDark ? 'mt-2 text-slate-400' : 'mt-2 text-gray-500'}>{text.message}</p>
        <p className={isDark ? 'mt-4 text-xs text-slate-500' : 'mt-4 text-xs text-gray-400'}>
          {text.orderId}: {outTradeNo || text.unknown}
        </p>
      </div>
    </div>
  );
}

function SuccessPageFallback() {
  const searchParams = useSearchParams();
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = searchParams.get('theme') === 'dark';

  return (
    <div className={`flex min-h-screen items-center justify-center ${isDark ? 'bg-slate-950' : 'bg-slate-50'}`}>
      <div className={isDark ? 'text-slate-400' : 'text-gray-500'}>
        {pickLocaleText(locale, '加载中...', 'Loading...')}
      </div>
    </div>
  );
}

export default function PaySuccessPage() {
  return (
    <Suspense fallback={<SuccessPageFallback />}>
      <SuccessContent />
    </Suspense>
  );
}
