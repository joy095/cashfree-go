package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	cashfree *CashfreeClient
	repo     *PaymentRepository
}

// Creates a payment session
func (h *PaymentHandler) CreatePaymentSession(c *gin.Context) {
	var req CreatePaymentSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create order in Cashfree
	cashfreeReq := CreateOrderRequest{
		OrderID:       req.OrderID,
		OrderAmount:   req.Amount,
		OrderCurrency: req.Currency,
		CustomerDetails: CustomerDetails{
			CustomerID:    req.CustomerID,
			CustomerName:  req.CustomerName,
			CustomerEmail: req.CustomerEmail,
			CustomerPhone: req.CustomerPhone,
		},
		OrderMeta: &OrderMeta{
			ReturnURL: req.ReturnURL,
			NotifyURL: req.NotifyURL,
		},
		OrderExpiryTime: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	// Handle optional description
	if req.Description != nil {
		cashfreeReq.OrderNote = *req.Description
	}

	cashfreeResp, err := h.cashfree.CreateOrder(cashfreeReq)
	if err != nil {
		log.Printf("Failed to create Cashfree order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment session"})
		return
	}

	// Save payment to database
	payment := &Payment{
		OrderID:       req.OrderID,
		CFOrderID:     cashfreeResp.CFOrderID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        "CREATED",
		CustomerID:    req.CustomerID,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		Description:   req.Description,
		PaymentURL:    &cashfreeResp.PaymentLink,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.repo.CreatePayment(ctx, payment); err != nil {
		log.Printf("Failed to save payment to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":     cashfreeResp.OrderID,
		"cf_order_id":  cashfreeResp.CFOrderID,
		"payment_link": cashfreeResp.PaymentLink,
		"order_status": cashfreeResp.OrderStatus,
		"amount":       req.Amount,
		"currency":     req.Currency,
	})
}

// Verifies a payment
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	var req VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get order status from Cashfree
	orderStatus, err := h.cashfree.GetOrderStatus(req.OrderID)
	if err != nil {
		log.Printf("Failed to get order status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}

	// Get payment details if order is paid
	var paymentDetails *CashfreePaymentResponse
	if orderStatus.OrderStatus == "PAID" {
		paymentDetails, err = h.cashfree.GetPayments(req.OrderID)
		if err != nil {
			log.Printf("Failed to get payment details: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment details"})
			return
		}
	}

	// Update payment status in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cfPaymentID *string
	var paymentMethod *string
	var paymentTime *time.Time

	if paymentDetails != nil {
		cfPaymentID = &paymentDetails.CFPaymentID
		paymentMethod = &paymentDetails.PaymentMethod
		paymentTime = &paymentDetails.PaymentTime
	}

	err = h.repo.UpdatePaymentStatus(ctx, req.OrderID, orderStatus.OrderStatus, cfPaymentID, paymentMethod, paymentTime)
	if err != nil {
		log.Printf("Failed to update payment status: %v", err)
		// Don't return error here as payment verification was successful
	}

	response := gin.H{
		"order_id":     orderStatus.OrderID,
		"cf_order_id":  orderStatus.CFOrderID,
		"order_status": orderStatus.OrderStatus,
		"order_amount": orderStatus.OrderAmount,
	}

	if paymentDetails != nil {
		response["cf_payment_id"] = paymentDetails.CFPaymentID
		response["payment_method"] = paymentDetails.PaymentMethod
		response["payment_time"] = paymentDetails.PaymentTime
		response["payment_amount"] = paymentDetails.PaymentAmount
	}

	c.JSON(http.StatusOK, response)
}

// Gets payment details
func (h *PaymentHandler) GetPaymentDetails(c *gin.Context) {
	orderID := c.Param("order_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get payment from database
	payment, err := h.repo.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		log.Printf("Failed to get payment from database: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	// Also get latest status from Cashfree
	orderStatus, err := h.cashfree.GetOrderStatus(orderID)
	if err != nil {
		log.Printf("Failed to get order status from Cashfree: %v", err)
		// Return database payment if Cashfree call fails
		c.JSON(http.StatusOK, payment)
		return
	}

	// Update status if different
	if payment.Status != orderStatus.OrderStatus {
		err = h.repo.UpdatePaymentStatus(ctx, orderID, orderStatus.OrderStatus, payment.CFPaymentID, payment.PaymentMethod, payment.PaymentTime)
		if err != nil {
			log.Printf("Failed to update payment status: %v", err)
		}
		payment.Status = orderStatus.OrderStatus
	}

	c.JSON(http.StatusOK, payment)
}

// Refunds a payment
func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	orderID := c.Param("order_id")

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate refund ID
	refundID := fmt.Sprintf("refund_%s_%d", orderID, time.Now().Unix())

	// Create refund request for Cashfree
	cashfreeRefundReq := CashfreeRefundRequest{
		OrderID:      orderID,
		RefundAmount: req.Amount,
		RefundID:     refundID,
	}

	if req.Reason != nil {
		cashfreeRefundReq.RefundNote = *req.Reason
	}

	// Create refund in Cashfree
	refundResp, err := h.cashfree.RefundPayment(cashfreeRefundReq)
	if err != nil {
		log.Printf("Failed to create refund in Cashfree: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create refund"})
		return
	}

	// Get payment details for cf_order_id
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payment, err := h.repo.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		log.Printf("Failed to get payment: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	// Save refund to database
	refund := &Refund{
		RefundID:   refundID,
		CFRefundID: refundResp.CFRefundID,
		OrderID:    orderID,
		CFOrderID:  payment.CFOrderID,
		Amount:     req.Amount,
		Status:     refundResp.RefundStatus,
		Reason:     req.Reason,
	}

	if err := h.repo.CreateRefund(ctx, refund); err != nil {
		log.Printf("Failed to save refund to database: %v", err)
		// Don't return error as refund was created successfully in Cashfree
	}

	c.JSON(http.StatusOK, gin.H{
		"refund_id":     refundResp.RefundID,
		"cf_refund_id":  refundResp.CFRefundID,
		"order_id":      refundResp.OrderID,
		"refund_amount": refundResp.RefundAmount,
		"refund_status": refundResp.RefundStatus,
	})
}

// Cancels a payment
func (h *PaymentHandler) CancelPayment(c *gin.Context) {
	orderID := c.Param("order_id")

	// Cancel order in Cashfree
	err := h.cashfree.CancelOrder(orderID)
	if err != nil {
		log.Printf("Failed to cancel order in Cashfree: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel payment"})
		return
	}

	// Update payment status in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.repo.UpdatePaymentStatus(ctx, orderID, "CANCELLED", nil, nil, nil)
	if err != nil {
		log.Printf("Failed to update payment status: %v", err)
		// Don't return error as cancellation was successful in Cashfree
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"status":   "CANCELLED",
		"message":  "Payment cancelled successfully",
	})
}

// Creates split settlement
func (h *PaymentHandler) CreateSplitSettlement(c *gin.Context) {
	orderID := c.Param("order_id")

	var req SplitSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get payment details
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payment, err := h.repo.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		log.Printf("Failed to get payment: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	// Convert splits for Cashfree API
	var cashfreeSplits []CashfreeSettlementSplit
	var dbSplits []SplitSettlement

	for _, split := range req.Splits {
		cashfreeSplit := CashfreeSettlementSplit{
			VendorID: split.VendorID,
		}

		dbSplit := SplitSettlement{
			OrderID:   orderID,
			CFOrderID: payment.CFOrderID,
			VendorID:  split.VendorID,
			Status:    "PENDING",
		}

		if split.Amount != nil {
			cashfreeSplit.Amount = split.Amount
			dbSplit.Amount = *split.Amount
			dbSplit.SplitType = "AMOUNT"
		} else if split.Percentage != nil {
			cashfreeSplit.Percentage = split.Percentage
			dbSplit.Percentage = split.Percentage
			dbSplit.Amount = (payment.Amount * *split.Percentage) / 100
			dbSplit.SplitType = "PERCENTAGE"
		}

		cashfreeSplits = append(cashfreeSplits, cashfreeSplit)
		dbSplits = append(dbSplits, dbSplit)
	}

	// Create settlement in Cashfree
	settlementReq := CashfreeSettlementRequest{
		OrderID: orderID,
		Splits:  cashfreeSplits,
	}

	settlementResp, err := h.cashfree.CreateSettlement(settlementReq)
	if err != nil {
		log.Printf("Failed to create settlement in Cashfree: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create split settlement"})
		return
	}

	// Save split settlement to database
	if err := h.repo.CreateSplitSettlement(ctx, dbSplits); err != nil {
		log.Printf("Failed to save split settlement to database: %v", err)
		// Don't return error as settlement was created in Cashfree
	}

	c.JSON(http.StatusOK, gin.H{
		"cf_settlement_id": settlementResp.CFSettlementID,
		"settlement_id":    settlementResp.SettlementID,
		"order_id":         settlementResp.OrderID,
		"settlement_status": settlementResp.SettlementStatus,
		"splits":           settlementResp.Splits,
	})
}

// Gets settlement details
func (h *PaymentHandler) GetSettlementDetails(c *gin.Context) {
	settlementID := c.Param("settlement_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	settlement, err := h.repo.GetSettlementByID(ctx, settlementID)
	if err != nil {
		log.Printf("Failed to get settlement: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Settlement not found"})
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// Handles webhook from Cashfree
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
	// Get webhook signature and timestamp from headers
	signature := c.GetHeader("x-webhook-signature")
	timestamp := c.GetHeader("x-webhook-timestamp")

	if signature == "" || timestamp == "" {
		log.Println("Missing webhook signature or timestamp")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing webhook headers"})
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify webhook signature
	if !h.cashfree.VerifyWebhookSignature(signature, timestamp, string(body)) {
		log.Println("Invalid webhook signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// Parse webhook data
	var webhookData WebhookData
	if err := json.Unmarshal(body, &webhookData); err != nil {
		log.Printf("Failed to parse webhook data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	// Log webhook for debugging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var orderID *string
	if oid, exists := webhookData.Data["order_id"]; exists {
		if oidStr, ok := oid.(string); ok {
			orderID = &oidStr
		}
	}

	webhook := &Webhook{
		EventType: webhookData.Type,
		OrderID:   orderID,
		Payload:   string(body),
		Status:    "RECEIVED",
	}

	if err := h.repo.CreateWebhookLog(ctx, webhook); err != nil {
		log.Printf("Failed to log webhook: %v", err)
	}

	// Process different webhook events
	switch webhookData.Type {
	case "PAYMENT_SUCCESS_WEBHOOK":
		h.handlePaymentSuccessWebhook(ctx, webhookData.Data)
	case "PAYMENT_FAILED_WEBHOOK":
		h.handlePaymentFailedWebhook(ctx, webhookData.Data)
	case "REFUND_STATUS_WEBHOOK":
		h.handleRefundStatusWebhook(ctx, webhookData.Data)
	case "SETTLEMENT_STATUS_WEBHOOK":
		h.handleSettlementStatusWebhook(ctx, webhookData.Data)
	default:
		log.Printf("Unknown webhook type: %s", webhookData.Type)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *PaymentHandler) handlePaymentSuccessWebhook(ctx context.Context, data map[string]interface{}) {
	orderID, ok := data["order_id"].(string)
	if !ok {
		log.Println("Missing order_id in payment success webhook")
		return
	}

	cfPaymentID, _ := data["cf_payment_id"].(string)
	paymentMethod, _ := data["payment_method"].(string)

	// Parse payment time
	var paymentTime *time.Time
	if pt, exists := data["payment_time"]; exists {
		if ptStr, ok := pt.(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, ptStr); err == nil {
				paymentTime = &parsedTime
			}
		}
	}

	err := h.repo.UpdatePaymentStatus(ctx, orderID, "SUCCESS", &cfPaymentID, &paymentMethod, paymentTime)
	if err != nil {
		log.Printf("Failed to update payment status for successful payment: %v", err)
	}
}

func (h *PaymentHandler) handlePaymentFailedWebhook(ctx context.Context, data map[string]interface{}) {
	orderID, ok := data["order_id"].(string)
	if !ok {
		log.Println("Missing order_id in payment failed webhook")
		return
	}

	err := h.repo.UpdatePaymentStatus(ctx, orderID, "FAILED", nil, nil, nil)
	if err != nil {
		log.Printf("Failed to update payment status for failed payment: %v", err)
	}
}

func (h *PaymentHandler) handleRefundStatusWebhook(ctx context.Context, data map[string]interface{}) {
	refundID, ok := data["refund_id"].(string)
	if !ok {
		log.Println("Missing refund_id in refund status webhook")
		return
	}

	refundStatus, _ := data["refund_status"].(string)

	// Parse processed time
	var processedAt *time.Time
	if pt, exists := data["processed_at"]; exists {
		if ptStr, ok := pt.(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, ptStr); err == nil {
				processedAt = &parsedTime
			}
		}
	}

	err := h.repo.UpdateRefundStatus(ctx, refundID, refundStatus, processedAt)
	if err != nil {
		log.Printf("Failed to update refund status: %v", err)
	}
}

func (h *PaymentHandler) handleSettlementStatusWebhook(ctx context.Context, data map[string]interface{}) {
	// Handle settlement status updates
	// This would involve updating settlement records in the database
	log.Printf("Settlement webhook received: %+v", data)
}

// Gets refund details
func (h *PaymentHandler) GetRefundDetails(c *gin.Context) {
	refundID := c.Param("refund_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	refund, err := h.repo.GetRefundByID(ctx, refundID)
	if err != nil {
		log.Printf("Failed to get refund: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Refund not found"})
		return
	}

	c.JSON(http.StatusOK, refund)
}

// Gets all payments
func (h *PaymentHandler) GetAllPayments(c *gin.Context) {
	// Parse query parameters for pagination
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Limit the maximum number of records per request
	if limit > 100 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payments, err := h.repo.GetAllPayments(ctx, limit, offset)
	if err != nil {
		log.Printf("Failed to get payments: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"limit":    limit,
		"offset":   offset,
		"count":    len(payments),
	})
}
