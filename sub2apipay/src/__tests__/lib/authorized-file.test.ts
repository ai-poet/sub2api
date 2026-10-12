import { describe, expect, it } from 'vitest';
import { filenameFromDisposition } from '@/lib/authorized-file';

describe('filenameFromDisposition', () => {
  it('prefers the RFC 5987 filename*', () => {
    expect(
      filenameFromDisposition(
        `attachment; filename="invoice.pdf"; filename*=UTF-8''%E5%8F%91%E7%A5%A8-2026.pdf`,
      ),
    ).toBe('发票-2026.pdf');
  });

  it('falls back to a quoted or bare filename', () => {
    expect(filenameFromDisposition('attachment; filename="invoice 1.pdf"')).toBe('invoice 1.pdf');
    expect(filenameFromDisposition('inline; filename=invoice.ofd')).toBe('invoice.ofd');
  });

  it('falls back to the plain filename when filename* is malformed', () => {
    expect(filenameFromDisposition(`attachment; filename="a.pdf"; filename*=UTF-8''%E5%ZZ`)).toBe('a.pdf');
  });

  it('returns an empty string when there is no filename', () => {
    expect(filenameFromDisposition(null)).toBe('');
    expect(filenameFromDisposition('attachment')).toBe('');
  });
});
