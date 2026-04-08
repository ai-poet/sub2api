import { headers } from 'next/headers';
import { resolveLocale } from '@/lib/locale';
import { normalizeBasePath, withPublicBasePath } from '@/lib/public-path';

const NAV_ITEMS = [
  { path: '/admin/orders', label: { zh: '订单管理', en: 'Orders' } },
  { path: '/admin/payment-config', label: { zh: '支付配置', en: 'Payment Config' } },
];

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const headerStore = await headers();
  const pathname = headerStore.get('x-pathname') || '/admin';
  const search = headerStore.get('x-search') || '';
  const basePath = normalizeBasePath(headerStore.get('x-forwarded-prefix'));
  const searchParams = new URLSearchParams(search);
  const token = searchParams.get('token') || '';
  const theme = searchParams.get('theme') || 'light';
  const uiMode = searchParams.get('ui_mode') || 'standalone';
  const locale = resolveLocale(searchParams.get('lang'));
  const isDark = theme === 'dark';
  const scopedPathname = basePath && pathname.startsWith(basePath) ? pathname.slice(basePath.length) || '/' : pathname;

  const buildUrl = (path: string) => {
    const params = new URLSearchParams();
    if (token) params.set('token', token);
    params.set('theme', theme);
    params.set('ui_mode', uiMode);
    if (locale !== 'zh') params.set('lang', locale);
    return `${withPublicBasePath(path, basePath)}?${params.toString()}`;
  };

  const isActive = (navPath: string) => {
    return scopedPathname.startsWith(navPath);
  };

  return (
    <div data-theme={theme} className={['min-h-screen', isDark ? 'bg-slate-950' : 'bg-slate-100'].join(' ')}>
      <div className="px-2 pt-2 sm:px-3 sm:pt-3">
        <nav
          className={[
            'mb-1 flex flex-wrap gap-1 rounded-xl border p-1',
            isDark ? 'border-slate-700 bg-slate-800/70' : 'border-slate-200 bg-slate-100/90',
          ].join(' ')}
        >
          {NAV_ITEMS.map((item) => (
            <a
              key={item.path}
              href={buildUrl(item.path)}
              className={[
                'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
                isActive(item.path)
                  ? isDark
                    ? 'bg-indigo-500/30 text-indigo-100 ring-1 ring-indigo-300/35'
                    : 'bg-white text-slate-900 ring-1 ring-slate-300 shadow-sm'
                  : isDark
                    ? 'text-slate-400 hover:text-slate-200 hover:bg-slate-700/50'
                    : 'text-slate-500 hover:text-slate-700 hover:bg-slate-200/70',
              ].join(' ')}
            >
              {item.label[locale]}
            </a>
          ))}
        </nav>
      </div>
      {children}
    </div>
  );
}
