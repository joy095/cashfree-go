# Cashfree Payment Gateway Integration

A complete Go-based REST API service for integrating with Cashfree Payment Gateway. This service provides a comprehensive implementation of payment processing, including order creation, payment verification, refunds, cancellations, split settlements, and webhook handling.

## Features

- ✅ **Payment Session Creation** - Create secure payment sessions with Cashfree
- ✅ **Payment Verification** - Verify payment status and update records
- ✅ **Refund Processing** - Handle partial and full refunds
- ✅ **Order Cancellation** - Cancel pending orders
- ✅ **Split Settlements** - Distribute payments to multiple vendors
- ✅ **Webhook Handling** - Process real-time payment notifications
- ✅ **Database Integration** - PostgreSQL for data persistence
- ✅ **Secure Signature Verification** - Verify webhook authenticity
- ✅ **Comprehensive Logging** - Track all payment operations
- ✅ **CORS Support** - Cross-origin request handling
- ✅ **Pagination** - Efficient data retrieval

## Tech Stack

- **Backend**: Go (Golang) with Gin framework
- **Database**: PostgreSQL with pgx driver
- **HTTP Client**: Resty for API calls
- **Environment**: dotenv for configuration
- **UUID**: Google UUID for unique identifiers

## Prerequisites

- Go 1.21+ installed
- PostgreSQL database running
- Cashfree merchant account (test/production)

## Installation

1. **Clone the repository**

```bash
git clone <repository-url>
cd cashfree-gateway
```

2. **Install dependencies**

```bash
go mod tidy
```

3. **Set up PostgreSQL database**

```bash
# Create database
createdb go_cashfree

# Run migrations
psql -d go_cashfree -f migrations.sql
```

4. **Configure environment variables**

```bash
cp .env.example .env
# Edit .env with your configuration
```

## Configuration

Update the `.env` file with your configurations:

```env
# Database Configuration
DATABASE_URL=postgresql://postgres:admin123@localhost:5432/go_cashfree

# Cashfree Configuration
CASHFREE_CLIENT_ID=
CASHFREE_CLIENT_SECRET=
CASHFREE_ENVIRONMENT=test  # or "prod" for production

# Server Configuration
PORT=8080
```

### Getting Cashfree Credentials

1. Sign up at [Cashfree Dashboard](https://payments.cashfree.com/)
2. Navigate to Developers → API Keys
3. Copy your `Client ID` and `Client Secret`
4. For testing, use sandbox credentials

## Running the Application

```bash
# Development
go run .

# Build and run
go build -o cashfree-gateway .
./cashfree-gateway
```

The server will start at `http://localhost:8080`

## API Endpoints

### Health Check

```
GET /health
```

### Payment Operations

#### 1. Create Payment Session

```
POST /api/v1/payments/create-session
```

**Request Body:**

```json
{
  "order_id": "order_123",
  "amount": 100.5,
  "currency": "INR",
  "customer_id": "customer_001",
  "customer_name": "John Doe",
  "customer_email": "john.doe@example.com",
  "customer_phone": "+919876543210",
  "description": "Test payment",
  "return_url": "https://your-domain.com/payment/success",
  "notify_url": "https://your-domain.com/api/v1/webhook/cashfree"
}
```

**Response:**

```json
{
  "order_id": "order_123",
  "cf_order_id": "cf_order_abc123",
  "payment_link": "https://payments.cashfree.com/links/abc123",
  "order_status": "ACTIVE",
  "amount": 100.5,
  "currency": "INR"
}
```

#### 2. Verify Payment

```
POST /api/v1/payments/verify
```

**Request Body:**

```json
{
  "order_id": "order_123"
}
```

#### 3. Get Payment Details

```
GET /api/v1/payments/{order_id}
```

#### 4. Refund Payment

```
POST /api/v1/payments/{order_id}/refund
```

**Request Body:**

```json
{
  "amount": 50.25,
  "reason": "Customer requested refund"
}
```

#### 5. Cancel Payment

```
POST /api/v1/payments/{order_id}/cancel
```

#### 6. Create Split Settlement

```
POST /api/v1/payments/{order_id}/split
```

**Request Body:**

```json
{
  "splits": [
    {
      "vendor_id": "vendor_001",
      "amount": 70.0
    },
    {
      "vendor_id": "vendor_002",
      "percentage": 30.0
    }
  ]
}
```

#### 7. Get All Payments (with pagination)

```
GET /api/v1/payments?limit=10&offset=0
```

### Settlement & Refund Operations

#### 8. Get Settlement Details

```
GET /api/v1/settlements/{settlement_id}
```

#### 9. Get Refund Details

```
GET /api/v1/refunds/{refund_id}
```

### Webhook Endpoint

#### 10. Handle Cashfree Webhooks

```
POST /api/v1/webhook/cashfree
```

This endpoint automatically processes webhook events from Cashfree including:

- Payment success/failure
- Refund status updates
- Settlement notifications

## Database Schema

The application uses the following main tables:

- **payments** - Store payment transactions
- **refunds** - Track refund operations
- **settlements** - Settlement information
- **split_settlements** - Split settlement configurations
- **webhooks** - Webhook event logs

## Testing

Use the provided `test_api.http` file with VS Code REST Client extension or any HTTP client like Postman or curl.

### Example Test Flow

1. **Create Payment Session**
2. **Simulate payment completion** (via Cashfree dashboard or test cards)
3. **Verify Payment** to check status
4. **Create Refund** if needed
5. **Check webhook logs** for real-time updates

## Security Features

- **Webhook Signature Verification**: All webhooks are verified using HMAC-SHA256
- **Environment Variable Protection**: Sensitive data stored in environment variables
- **SQL Injection Prevention**: Parameterized queries used throughout
- **CORS Configuration**: Configurable cross-origin support
- **Request Validation**: Input validation for all API endpoints

## Error Handling

The API returns appropriate HTTP status codes and error messages:

- `200 OK` - Successful operation
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Invalid webhook signature
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server errors

## Logging

Comprehensive logging is implemented throughout the application:

- Payment operations
- Webhook events
- Database operations
- API errors

## Production Deployment

### Environment Setup

1. Set `CASHFREE_ENVIRONMENT=prod`
2. Use production database
3. Configure proper SSL certificates
4. Set up reverse proxy (nginx/cloudflare)
5. Enable database connection pooling

### Monitoring

- Monitor webhook endpoint health
- Set up alerts for failed payments
- Track database performance
- Monitor API response times

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- Check [Cashfree Documentation](https://docs.cashfree.com/)
- Review the test files and examples
- Open an issue in this repository

## Changelog

### v1.0.0

- Initial implementation with complete Cashfree integration
- Payment creation, verification, and management
- Refund and settlement processing
- Webhook handling with signature verification
- PostgreSQL database integration
- Comprehensive API documentation
  "# cashfree-go"

# cashfree-go
"# cashfree-go" 
