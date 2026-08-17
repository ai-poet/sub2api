-- 合并开票：一张发票可覆盖多张订单。
--
-- invoice_requests.order_id 保留为「主订单」（审计日志 / 通知邮件 / 管理端展示仍按它索引），
-- 完整清单落在新表里。已有发票回填一行，保证查询路径统一，无需在代码里区分新旧数据。

CREATE TABLE IF NOT EXISTS "invoice_request_orders" (
    "id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "amount" DECIMAL(10,2) NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "invoice_request_orders_pkey" PRIMARY KEY ("id")
);

-- 「一单只进一张有效发票」由唯一约束兜底：并发提交时后到者拿 P2002，映射为 409。
CREATE UNIQUE INDEX IF NOT EXISTS "invoice_request_orders_order_id_key" ON "invoice_request_orders"("order_id");
CREATE INDEX IF NOT EXISTS "invoice_request_orders_invoice_id_idx" ON "invoice_request_orders"("invoice_id");

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoice_request_orders_invoice_id_fkey'
  ) THEN
    ALTER TABLE "invoice_request_orders"
      ADD CONSTRAINT "invoice_request_orders_invoice_id_fkey"
      FOREIGN KEY ("invoice_id") REFERENCES "invoice_requests"("id") ON DELETE CASCADE ON UPDATE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoice_request_orders_order_id_fkey'
  ) THEN
    ALTER TABLE "invoice_request_orders"
      ADD CONSTRAINT "invoice_request_orders_order_id_fkey"
      FOREIGN KEY ("order_id") REFERENCES "orders"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
  END IF;
END
$$;

-- 回填存量发票：每张发票一行，金额取发票金额（此前一票一单，两者相等）。
INSERT INTO "invoice_request_orders" ("id", "invoice_id", "order_id", "amount", "created_at")
SELECT
    'ivo_' || "id",
    "id",
    "order_id",
    "amount",
    "created_at"
FROM "invoice_requests"
ON CONFLICT ("order_id") DO NOTHING;
