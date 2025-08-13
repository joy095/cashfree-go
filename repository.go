package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// CreatePayment creates a new payment record
func (r *PaymentRepository) CreatePayment(ctx context.Context, payment *Payment) error {
	query := `
		INSERT INTO payments (
			id, order_id, cf_order_id, amount, currency, status,
			customer_id, customer_name, customer_email, customer_phone,
			description, payment_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	now := time.Now()
	payment.ID = uuid.New()
	payment.CreatedAt = now
	payment.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		payment.ID, payment.OrderID, payment.CFOrderID, payment.Amount,
		payment.Currency, payment.Status, payment.CustomerID, payment.CustomerName,
		payment.CustomerEmail, payment.CustomerPhone, payment.Description,
		payment.PaymentURL, payment.CreatedAt, payment.UpdatedAt,
	)

	return err
}

// GetPaymentByOrderID retrieves a payment by order ID
func (r *PaymentRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	query := `
		SELECT id, order_id, cf_order_id, amount, currency, status,
			   payment_method, customer_id, customer_name, customer_email,
			   customer_phone, description, payment_url, cf_payment_id,
			   payment_time, created_at, updated_at
		FROM payments
		WHERE order_id = $1
	`

	var payment Payment
	row := r.db.QueryRow(ctx, query, orderID)

	err := row.Scan(
		&payment.ID, &payment.OrderID, &payment.CFOrderID, &payment.Amount,
		&payment.Currency, &payment.Status, &payment.PaymentMethod,
		&payment.CustomerID, &payment.CustomerName, &payment.CustomerEmail,
		&payment.CustomerPhone, &payment.Description, &payment.PaymentURL,
		&payment.CFPaymentID, &payment.PaymentTime, &payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("payment not found for order_id: %s", orderID)
		}
		return nil, err
	}

	return &payment, nil
}

// UpdatePaymentStatus updates payment status and related fields
func (r *PaymentRepository) UpdatePaymentStatus(ctx context.Context, orderID, status string, cfPaymentID *string, paymentMethod *string, paymentTime *time.Time) error {
	query := `
		UPDATE payments 
		SET status = $1, cf_payment_id = $2, payment_method = $3, 
			payment_time = $4, updated_at = $5
		WHERE order_id = $6
	`

	_, err := r.db.Exec(ctx, query, status, cfPaymentID, paymentMethod, paymentTime, time.Now(), orderID)
	return err
}

// GetAllPayments retrieves all payments with pagination
func (r *PaymentRepository) GetAllPayments(ctx context.Context, limit, offset int) ([]Payment, error) {
	query := `
		SELECT id, order_id, cf_order_id, amount, currency, status,
			   payment_method, customer_id, customer_name, customer_email,
			   customer_phone, description, payment_url, cf_payment_id,
			   payment_time, created_at, updated_at
		FROM payments
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var payment Payment
		err := rows.Scan(
			&payment.ID, &payment.OrderID, &payment.CFOrderID, &payment.Amount,
			&payment.Currency, &payment.Status, &payment.PaymentMethod,
			&payment.CustomerID, &payment.CustomerName, &payment.CustomerEmail,
			&payment.CustomerPhone, &payment.Description, &payment.PaymentURL,
			&payment.CFPaymentID, &payment.PaymentTime, &payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, payment)
	}

	return payments, rows.Err()
}

// CreateRefund creates a new refund record
func (r *PaymentRepository) CreateRefund(ctx context.Context, refund *Refund) error {
	query := `
		INSERT INTO refunds (
			id, refund_id, cf_refund_id, order_id, cf_order_id, amount,
			status, reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	refund.ID = uuid.New()
	refund.CreatedAt = now
	refund.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		refund.ID, refund.RefundID, refund.CFRefundID, refund.OrderID,
		refund.CFOrderID, refund.Amount, refund.Status, refund.Reason,
		refund.CreatedAt, refund.UpdatedAt,
	)

	return err
}

// UpdateRefundStatus updates refund status
func (r *PaymentRepository) UpdateRefundStatus(ctx context.Context, refundID, status string, processedAt *time.Time) error {
	query := `
		UPDATE refunds 
		SET status = $1, processed_at = $2, updated_at = $3
		WHERE refund_id = $4
	`

	_, err := r.db.Exec(ctx, query, status, processedAt, time.Now(), refundID)
	return err
}

// GetRefundByID retrieves a refund by refund ID
func (r *PaymentRepository) GetRefundByID(ctx context.Context, refundID string) (*Refund, error) {
	query := `
		SELECT id, refund_id, cf_refund_id, order_id, cf_order_id, amount,
			   status, reason, processed_at, created_at, updated_at
		FROM refunds
		WHERE refund_id = $1
	`

	var refund Refund
	row := r.db.QueryRow(ctx, query, refundID)

	err := row.Scan(
		&refund.ID, &refund.RefundID, &refund.CFRefundID, &refund.OrderID,
		&refund.CFOrderID, &refund.Amount, &refund.Status, &refund.Reason,
		&refund.ProcessedAt, &refund.CreatedAt, &refund.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("refund not found for refund_id: %s", refundID)
		}
		return nil, err
	}

	return &refund, nil
}

// CreateSplitSettlement creates split settlement records
func (r *PaymentRepository) CreateSplitSettlement(ctx context.Context, splits []SplitSettlement) error {
	query := `
		INSERT INTO split_settlements (
			id, order_id, cf_order_id, vendor_id, amount, percentage,
			split_type, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	for i := range splits {
		splits[i].ID = uuid.New()
		splits[i].CreatedAt = now
		splits[i].UpdatedAt = now

		_, err := tx.Exec(ctx, query,
			splits[i].ID, splits[i].OrderID, splits[i].CFOrderID,
			splits[i].VendorID, splits[i].Amount, splits[i].Percentage,
			splits[i].SplitType, splits[i].Status, splits[i].CreatedAt,
			splits[i].UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// CreateSettlement creates a settlement record
func (r *PaymentRepository) CreateSettlement(ctx context.Context, settlement *Settlement) error {
	query := `
		INSERT INTO settlements (
			id, settlement_id, order_id, cf_order_id, amount, status,
			utr, settled_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	settlement.ID = uuid.New()
	settlement.CreatedAt = now
	settlement.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		settlement.ID, settlement.SettlementID, settlement.OrderID,
		settlement.CFOrderID, settlement.Amount, settlement.Status,
		settlement.UTR, settlement.SettledAt, settlement.CreatedAt,
		settlement.UpdatedAt,
	)

	return err
}

// GetSettlementByID retrieves a settlement by settlement ID
func (r *PaymentRepository) GetSettlementByID(ctx context.Context, settlementID string) (*Settlement, error) {
	query := `
		SELECT id, settlement_id, order_id, cf_order_id, amount, status,
			   utr, settled_at, created_at, updated_at
		FROM settlements
		WHERE settlement_id = $1
	`

	var settlement Settlement
	row := r.db.QueryRow(ctx, query, settlementID)

	err := row.Scan(
		&settlement.ID, &settlement.SettlementID, &settlement.OrderID,
		&settlement.CFOrderID, &settlement.Amount, &settlement.Status,
		&settlement.UTR, &settlement.SettledAt, &settlement.CreatedAt,
		&settlement.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("settlement not found for settlement_id: %s", settlementID)
		}
		return nil, err
	}

	return &settlement, nil
}

// CreateWebhookLog creates a webhook log entry
func (r *PaymentRepository) CreateWebhookLog(ctx context.Context, webhook *Webhook) error {
	query := `
		INSERT INTO webhooks (id, event_type, order_id, payload, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	webhook.ID = uuid.New()
	webhook.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		webhook.ID, webhook.EventType, webhook.OrderID,
		webhook.Payload, webhook.Status, webhook.CreatedAt,
	)

	return err
}
