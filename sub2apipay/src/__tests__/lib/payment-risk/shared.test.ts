import { describe, expect, it } from 'vitest';
import fs from 'fs';
import path from 'path';
import {
  isPaymentRiskAction,
  PAYMENT_AMOUNT_MISMATCH,
  PAYMENT_CONFIRM_FAILED,
  PAYMENT_NOTIFY_REJECTED,
  PAYMENT_RISK_ACTIONS,
  PAYMENT_RISK_ACTION_META,
  PAYMENT_RISK_REASON_LABELS,
} from '@/lib/payment-risk/shared';

describe('payment risk shared definitions', () => {
  it('每类风险事件都有严重程度、配色和中英文文案', () => {
    for (const action of PAYMENT_RISK_ACTIONS) {
      const meta = PAYMENT_RISK_ACTION_META[action];
      expect(['high', 'medium', 'low']).toContain(meta.severity);
      expect(meta.color).toMatch(/^#[0-9a-f]{6}$/i);
      expect(meta.label.zh && meta.label.en).toBeTruthy();
      expect(meta.description.zh && meta.description.en).toBeTruthy();
    }
  });

  it('审计和复核里会出现的每种原因都有中英文文案', () => {
    const reasons = [
      'unexpected_param',
      'bad_signature',
      'pid_mismatch',
      'invalid_amount',
      'non_positive_amount',
      'missing_trade_no',
      'trade_no_mismatch',
      'upstream_not_paid',
      'upstream_order_not_found',
      'upstream_amount_mismatch',
      'upstream_trade_no_mismatch',
      'provider_unavailable',
    ];
    for (const reason of reasons) {
      expect(PAYMENT_RISK_REASON_LABELS[reason]?.zh).toBeTruthy();
      expect(PAYMENT_RISK_REASON_LABELS[reason]?.en).toBeTruthy();
    }
  });

  it('isPaymentRiskAction 只认风险事件', () => {
    expect(isPaymentRiskAction('PAYMENT_UPSTREAM_MISMATCH')).toBe(true);
    expect(isPaymentRiskAction('ORDER_PAID')).toBe(false);
    expect(isPaymentRiskAction('')).toBe(false);
  });

  // 这三类审计由入账与到期扫描用字面量写入；改名时风控面板会悄悄统计不到，这里钉住
  it('入账与到期扫描写入的审计动作名与风控面板一致', () => {
    const read = (file: string) => fs.readFileSync(path.resolve(__dirname, '../../../lib/order', file), 'utf8');
    const service = read('service.ts');
    const timeout = read('timeout.ts');

    expect(service).toContain(`action: '${PAYMENT_NOTIFY_REJECTED}'`);
    expect(service).toContain(`action: '${PAYMENT_AMOUNT_MISMATCH}'`);
    expect(timeout).toContain(`action: '${PAYMENT_CONFIRM_FAILED}'`);
  });
});
