/**
 * 管理员上传发票时核对金额：从 PDF 的文字层读出「价税合计」，和申请的开票金额比对，
 * 防止把别人的发票传给这个申请。
 *
 * 结果只用于提示（管理员仍可照常开票），所以宁可读不出也不能读错。按可靠程度依次取：
 * 1. 「（小写）¥123.45」；
 * 2. 「价税合计」后面的大写金额（壹佰贰拾叁圆肆角伍分）；
 * 3. 前两者都读不到时，全文最大的 ¥ 金额：发票上带 ¥ 的只有合计金额、合计税额和
 *    价税合计，税额不为负，价税合计最大。
 * 扫描件没有文字层；老式税控发票的预定义 CMap 字体（STSong-Light / UniGB-UCS2-H）
 * 浏览器里没有 cMap 文件也读不出字。这两种都返回 unknown，由管理员人工核对。
 */

export type InvoiceAmountCheck =
  | { status: 'match'; foundCents: number }
  | { status: 'mismatch'; foundCents: number }
  | { status: 'unknown' };

const AMOUNT = String.raw`(\d{1,3}(?:,\d{3})+|\d+)\.(\d{2})(?![\d.])`;
const LOWER_CASE_TOTAL = new RegExp(String.raw`[（(]\s*小\s*写\s*[）)]\s*[:：]?\s*[¥￥]?\s*` + AMOUNT);
const YEN_AMOUNT = new RegExp(String.raw`[¥￥]\s*` + AMOUNT, 'g');
const UPPER_CASE_TOTAL = /[零壹贰叁肆伍陆柒捌玖拾佰仟万亿]+[圆元](?:[零壹贰叁肆伍陆柒捌玖]+[角分])*(?:整|正)?/;

const UPPER_DIGITS: Record<string, number> = {
  零: 0, 壹: 1, 贰: 2, 叁: 3, 肆: 4, 伍: 5, 陆: 6, 柒: 7, 捌: 8, 玖: 9,
};
const UPPER_UNITS: Record<string, number> = { 拾: 10, 佰: 100, 仟: 1000 };

function centsOf(integer: string, fraction: string): number {
  return Number(integer.replace(/,/g, '')) * 100 + Number(fraction);
}

/** 大写金额转成分；不是合法的大写金额时返回 null。 */
export function parseUpperCaseAmountCents(value: string): number | null {
  const match = /^([零壹贰叁肆伍陆柒捌玖拾佰仟万亿]+)[圆元]([零壹贰叁肆伍陆柒捌玖角分]*)(?:整|正)?$/.exec(value);
  if (!match || !/[壹贰叁肆伍陆柒捌玖拾]/.test(match[1] + match[2])) return null;

  let total = 0;
  let section = 0;
  let digit = 0;
  for (const char of match[1]) {
    if (char in UPPER_DIGITS) {
      digit = UPPER_DIGITS[char];
    } else if (char in UPPER_UNITS) {
      // 「拾圆」「壹佰拾圆」里省掉的壹
      section += (digit || (char === '拾' ? 1 : 0)) * UPPER_UNITS[char];
      digit = 0;
    } else if (char === '万') {
      total += (section + digit) * 10_000;
      section = 0;
      digit = 0;
    } else {
      total = (total + section + digit) * 100_000_000;
      section = 0;
      digit = 0;
    }
  }
  total += section + digit;

  let jiao = 0;
  let fen = 0;
  digit = 0;
  for (const char of match[2]) {
    if (char in UPPER_DIGITS) digit = UPPER_DIGITS[char];
    else if (char === '角') jiao = digit;
    else fen = digit;
    if (char === '角' || char === '分') digit = 0;
  }
  return total * 100 + jiao * 10 + fen;
}

/** 从发票文字里读出价税合计（分），按可靠程度排序、去重；读不到时返回空数组。 */
export function extractInvoiceTotalCents(text: string): number[] {
  const found: number[] = [];

  const lower = LOWER_CASE_TOTAL.exec(text);
  if (lower) found.push(centsOf(lower[1], lower[2]));

  const compact = text.replace(/\s+/g, '');
  const totalAt = compact.indexOf('价税合计');
  if (totalAt >= 0) {
    const upper = UPPER_CASE_TOTAL.exec(compact.slice(totalAt, totalAt + 80));
    const cents = upper ? parseUpperCaseAmountCents(upper[0]) : null;
    if (cents !== null) found.push(cents);
  }

  if (found.length === 0) {
    const yen = [...text.matchAll(YEN_AMOUNT)].map((m) => centsOf(m[1], m[2]));
    if (yen.length > 0) found.push(Math.max(...yen));
  }

  return [...new Set(found)];
}

export function checkInvoiceAmount(text: string, expectedYuan: number): InvoiceAmountCheck {
  const found = extractInvoiceTotalCents(text);
  if (found.length === 0) return { status: 'unknown' };
  const expectedCents = Math.round(expectedYuan * 100);
  if (found.includes(expectedCents)) return { status: 'match', foundCents: expectedCents };
  return { status: 'mismatch', foundCents: found[0] };
}

/**
 * 读出 PDF 的全部文字。文字块之间补空格：unpdf 的 extractText 直接拼接，
 * 「¥123.45」后面紧跟的数字会粘成一个数。
 */
export async function readPdfText(data: Uint8Array): Promise<string> {
  const { extractTextItems } = await import('unpdf');
  const { items } = await extractTextItems(data);
  return items.flat().map((item) => item.str).join(' ');
}

export function isPdfFile(file: File): boolean {
  return file.type === 'application/pdf' || /\.pdf$/i.test(file.name);
}
