-- 充值活动（充 X 送 Y）：活动表 + 订单上的赠送/活动快照。
--
-- 赠送额在履约时与 amount 一起入账，退款时一并扣回；orders.promotion_name 是下单时的名称快照，
-- 活动删除后（promotion_id 置空）订单仍能展示当时参与的活动。

CREATE TABLE IF NOT EXISTS "recharge_promotions" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "min_amount" DECIMAL(10,2) NOT NULL,
    "bonus_type" TEXT NOT NULL DEFAULT 'fixed',
    "bonus_value" DECIMAL(10,2) NOT NULL,
    "max_bonus" DECIMAL(10,2),
    "starts_at" TIMESTAMP(3),
    "ends_at" TIMESTAMP(3),
    "per_user_limit" INTEGER NOT NULL DEFAULT 0,
    "total_limit" INTEGER NOT NULL DEFAULT 0,
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "recharge_promotions_pkey" PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "recharge_promotions_enabled_sort_order_idx"
  ON "recharge_promotions"("enabled", "sort_order");

ALTER TABLE "orders" ADD COLUMN IF NOT EXISTS "bonus_amount" DECIMAL(10,2);
ALTER TABLE "orders" ADD COLUMN IF NOT EXISTS "promotion_id" TEXT;
ALTER TABLE "orders" ADD COLUMN IF NOT EXISTS "promotion_name" TEXT;

CREATE INDEX IF NOT EXISTS "orders_promotion_id_idx" ON "orders"("promotion_id");

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'orders_promotion_id_fkey'
  ) THEN
    ALTER TABLE "orders"
      ADD CONSTRAINT "orders_promotion_id_fkey"
      FOREIGN KEY ("promotion_id") REFERENCES "recharge_promotions"("id")
      ON DELETE SET NULL ON UPDATE CASCADE;
  END IF;
END
$$;
