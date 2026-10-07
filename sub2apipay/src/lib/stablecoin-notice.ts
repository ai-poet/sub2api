import { isStablecoinPaymentType } from '@/lib/currency';
import type { Locale } from '@/lib/locale';
import { getPaymentDisplayInfo } from '@/lib/pay-utils';

/**
 * USDT / USDC 下单前的二次确认文案。链上收款只认按正确网络直接转到收款地址的转账，
 * 交易所站内转账、支付类功能、合约或跨链桥转出的钱，以及转错网络的钱，都可能不会自动到账。
 */
export interface StablecoinPaymentNotice {
  title: string;
  intro: string;
  points: { label: string; text: string }[];
  warning: string;
  cancelLabel: string;
  confirmLabel: string;
}

/** 非 USDT / USDC 支付方式返回 null，调用方据此决定是否弹确认框。 */
export function getStablecoinPaymentNotice(paymentType: string, locale: Locale): StablecoinPaymentNotice | null {
  if (!isStablecoinPaymentType(paymentType)) return null;

  const { channel: token, sublabel: network } = getPaymentDisplayInfo(paymentType, locale);

  if (locale === 'en') {
    return {
      title: `Before you pay with ${token}`,
      intro: 'Please make sure you pay as follows:',
      points: [
        {
          label: 'Direct transfer',
          text: `Send ${token} to the payment address with a regular transfer (Regular Transfer / withdrawal) from your wallet or exchange. Do not use exchange internal transfers, payment features such as Binance Pay, contract calls or cross-chain bridges.`,
        },
        {
          label: 'Correct network',
          text: network
            ? `Send on the ${network} network only. A transfer on any other network will not be detected.`
            : 'Send on the network shown on the payment page only. A transfer on any other network will not be detected.',
        },
      ],
      warning: 'Otherwise your payment may not be credited automatically.',
      cancelLabel: 'Back',
      confirmLabel: 'I understand, continue',
    };
  }

  return {
    title: `${token} 支付须知`,
    intro: '请确认你会按下面的方式付款：',
    points: [
      {
        label: '直接转账',
        text: `用钱包或交易所的直接转账（Regular Transfer / 提现）把 ${token} 转到收款地址。不要用交易所内部转账、Binance Pay 等支付功能，也不要通过合约调用或跨链桥转账。`,
      },
      {
        label: '正确网络',
        text: network
          ? `只能选择 ${network} 网络转账，选其他网络转账无法识别。`
          : '只能选择支付页面显示的网络转账，选其他网络转账无法识别。',
      },
    ],
    warning: '否则可能不会自动到账。',
    cancelLabel: '返回',
    confirmLabel: '我已了解，继续支付',
  };
}
