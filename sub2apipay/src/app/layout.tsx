import type { Metadata } from 'next';
import { headers } from 'next/headers';
import { normalizeBasePath } from '@/lib/public-path';
import './globals.css';

export const metadata: Metadata = {
  title: 'Sub2API Recharge',
  description: 'Sub2API balance recharge platform',
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const headerStore = await headers();
  const pathname = headerStore.get('x-pathname') || '';
  const search = headerStore.get('x-search') || '';
  const basePath = normalizeBasePath(headerStore.get('x-forwarded-prefix'));
  const locale = new URLSearchParams(search).get('lang')?.trim().toLowerCase() === 'en' ? 'en' : 'zh';
  const htmlLang = locale === 'en' ? 'en' : 'zh-CN';

  return (
    <html lang={htmlLang} data-pathname={pathname} data-base-path={basePath || undefined}>
      <body className="antialiased">{children}</body>
    </html>
  );
}
