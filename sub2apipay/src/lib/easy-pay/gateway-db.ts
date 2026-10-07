import type { Pool, RowDataPacket } from 'mysql2/promise';
import { getEnv } from '@/lib/config';

/**
 * 自建易支付网关的只读数据库查询，用于平台复核的兜底。
 *
 * 彩虹易支付的商户查单接口按"订单号 + 商户号"找单：实例换过商户号之后，挂在这个实例上的
 * 老订单就会被回"订单号不存在"，复核把真实付过款的订单误判为伪造。网关是自建的时候，
 * 可以直接按订单号查网关库来确认。
 *
 * - 只在配置了 EASY_PAY_GATEWAY_DB_URL 时启用；连接串只从环境变量读取，不落库、不打印。
 * - 每次查询都在 START TRANSACTION READ ONLY 里执行并回滚；建议再给这个账号只授予 SELECT。
 * - 只读订单表 pay_order 的五个字段。
 */

/** 易支付网关的订单表，固定不变。 */
const GATEWAY_ORDER_TABLE = 'pay_order';
const QUERY_TIMEOUT_MS = 10_000;

export interface GatewayOrder {
  outTradeNo: string;
  tradeNo: string;
  merchantId: string;
  amount: number;
  /** 网关库里的原始状态值，彩虹易支付 1 表示已支付 */
  status: string;
  paid: boolean;
}

let pool: Pool | null = null;
let poolUrl: string | null = null;

export function isGatewayDbConfigured(): boolean {
  return Boolean(getEnv().EASY_PAY_GATEWAY_DB_URL);
}

/** 配置的商户号列表；未配置时返回 null，表示网关库里任何商户的订单都接受。 */
export function gatewayMerchantIds(): Set<string> | null {
  const raw = getEnv().EASY_PAY_GATEWAY_MERCHANT_IDS;
  if (!raw) return null;
  const ids = raw
    .split(',')
    .map((id) => id.trim())
    .filter(Boolean);
  return ids.length > 0 ? new Set(ids) : null;
}

async function getPool(url: string): Promise<Pool> {
  if (pool && poolUrl === url) return pool;
  if (pool) {
    await pool.end().catch(() => undefined);
  }
  // 按需加载：没配置网关库的部署不会加载 MySQL 驱动
  const mysql = await import('mysql2/promise');
  pool = mysql.createPool({
    uri: url,
    connectionLimit: 2,
    queueLimit: 20,
    connectTimeout: QUERY_TIMEOUT_MS,
    dateStrings: true,
  });
  poolUrl = url;
  return pool;
}

/**
 * 按我们的订单号（即网关的 out_trade_no）查网关订单。查不到返回 null；
 * 同一个订单号在网关里对应多行时无法判定，抛错交给调用方按"查单失败"处理。
 */
export async function findGatewayOrder(outTradeNo: string): Promise<GatewayOrder | null> {
  const url = getEnv().EASY_PAY_GATEWAY_DB_URL;
  if (!url) {
    throw new Error('EASY_PAY_GATEWAY_DB_URL is not configured');
  }
  const connection = await (await getPool(url)).getConnection();
  try {
    await connection.query('START TRANSACTION READ ONLY');
    const [rows] = await connection.query<RowDataPacket[]>(
      {
        sql: `SELECT trade_no, out_trade_no, uid, money, status FROM ${GATEWAY_ORDER_TABLE} WHERE out_trade_no = ? LIMIT 2`,
        timeout: QUERY_TIMEOUT_MS,
      },
      [outTradeNo],
    );
    if (rows.length === 0) {
      return null;
    }
    if (rows.length > 1) {
      throw new Error(`gateway order ${outTradeNo} matches more than one row`);
    }
    const row = rows[0];
    const status = String(row.status ?? '');
    return {
      outTradeNo: String(row.out_trade_no ?? ''),
      tradeNo: String(row.trade_no ?? ''),
      merchantId: String(row.uid ?? ''),
      amount: Number(row.money),
      status,
      paid: status === '1',
    };
  } finally {
    await connection.query('ROLLBACK').catch(() => undefined);
    connection.release();
  }
}

/** 仅供测试：丢弃缓存的连接池。 */
export async function resetGatewayDbPoolForTests(): Promise<void> {
  if (pool) {
    await pool.end().catch(() => undefined);
  }
  pool = null;
  poolUrl = null;
}
