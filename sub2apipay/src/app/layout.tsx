import type { Metadata } from 'next';
import { headers } from 'next/headers';
import { PAY_CENTER_METADATA_DESCRIPTION, PAY_CENTER_METADATA_TITLE } from '@/lib/branding';
import { normalizeBasePath } from '@/lib/public-path';
import './globals.css';

export const metadata: Metadata = {
  title: PAY_CENTER_METADATA_TITLE,
  description: PAY_CENTER_METADATA_DESCRIPTION,
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
