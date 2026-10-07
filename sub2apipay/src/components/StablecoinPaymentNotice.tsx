'use client';

import { useEffect, useRef } from 'react';
import type { Locale } from '@/lib/locale';
import { getStablecoinPaymentNotice } from '@/lib/stablecoin-notice';

interface StablecoinPaymentNoticeProps {
  paymentType: string;
  isDark: boolean;
  locale: Locale;
  onCancel: () => void;
  onConfirm: () => void;
}

/** USDT / USDC 点击支付后的二次确认框。由调用方按需挂载，挂载即显示。 */
export default function StablecoinPaymentNotice({
  paymentType,
  isDark,
  locale,
  onCancel,
  onConfirm,
}: StablecoinPaymentNoticeProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const notice = getStablecoinPaymentNotice(paymentType, locale);

  // 焦点放在对话框本身而不是"继续支付"上，免得用户没看内容一个回车就确认了
  useEffect(() => {
    dialogRef.current?.focus();
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onCancel]);

  if (!notice) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        ref={dialogRef}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="stablecoin-notice-title"
        tabIndex={-1}
        className={[
          'max-h-[90vh] w-full max-w-md overflow-y-auto rounded-xl p-6 shadow-xl focus:outline-none',
          isDark ? 'bg-slate-900 text-slate-200' : 'bg-white text-gray-700',
        ].join(' ')}
      >
        <h3
          id="stablecoin-notice-title"
          className={['text-lg font-bold', isDark ? 'text-slate-100' : 'text-gray-900'].join(' ')}
        >
          {notice.title}
        </h3>
        <p className={['mt-1 text-sm', isDark ? 'text-slate-400' : 'text-gray-500'].join(' ')}>{notice.intro}</p>

        <ol className="mt-4 space-y-3 text-sm">
          {notice.points.map((point, index) => (
            <li key={point.label} className={['rounded-lg p-3', isDark ? 'bg-slate-800' : 'bg-gray-50'].join(' ')}>
              <div className={['font-semibold', isDark ? 'text-slate-100' : 'text-gray-900'].join(' ')}>
                {index + 1}. {point.label}
              </div>
              <div className="mt-1 leading-6">{point.text}</div>
            </li>
          ))}
        </ol>

        <p
          className={[
            'mt-4 rounded-lg border p-3 text-sm font-medium',
            isDark ? 'border-amber-700 bg-amber-900/30 text-amber-300' : 'border-amber-200 bg-amber-50 text-amber-700',
          ].join(' ')}
        >
          {notice.warning}
        </p>

        <div className="mt-5 flex gap-3">
          <button
            type="button"
            onClick={onCancel}
            className={[
              'flex-1 rounded-lg border py-2.5 text-sm font-medium transition-colors',
              isDark
                ? 'border-slate-600 text-slate-200 hover:bg-slate-800'
                : 'border-gray-300 text-gray-700 hover:bg-gray-50',
            ].join(' ')}
          >
            {notice.cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="flex-1 rounded-lg bg-emerald-500 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-emerald-600 active:bg-emerald-700"
          >
            {notice.confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
