package main

import (
	"time"
	"github.com/google/uuid"
)

// Payment represents a payment transaction
type Payment struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrderID        string     `json:"order_id" db:"order_id"`
	CFOrderID      string     `json:"cf_order_id" db:"cf_order_id"`
	Amount         float64    `json:"amount" db:"amount"`
	Currency       string     `json:"currency" db:"currency"`
	Status         string     `json:"status" db:"status"`
	PaymentMethod  *string    `json:"payment_method,omitempty" db:"payment_method"`
	CustomerID     string     `json:"customer_id" db:"customer_id"`
	CustomerName   string     `json:"customer_name" db:"customer_name"`
	CustomerEmail  string     `json:"customer_email" db:"customer_email"`
	CustomerPhone  string     `json:"customer_phone" db:"customer_phone"`
	Description    *string    `json:"description,omitempty" db:"description"`
	PaymentURL     *string    `json:"payment_url,omitempty" db:"payment_url"`
	CFPaymentID    *string    `json:"cf_payment_id,omitempty" db:"cf_payment_id"`
	PaymentTime    *time.Time `json:"payment_time,omitempty" db:"payment_time"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Refund represents a refund transaction
type Refund struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	RefundID    string     `json:"refund_id" db:"refund_id"`
	CFRefundID  string     `json:"cf_refund_id" db:"cf_refund_id"`
	OrderID     string     `json:"order_id" db:"order_id"`
	CFOrderID   string     `json:"cf_order_id" db:"cf_order_id"`
	Amount      float64    `json:"amount" db:"amount"`
	Status      string     `json:"status" db:"status"`
	Reason      *string    `json:"reason,omitempty" db:"reason"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Settlement represents settlement information
type Settlement struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SettlementID string    `json:"settlement_id" db:"settlement_id"`
	OrderID      string    `json:"order_id" db:"order_id"`
	CFOrderID    string    `json:"cf_order_id" db:"cf_order_id"`
	Amount       float64   `json:"amount" db:"amount"`
	Status       string    `json:"status" db:"status"`
	UTR          *string   `json:"utr,omitempty" db:"utr"`
	SettledAt    *time.Time `json:"settled_at,omitempty" db:"settled_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// SplitSettlement represents split settlement configuration
type SplitSettlement struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrderID         string    `json:"order_id" db:"order_id"`
	CFOrderID       string    `json:"cf_order_id" db:"cf_order_id"`
	VendorID        string    `json:"vendor_id" db:"vendor_id"`
	Amount          float64   `json:"amount" db:"amount"`
	Percentage      *float64  `json:"percentage,omitempty" db:"percentage"`
	SplitType       string    `json:"split_type" db:"split_type"` // "PERCENTAGE" or "AMOUNT"
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Webhook represents webhook logs
type Webhook struct {
	ID        uuid.UUID `json:"id" db:"id"`
	EventType string    `json:"event_type" db:"event_type"`
	OrderID   *string   `json:"order_id,omitempty" db:"order_id"`
	Payload   string    `json:"payload" db:"payload"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CreatePaymentSessionRequest represents the request to create a payment session
type CreatePaymentSessionRequest struct {
	OrderID       string  `json:"order_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Currency      string  `json:"currency" binding:"required"`
	CustomerID    string  `json:"customer_id" binding:"required"`
	CustomerName  string  `json:"customer_name" binding:"required"`
	CustomerEmail string  `json:"customer_email" binding:"required,email"`
	CustomerPhone string  `json:"customer_phone" binding:"required"`
	Description   *string `json:"description,omitempty"`
	ReturnURL     string  `json:"return_url" binding:"required,url"`
	NotifyURL     string  `json:"notify_url" binding:"required,url"`
}

// RefundRequest represents a refund request
type RefundRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason *string `json:"reason,omitempty"`
}

// SplitSettlementRequest represents a split settlement request
type SplitSettlementRequest struct {
	Splits []SplitConfig `json:"splits" binding:"required,dive"`
}

type SplitConfig struct {
	VendorID   string   `json:"vendor_id" binding:"required"`
	Amount     *float64 `json:"amount,omitempty"`
	Percentage *float64 `json:"percentage,omitempty"`
}

// VerifyPaymentRequest represents payment verification request
type VerifyPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

// CashfreeOrderResponse represents Cashfree order creation response
type CashfreeOrderResponse struct {
	CFOrderID      string `json:"cf_order_id"`
	OrderID        string `json:"order_id"`
	PaymentLink    string `json:"payment_link"`
	OrderStatus    string `json:"order_status"`
	OrderExpiryTime string `json:"order_expiry_time"`
}

// CashfreePaymentResponse represents Cashfree payment response
type CashfreePaymentResponse struct {
	CFOrderID     string    `json:"cf_order_id"`
	OrderID       string    `json:"order_id"`
	CFPaymentID   string    `json:"cf_payment_id"`
	PaymentStatus string    `json:"payment_status"`
	PaymentAmount float64   `json:"payment_amount"`
	PaymentTime   time.Time `json:"payment_time"`
	PaymentMethod string    `json:"payment_method"`
}
