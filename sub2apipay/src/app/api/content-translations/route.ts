import { NextRequest, NextResponse } from 'next/server';
import { resolveLocale, type Locale } from '@/lib/locale';
import {
  lookupContentTranslations,
  MAX_LOOKUP_TEXTS,
  MAX_TRANSLATION_TEXT_LENGTH,
} from '@/lib/sub2api/content-translations';

/** 与后端公开查询接口的请求体上限一致 */
const MAX_LOOKUP_BODY_BYTES = 256 * 1024;

class BodyTooLargeError extends Error {}

/** 读请求体，超过上限立即停止（不信任也不依赖 Content-Length） */
async function readBodyWithLimit(request: NextRequest, limit: number): Promise<string> {
  const declared = Number(request.headers.get('content-length'));
  if (Number.isFinite(declared) && declared > limit) throw new BodyTooLargeError();
  if (!request.body) return '';

  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > limit) {
      await reader.cancel().catch(() => undefined);
      throw new BodyTooLargeError();
    }
    chunks.push(value);
  }
  return Buffer.concat(chunks).toString('utf8');
}

function errorResponse(locale: Locale, zh: string, en: string, status: number) {
  return NextResponse.json({ error: locale === 'en' ? en : zh }, { status });
}

/**
 * POST /api/content-translations  { lang, texts[] } → { translations, pending }
 *
 * 支付页查管理员文案的译文（内容自动翻译，见 docs/CONTENT_TRANSLATION.md）。
 * 转发到后端的公开只读查询接口：只查缓存、不触发模型调用，所以不需要登录；这里只做大小限制。
 */
export async function POST(request: NextRequest) {
  let locale = resolveLocale(request.nextUrl.searchParams.get('lang'));

  let raw: string;
  try {
    raw = await readBodyWithLimit(request, MAX_LOOKUP_BODY_BYTES);
  } catch (error) {
    if (error instanceof BodyTooLargeError) {
      return errorResponse(locale, '请求体过大', 'Request body too large', 413);
    }
    return errorResponse(locale, '读取请求失败', 'Failed to read request', 400);
  }

  let body: unknown;
  try {
    body = JSON.parse(raw);
  } catch {
    return errorResponse(locale, '请求体不是有效的 JSON', 'Invalid JSON body', 400);
  }
  if (!body || typeof body !== 'object' || Array.isArray(body)) {
    return errorResponse(locale, '参数错误', 'Invalid parameters', 400);
  }

  const { lang, texts } = body as { lang?: unknown; texts?: unknown };
  if (lang !== undefined && lang !== null && typeof lang !== 'string') {
    return errorResponse(locale, 'lang 参数无效', 'Invalid lang', 400);
  }
  locale = resolveLocale(typeof lang === 'string' ? lang : null);

  if (!Array.isArray(texts) || !texts.every((text) => typeof text === 'string')) {
    return errorResponse(locale, 'texts 必须是字符串数组', 'texts must be an array of strings', 400);
  }
  if (texts.length > MAX_LOOKUP_TEXTS) {
    return errorResponse(
      locale,
      `texts 最多 ${MAX_LOOKUP_TEXTS} 条`,
      `texts must not exceed ${MAX_LOOKUP_TEXTS} items`,
      400,
    );
  }

  // 空串与超长文本后端本来就不会返回译文，不必转发
  const unique = [
    ...new Set(
      (texts as string[]).filter((text) => text.trim() !== '' && text.length <= MAX_TRANSLATION_TEXT_LENGTH),
    ),
  ];
  if (unique.length === 0) {
    return NextResponse.json({ translations: {}, pending: false });
  }

  const result = await lookupContentTranslations(locale, unique);
  return NextResponse.json({ translations: result.translations, pending: result.pending });
}
