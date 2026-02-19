-- 商品表
CREATE TABLE shop_products (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description TEXT,
  price DECIMAL(10,2) NOT NULL,
  currency VARCHAR(10) DEFAULT 'CNY',
  redeem_type VARCHAR(20) NOT NULL,
  redeem_value DECIMAL(20,8) DEFAULT 0,
  group_id BIGINT,
  validity_days INT DEFAULT 30,
  stock_count INT DEFAULT 0,
  is_active BOOLEAN DEFAULT true,
  sort_order INT DEFAULT 0,
  creem_product_id VARCHAR(100) DEFAULT '',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 商品库存表
CREATE TABLE shop_product_stocks (
  id BIGSERIAL PRIMARY KEY,
  product_id BIGINT NOT NULL REFERENCES shop_products(id),
  redeem_code_id BIGINT NOT NULL REFERENCES redeem_codes(id),
  status VARCHAR(20) DEFAULT 'available',
  order_id BIGINT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_shop_product_stocks_product_status ON shop_product_stocks(product_id, status);

-- 订单表
CREATE TABLE shop_orders (
  id BIGSERIAL PRIMARY KEY,
  order_no VARCHAR(64) UNIQUE NOT NULL,
  user_id BIGINT NOT NULL REFERENCES users(id),
  product_id BIGINT NOT NULL REFERENCES shop_products(id),
  product_name VARCHAR(100) NOT NULL,
  amount DECIMAL(10,2) NOT NULL,
  currency VARCHAR(10) DEFAULT 'CNY',
  payment_method VARCHAR(20),
  status VARCHAR(20) DEFAULT 'pending',
  redeem_code_id BIGINT,
  paid_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_shop_orders_user_status ON shop_orders(user_id, status);
CREATE INDEX idx_shop_orders_order_no ON shop_orders(order_no);
