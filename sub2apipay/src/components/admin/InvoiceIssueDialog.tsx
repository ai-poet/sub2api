'use client';

import { useEffect, useRef, useState } from 'react';
import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';
import { checkInvoiceAmount, isPdfFile, readPdfText, type InvoiceAmountCheck } from '@/lib/invoice/pdf-amount';
import type { AdminInvoice } from './InvoiceTable';

const ACCEPTED = '.pdf,.ofd,.zip,.jpg,.jpeg,.png';
const MAX_BYTES = 10 * 1024 * 1024;
const TOAST_MS = 8000;

type AmountCheckState = InvoiceAmountCheck | { status: 'checking' } | { status: 'unsupported' } | null;

const yuan = (cents: number) => (cents / 100).toFixed(2);

interface InvoiceIssueDialogProps {
  isDark: boolean;
  locale: Locale;
  invoice: AdminInvoice;
  submitting: boolean;
  error: string;
  onSubmit: (file: File, adminNote: string) => void;
  onClose: () => void;
}

export default function InvoiceIssueDialog({
  isDark,
  locale,
  invoice,
  submitting,
  error,
  onSubmit,
  onClose,
}: InvoiceIssueDialogProps) {
  const text = {
    title: pickLocaleText(locale, '上传并开票', 'Upload & issue'),
    titleName: pickLocaleText(locale, '单位名称', 'Company name'),
    taxNo: pickLocaleText(locale, '税号', 'Tax ID'),
    amount: pickLocaleText(locale, '开票金额', 'Amount'),
    orderId: pickLocaleText(locale, '订单号', 'Order ID'),
    remark: pickLocaleText(locale, '客户备注', 'Customer remark'),
    file: pickLocaleText(locale, '发票文件', 'Invoice file'),
    fileHint: pickLocaleText(locale, '支持 PDF / OFD / ZIP / JPG / PNG，不超过 10MB', 'PDF, OFD, ZIP, JPG or PNG, up to 10MB'),
    adminNote: pickLocaleText(locale, '内部备注', 'Internal note'),
    adminNotePlaceholder: pickLocaleText(locale, '选填，仅管理员可见', 'Optional, admin only'),
    tooLarge: pickLocaleText(locale, '文件不能超过 10MB', 'File must not exceed 10MB'),
    notice: pickLocaleText(
      locale,
      '开票后将自动向用户发送通知邮件。',
      'The customer will be emailed automatically once issued.',
    ),
    cancel: pickLocaleText(locale, '取消', 'Cancel'),
    submit: pickLocaleText(locale, '确认开票', 'Issue invoice'),
    submitting: pickLocaleText(locale, '处理中...', 'Processing...'),
    amountChecking: pickLocaleText(locale, '正在核对发票金额...', 'Checking the invoice total...'),
    amountMatch: (found: string) =>
      pickLocaleText(locale, `发票金额 ¥${found}，与开票金额一致`, `Invoice total ¥${found} matches the requested amount`),
    amountMismatch: (found: string, expected: string) =>
      pickLocaleText(
        locale,
        `发票金额 ¥${found} 与用户申请的开票金额 ¥${expected} 不一致，请确认是否传错了发票`,
        `The invoice total ¥${found} does not match the requested ¥${expected}. Check that this is the right invoice.`,
      ),
    amountUnknown: pickLocaleText(
      locale,
      '未能从 PDF 中识别发票金额，请人工核对',
      "Couldn't read the total from this PDF. Check it manually.",
    ),
    amountUnsupported: pickLocaleText(
      locale,
      '仅 PDF 发票可自动核对金额，请人工核对',
      'Only PDF invoices are checked automatically. Check this file manually.',
    ),
    dismiss: pickLocaleText(locale, '关闭', 'Dismiss'),
  };

  const [file, setFile] = useState<File | null>(null);
  const [adminNote, setAdminNote] = useState('');
  const [localError, setLocalError] = useState('');
  const [amountCheck, setAmountCheck] = useState<AmountCheckState>(null);
  const [toast, setToast] = useState('');
  // 连续换文件时，只认最后一次选择的核对结果。
  const checkRun = useRef(0);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !submitting) onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [submitting, onClose]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(''), TOAST_MS);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const labelClass = ['mb-1 block text-sm font-medium', isDark ? 'text-slate-300' : 'text-gray-700'].join(' ');
  const inputClass = [
    'w-full rounded-lg border px-3 py-2 text-sm focus:border-blue-500 focus:outline-none',
    isDark ? 'border-slate-600 bg-slate-800 text-slate-100' : 'border-gray-300 bg-white text-gray-900',
  ].join(' ');

  const handleFileChange = (selected: File | null) => {
    const run = ++checkRun.current;
    setLocalError('');
    setToast('');
    setAmountCheck(null);
    if (selected && selected.size > MAX_BYTES) {
      setLocalError(text.tooLarge);
      setFile(null);
      return;
    }
    setFile(selected);
    if (!selected) return;
    if (!isPdfFile(selected)) {
      setAmountCheck({ status: 'unsupported' });
      return;
    }

    // 只提示不拦截：读错金额时管理员仍能照常开票。
    setAmountCheck({ status: 'checking' });
    void selected
      .arrayBuffer()
      .then((data) => readPdfText(new Uint8Array(data)))
      .then((pdfText) => checkInvoiceAmount(pdfText, invoice.amount))
      .catch((): InvoiceAmountCheck => ({ status: 'unknown' }))
      .then((result) => {
        if (run !== checkRun.current) return;
        setAmountCheck(result);
        if (result.status === 'mismatch') {
          setToast(text.amountMismatch(yuan(result.foundCents), invoice.amount.toFixed(2)));
        }
      });
  };

  const amountHint = ((): { message: string; tone: 'ok' | 'error' | 'muted' } | null => {
    switch (amountCheck?.status) {
      case 'checking':
        return { message: text.amountChecking, tone: 'muted' };
      case 'match':
        return { message: text.amountMatch(yuan(amountCheck.foundCents)), tone: 'ok' };
      case 'mismatch':
        return { message: text.amountMismatch(yuan(amountCheck.foundCents), invoice.amount.toFixed(2)), tone: 'error' };
      case 'unknown':
        return { message: text.amountUnknown, tone: 'muted' };
      case 'unsupported':
        return { message: text.amountUnsupported, tone: 'muted' };
      default:
        return null;
    }
  })();
  const hintToneClass = {
    ok: isDark ? 'text-emerald-400' : 'text-emerald-600',
    error: isDark ? 'text-red-400' : 'text-red-600',
    muted: isDark ? 'text-slate-400' : 'text-gray-500',
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      {toast && (
        <div
          role="alert"
          className={[
            'fixed left-1/2 top-4 z-10 flex w-[calc(100%-2rem)] max-w-md -translate-x-1/2 items-start gap-3 rounded-lg border px-4 py-3 text-sm shadow-lg',
            isDark ? 'border-red-500/40 bg-red-950 text-red-200' : 'border-red-300 bg-red-50 text-red-700',
          ].join(' ')}
        >
          <span className="flex-1">{toast}</span>
          <button
            type="button"
            onClick={() => setToast('')}
            aria-label={text.dismiss}
            className="shrink-0 leading-none opacity-70 hover:opacity-100"
          >
            ✕
          </button>
        </div>
      )}
      <div
        className={[
          'max-h-[90vh] w-full max-w-md overflow-y-auto rounded-xl p-6 shadow-xl',
          isDark ? 'bg-slate-900' : 'bg-white',
        ].join(' ')}
      >
        <h3 className={['text-lg font-bold', isDark ? 'text-slate-100' : 'text-gray-900'].join(' ')}>{text.title}</h3>

        <dl className={['mt-4 space-y-2 rounded-lg p-3 text-sm', isDark ? 'bg-slate-800' : 'bg-gray-50'].join(' ')}>
          <div className="flex justify-between gap-3">
            <dt className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.titleName}</dt>
            <dd className="truncate font-medium">{invoice.titleName}</dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.taxNo}</dt>
            <dd className="truncate font-mono text-xs">{invoice.taxNo}</dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.amount}</dt>
            <dd className="font-semibold">¥{invoice.amount.toFixed(2)}</dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.orderId}</dt>
            {/* 合并开票时列全部订单号：管理员要靠它核对合计金额。 */}
            <dd className="max-h-24 overflow-y-auto text-right font-mono text-xs">
              {(invoice.orderIds?.length ? invoice.orderIds : [invoice.orderId]).map((id) => (
                <div key={id} className="truncate">
                  #{id}
                </div>
              ))}
            </dd>
          </div>
          {invoice.remark && (
            <div className="flex justify-between gap-3">
              <dt className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.remark}</dt>
              <dd className="truncate">{invoice.remark}</dd>
            </div>
          )}
        </dl>

        <div className="mt-4 space-y-3">
          <div>
            <label className={labelClass}>
              {text.file} <span className="text-red-500">*</span>
            </label>
            <input
              type="file"
              accept={ACCEPTED}
              onChange={(e) => handleFileChange(e.target.files?.[0] ?? null)}
              className={`${inputClass} file:mr-3 file:rounded file:border-0 file:bg-slate-200 file:px-2 file:py-1 file:text-xs`}
            />
            <div className={['mt-1 text-xs', isDark ? 'text-slate-400' : 'text-gray-500'].join(' ')}>{text.fileHint}</div>
            {amountHint && (
              <div className={['mt-1 text-xs', hintToneClass[amountHint.tone]].join(' ')}>{amountHint.message}</div>
            )}
          </div>

          <div>
            <label className={labelClass}>{text.adminNote}</label>
            <textarea
              value={adminNote}
              rows={2}
              maxLength={200}
              onChange={(e) => setAdminNote(e.target.value)}
              placeholder={text.adminNotePlaceholder}
              className={inputClass}
            />
          </div>
        </div>

        <p className={['mt-3 text-xs', isDark ? 'text-slate-400' : 'text-gray-500'].join(' ')}>{text.notice}</p>
        {(localError || error) && (
          <div className={['mt-2 text-xs', isDark ? 'text-red-400' : 'text-red-600'].join(' ')}>{localError || error}</div>
        )}

        <div className="mt-6 flex gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={submitting}
            className={[
              'flex-1 rounded-lg border py-2 text-sm',
              isDark ? 'border-slate-600 text-slate-300 hover:bg-slate-800' : 'border-gray-300 text-gray-600 hover:bg-gray-50',
            ].join(' ')}
          >
            {text.cancel}
          </button>
          <button
            type="button"
            onClick={() => file && onSubmit(file, adminNote.trim())}
            disabled={!file || submitting}
            className={[
              'flex-1 rounded-lg py-2 text-sm font-medium text-white disabled:cursor-not-allowed',
              isDark
                ? 'bg-blue-600/90 hover:bg-blue-700 disabled:bg-slate-700 disabled:text-slate-500'
                : 'bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 disabled:text-gray-400',
            ].join(' ')}
          >
            {submitting ? text.submitting : text.submit}
          </button>
        </div>
      </div>
    </div>
  );
}
