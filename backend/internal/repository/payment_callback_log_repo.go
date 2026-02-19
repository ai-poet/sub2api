package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type paymentCallbackLogRepository struct {
	db *sql.DB
}

func NewPaymentCallbackLogRepository(db *sql.DB) service.PaymentCallbackLogRepository {
	return &paymentCallbackLogRepository{db: db}
}

func (r *paymentCallbackLogRepository) Create(ctx context.Context, callbackLog *service.PaymentCallbackLog) error {
	rawDataJSON, err := json.Marshal(callbackLog.RawData)
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
		callbackLog.OrderNo,
		callbackLog.Provider,
		rawDataJSON,
		callbackLog.Signature,
		callbackLog.Verified,
		callbackLog.Processed,
		callbackLog.ResultMessage,
		callbackLog.ClientIP,
		callbackLog.CreatedAt,
	).Scan(&callbackLog.ID)
}

func (r *paymentCallbackLogRepository) GetByOrderNo(ctx context.Context, orderNo string) ([]service.PaymentCallbackLog, error) {
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

	var logs []service.PaymentCallbackLog
	for rows.Next() {
		var callbackLog service.PaymentCallbackLog
		var rawDataJSON []byte
		var processedAt sql.NullTime
		var signature, resultMessage, clientIP sql.NullString

		err := rows.Scan(
			&callbackLog.ID,
			&callbackLog.OrderNo,
			&callbackLog.Provider,
			&rawDataJSON,
			&signature,
			&callbackLog.Verified,
			&callbackLog.Processed,
			&resultMessage,
			&clientIP,
			&callbackLog.CreatedAt,
			&processedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(rawDataJSON, &callbackLog.RawData); err != nil {
			callbackLog.RawData = make(map[string]string)
		}

		if signature.Valid {
			callbackLog.Signature = &signature.String
		}
		if resultMessage.Valid {
			callbackLog.ResultMessage = &resultMessage.String
		}
		if clientIP.Valid {
			callbackLog.ClientIP = &clientIP.String
		}
		if processedAt.Valid {
			callbackLog.ProcessedAt = &processedAt.Time
		}

		logs = append(logs, callbackLog)
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
