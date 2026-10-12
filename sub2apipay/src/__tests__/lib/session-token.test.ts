import { afterEach, describe, expect, it, vi } from 'vitest';
import { bearerHeaders, freshSessionToken, readMainSiteToken, requestToken, tokenUserId } from '@/lib/session-token';

function jwt(payload: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(payload)}.signature`;
}

describe('tokenUserId', () => {
  it('reads user_id from the payload', () => {
    expect(tokenUserId(jwt({ user_id: 42, email: 'a@example.com' }))).toBe(42);
  });

  it('decodes base64url payloads with non-ASCII claims', () => {
    expect(tokenUserId(jwt({ user_id: 7, username: '测试用户~~??' }))).toBe(7);
  });

  it('returns null for anything that is not a usable JWT', () => {
    expect(tokenUserId('')).toBeNull();
    expect(tokenUserId('not-a-jwt')).toBeNull();
    expect(tokenUserId('a.%%%.c')).toBeNull();
    expect(tokenUserId(jwt({ user_id: '42' }))).toBeNull();
    expect(tokenUserId(jwt({ user_id: 0 }))).toBeNull();
    expect(tokenUserId(jwt({ sub: 42 }))).toBeNull();
  });
});

describe('freshSessionToken', () => {
  const userA1 = jwt({ user_id: 1, iat: 1 });
  const userA2 = jwt({ user_id: 1, iat: 2 });
  const userB = jwt({ user_id: 2, iat: 3 });

  it('follows a rotated token of the same account', () => {
    expect(freshSessionToken(userA1, userA2)).toBe(userA2);
  });

  it('never switches to another account signed in elsewhere', () => {
    expect(freshSessionToken(userA1, userB)).toBe(userA1);
  });

  it('keeps the pinned token when nothing newer is available', () => {
    expect(freshSessionToken(userA1, '')).toBe(userA1);
    expect(freshSessionToken(userA1, userA1)).toBe(userA1);
  });

  it('keeps an undecodable pinned token instead of guessing', () => {
    expect(freshSessionToken('opaque-token', userA2)).toBe('opaque-token');
    expect(freshSessionToken(userA1, 'opaque-token')).toBe(userA1);
  });

  it('never invents a token when none was pinned', () => {
    expect(freshSessionToken('', userA2)).toBe('');
  });
});

describe('requestToken', () => {
  const desktopToken = jwt({ user_id: 1, iat: 10 });
  const staleWebToken = jwt({ user_id: 1, iat: 1 });
  const rotatedWebToken = jwt({ user_id: 1, iat: 20 });

  // 桌面客户端的支付中心以 ?token= 打开系统浏览器；那个浏览器里可能还留着同一账号过期的网页登录态。
  it('uses a token from the URL exactly as given, ignoring stored sessions', () => {
    expect(requestToken(desktopToken, desktopToken, staleWebToken)).toBe(desktopToken);
    expect(requestToken(desktopToken, desktopToken, '')).toBe(desktopToken);
  });

  it('follows same-account rotation only for a token taken from the main site session', () => {
    expect(requestToken('', staleWebToken, rotatedWebToken)).toBe(rotatedWebToken);
  });
});

describe('bearerHeaders', () => {
  it('adds the Authorization header and keeps extra headers', () => {
    expect(bearerHeaders('t', { 'Content-Type': 'application/json' })).toEqual({
      'Content-Type': 'application/json',
      Authorization: 'Bearer t',
    });
  });

  it('sends no Authorization header without a token', () => {
    expect(bearerHeaders('')).toEqual({});
  });
});

describe('readMainSiteToken', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('returns an empty string outside the browser', () => {
    expect(readMainSiteToken()).toBe('');
  });

  it('reads the main site access token from localStorage', () => {
    vi.stubGlobal('window', { localStorage: { getItem: (key: string) => (key === 'auth_token' ? ' abc ' : null) } });
    expect(readMainSiteToken()).toBe('abc');
  });

  it('treats blocked storage as signed out', () => {
    vi.stubGlobal('window', {
      localStorage: {
        getItem: () => {
          throw new Error('SecurityError');
        },
      },
    });
    expect(readMainSiteToken()).toBe('');
  });
});
