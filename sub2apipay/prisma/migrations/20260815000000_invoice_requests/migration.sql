-- 发票（增值税普通发票）：开票申请 + 抬头记忆

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'InvoiceStatus') THEN
    CREATE TYPE "InvoiceStatus" AS ENUM ('PENDING', 'ISSUED', 'REJECTED', 'CANCELLED');
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS "invoice_requests" (
    "id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "user_id" INTEGER NOT NULL,
    "title_name" TEXT NOT NULL,
    "tax_no" TEXT NOT NULL,
    "remark" TEXT,
    "contact_email" TEXT,
    "amount" DECIMAL(10,2) NOT NULL,
    "status" "InvoiceStatus" NOT NULL DEFAULT 'PENDING',
    "file_key" TEXT,
    "file_name" TEXT,
    "file_size" INTEGER,
    "file_content_type" TEXT,
    "reject_reason" TEXT,
    "admin_note" TEXT,
    "issued_at" TIMESTAMP(3),
    "issued_by" TEXT,
    "rejected_at" TIMESTAMP(3),
    "notified_at" TIMESTAMP(3),
    "download_count" INTEGER NOT NULL DEFAULT 0,
    "last_downloaded_at" TIMESTAMP(3),
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "invoice_requests_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "invoice_titles" (
    "id" TEXT NOT NULL,
    "user_id" INTEGER NOT NULL,
    "title_name" TEXT NOT NULL,
    "tax_no" TEXT NOT NULL,
    "remark" TEXT,
    "contact_email" TEXT,
    "last_used_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "invoice_titles_pkey" PRIMARY KEY ("id")
);

-- 「一单一票」由唯一约束兜底：并发申请时后到者拿 P2002，映射为 409。
CREATE UNIQUE INDEX IF NOT EXISTS "invoice_requests_order_id_key" ON "invoice_requests"("order_id");
CREATE INDEX IF NOT EXISTS "invoice_requests_user_id_created_at_idx" ON "invoice_requests"("user_id", "created_at");
CREATE INDEX IF NOT EXISTS "invoice_requests_status_created_at_idx" ON "invoice_requests"("status", "created_at");

CREATE UNIQUE INDEX IF NOT EXISTS "invoice_titles_user_id_tax_no_key" ON "invoice_titles"("user_id", "tax_no");
CREATE INDEX IF NOT EXISTS "invoice_titles_user_id_last_used_at_idx" ON "invoice_titles"("user_id", "last_used_at");

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'invoice_requests_order_id_fkey'
  ) THEN
    ALTER TABLE "invoice_requests"
      ADD CONSTRAINT "invoice_requests_order_id_fkey"
      FOREIGN KEY ("order_id") REFERENCES "orders"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
  END IF;
END
$$;
