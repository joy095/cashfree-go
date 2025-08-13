package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	connectDB()
	defer closeDB()

	// Initialize Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(CORSMiddleware())

	// Initialize Cashfree client
	cashfreeClient := NewCashfreeClient(
		os.Getenv("CASHFREE_CLIENT_ID"),
		os.Getenv("CASHFREE_CLIENT_SECRET"),
		os.Getenv("CASHFREE_ENVIRONMENT"), // "TEST" or "PROD"
	)

	// Initialize repository
	paymentRepo := NewPaymentRepository(dbPool)

	// Initialize payment handler
	paymentHandler := &PaymentHandler{
		cashfree: cashfreeClient,
		repo:     paymentRepo,
	}

	// Payment routes
	api := r.Group("/api/v1")
	{
		// Create payment session
		api.POST("/payments/create-session", paymentHandler.CreatePaymentSession)
		
		// Verify payment
		api.POST("/payments/verify", paymentHandler.VerifyPayment)
		
		// Get payment details
		api.GET("/payments/:order_id", paymentHandler.GetPaymentDetails)
		
		// Refund payment
		api.POST("/payments/:order_id/refund", paymentHandler.RefundPayment)
		
		// Cancel payment
		api.POST("/payments/:order_id/cancel", paymentHandler.CancelPayment)
		
		// Split settlement
		api.POST("/payments/:order_id/split", paymentHandler.CreateSplitSettlement)
		
		// Get settlement details
		api.GET("/settlements/:settlement_id", paymentHandler.GetSettlementDetails)
		
		// Webhook handler
		api.POST("/webhook/cashfree", paymentHandler.HandleWebhook)
		
		// Get refund details
		api.GET("/refunds/:refund_id", paymentHandler.GetRefundDetails)
		
		// Get all payments
		api.GET("/payments", paymentHandler.GetAllPayments)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK", "service": "Cashfree Payment Gateway"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
