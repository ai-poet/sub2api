import { beforeEach, describe, expect, it, vi } from 'vitest';

const env: Record<string, string | undefined> = {};

vi.mock('@/lib/config', () => ({
  getEnv: () => env,
}));

const mockQuery = vi.fn();
const mockRelease = vi.fn();
const mockPoolEnd = vi.fn();
const mockCreatePool = vi.fn();

vi.mock('mysql2/promise', () => ({
  createPool: (...args: unknown[]) => mockCreatePool(...args),
}));

import {
  findGatewayOrder,
  gatewayMerchantIds,
  isGatewayDbConfigured,
  resetGatewayDbPoolForTests,
} from '@/lib/easy-pay/gateway-db';

const GATEWAY_URL = 'mysql://readonly:pw@gateway.example.com:3306/epay';

function selectCalls() {
  return mockQuery.mock.calls.filter(([query]) => typeof query === 'object');
}

describe('gateway-db', () => {
  beforeEach(async () => {
    mockPoolEnd.mockResolvedValue(undefined);
    await resetGatewayDbPoolForTests();
    vi.clearAllMocks();
    env.EASY_PAY_GATEWAY_DB_URL = GATEWAY_URL;
    env.EASY_PAY_GATEWAY_MERCHANT_IDS = undefined;
    mockCreatePool.mockReturnValue({
      getConnection: () => Promise.resolve({ query: mockQuery, release: mockRelease }),
      end: mockPoolEnd,
    });
    mockQuery.mockImplementation((query: unknown) =>
      typeof query === 'object' ? Promise.resolve([[]]) : Promise.resolve([{}]),
    );
  });

  it('只有配置了连接串才启用', () => {
    expect(isGatewayDbConfigured()).toBe(true);
    env.EASY_PAY_GATEWAY_DB_URL = undefined;
    expect(isGatewayDbConfigured()).toBe(false);
  });

  it('解析商户号列表，未配置时返回 null', () => {
    expect(gatewayMerchantIds()).toBeNull();
    env.EASY_PAY_GATEWAY_MERCHANT_IDS = ' 1001, 1002 ,, ';
    expect(gatewayMerchantIds()).toEqual(new Set(['1001', '1002']));
    env.EASY_PAY_GATEWAY_MERCHANT_IDS = ' , ';
    expect(gatewayMerchantIds()).toBeNull();
  });

  it('在只读事务里按订单号查 pay_order，结束后回滚并归还连接', async () => {
    mockQuery.mockImplementation((query: unknown) =>
      typeof query === 'object'
        ? Promise.resolve([[{ trade_no: '2026091010411597608', out_trade_no: 'order-1', uid: 1001, money: '10.00', status: 1 }]])
        : Promise.resolve([{}]),
    );

    const order = await findGatewayOrder('order-1');

    expect(order).toEqual({
      outTradeNo: 'order-1',
      tradeNo: '2026091010411597608',
      merchantId: '1001',
      amount: 10,
      status: '1',
      paid: true,
    });
    expect(mockQuery.mock.calls.map(([query]) => (typeof query === 'string' ? query : 'SELECT'))).toEqual([
      'START TRANSACTION READ ONLY',
      'SELECT',
      'ROLLBACK',
    ]);
    const [[select, params]] = selectCalls();
    expect(select.sql).toBe(
      'SELECT trade_no, out_trade_no, uid, money, status FROM pay_order WHERE out_trade_no = ? LIMIT 2',
    );
    expect(select.timeout).toBe(10_000);
    expect(params).toEqual(['order-1']);
    expect(mockRelease).toHaveBeenCalledTimes(1);
  });

  it('未支付的订单 paid 为 false', async () => {
    mockQuery.mockImplementation((query: unknown) =>
      typeof query === 'object'
        ? Promise.resolve([[{ trade_no: 'T', out_trade_no: 'order-1', uid: 1001, money: '10.00', status: 0 }]])
        : Promise.resolve([{}]),
    );

    expect(await findGatewayOrder('order-1')).toMatchObject({ status: '0', paid: false });
  });

  it('查不到返回 null', async () => {
    expect(await findGatewayOrder('missing')).toBeNull();
    expect(mockRelease).toHaveBeenCalledTimes(1);
  });

  it('同一订单号对应多行时无法判定，抛错，但仍回滚并归还连接', async () => {
    mockQuery.mockImplementation((query: unknown) =>
      typeof query === 'object'
        ? Promise.resolve([[{ out_trade_no: 'order-1' }, { out_trade_no: 'order-1' }]])
        : Promise.resolve([{}]),
    );

    await expect(findGatewayOrder('order-1')).rejects.toThrow('more than one row');
    expect(mockQuery).toHaveBeenLastCalledWith('ROLLBACK');
    expect(mockRelease).toHaveBeenCalledTimes(1);
  });

  it('连接池只建一次，连接串变化时重建', async () => {
    await findGatewayOrder('a');
    await findGatewayOrder('b');
    expect(mockCreatePool).toHaveBeenCalledTimes(1);
    expect(mockCreatePool).toHaveBeenCalledWith(
      expect.objectContaining({ uri: GATEWAY_URL, connectionLimit: 2, connectTimeout: 10_000 }),
    );

    env.EASY_PAY_GATEWAY_DB_URL = 'mysql://readonly:pw@other.example.com:3306/epay';
    await findGatewayOrder('c');
    expect(mockPoolEnd).toHaveBeenCalledTimes(1);
    expect(mockCreatePool).toHaveBeenCalledTimes(2);
  });

  it('未配置连接串时调用直接报错，不连库', async () => {
    env.EASY_PAY_GATEWAY_DB_URL = undefined;

    await expect(findGatewayOrder('order-1')).rejects.toThrow('not configured');
    expect(mockCreatePool).not.toHaveBeenCalled();
  });
});
