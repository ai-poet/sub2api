export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    const { startTimeoutScheduler } = await import('@/lib/order/timeout');
    const { startReconcileScheduler } = await import('@/lib/order/reconcile');
    startTimeoutScheduler();
    // 待支付订单周期对账：异步通知丢失时一分钟内自愈，不再依赖用户停留在页面上
    startReconcileScheduler();
  }
}
