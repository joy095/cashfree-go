package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	CashfreeTestURL = "https://sandbox.cashfree.com/pg"
	CashfreeProdURL = "https://api.cashfree.com/pg"
)

// CashfreeClient represents the Cashfree payment gateway client
type CashfreeClient struct {
	ClientID     string
	ClientSecret string
	Environment  string
	BaseURL      string
	Client       *resty.Client
}

// NewCashfreeClient creates a new Cashfree client
func NewCashfreeClient(clientID, clientSecret, environment string) *CashfreeClient {
	baseURL := CashfreeTestURL
	if strings.ToUpper(environment) == "PROD" {
		baseURL = CashfreeProdURL
	}

	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(5 * time.Second)

	return &CashfreeClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Environment:  environment,
		BaseURL:      baseURL,
		Client:       client,
	}
}

// CreateOrder creates a new order in Cashfree
func (c *CashfreeClient) CreateOrder(req CreateOrderRequest) (*CashfreeOrderResponse, error) {
	url := fmt.Sprintf("%s/orders", c.BaseURL)

	// Prepare headers
	headers := c.getAuthHeaders()

	var response CashfreeOrderResponse
	resp, err := c.Client.R().
		SetHeaders(headers).
		SetBody(req).
		SetResult(&response).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &response, nil
}

// GetOrderStatus gets the status of an order
func (c *CashfreeClient) GetOrderStatus(orderID string) (*CashfreeOrderStatusResponse, error) {
	url := fmt.Sprintf("%s/orders/%s", c.BaseURL, orderID)

	headers := c.getAuthHeaders()

	var response CashfreeOrderStatusResponse
	resp, err := c.Client.R().
		SetHeaders(headers).
		SetResult(&response).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &response, nil
}

// GetPayments gets payment details for an order
func (c *CashfreeClient) GetPayments(orderID string) (*CashfreePaymentResponse, error) {
	url := fmt.Sprintf("%s/orders/%s/payments", c.BaseURL, orderID)

	headers := c.getAuthHeaders()

	var payments []CashfreePaymentResponse
	resp, err := c.Client.R().
		SetHeaders(headers).
		SetResult(&payments).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to get payments: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	if len(payments) == 0 {
		return nil, fmt.Errorf("no payments found for order %s", orderID)
	}

	return &payments[0], nil
}

// RefundPayment creates a refund for a payment
func (c *CashfreeClient) RefundPayment(req CashfreeRefundRequest) (*CashfreeRefundResponse, error) {
	url := fmt.Sprintf("%s/orders/%s/refunds", c.BaseURL, req.OrderID)

	headers := c.getAuthHeaders()

	var response CashfreeRefundResponse
	resp, err := c.Client.R().
		SetHeaders(headers).
		SetBody(req).
		SetResult(&response).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &response, nil
}

// GetRefundStatus gets the status of a refund
func (c *CashfreeClient) GetRefundStatus(orderID, refundID string) (*CashfreeRefundResponse, error) {
	url := fmt.Sprintf("%s/orders/%s/refunds/%s", c.BaseURL, orderID, refundID)

	headers := c.getAuthHeaders()

	var response CashfreeRefundResponse
	resp, err := c.Client.R().
		SetHeaders(headers).
		SetResult(&response).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to get refund status: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &response, nil
}

// CancelOrder cancels an order
func (c *CashfreeClient) CancelOrder(orderID string) error {
	url := fmt.Sprintf("%s/orders/%s/cancel", c.BaseURL, orderID)

	headers := c.getAuthHeaders()

	resp, err := c.Client.R().
		SetHeaders(headers).
		Patch(url)

	if err != nil {
		return fmt.Errorf("failed to cancel order: %v", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// CreateSettlement creates split settlement
func (c *CashfreeClient) CreateSettlement(req CashfreeSettlementRequest) (*CashfreeSettlementResponse, error) {
	url := fmt.Sprintf("%s/orders/%s/settlements", c.BaseURL, req.OrderID)

	headers := c.getAuthHeaders()

	var response CashfreeSettlementResponse
	resp, err := c.Client.R().
		SetHeaders(headers).
		SetBody(req).
		SetResult(&response).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("failed to create settlement: %v", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("cashfree API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &response, nil
}

// VerifyWebhookSignature verifies the webhook signature
func (c *CashfreeClient) VerifyWebhookSignature(signature, timestamp, payload string) bool {
	// Create the string to sign
	stringToSign := timestamp + payload

	// Create HMAC SHA256 hash
	h := hmac.New(sha256.New, []byte(c.ClientSecret))
	h.Write([]byte(stringToSign))
	hash := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return hash == signature
}

// getAuthHeaders returns the authentication headers for Cashfree API
func (c *CashfreeClient) getAuthHeaders() map[string]string {
	return map[string]string{
		"X-Client-Id":     c.ClientID,
		"X-Client-Secret": c.ClientSecret,
		"Content-Type":    "application/json",
		"Accept":          "application/json",
		"x-api-version":   "2023-08-01",
	}
}

// CreateOrderRequest represents the request to create an order in Cashfree
type CreateOrderRequest struct {
	OrderID     string                  `json:"order_id"`
	OrderAmount float64                 `json:"order_amount"`
	OrderCurrency string                `json:"order_currency"`
	CustomerDetails CustomerDetails     `json:"customer_details"`
	OrderMeta   *OrderMeta              `json:"order_meta,omitempty"`
	OrderNote   string                  `json:"order_note,omitempty"`
	OrderExpiryTime string              `json:"order_expiry_time,omitempty"`
}

type CustomerDetails struct {
	CustomerID    string `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	CustomerPhone string `json:"customer_phone"`
}

type OrderMeta struct {
	ReturnURL    string `json:"return_url,omitempty"`
	NotifyURL    string `json:"notify_url,omitempty"`
	PaymentMethods string `json:"payment_methods,omitempty"`
}

// CashfreeOrderStatusResponse represents order status response
type CashfreeOrderStatusResponse struct {
	CFOrderID       string    `json:"cf_order_id"`
	OrderID         string    `json:"order_id"`
	OrderStatus     string    `json:"order_status"`
	OrderAmount     float64   `json:"order_amount"`
	OrderCurrency   string    `json:"order_currency"`
	OrderExpiryTime time.Time `json:"order_expiry_time"`
	PaymentLink     string    `json:"payment_link"`
}

// CashfreeRefundRequest represents refund request
type CashfreeRefundRequest struct {
	OrderID      string  `json:"-"` // Used in URL, not in body
	RefundAmount float64 `json:"refund_amount"`
	RefundID     string  `json:"refund_id"`
	RefundNote   string  `json:"refund_note,omitempty"`
}

// CashfreeRefundResponse represents refund response
type CashfreeRefundResponse struct {
	CFRefundID    string  `json:"cf_refund_id"`
	RefundID      string  `json:"refund_id"`
	OrderID       string  `json:"order_id"`
	RefundAmount  float64 `json:"refund_amount"`
	RefundStatus  string  `json:"refund_status"`
	RefundMode    string  `json:"refund_mode"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	RefundNote    string  `json:"refund_note,omitempty"`
}

// CashfreeSettlementRequest represents settlement request
type CashfreeSettlementRequest struct {
	OrderID string                      `json:"-"` // Used in URL
	Splits  []CashfreeSettlementSplit   `json:"splits"`
}

type CashfreeSettlementSplit struct {
	VendorID   string  `json:"vendor_id"`
	Amount     *float64 `json:"amount,omitempty"`
	Percentage *float64 `json:"percentage,omitempty"`
}

// CashfreeSettlementResponse represents settlement response
type CashfreeSettlementResponse struct {
	CFSettlementID string    `json:"cf_settlement_id"`
	SettlementID   string    `json:"settlement_id"`
	OrderID        string    `json:"order_id"`
	SettlementStatus string  `json:"settlement_status"`
	Splits         []CashfreeSettlementSplit `json:"splits"`
}

// WebhookData represents webhook payload
type WebhookData struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Signature string                 `json:"-"`
	Timestamp string                 `json:"-"`
}
