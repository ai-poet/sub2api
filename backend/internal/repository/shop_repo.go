package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// shopSQLTxKey is the context key for passing *sql.Tx to shop repositories
type shopSQLTxKey = service.ShopSQLTxKey

// ShopTxContext injects a *sql.Tx into context for shop repositories.
func ShopTxContext(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, shopSQLTxKey{}, tx)
}

type execer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func shopExecer(ctx context.Context, db *sql.DB) execer {
	if tx, ok := ctx.Value(shopSQLTxKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return db
}

// --- ShopProduct ---

type shopProductRepository struct{ db *sql.DB }

func NewShopProductRepository(db *sql.DB) service.ShopProductRepository {
	return &shopProductRepository{db: db}
}

func (r *shopProductRepository) Create(ctx context.Context, p *service.ShopProduct) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO shop_products (name,description,price,currency,redeem_type,redeem_value,group_id,validity_days,stock_count,is_active,sort_order,creem_product_id,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW()) RETURNING id,created_at,updated_at`,
		p.Name, p.Description, p.Price, p.Currency, p.RedeemType, p.RedeemValue,
		p.GroupID, p.ValidityDays, p.StockCount, p.IsActive, p.SortOrder, p.CreemProductID,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *shopProductRepository) Update(ctx context.Context, p *service.ShopProduct) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE shop_products SET name=$1,description=$2,price=$3,currency=$4,redeem_type=$5,redeem_value=$6,
group_id=$7,validity_days=$8,is_active=$9,sort_order=$10,creem_product_id=$11,updated_at=NOW() WHERE id=$12`,
		p.Name, p.Description, p.Price, p.Currency, p.RedeemType, p.RedeemValue,
		p.GroupID, p.ValidityDays, p.IsActive, p.SortOrder, p.CreemProductID, p.ID)
	return err
}

func (r *shopProductRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM shop_products WHERE id=$1`, id)
	return err
}

func (r *shopProductRepository) GetByID(ctx context.Context, id int64) (*service.ShopProduct, error) {
	p := &service.ShopProduct{}
	err := r.db.QueryRowContext(ctx, `
SELECT id,name,description,price,currency,redeem_type,redeem_value,group_id,validity_days,stock_count,is_active,sort_order,creem_product_id,created_at,updated_at
FROM shop_products WHERE id=$1`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.Currency, &p.RedeemType, &p.RedeemValue,
		&p.GroupID, &p.ValidityDays, &p.StockCount, &p.IsActive, &p.SortOrder, &p.CreemProductID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrProductNotFound
	}
	return p, err
}

func (r *shopProductRepository) List(ctx context.Context, activeOnly bool) ([]service.ShopProduct, error) {
	q := `SELECT id,name,description,price,currency,redeem_type,redeem_value,group_id,validity_days,stock_count,is_active,sort_order,creem_product_id,created_at,updated_at FROM shop_products`
	if activeOnly {
		q += ` WHERE is_active=true`
	}
	q += ` ORDER BY sort_order ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.ShopProduct
	for rows.Next() {
		var p service.ShopProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Currency, &p.RedeemType, &p.RedeemValue,
			&p.GroupID, &p.ValidityDays, &p.StockCount, &p.IsActive, &p.SortOrder, &p.CreemProductID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *shopProductRepository) UpdateStockCount(ctx context.Context, id int64, delta int) error {
	_, err := shopExecer(ctx, r.db).ExecContext(ctx, `UPDATE shop_products SET stock_count=stock_count+$1,updated_at=NOW() WHERE id=$2`, delta, id)
	return err
}

// --- ShopProductStock ---

type shopProductStockRepository struct{ db *sql.DB }

func NewShopProductStockRepository(db *sql.DB) service.ShopProductStockRepository {
	return &shopProductStockRepository{db: db}
}

func (r *shopProductStockRepository) CreateBatch(ctx context.Context, stocks []service.ShopProductStock) error {
	if len(stocks) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO shop_product_stocks (product_id,redeem_code_id,status,created_at) VALUES ($1,$2,$3,NOW())`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, s := range stocks {
		if _, err := stmt.ExecContext(ctx, s.ProductID, s.RedeemCodeID, s.Status); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *shopProductStockRepository) ListByProduct(ctx context.Context, productID int64) ([]service.ShopProductStock, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id,product_id,redeem_code_id,status,order_id,created_at FROM shop_product_stocks WHERE product_id=$1 ORDER BY id ASC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.ShopProductStock
	for rows.Next() {
		var s service.ShopProductStock
		if err := rows.Scan(&s.ID, &s.ProductID, &s.RedeemCodeID, &s.Status, &s.OrderID, &s.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *shopProductStockRepository) Delete(ctx context.Context, id int64) (productID int64, deleted bool, err error) {
	err = shopExecer(ctx, r.db).QueryRowContext(ctx, `
	DELETE FROM shop_product_stocks
	WHERE id=$1 AND status='available'
	RETURNING product_id`, id).Scan(&productID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return productID, true, nil
}

func (r *shopProductStockRepository) TakeOne(ctx context.Context, productID int64, orderID int64) (*service.ShopProductStock, error) {
	s := &service.ShopProductStock{}
	err := shopExecer(ctx, r.db).QueryRowContext(ctx, `
UPDATE shop_product_stocks SET status='sold', order_id=$1
WHERE id = (
  SELECT id FROM shop_product_stocks WHERE product_id=$2 AND status='available' ORDER BY created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED
)
RETURNING id,product_id,redeem_code_id,status,order_id,created_at`, orderID, productID).Scan(
		&s.ID, &s.ProductID, &s.RedeemCodeID, &s.Status, &s.OrderID, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrProductOutOfStock
	}
	return s, err
}

func (r *shopProductStockRepository) CountAvailable(ctx context.Context, productID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shop_product_stocks WHERE product_id=$1 AND status='available'`, productID).Scan(&count)
	return count, err
}

// --- ShopOrder ---

type shopOrderRepository struct{ db *sql.DB }

func NewShopOrderRepository(db *sql.DB) service.ShopOrderRepository {
	return &shopOrderRepository{db: db}
}

func (r *shopOrderRepository) Create(ctx context.Context, o *service.ShopOrder) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO shop_orders (order_no,user_id,product_id,product_name,amount,currency,payment_method,status,expires_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW()) RETURNING id,created_at,updated_at`,
		o.OrderNo, o.UserID, o.ProductID, o.ProductName, o.Amount, o.Currency,
		o.PaymentMethod, o.Status, o.ExpiresAt,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (r *shopOrderRepository) Update(ctx context.Context, o *service.ShopOrder) error {
	_, err := shopExecer(ctx, r.db).ExecContext(ctx, `
UPDATE shop_orders SET status=$1,redeem_code_id=$2,paid_at=$3,updated_at=NOW() WHERE id=$4`,
		o.Status, o.RedeemCodeID, o.PaidAt, o.ID)
	return err
}

func (r *shopOrderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*service.ShopOrder, error) {
	o := &service.ShopOrder{}
	err := shopExecer(ctx, r.db).QueryRowContext(ctx, `
	SELECT id,order_no,user_id,product_id,product_name,amount,currency,payment_method,status,redeem_code_id,paid_at,expires_at,created_at,updated_at
	FROM shop_orders WHERE order_no=$1`, orderNo).Scan(
		&o.ID, &o.OrderNo, &o.UserID, &o.ProductID, &o.ProductName, &o.Amount, &o.Currency,
		&o.PaymentMethod, &o.Status, &o.RedeemCodeID, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrOrderNotFound
	}
	return o, err
}

func (r *shopOrderRepository) GetByOrderNoForUpdate(ctx context.Context, orderNo string) (*service.ShopOrder, error) {
	o := &service.ShopOrder{}
	err := shopExecer(ctx, r.db).QueryRowContext(ctx, `
	SELECT id,order_no,user_id,product_id,product_name,amount,currency,payment_method,status,redeem_code_id,paid_at,expires_at,created_at,updated_at
	FROM shop_orders
	WHERE order_no=$1
	FOR UPDATE`, orderNo).Scan(
		&o.ID, &o.OrderNo, &o.UserID, &o.ProductID, &o.ProductName, &o.Amount, &o.Currency,
		&o.PaymentMethod, &o.Status, &o.RedeemCodeID, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrOrderNotFound
	}
	return o, err
}

func (r *shopOrderRepository) ListByUser(ctx context.Context, userID int64) ([]service.ShopOrder, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id,order_no,user_id,product_id,product_name,amount,currency,payment_method,status,redeem_code_id,paid_at,expires_at,created_at,updated_at
FROM shop_orders WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.ShopOrder
	for rows.Next() {
		var o service.ShopOrder
		if err := rows.Scan(&o.ID, &o.OrderNo, &o.UserID, &o.ProductID, &o.ProductName, &o.Amount, &o.Currency,
			&o.PaymentMethod, &o.Status, &o.RedeemCodeID, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

func (r *shopOrderRepository) ListByUserAndStatus(ctx context.Context, userID int64, status string) ([]service.ShopOrder, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id,order_no,user_id,product_id,product_name,amount,currency,payment_method,status,redeem_code_id,paid_at,expires_at,created_at,updated_at
FROM shop_orders WHERE user_id=$1 AND status=$2 ORDER BY created_at DESC`, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.ShopOrder
	for rows.Next() {
		var o service.ShopOrder
		if err := rows.Scan(&o.ID, &o.OrderNo, &o.UserID, &o.ProductID, &o.ProductName, &o.Amount, &o.Currency,
			&o.PaymentMethod, &o.Status, &o.RedeemCodeID, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}
