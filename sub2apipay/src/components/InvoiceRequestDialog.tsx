'use client';

import { useEffect, useState } from 'react';
import type { Locale } from '@/lib/locale';
import { pickLocaleText } from '@/lib/locale';

export interface SavedInvoiceTitleOption {
  id: string;
  titleName: string;
  taxNo: string;
  remark: string | null;
  contactEmail: string | null;
}

export interface InvoiceRequestPayload {
  titleName: string;
  taxNo: string;
  remark: string;
  contactEmail: string;
}

interface InvoiceRequestDialogProps {
  isDark: boolean;
  locale: Locale;
  /** 开票金额（人民币），来自订单的实付金额。 */
  amount: number | null;
  orderId: string;
  savedTitles: SavedInvoiceTitleOption[];
  submitting: boolean;
  error: string;
  onSubmit: (payload: InvoiceRequestPayload) => void;
  onClose: () => void;
}

const TAX_NO_PATTERN = /^[0-9A-Za-z]{15,20}$/;

export default function InvoiceRequestDialog({
  isDark,
  locale,
  amount,
  orderId,
  savedTitles,
  submitting,
  error,
  onSubmit,
  onClose,
}: InvoiceRequestDialogProps) {
  const text = {
    title: pickLocaleText(locale, '申请开票', 'Request Invoice'),
    subtitle: pickLocaleText(locale, '增值税普通发票', 'VAT general invoice'),
    savedTitles: pickLocaleText(locale, '常用抬头', 'Saved titles'),
    titleName: pickLocaleText(locale, '单位名称', 'Company name'),
    titleNamePlaceholder: pickLocaleText(locale, '请输入营业执照上的单位全称', 'Full company name as registered'),
    taxNo: pickLocaleText(locale, '税号', 'Tax ID'),
    taxNoPlaceholder: pickLocaleText(locale, '统一社会信用代码', 'Unified social credit code'),
    remark: pickLocaleText(locale, '备注', 'Remark'),
    remarkPlaceholder: pickLocaleText(locale, '选填，如需在发票备注栏体现的信息', 'Optional'),
    contactEmail: pickLocaleText(locale, '收票邮箱', 'Contact email'),
    contactEmailPlaceholder: pickLocaleText(locale, '选填，便于客服联系', 'Optional'),
    amount: pickLocaleText(locale, '开票金额', 'Invoice amount'),
    orderId: pickLocaleText(locale, '订单号', 'Order ID'),
    cancel: pickLocaleText(locale, '取消', 'Cancel'),
    submit: pickLocaleText(locale, '提交申请', 'Submit'),
    submitting: pickLocaleText(locale, '提交中...', 'Submitting...'),
    titleNameInvalid: pickLocaleText(locale, '请输入 2-100 个字符的单位名称', 'Company name must be 2-100 characters'),
    taxNoInvalid: pickLocaleText(locale, '税号应为 15-20 位字母或数字', 'Tax ID must be 15-20 letters or digits'),
    notice: pickLocaleText(
      locale,
      '提交后由客服人工开具，完成后将邮件通知您回到本页下载。',
      'An admin will issue the invoice manually. You will get an email when it is ready to download here.',
    ),
  };

  // 抬头记忆：挂载时就回填最近一次使用的抬头，用户仍可切换或改写。
  // 调用方保证抬头在打开对话框前已取回，因此这里用初始化器而不是 effect——
  // 在 effect 里 setState 会触发级联渲染，也会覆盖用户已经敲进去的内容。
  const [titleName, setTitleName] = useState(() => savedTitles[0]?.titleName ?? '');
  const [taxNo, setTaxNo] = useState(() => savedTitles[0]?.taxNo ?? '');
  const [remark, setRemark] = useState(() => savedTitles[0]?.remark ?? '');
  const [contactEmail, setContactEmail] = useState(() => savedTitles[0]?.contactEmail ?? '');

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !submitting) onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [submitting, onClose]);

  const trimmedTitle = titleName.trim();
  const trimmedTaxNo = taxNo.trim();
  const titleError = trimmedTitle.length > 0 && (trimmedTitle.length < 2 || trimmedTitle.length > 100) ? text.titleNameInvalid : '';
  const taxNoError = trimmedTaxNo.length > 0 && !TAX_NO_PATTERN.test(trimmedTaxNo) ? text.taxNoInvalid : '';
  const canSubmit =
    !submitting && trimmedTitle.length >= 2 && trimmedTitle.length <= 100 && TAX_NO_PATTERN.test(trimmedTaxNo);

  const applySavedTitle = (saved: SavedInvoiceTitleOption) => {
    setTitleName(saved.titleName);
    setTaxNo(saved.taxNo);
    setRemark(saved.remark ?? '');
    setContactEmail(saved.contactEmail ?? '');
  };

  const inputClass = [
    'w-full rounded-lg border px-3 py-2 text-sm focus:border-blue-500 focus:outline-none',
    isDark ? 'border-slate-600 bg-slate-800 text-slate-100' : 'border-gray-300 bg-white text-gray-900',
  ].join(' ');
  const labelClass = ['mb-1 block text-sm font-medium', isDark ? 'text-slate-300' : 'text-gray-700'].join(' ');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div
        className={[
          'max-h-[90vh] w-full max-w-md overflow-y-auto rounded-xl p-6 shadow-xl',
          isDark ? 'bg-slate-900' : 'bg-white',
        ].join(' ')}
      >
        <h3 className={['text-lg font-bold', isDark ? 'text-slate-100' : 'text-gray-900'].join(' ')}>{text.title}</h3>
        <p className={['mt-1 text-xs', isDark ? 'text-slate-400' : 'text-gray-500'].join(' ')}>{text.subtitle}</p>

        <div className={['mt-4 grid grid-cols-2 gap-3 text-sm', isDark ? 'text-slate-300' : 'text-gray-700'].join(' ')}>
          <div className={['rounded-lg p-3', isDark ? 'bg-slate-800' : 'bg-gray-50'].join(' ')}>
            <div className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.amount}</div>
            <div className="mt-1 font-semibold">{amount == null ? '-' : `¥${amount.toFixed(2)}`}</div>
          </div>
          <div className={['rounded-lg p-3', isDark ? 'bg-slate-800' : 'bg-gray-50'].join(' ')}>
            <div className={isDark ? 'text-slate-400' : 'text-gray-500'}>{text.orderId}</div>
            <div className="mt-1 truncate font-mono text-xs">#{orderId.slice(0, 12)}</div>
          </div>
        </div>

        {savedTitles.length > 0 && (
          <div className="mt-4">
            <div className={labelClass}>{text.savedTitles}</div>
            <div className="flex flex-wrap gap-2">
              {savedTitles.map((saved) => {
                const active = saved.taxNo === trimmedTaxNo;
                return (
                  <button
                    key={saved.id}
                    type="button"
                    onClick={() => applySavedTitle(saved)}
                    className={[
                      'max-w-full truncate rounded-full border px-3 py-1 text-xs transition-colors',
                      active
                        ? isDark
                          ? 'border-blue-400 bg-blue-500/20 text-blue-200'
                          : 'border-blue-400 bg-blue-50 text-blue-700'
                        : isDark
                          ? 'border-slate-600 text-slate-300 hover:bg-slate-800'
                          : 'border-gray-300 text-gray-600 hover:bg-gray-50',
                    ].join(' ')}
                  >
                    {saved.titleName}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        <div className="mt-4 space-y-3">
          <div>
            <label className={labelClass}>
              {text.titleName} <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={titleName}
              maxLength={100}
              onChange={(e) => setTitleName(e.target.value)}
              placeholder={text.titleNamePlaceholder}
              className={inputClass}
            />
            {titleError && <div className={['mt-1 text-xs', isDark ? 'text-red-400' : 'text-red-600'].join(' ')}>{titleError}</div>}
          </div>

          <div>
            <label className={labelClass}>
              {text.taxNo} <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={taxNo}
              maxLength={20}
              onChange={(e) => setTaxNo(e.target.value.toUpperCase())}
              placeholder={text.taxNoPlaceholder}
              className={`${inputClass} font-mono`}
            />
            {taxNoError && <div className={['mt-1 text-xs', isDark ? 'text-red-400' : 'text-red-600'].join(' ')}>{taxNoError}</div>}
          </div>

          <div>
            <label className={labelClass}>{text.remark}</label>
            <textarea
              value={remark}
              maxLength={200}
              rows={2}
              onChange={(e) => setRemark(e.target.value)}
              placeholder={text.remarkPlaceholder}
              className={inputClass}
            />
          </div>

          <div>
            <label className={labelClass}>{text.contactEmail}</label>
            <input
              type="email"
              value={contactEmail}
              maxLength={200}
              onChange={(e) => setContactEmail(e.target.value)}
              placeholder={text.contactEmailPlaceholder}
              className={inputClass}
            />
          </div>
        </div>

        <p className={['mt-3 text-xs', isDark ? 'text-slate-400' : 'text-gray-500'].join(' ')}>{text.notice}</p>
        {error && <div className={['mt-2 text-xs', isDark ? 'text-red-400' : 'text-red-600'].join(' ')}>{error}</div>}

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
            onClick={() =>
              onSubmit({
                titleName: trimmedTitle,
                taxNo: trimmedTaxNo,
                remark: remark.trim(),
                contactEmail: contactEmail.trim(),
              })
            }
            disabled={!canSubmit}
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
