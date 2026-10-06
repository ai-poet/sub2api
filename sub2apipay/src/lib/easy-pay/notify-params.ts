/**
 * 易支付异步通知的参数名约定，供验签（provider.ts）与旁路观测（notify-audit.ts）共用。
 */

/** 彩虹易支付标准异步通知字段。 */
export const EASY_PAY_NOTIFY_FIELDS: ReadonlySet<string> = new Set([
  'pid',
  'trade_no',
  'out_trade_no',
  'type',
  'name',
  'money',
  'trade_status',
  'param',
  'sign',
  'sign_type',
]);

/** 本地路由参数：写在 notify_url 的 query 里，用来挑选支付实例，不参与验签。 */
export const EASY_PAY_ROUTING_PARAMS: ReadonlySet<string> = new Set(['inst']);

/**
 * 只出现在下单请求（mapi.php）里、真实异步通知从不回传的参数名。
 *
 * 易支付的 MD5 签名串是 `k=v&k=v…` 原样拼接、不做转义，同一段已签名的串可以按另一种方式切分成
 * 另一组参数而签名不变。我们的下单签名串按参数名排序后总以 `cid=` 或 `clientip=` 开头
 * （client.ts 保证 clientip 非空），参数名又不允许含 `=`，所以任何由下单签名派生出的回调都必然
 * 带有 `cid` 或 `clientip`。验签前拒绝这些参数名即可封住这一整类伪造，且不会误伤真实通知。
 *
 * 给下单请求新增参数时，若它按名字排序会排到最前面，必须同时加进这里（client.test.ts 钉住了这一点）。
 */
export const EASY_PAY_CREATE_ONLY_PARAMS: ReadonlySet<string> = new Set([
  'cid',
  'clientip',
  'device',
  'notify_url',
  'return_url',
]);

/**
 * 参数名只能是纯标识符。URLSearchParams 会解码参数名里的 %3D / %26，
 * 放任 `=` / `&` 出现在参数名里，上面"派生回调必带 cid / clientip"的结论就不再成立。
 */
const NOTIFY_PARAM_NAME = /^[A-Za-z0-9_]+$/;

/**
 * 回调参数名是否可以接受。只拒绝真实通知不可能出现的名字：非纯标识符、下单专有参数。
 *
 * 克隆平台多带的字段（buyer、addtime 等）照常放行，不做严格白名单：已到期 / 已取消订单的
 * 晚付款只能靠异步通知入账（周期对账只扫未到期的 PENDING 订单），误拒会让钱到了平台却无人补单。
 */
export function isAcceptableNotifyParamName(name: string): boolean {
  return NOTIFY_PARAM_NAME.test(name) && !EASY_PAY_CREATE_ONLY_PARAMS.has(name);
}
