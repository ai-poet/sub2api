import { describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/config', () => ({
  getEnv: () => ({
    EASY_PAY_PID: '1001',
    EASY_PAY_PKEY: 'test-merchant-secret-key',
    EASY_PAY_API_BASE: 'https://pay.example.com',
    EASY_PAY_NOTIFY_URL: 'https://site.example.com/api/easy-pay/notify',
    EASY_PAY_RETURN_URL: 'https://site.example.com/pay/result',
  }),
}));

// 不 mock sign：用真实 MD5 签名，被拒的用例签名本身都是对的，证明拒绝来自参数名守卫而不是验签
import { generateSign } from '@/lib/easy-pay/sign';
import { EasyPayProvider } from '@/lib/easy-pay/provider';
import { EASY_PAY_CREATE_ONLY_PARAMS } from '@/lib/easy-pay/notify-params';
import { EasyPayNotifyRejectedError } from '@/lib/easy-pay/notify-errors';

const PKEY = 'test-merchant-secret-key';

const STANDARD_NOTIFY: Record<string, string> = {
  pid: '1001',
  trade_no: '2026100612345678',
  out_trade_no: 'order-001',
  type: 'alipay',
  name: 'Balance recharge 10.00',
  money: '10.00',
  trade_status: 'TRADE_SUCCESS',
};

/** 按平台的方式签名：sign 覆盖除 sign / sign_type 外的全部字段；路由参数不参与签名。 */
function signedBody(fields: Record<string, string>, routing: Record<string, string> = {}): string {
  const sign = generateSign(fields, PKEY);
  return new URLSearchParams({ ...routing, ...fields, sign, sign_type: 'MD5' }).toString();
}

describe('EasyPayProvider.verifyNotification 参数名守卫', () => {
  const provider = new EasyPayProvider();

  it('标准通知照常通过', async () => {
    const notification = await provider.verifyNotification(signedBody(STANDARD_NOTIFY), {});

    expect(notification).toMatchObject({
      orderId: 'order-001',
      tradeNo: '2026100612345678',
      amount: 10,
      status: 'success',
    });
  });

  it('带 inst 路由参数的通知照常通过（inst 不参与签名）', async () => {
    const notification = await provider.verifyNotification(signedBody(STANDARD_NOTIFY, { inst: 'epay-main' }), {});

    expect(notification.status).toBe('success');
    expect(notification.orderId).toBe('order-001');
  });

  it('克隆平台多带的已签名字段照常通过，不误伤真实通知', async () => {
    const fields = {
      ...STANDARD_NOTIFY,
      param: 'custom',
      buyer: 'buyer@example.com',
      addtime: '2026-10-06 10:00:00',
      endtime: '2026-10-06 10:01:00',
      api_trade_no: '4200001234',
      // EPUSDT 的 epay 兼容通知
      trade_id: 'EPUSDT-0001',
    };

    const notification = await provider.verifyNotification(signedBody(fields), {});

    expect(notification.status).toBe('success');
    expect(notification.amount).toBe(10);
  });

  it.each([...EASY_PAY_CREATE_ONLY_PARAMS])('签名正确但带下单专有参数 %s：拒绝', async (field) => {
    const body = signedBody({ ...STANDARD_NOTIFY, [field]: 'x' });

    await expect(provider.verifyNotification(body, {})).rejects.toThrow(`unexpected param "${field}"`);
  });

  it.each([...EASY_PAY_CREATE_ONLY_PARAMS])('下单专有参数 %s 即使为空值也拒绝', async (field) => {
    // 空值不参与签名，签名仍是标准通知的签名
    const body = `${signedBody(STANDARD_NOTIFY)}&${field}=`;

    await expect(provider.verifyNotification(body, {})).rejects.toThrow(`unexpected param "${field}"`);
  });

  it.each(['trade.status', 'x=y', 'a&b', 'name ', ''])('签名正确但参数名不是纯标识符 %j：拒绝', async (name) => {
    const body = signedBody({ ...STANDARD_NOTIFY, [name]: 'v' });

    await expect(provider.verifyNotification(body, {})).rejects.toThrow('unexpected param');
  });

  it('拒绝时错误信息带上 out_trade_no，便于按订单号排查', async () => {
    const body = signedBody({ ...STANDARD_NOTIFY, clientip: '127.0.0.1' });

    await expect(provider.verifyNotification(body, {})).rejects.toThrow('out_trade_no="order-001"');
  });

  it('原有校验不变：签名错误仍拒绝', async () => {
    const body = new URLSearchParams({ ...STANDARD_NOTIFY, sign: '0'.repeat(32), sign_type: 'MD5' }).toString();

    await expect(provider.verifyNotification(body, {})).rejects.toThrow('signature verification failed');
  });

  it('原有校验不变：pid 与配置不一致仍拒绝', async () => {
    const body = signedBody({ ...STANDARD_NOTIFY, pid: '2002' });

    await expect(provider.verifyNotification(body, {})).rejects.toThrow('pid mismatch');
  });

  it('各类拒绝都抛出带原因与订单号的 EasyPayNotifyRejectedError，供风控面板记录', async () => {
    const reject = (body: string) => provider.verifyNotification(body, {}).catch((error: unknown) => error);

    const guard = await reject(signedBody({ ...STANDARD_NOTIFY, clientip: '127.0.0.1' }));
    expect(guard).toBeInstanceOf(EasyPayNotifyRejectedError);
    expect(guard).toMatchObject({ reason: 'unexpected_param', outTradeNo: 'order-001', param: 'clientip' });

    const badSign = await reject(
      new URLSearchParams({ ...STANDARD_NOTIFY, sign: '0'.repeat(32), sign_type: 'MD5' }).toString(),
    );
    expect(badSign).toMatchObject({ reason: 'bad_signature', outTradeNo: 'order-001' });

    expect(await reject(signedBody({ ...STANDARD_NOTIFY, pid: '2002' }))).toMatchObject({ reason: 'pid_mismatch' });
    expect(await reject(signedBody({ ...STANDARD_NOTIFY, money: '0' }))).toMatchObject({ reason: 'invalid_amount' });
  });
});
