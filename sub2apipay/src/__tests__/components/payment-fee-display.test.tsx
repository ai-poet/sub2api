import { describe, expect, it, vi } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import PaymentForm, { type MethodLimitInfo } from '@/components/PaymentForm';
import SubscriptionConfirm from '@/components/SubscriptionConfirm';
import PaymentQRCode from '@/components/PaymentQRCode';
import type { PlanInfo } from '@/components/SubscriptionPlanCard';

const feeLimit: MethodLimitInfo = {
  available: true,
  remaining: null,
  feeEnabled: true,
  feeRate: 3.9,
  fixedFee: 3.6,
};

const plan: PlanInfo = {
  id: 'plan-1',
  groupId: 1,
  groupName: 'Pro',
  name: 'Pro',
  price: 100,
  originalPrice: null,
  validityDays: 30,
  features: [],
  description: null,
  platform: null,
  rateMultiplier: null,
  limits: null,
  allowMessagesDispatch: false,
  defaultMappedModel: null,
};

describe('支付手续费前端展示', () => {
  it('余额充值使用共享反推结果并展示独立手续费行', () => {
    const html = renderToStaticMarkup(
      <PaymentForm
        userId={1}
        fixedAmount={100}
        enabledPaymentTypes={['alipay']}
        methodLimits={{ alipay: feeLimit }}
        minAmount={0}
        maxAmount={0}
        balanceCreditCnyPerUsd={1}
        onSubmit={vi.fn()}
        locale="zh"
      />,
    );
    expect(html).toContain('支付手续费');
    expect(html).toContain('3.9% + ¥3.60');
    expect(html).toContain('¥107.81');
    expect(html).toContain('¥7.81');
  });

  it('未输入充值金额时不提前展示固定手续费', () => {
    const html = renderToStaticMarkup(
      <PaymentForm
        userId={1}
        enabledPaymentTypes={['alipay']}
        methodLimits={{ alipay: feeLimit }}
        minAmount={0}
        maxAmount={0}
        balanceCreditCnyPerUsd={1}
        onSubmit={vi.fn()}
        locale="zh"
      />,
    );
    expect(html).toContain('立即支付 ¥0.00');
    expect(html).not.toContain('立即支付 ¥3.60');
    expect(html).not.toContain('支付手续费');
  });

  it('套餐确认页与余额充值使用同一费用口径', () => {
    const html = renderToStaticMarkup(
      <SubscriptionConfirm
        plan={plan}
        paymentTypes={['alipay']}
        methodLimits={{ alipay: feeLimit }}
        onBack={vi.fn()}
        onSubmit={vi.fn()}
        loading={false}
        isDark={false}
        locale="zh"
      />,
    );
    expect(html).toContain('支付手续费');
    expect(html).toContain('¥107.81');
    expect(html).toContain('¥7.81');
  });

  it('支付页使用下单返回的手续费快照', () => {
    const html = renderToStaticMarkup(
      <PaymentQRCode
        orderId="order-1"
        paymentType="alipay"
        amount={100}
        payAmount={107.81}
        feeAmount={7.81}
        orderType="balance"
        expiresAt={new Date(Date.now() + 60000).toISOString()}
        onStatusChange={vi.fn()}
        onBack={vi.fn()}
        locale="zh"
      />,
    );
    expect(html).toContain('支付手续费');
    expect(html).toContain('¥107.81');
    expect(html).toContain('¥7.81');
  });
});
