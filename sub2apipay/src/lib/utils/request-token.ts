import type { NextRequest } from 'next/server';

/**
 * The sub2api user JWT a request carries.
 *
 * `Authorization: Bearer` comes first: the desktop client sends the token
 * there so it stays out of URLs, access logs and process argument lists. The
 * `token` query parameter and a token in the JSON body remain accepted for the
 * web pay page and older clients.
 */
export function readUserToken(request: NextRequest, bodyToken?: string | null): string | null {
  const header = request.headers.get('authorization')?.trim() ?? '';
  const bearer = /^bearer\s+(.+)$/i.exec(header)?.[1]?.trim();
  if (bearer) return bearer;

  const fromQuery = request.nextUrl.searchParams.get('token')?.trim();
  if (fromQuery) return fromQuery;

  return bodyToken?.trim() || null;
}
