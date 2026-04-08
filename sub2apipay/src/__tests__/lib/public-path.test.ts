import { describe, expect, it } from 'vitest';
import { inferPublicBasePathFromPathname, withPublicBasePath } from '@/lib/public-path';

describe('public-path', () => {
  it('infers /pay base path from proxied admin pathname', () => {
    expect(inferPublicBasePathFromPathname('/pay/admin')).toBe('/pay');
    expect(inferPublicBasePathFromPathname('/pay/admin/orders')).toBe('/pay');
  });

  it('does not infer base path for standalone admin pathname', () => {
    expect(inferPublicBasePathFromPathname('/admin')).toBe('');
    expect(inferPublicBasePathFromPathname('/admin/orders')).toBe('');
  });

  it('prefixes admin routes with inferred public base path', () => {
    expect(withPublicBasePath('/admin/orders', '/pay')).toBe('/pay/admin/orders');
    expect(withPublicBasePath('/pay/admin/orders', '/pay')).toBe('/pay/admin/orders');
  });
});
