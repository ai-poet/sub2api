import { describe, expect, it } from 'vitest';
import {
  checkInvoiceAmount,
  extractInvoiceTotalCents,
  parseUpperCaseAmountCents,
  readPdfText,
} from '@/lib/invoice/pdf-amount';

// pdf.js 取出的数电票文字层（文字块以空格相连），金额 ¥106.00
const DIGITAL_INVOICE_TEXT = [
  '电子发票（普通发票）',
  '发票号码：',
  '26112000000012345678',
  '开票日期：',
  '2026年10月12日',
  '*信息技术服务*AI 模型调用服务',
  '100.00',
  '6%',
  '6.00',
  '合 计',
  '¥100.00',
  '¥6.00',
  '价税合计（大写）',
  '⊗壹佰零陆圆整',
  '（小写）',
  '¥106.00',
].join(' ');

describe('parseUpperCaseAmountCents', () => {
  it.each([
    ['壹佰零陆圆整', 10600],
    ['壹佰贰拾叁圆肆角伍分', 12345],
    ['拾圆整', 1000],
    ['壹拾贰万叁仟肆佰伍拾陆圆柒角捌分', 12345678],
    ['壹亿零伍佰万圆整', 10500000000],
    ['伍圆零伍分', 505],
    ['零圆玖角', 90],
  ])('reads %s', (value, cents) => {
    expect(parseUpperCaseAmountCents(value)).toBe(cents);
  });

  it.each(['零圆整', '壹佰', '一百元', '圆整'])('rejects %s', (value) => {
    expect(parseUpperCaseAmountCents(value)).toBeNull();
  });
});

describe('extractInvoiceTotalCents', () => {
  it('reads the lower-case and upper-case total of a digital invoice', () => {
    expect(extractInvoiceTotalCents(DIGITAL_INVOICE_TEXT)).toEqual([10600]);
  });

  it('reads thousands separators and half-width brackets', () => {
    expect(extractInvoiceTotalCents('价税合计(大写) 壹仟贰佰叁拾肆圆伍角陆分 (小写) ￥1,234.56')).toEqual([123456]);
  });

  it('keeps both readings when they disagree, lower case first', () => {
    expect(extractInvoiceTotalCents('价税合计（大写）壹佰圆整（小写）¥200.00')).toEqual([20000, 10000]);
  });

  it('falls back to the largest ¥ amount when the labels are unreadable', () => {
    expect(extractInvoiceTotalCents('��� ¥100.00 ¥6.00 ��� ¥106.00 2026')).toEqual([10600]);
  });

  it('does not glue an amount to a following number', () => {
    expect(extractInvoiceTotalCents('（小写）¥106.002026')).toEqual([]);
  });

  it('finds nothing in a PDF without a text layer', () => {
    expect(extractInvoiceTotalCents('')).toEqual([]);
    expect(extractInvoiceTotalCents('发票号码 26112000000012345678 2026年10月12日')).toEqual([]);
  });
});

describe('checkInvoiceAmount', () => {
  it('matches the requested amount', () => {
    expect(checkInvoiceAmount(DIGITAL_INVOICE_TEXT, 106)).toEqual({ status: 'match', foundCents: 10600 });
  });

  it('compares in cents', () => {
    expect(checkInvoiceAmount('（小写）¥0.30', 0.1 + 0.2)).toEqual({ status: 'match', foundCents: 30 });
  });

  it('reports the invoice total on a mismatch', () => {
    expect(checkInvoiceAmount(DIGITAL_INVOICE_TEXT, 128)).toEqual({ status: 'mismatch', foundCents: 10600 });
  });

  it('matches when either reading agrees', () => {
    expect(checkInvoiceAmount('价税合计（大写）壹佰圆整（小写）¥200.00', 100).status).toBe('match');
  });

  it('is unknown when no total can be read', () => {
    expect(checkInvoiceAmount('', 106)).toEqual({ status: 'unknown' });
  });
});

/**
 * 最小 PDF：每段文字一个文字对象，Type0 / Identity-H 字体 + ToUnicode，不嵌字形 ——
 * 数电票的字体就是这种结构，pdf.js 抽文字只靠 ToUnicode。
 */
function buildInvoicePdf(lines: string[]): Uint8Array {
  const chars = [...new Set(lines.join(''))];
  const code = (char: string) => (chars.indexOf(char) + 1).toString(16).padStart(4, '0');
  const content = lines
    .map((line, i) => `BT /F1 10 Tf 40 ${800 - i * 16} Td <${[...line].map(code).join('')}> Tj ET`)
    .join('\n');
  const toUnicode = [
    '/CIDInit /ProcSet findresource begin 12 dict begin begincmap',
    '/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def',
    '/CMapName /Adobe-Identity-UCS def /CMapType 2 def',
    '1 begincodespacerange <0000> <FFFF> endcodespacerange',
    `${chars.length} beginbfchar`,
    ...chars.map((char) => `<${code(char)}> <${char.codePointAt(0)!.toString(16).padStart(4, '0')}>`),
    'endbfchar endcmap CMapName currentdict /CMap defineresource pop end end',
  ].join('\n');
  const stream = (body: string) => `<< /Length ${body.length} >>\nstream\n${body}\nendstream`;
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>',
    stream(content),
    '<< /Type /Font /Subtype /Type0 /BaseFont /SimSun /Encoding /Identity-H /DescendantFonts [6 0 R] /ToUnicode 7 0 R >>',
    '<< /Type /Font /Subtype /CIDFontType2 /BaseFont /SimSun /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor 8 0 R /DW 1000 >>',
    stream(toUnicode),
    '<< /Type /FontDescriptor /FontName /SimSun /Flags 4 /FontBBox [0 -200 1000 900] /ItalicAngle 0 /Ascent 900 /Descent -200 /CapHeight 700 /StemV 80 >>',
  ];

  let pdf = '%PDF-1.7\n';
  const offsets = objects.map((body, i) => {
    const offset = pdf.length;
    pdf += `${i + 1} 0 obj\n${body}\nendobj\n`;
    return offset;
  });
  const xref = pdf.length;
  pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
  pdf += offsets.map((offset) => `${String(offset).padStart(10, '0')} 00000 n \n`).join('');
  pdf += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`;
  return new TextEncoder().encode(pdf);
}

describe('readPdfText', () => {
  it('reads the total out of a PDF through pdf.js', async () => {
    const pdf = buildInvoicePdf(['合 计', '¥100.00', '¥6.00', '价税合计（大写）', '⊗壹佰零陆圆整', '（小写）', '¥106.00']);
    const text = await readPdfText(pdf);

    expect(text).toContain('价税合计');
    expect(checkInvoiceAmount(text, 106)).toEqual({ status: 'match', foundCents: 10600 });
    expect(checkInvoiceAmount(text, 128)).toEqual({ status: 'mismatch', foundCents: 10600 });
  });
});
