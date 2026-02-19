package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// PaymentCallbackLog 支付回调日志
type PaymentCallbackLog struct {
	ID             int64
	OrderNo        string
	Provider       string // epay, creem
	RawData        map[string]string
	Signature      *string
	Verified       bool
	Processed      bool
	ResultMessage  *string
	ClientIP       *string
	CreatedAt      time.Time
	ProcessedAt    *time.Time
}

// PaymentCallbackLogRepository 支付回调日志仓库接口
type PaymentCallbackLogRepository interface {
	Create(ctx context.Context, log *PaymentCallbackLog) error
	GetByOrderNo(ctx context.Context, orderNo string) ([]PaymentCallbackLog, error)
	UpdateProcessed(ctx context.Context, id int64, processed bool, resultMessage string) error
}

type paymentCallbackLogRepository struct {
	db *sql.DB
}

func NewPaymentCallbackLogRepository(db *sql.DB) PaymentCallbackLogRepository {
	return &paymentCallbackLogRepository{db: db}
}

func (r *paymentCallbackLogRepository) Create(ctx context.Context, log *PaymentCallbackLog) error {
	rawDataJSON, err := json.Marshal(log.RawData)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO payment_callback_logs (
			order_no, provider, raw_data, signature, verified, 
			processed, result_message, client_ip, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return r.db.QueryRowContext(ctx, query,
		log.OrderNo,
		log.Provider,
		rawDataJSON,
		log.Signature,
		log.Verified,
		log.Processed,
		log.ResultMessage,
		log.ClientIP,
		log.CreatedAt,
	).Scan(&log.ID)
}

func (r *paymentCallbackLogRepository) GetByOrderNo(ctx context.Context, orderNo string) ([]PaymentCallbackLog, error) {
	query := `
		SELECT id, order_no, provider, raw_data, signature, verified,
			   processed, result_message, client_ip, created_at, processed_at
		FROM payment_callback_logs
		WHERE order_no = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, orderNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []PaymentCallbackLog
	for rows.Next() {
		var log PaymentCallbackLog
		var rawDataJSON []byte
		var processedAt sql.NullTime
		var signature, resultMessage, clientIP sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.OrderNo,
			&log.Provider,
			&rawDataJSON,
			&signature,
			&log.Verified,
			&log.Processed,
			&resultMessage,
			&clientIP,
			&log.CreatedAt,
			&processedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(rawDataJSON, &log.RawData); err != nil {
			log.RawData = make(map[string]string)
		}

		if signature.Valid {
			log.Signature = &signature.String
		}
		if resultMessage.Valid {
			log.ResultMessage = &resultMessage.String
		}
		if clientIP.Valid {
			log.ClientIP = &clientIP.String
		}
		if processedAt.Valid {
			log.ProcessedAt = &processedAt.Time
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

func (r *paymentCallbackLogRepository) UpdateProcessed(ctx context.Context, id int64, processed bool, resultMessage string) error {
	query := `
		UPDATE payment_callback_logs
		SET processed = $1, result_message = $2, processed_at = $3
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, processed, resultMessage, time.Now(), id)
	return err
}
