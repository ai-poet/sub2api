'use client';

import { useSearchParams } from 'next/navigation';
import { useState, useEffect, useCallback, useMemo, Suspense } from 'react';
import PayPageLayout from '@/components/PayPageLayout';
import { getAdminAccessHint } from '@/lib/branding';
import { resolveLocale, type Locale } from '@/lib/locale';
import { buildAppApiPath } from '@/lib/public-path';
import { describePromotion, formatAmount, type BonusType, type PromotionRule } from '@/lib/promotion/calc';

// ── Types ──

type PromotionStatus = 'active' | 'scheduled' | 'ended' | 'disabled' | 'exhausted';

interface Promotion {
  id: string;
  name: string;
  description: string | null;
  minAmount: number;
  bonusType: BonusType;
  bonusValue: number;
  maxBonus: number | null;
  startsAt: string | null;
  endsAt: string | null;
  perUserLimit: number;
  totalLimit: number;
  enabled: boolean;
  sortOrder: number;
  usedCount: number;
  bonusTotal: number;
  status: PromotionStatus;
  createdAt: string;
  updatedAt: string;
}

interface PromotionFormData {
  name: string;
  description: string;
  min_amount: string;
  bonus_type: BonusType;
  bonus_value: string;
  max_bonus: string;
  starts_at: string; // datetime-local
  ends_at: string; // datetime-local
  per_user_limit: string;
  total_limit: string;
  sort_order: string;
  enabled: boolean;
}

// ── i18n ──

function getTexts(locale: Locale) {
  return locale === 'en'
    ? {
        missingToken: 'Missing admin token',
        missingTokenHint: getAdminAccessHint(locale),
        invalidToken: 'Invalid admin token',
        title: 'Top-up Promotions',
        subtitle: 'Configure "top up X, get Y" bonus campaigns shown on the recharge page',
        refresh: 'Refresh',
        loading: 'Loading...',
        noPromotions: 'No promotions yet',
        noPromotionsHint:
          'Click "New Promotion" to create one. Bonus is credited to the balance together with the top-up.',
        newPromotion: 'New Promotion',
        editPromotion: 'Edit Promotion',
        colName: 'Name',
        colRule: 'Rule',
        colWindow: 'Period',
        colLimits: 'Limits',
        colUsage: 'Used',
        colStatus: 'Status',
        colSortOrder: 'Sort',
        colEnabled: 'Enabled',
        colActions: 'Actions',
        edit: 'Edit',
        delete: 'Delete',
        deleteConfirm: 'Delete this promotion? Completed orders keep their bonus records.',
        fieldName: 'Promotion Name',
        fieldDescription: 'Description (shown to users)',
        fieldMinAmount: 'Credited Amount Threshold (USD)',
        fieldMinAmountHint: 'Applies when the credited amount is greater than or equal to this value.',
        fieldBonusType: 'Bonus Type',
        bonusTypeFixed: 'Fixed amount (USD)',
        bonusTypePercent: 'Percentage of credited amount',
        fieldBonusValue: 'Bonus Value',
        fieldBonusValueFixedHint: 'USD credited on top of the top-up amount.',
        fieldBonusValuePercentHint: 'Percent of the credited amount, e.g. 10 = 10%.',
        fieldMaxBonus: 'Bonus Cap (USD, optional)',
        percentNoCapWarning: 'No cap set: a percentage bonus grows with the top-up amount. Consider setting a cap.',
        fieldStartsAt: 'Starts At (optional)',
        fieldEndsAt: 'Ends At (optional)',
        fieldPerUserLimit: 'Per-user Limit (0 = unlimited)',
        fieldTotalLimit: 'Total Quota (0 = unlimited)',
        fieldSortOrder: 'Sort Order',
        fieldEnabled: 'Enable Promotion',
        preview: 'Preview',
        creditNote: 'Total credited = credited amount + bonus. It may exceed the max credited balance setting.',
        stackingNote: 'When several promotions match one order, only the one with the highest bonus applies.',
        unlimited: 'Unlimited',
        perUser: (n: number) => `${n}/user`,
        total: (n: number) => `${n} total`,
        usage: (count: number, bonus: number) => `${count} orders · $${bonus.toFixed(2)} bonus`,
        noWindow: 'Always on',
        from: 'From',
        until: 'Until',
        cancel: 'Cancel',
        save: 'Save',
        saving: 'Saving...',
        loadFailed: 'Failed to load promotions',
        saveFailed: 'Failed to save promotion',
        deleteFailed: 'Failed to delete promotion',
        validationName: 'Name is required',
        validationMinAmount: 'Threshold must be greater than 0',
        validationBonusValue: 'Bonus value must be greater than 0',
        validationWindow: 'End time must be later than start time',
        status: {
          active: 'Active',
          scheduled: 'Scheduled',
          ended: 'Ended',
          disabled: 'Disabled',
          exhausted: 'Quota full',
        } as Record<PromotionStatus, string>,
      }
    : {
        missingToken: '缺少管理员凭证',
        missingTokenHint: getAdminAccessHint(locale),
        invalidToken: '管理员凭证无效',
        title: '充值活动',
        subtitle: '配置「充多少送多少」活动，用户在充值页面可见',
        refresh: '刷新',
        loading: '加载中...',
        noPromotions: '暂无充值活动',
        noPromotionsHint: '点击「新建活动」创建。赠送金额会随充值一起计入用户余额。',
        newPromotion: '新建活动',
        editPromotion: '编辑活动',
        colName: '名称',
        colRule: '规则',
        colWindow: '有效期',
        colLimits: '限制',
        colUsage: '已使用',
        colStatus: '状态',
        colSortOrder: '排序',
        colEnabled: '启用',
        colActions: '操作',
        edit: '编辑',
        delete: '删除',
        deleteConfirm: '确定删除该活动？已完成订单的赠送记录会保留。',
        fieldName: '活动名称',
        fieldDescription: '活动说明（展示给用户）',
        fieldMinAmount: '到账金额门槛（USD）',
        fieldMinAmountHint: '用户选择的到账金额大于等于该值时生效。',
        fieldBonusType: '赠送方式',
        bonusTypeFixed: '固定金额（USD）',
        bonusTypePercent: '按到账金额比例',
        fieldBonusValue: '赠送值',
        fieldBonusValueFixedHint: '在到账金额基础上额外赠送的 USD。',
        fieldBonusValuePercentHint: '到账金额的百分比，如 10 表示 10%。',
        fieldMaxBonus: '赠送封顶（USD，可选）',
        percentNoCapWarning: '未设置封顶：按比例赠送会随充值金额增长，建议设置封顶。',
        fieldStartsAt: '开始时间（可选）',
        fieldEndsAt: '结束时间（可选）',
        fieldPerUserLimit: '每人可参与次数（0 = 不限）',
        fieldTotalLimit: '总名额（0 = 不限）',
        fieldSortOrder: '排序',
        fieldEnabled: '启用活动',
        preview: '预览',
        creditNote: '实际到账 = 到账金额 + 赠送，可能超过「最大到账金额」配置。',
        stackingNote: '一笔充值同时满足多个活动时，只按赠送最高的一个发放，不叠加。',
        unlimited: '不限',
        perUser: (n: number) => `每人 ${n} 次`,
        total: (n: number) => `总 ${n} 名额`,
        usage: (count: number, bonus: number) => `${count} 单 · 赠送 $${bonus.toFixed(2)}`,
        noWindow: '长期有效',
        from: '自',
        until: '至',
        cancel: '取消',
        save: '保存',
        saving: '保存中...',
        loadFailed: '加载充值活动失败',
        saveFailed: '保存充值活动失败',
        deleteFailed: '删除充值活动失败',
        validationName: '请填写活动名称',
        validationMinAmount: '门槛金额必须大于 0',
        validationBonusValue: '赠送值必须大于 0',
        validationWindow: '结束时间必须晚于开始时间',
        status: {
          active: '进行中',
          scheduled: '未开始',
          ended: '已结束',
          disabled: '已停用',
          exhausted: '名额已满',
        } as Record<PromotionStatus, string>,
      };
}

// ── Helpers ──

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

/** ISO → datetime-local（本地时区） */
function toLocalInput(iso: string | null): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** datetime-local → ISO；空值返回 null */
function fromLocalInput(value: string): string | null {
  if (!value.trim()) return null;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

function formatDateTime(iso: string, locale: Locale): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(locale === 'en' ? 'en-US' : 'zh-CN', {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

const emptyForm: PromotionFormData = {
  name: '',
  description: '',
  min_amount: '100',
  bonus_type: 'fixed',
  bonus_value: '10',
  max_bonus: '',
  starts_at: '',
  ends_at: '',
  per_user_limit: '0',
  total_limit: '0',
  sort_order: '0',
  enabled: true,
};

function formToRule(form: PromotionFormData): PromotionRule {
  return {
    id: 'preview',
    name: form.name || '',
    description: form.description || null,
    minAmount: parseFloat(form.min_amount) || 0,
    bonusType: form.bonus_type,
    bonusValue: parseFloat(form.bonus_value) || 0,
    maxBonus: form.bonus_type === 'percent' && form.max_bonus.trim() ? parseFloat(form.max_bonus) || null : null,
    sortOrder: parseInt(form.sort_order, 10) || 0,
  };
}

// ── Main Content ──

function PromotionsContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';
  const theme = searchParams.get('theme') === 'dark' ? 'dark' : 'light';
  const uiMode = searchParams.get('ui_mode') || 'standalone';
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = theme === 'dark';
  const isEmbedded = uiMode === 'embedded';
  const t = getTexts(locale);

  const [promotions, setPromotions] = useState<Promotion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Promotion | null>(null);
  const [form, setForm] = useState<PromotionFormData>(emptyForm);
  const [saving, setSaving] = useState(false);

  const authHeaders = useCallback(
    (extra?: Record<string, string>): Record<string, string> => ({
      Authorization: `Bearer ${token}`,
      ...(extra ?? {}),
    }),
    [token],
  );

  const langQuery = locale === 'en' ? '?lang=en' : '';

  // ── Fetch ──

  const fetchPromotions = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch(buildAppApiPath(`/api/admin/promotions${langQuery}`), { headers: authHeaders() });
      if (!res.ok) {
        if (res.status === 401) {
          setError(t.invalidToken);
          return;
        }
        throw new Error();
      }
      const data = await res.json();
      setPromotions(data.promotions ?? []);
    } catch {
      setError(t.loadFailed);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, langQuery]);

  useEffect(() => {
    fetchPromotions();
  }, [fetchPromotions]);

  // ── Form helpers ──

  const previewRule = useMemo(() => formToRule(form), [form]);
  const previewText =
    previewRule.minAmount > 0 && previewRule.bonusValue > 0 ? describePromotion(previewRule, locale) : '';
  const percentWithoutCap = form.bonus_type === 'percent' && !form.max_bonus.trim();

  const formError = (() => {
    if (!form.name.trim()) return t.validationName;
    if (!(parseFloat(form.min_amount) > 0)) return t.validationMinAmount;
    if (!(parseFloat(form.bonus_value) > 0)) return t.validationBonusValue;
    const starts = fromLocalInput(form.starts_at);
    const ends = fromLocalInput(form.ends_at);
    if (starts && ends && new Date(ends).getTime() <= new Date(starts).getTime()) return t.validationWindow;
    return '';
  })();

  const openCreateModal = () => {
    setEditing(null);
    setForm(emptyForm);
    setModalOpen(true);
  };

  const openEditModal = (promo: Promotion) => {
    setEditing(promo);
    setForm({
      name: promo.name,
      description: promo.description ?? '',
      min_amount: String(promo.minAmount),
      bonus_type: promo.bonusType,
      bonus_value: String(promo.bonusValue),
      max_bonus: promo.maxBonus != null ? String(promo.maxBonus) : '',
      starts_at: toLocalInput(promo.startsAt),
      ends_at: toLocalInput(promo.endsAt),
      per_user_limit: String(promo.perUserLimit),
      total_limit: String(promo.totalLimit),
      sort_order: String(promo.sortOrder),
      enabled: promo.enabled,
    });
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditing(null);
  };

  const handleSave = async () => {
    if (formError) return;
    setSaving(true);
    setError('');

    const body = {
      name: form.name.trim(),
      description: form.description.trim() || null,
      min_amount: parseFloat(form.min_amount),
      bonus_type: form.bonus_type,
      bonus_value: parseFloat(form.bonus_value),
      max_bonus: form.bonus_type === 'percent' && form.max_bonus.trim() ? parseFloat(form.max_bonus) : null,
      starts_at: fromLocalInput(form.starts_at),
      ends_at: fromLocalInput(form.ends_at),
      per_user_limit: parseInt(form.per_user_limit, 10) || 0,
      total_limit: parseInt(form.total_limit, 10) || 0,
      sort_order: parseInt(form.sort_order, 10) || 0,
      enabled: form.enabled,
    };

    try {
      const url = editing ? `/api/admin/promotions/${editing.id}${langQuery}` : `/api/admin/promotions${langQuery}`;
      const res = await fetch(buildAppApiPath(url), {
        method: editing ? 'PUT' : 'POST',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || t.saveFailed);
        return;
      }
      closeModal();
      fetchPromotions();
    } catch {
      setError(t.saveFailed);
    } finally {
      setSaving(false);
    }
  };

  const handleToggleEnabled = async (promo: Promotion) => {
    try {
      const res = await fetch(buildAppApiPath(`/api/admin/promotions/${promo.id}${langQuery}`), {
        method: 'PUT',
        headers: authHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify({ enabled: !promo.enabled }),
      });
      if (res.ok) {
        fetchPromotions();
      } else {
        const data = await res.json().catch(() => ({}));
        setError(data.error || t.saveFailed);
      }
    } catch {
      setError(t.saveFailed);
    }
  };

  const handleDelete = async (promo: Promotion) => {
    if (!confirm(t.deleteConfirm)) return;
    try {
      const res = await fetch(buildAppApiPath(`/api/admin/promotions/${promo.id}${langQuery}`), {
        method: 'DELETE',
        headers: authHeaders(),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || t.deleteFailed);
        return;
      }
      fetchPromotions();
    } catch {
      setError(t.deleteFailed);
    }
  };

  // ── Missing token ──

  if (!token) {
    return (
      <div className={`flex min-h-screen items-center justify-center p-4 ${isDark ? 'bg-slate-950' : 'bg-slate-50'}`}>
        <div className="text-center text-red-500">
          <p className="text-lg font-medium">{t.missingToken}</p>
          <p className={`mt-2 text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{t.missingTokenHint}</p>
        </div>
      </div>
    );
  }

  // ── Styles ──

  const btnBase = [
    'inline-flex items-center rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors',
    isDark
      ? 'border-slate-600 text-slate-200 hover:bg-slate-800'
      : 'border-slate-300 text-slate-700 hover:bg-slate-100',
  ].join(' ');

  const inputCls = [
    'w-full rounded-lg border px-3 py-2 text-sm transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500/50',
    isDark
      ? 'border-slate-600 bg-slate-700 text-slate-100 placeholder-slate-400'
      : 'border-slate-300 bg-white text-slate-900 placeholder-slate-400',
  ].join(' ');

  const labelCls = ['block text-sm font-medium mb-1', isDark ? 'text-slate-300' : 'text-slate-700'].join(' ');
  const hintCls = ['mt-1 text-xs', isDark ? 'text-slate-500' : 'text-slate-400'].join(' ');

  const statusBadgeCls = (status: PromotionStatus) => {
    const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium';
    switch (status) {
      case 'active':
        return `${base} ${isDark ? 'bg-emerald-500/20 text-emerald-300' : 'bg-emerald-100 text-emerald-700'}`;
      case 'scheduled':
        return `${base} ${isDark ? 'bg-blue-500/20 text-blue-300' : 'bg-blue-100 text-blue-700'}`;
      case 'exhausted':
        return `${base} ${isDark ? 'bg-amber-500/20 text-amber-300' : 'bg-amber-100 text-amber-700'}`;
      default:
        return `${base} ${isDark ? 'bg-slate-700 text-slate-300' : 'bg-slate-100 text-slate-600'}`;
    }
  };

  const renderWindow = (promo: Promotion) => {
    if (!promo.startsAt && !promo.endsAt) return t.noWindow;
    const parts: string[] = [];
    if (promo.startsAt) parts.push(`${t.from} ${formatDateTime(promo.startsAt, locale)}`);
    if (promo.endsAt) parts.push(`${t.until} ${formatDateTime(promo.endsAt, locale)}`);
    return parts.join(' ');
  };

  const renderLimits = (promo: Promotion) => {
    const parts: string[] = [];
    if (promo.perUserLimit > 0) parts.push(t.perUser(promo.perUserLimit));
    if (promo.totalLimit > 0) parts.push(t.total(promo.totalLimit));
    return parts.length > 0 ? parts.join(' · ') : t.unlimited;
  };

  // ── Render ──

  return (
    <PayPageLayout
      isDark={isDark}
      isEmbedded={isEmbedded}
      maxWidth="full"
      title={t.title}
      subtitle={t.subtitle}
      locale={locale}
    >
      {error && (
        <div
          className={`mb-4 rounded-lg border p-3 text-sm ${isDark ? 'border-red-800 bg-red-950/50 text-red-400' : 'border-red-200 bg-red-50 text-red-600'}`}
        >
          {error}
          <button onClick={() => setError('')} className="ml-2 opacity-60 hover:opacity-100">
            ✕
          </button>
        </div>
      )}

      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <p className={`text-xs ${isDark ? 'text-slate-500' : 'text-slate-500'}`}>{t.stackingNote}</p>
        <div className="flex gap-2">
          <button type="button" onClick={fetchPromotions} className={btnBase}>
            {t.refresh}
          </button>
          <button
            type="button"
            onClick={openCreateModal}
            className="inline-flex items-center rounded-lg border border-emerald-500 bg-emerald-500 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-emerald-600"
          >
            {t.newPromotion}
          </button>
        </div>
      </div>

      <div
        className={[
          'overflow-x-auto rounded-xl border',
          isDark ? 'border-slate-700 bg-slate-800/70' : 'border-slate-200 bg-white shadow-sm',
        ].join(' ')}
      >
        {loading ? (
          <div className={`py-12 text-center ${isDark ? 'text-slate-400' : 'text-gray-500'}`}>{t.loading}</div>
        ) : promotions.length === 0 ? (
          <div className={`py-12 text-center ${isDark ? 'text-slate-400' : 'text-gray-500'}`}>
            <p className="text-base font-medium">{t.noPromotions}</p>
            <p className="mt-1 text-sm opacity-70">{t.noPromotionsHint}</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr
                className={
                  isDark ? 'border-b border-slate-700 text-slate-400' : 'border-b border-slate-200 text-slate-500'
                }
              >
                <th className="px-4 py-3 text-left font-medium">{t.colName}</th>
                <th className="px-4 py-3 text-left font-medium">{t.colRule}</th>
                <th className="px-4 py-3 text-left font-medium">{t.colWindow}</th>
                <th className="px-4 py-3 text-left font-medium">{t.colLimits}</th>
                <th className="px-4 py-3 text-left font-medium">{t.colUsage}</th>
                <th className="px-4 py-3 text-center font-medium">{t.colStatus}</th>
                <th className="px-4 py-3 text-center font-medium">{t.colSortOrder}</th>
                <th className="px-4 py-3 text-center font-medium">{t.colEnabled}</th>
                <th className="px-4 py-3 text-right font-medium">{t.colActions}</th>
              </tr>
            </thead>
            <tbody>
              {promotions.map((promo) => (
                <tr
                  key={promo.id}
                  className={[
                    'border-b transition-colors',
                    isDark ? 'border-slate-700/50 hover:bg-slate-700/30' : 'border-slate-100 hover:bg-slate-50',
                  ].join(' ')}
                >
                  <td className={`px-4 py-3 font-medium ${isDark ? 'text-slate-100' : 'text-slate-900'}`}>
                    <div>{promo.name}</div>
                    {promo.description && (
                      <div className={`max-w-xs truncate text-xs ${isDark ? 'text-slate-500' : 'text-slate-400'}`}>
                        {promo.description}
                      </div>
                    )}
                  </td>
                  <td className={`px-4 py-3 ${isDark ? 'text-emerald-300' : 'text-emerald-700'}`}>
                    {describePromotion(promo, locale)}
                  </td>
                  <td className={`whitespace-nowrap px-4 py-3 text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                    {renderWindow(promo)}
                  </td>
                  <td className={`whitespace-nowrap px-4 py-3 text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                    {renderLimits(promo)}
                  </td>
                  <td className={`whitespace-nowrap px-4 py-3 text-xs ${isDark ? 'text-slate-300' : 'text-slate-600'}`}>
                    {t.usage(promo.usedCount, promo.bonusTotal)}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className={statusBadgeCls(promo.status)}>{t.status[promo.status]}</span>
                  </td>
                  <td className={`px-4 py-3 text-center ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                    {promo.sortOrder}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <button
                      type="button"
                      onClick={() => handleToggleEnabled(promo)}
                      className={[
                        'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                        promo.enabled ? 'bg-emerald-500' : isDark ? 'bg-slate-600' : 'bg-slate-300',
                      ].join(' ')}
                    >
                      <span
                        className={[
                          'inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform',
                          promo.enabled ? 'translate-x-4.5' : 'translate-x-0.5',
                        ].join(' ')}
                      />
                    </button>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="inline-flex gap-1">
                      <button
                        type="button"
                        onClick={() => openEditModal(promo)}
                        className={[
                          'rounded-md px-2 py-1 text-xs font-medium transition-colors',
                          isDark ? 'text-indigo-400 hover:bg-indigo-500/20' : 'text-indigo-600 hover:bg-indigo-50',
                        ].join(' ')}
                      >
                        {t.edit}
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(promo)}
                        className={[
                          'rounded-md px-2 py-1 text-xs font-medium transition-colors',
                          isDark ? 'text-red-400 hover:bg-red-500/20' : 'text-red-600 hover:bg-red-50',
                        ].join(' ')}
                      >
                        {t.delete}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* ── Create / Edit Modal ── */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div
            className={[
              'relative w-full max-w-lg overflow-y-auto rounded-2xl border p-6 shadow-2xl',
              isDark ? 'border-slate-700 bg-slate-800' : 'border-slate-200 bg-white',
            ].join(' ')}
            style={{ maxHeight: '90vh' }}
          >
            <h2 className={`mb-5 text-lg font-semibold ${isDark ? 'text-slate-100' : 'text-slate-900'}`}>
              {editing ? t.editPromotion : t.newPromotion}
            </h2>

            <div className="space-y-4">
              <div>
                <label className={labelCls}>{t.fieldName}</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className={inputCls}
                  maxLength={100}
                  required
                />
              </div>

              <div>
                <label className={labelCls}>{t.fieldDescription}</label>
                <textarea
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  rows={2}
                  className={inputCls}
                  maxLength={2000}
                />
              </div>

              <div>
                <label className={labelCls}>{t.fieldMinAmount}</label>
                <input
                  type="number"
                  step="0.01"
                  min="0.01"
                  value={form.min_amount}
                  onChange={(e) => setForm({ ...form, min_amount: e.target.value })}
                  className={inputCls}
                  required
                />
                <p className={hintCls}>{t.fieldMinAmountHint}</p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelCls}>{t.fieldBonusType}</label>
                  <select
                    value={form.bonus_type}
                    onChange={(e) => setForm({ ...form, bonus_type: e.target.value as BonusType })}
                    className={inputCls}
                  >
                    <option value="fixed">{t.bonusTypeFixed}</option>
                    <option value="percent">{t.bonusTypePercent}</option>
                  </select>
                </div>
                <div>
                  <label className={labelCls}>{t.fieldBonusValue}</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0.01"
                    value={form.bonus_value}
                    onChange={(e) => setForm({ ...form, bonus_value: e.target.value })}
                    className={inputCls}
                    required
                  />
                  <p className={hintCls}>
                    {form.bonus_type === 'percent' ? t.fieldBonusValuePercentHint : t.fieldBonusValueFixedHint}
                  </p>
                </div>
              </div>

              {form.bonus_type === 'percent' && (
                <div>
                  <label className={labelCls}>{t.fieldMaxBonus}</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0.01"
                    value={form.max_bonus}
                    onChange={(e) => setForm({ ...form, max_bonus: e.target.value })}
                    className={inputCls}
                    placeholder={t.unlimited}
                  />
                  {percentWithoutCap && (
                    <p className={['mt-1 text-xs', isDark ? 'text-amber-300' : 'text-amber-600'].join(' ')}>
                      {t.percentNoCapWarning}
                    </p>
                  )}
                </div>
              )}

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelCls}>{t.fieldStartsAt}</label>
                  <input
                    type="datetime-local"
                    value={form.starts_at}
                    onChange={(e) => setForm({ ...form, starts_at: e.target.value })}
                    className={inputCls}
                  />
                </div>
                <div>
                  <label className={labelCls}>{t.fieldEndsAt}</label>
                  <input
                    type="datetime-local"
                    value={form.ends_at}
                    onChange={(e) => setForm({ ...form, ends_at: e.target.value })}
                    className={inputCls}
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className={labelCls}>{t.fieldPerUserLimit}</label>
                  <input
                    type="number"
                    min="0"
                    step="1"
                    value={form.per_user_limit}
                    onChange={(e) => setForm({ ...form, per_user_limit: e.target.value })}
                    className={inputCls}
                  />
                </div>
                <div>
                  <label className={labelCls}>{t.fieldTotalLimit}</label>
                  <input
                    type="number"
                    min="0"
                    step="1"
                    value={form.total_limit}
                    onChange={(e) => setForm({ ...form, total_limit: e.target.value })}
                    className={inputCls}
                  />
                </div>
                <div>
                  <label className={labelCls}>{t.fieldSortOrder}</label>
                  <input
                    type="number"
                    min="0"
                    step="1"
                    value={form.sort_order}
                    onChange={(e) => setForm({ ...form, sort_order: e.target.value })}
                    className={inputCls}
                  />
                </div>
              </div>

              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => setForm({ ...form, enabled: !form.enabled })}
                  className={[
                    'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                    form.enabled ? 'bg-emerald-500' : isDark ? 'bg-slate-600' : 'bg-slate-300',
                  ].join(' ')}
                >
                  <span
                    className={[
                      'inline-block h-4 w-4 rounded-full bg-white transition-transform',
                      form.enabled ? 'translate-x-6' : 'translate-x-1',
                    ].join(' ')}
                  />
                </button>
                <span className={`text-sm ${isDark ? 'text-slate-300' : 'text-slate-700'}`}>{t.fieldEnabled}</span>
              </div>

              <div
                className={[
                  'rounded-lg border p-3 text-sm',
                  isDark ? 'border-slate-700 bg-slate-900/60' : 'border-slate-200 bg-slate-50',
                ].join(' ')}
              >
                <div className={`text-xs ${isDark ? 'text-slate-500' : 'text-slate-400'}`}>{t.preview}</div>
                <div className={`mt-1 font-medium ${isDark ? 'text-emerald-300' : 'text-emerald-700'}`}>
                  {previewText || '—'}
                  {previewText && previewRule.minAmount > 0 && (
                    <span className={`ml-2 text-xs font-normal ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                      {locale === 'en'
                        ? `e.g. top up $${formatAmount(previewRule.minAmount)} → credited $${formatAmount(previewRule.minAmount + (previewRule.bonusType === 'percent' ? Math.min((previewRule.minAmount * previewRule.bonusValue) / 100, previewRule.maxBonus ?? Number.POSITIVE_INFINITY) : previewRule.bonusValue))}`
                        : `如充 $${formatAmount(previewRule.minAmount)} → 到账 $${formatAmount(previewRule.minAmount + (previewRule.bonusType === 'percent' ? Math.min((previewRule.minAmount * previewRule.bonusValue) / 100, previewRule.maxBonus ?? Number.POSITIVE_INFINITY) : previewRule.bonusValue))}`}
                    </span>
                  )}
                </div>
                <p className={`mt-2 text-xs ${isDark ? 'text-slate-500' : 'text-slate-400'}`}>{t.creditNote}</p>
              </div>

              {formError && (
                <p className={['text-xs', isDark ? 'text-amber-300' : 'text-amber-600'].join(' ')}>{formError}</p>
              )}
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={closeModal}
                className={[
                  'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
                  isDark ? 'text-slate-400 hover:bg-slate-700' : 'text-slate-600 hover:bg-slate-100',
                ].join(' ')}
              >
                {t.cancel}
              </button>
              <button
                type="button"
                onClick={handleSave}
                disabled={saving || !!formError}
                className="rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? t.saving : t.save}
              </button>
            </div>
          </div>
        </div>
      )}
    </PayPageLayout>
  );
}

function PromotionsPageFallback() {
  const searchParams = useSearchParams();
  const locale = resolveLocale(searchParams.get('lang'));

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-slate-500">{locale === 'en' ? 'Loading...' : '加载中...'}</div>
    </div>
  );
}

export default function PromotionsPage() {
  return (
    <Suspense fallback={<PromotionsPageFallback />}>
      <PromotionsContent />
    </Suspense>
  );
}
