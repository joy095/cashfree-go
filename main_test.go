package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK", "service": "Cashfree Payment Gateway"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "OK", response["status"])
	assert.Equal(t, "Cashfree Payment Gateway", response["service"])
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.Default()
	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSOptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.Default()
	router.Use(CORSMiddleware())
	router.OPTIONS("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "options"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)
}

func TestCreatePaymentSessionValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.Default()
	
	// Mock handler that just validates the request
	router.POST("/api/v1/payments/create-session", func(c *gin.Context) {
		var req CreatePaymentSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "validation passed"})
	})

	// Test with valid request
	validRequest := CreatePaymentSessionRequest{
		OrderID:       "test_order_123",
		Amount:        100.50,
		Currency:      "INR",
		CustomerID:    "customer_001",
		CustomerName:  "John Doe",
		CustomerEmail: "john.doe@example.com",
		CustomerPhone: "+919876543210",
		ReturnURL:     "https://example.com/return",
		NotifyURL:     "https://example.com/notify",
	}

	jsonBody, _ := json.Marshal(validRequest)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/payments/create-session", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Test with invalid request (missing required fields)
	invalidRequest := map[string]interface{}{
		"amount": 100.50,
		// Missing required fields
	}

	jsonBody, _ = json.Marshal(invalidRequest)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/payments/create-session", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestVerifyPaymentRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.Default()
	
	router.POST("/api/v1/payments/verify", func(c *gin.Context) {
		var req VerifyPaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "validation passed"})
	})

	// Test with valid request
	validRequest := VerifyPaymentRequest{
		OrderID: "test_order_123",
	}

	jsonBody, _ := json.Marshal(validRequest)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/payments/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// Test with invalid request (empty order_id)
	invalidRequest := VerifyPaymentRequest{
		OrderID: "",
	}

	jsonBody, _ = json.Marshal(invalidRequest)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/payments/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestCashfreeClientCreation(t *testing.T) {
	client := NewCashfreeClient("test_client_id", "test_secret", "test")
	
	assert.NotNil(t, client)
	assert.Equal(t, "test_client_id", client.ClientID)
	assert.Equal(t, "test_secret", client.ClientSecret)
	assert.Equal(t, "test", client.Environment)
	assert.Equal(t, CashfreeTestURL, client.BaseURL)
	assert.NotNil(t, client.Client)
}

func TestCashfreeClientProductionURL(t *testing.T) {
	client := NewCashfreeClient("prod_client_id", "prod_secret", "PROD")
	
	assert.Equal(t, CashfreeProdURL, client.BaseURL)
}

func TestWebhookSignatureVerification(t *testing.T) {
	client := NewCashfreeClient("test_id", "test_secret", "test")
	
	// Test with correct signature
	timestamp := "1640995200"
	payload := `{"type":"PAYMENT_SUCCESS_WEBHOOK","data":{"order_id":"test_order"}}`
	
	// This is a mock test - in real implementation you'd calculate the actual signature
	// For now, we'll test the structure
	isValid := client.VerifyWebhookSignature("mock_signature", timestamp, payload)
	
	// Since we're using a mock signature, it should be false
	assert.False(t, isValid)
}
