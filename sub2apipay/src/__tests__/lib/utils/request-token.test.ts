import { describe, expect, it } from 'vitest';
import { NextRequest } from 'next/server';
import { readUserToken } from '@/lib/utils/request-token';

function request(url: string, headers?: Record<string, string>) {
  return new NextRequest(url, { headers });
}

describe('readUserToken', () => {
  it('prefers the Authorization bearer header', () => {
    const req = request('https://pay.example.com/api/user?token=from-query', {
      Authorization: 'Bearer from-header',
    });
    expect(readUserToken(req, 'from-body')).toBe('from-header');
  });

  it('accepts a lowercase scheme and surrounding whitespace', () => {
    const req = request('https://pay.example.com/api/user', { authorization: '  bearer   abc.def  ' });
    expect(readUserToken(req)).toBe('abc.def');
  });

  it('falls back to the token query parameter, then the body token', () => {
    expect(readUserToken(request('https://pay.example.com/api/user?token=q'), 'b')).toBe('q');
    expect(readUserToken(request('https://pay.example.com/api/orders'), ' b ')).toBe('b');
  });

  it('ignores non-bearer schemes and empty values', () => {
    const req = request('https://pay.example.com/api/user?token=%20', { Authorization: 'Basic abc' });
    expect(readUserToken(req, '')).toBeNull();
  });
});
