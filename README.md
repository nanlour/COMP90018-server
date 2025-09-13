# Mobile Bill App Server

This is a RESTful API server for a mobile bill application that allows users to manage ledgers for tracking expenses collaboratively.

## Features

- User authentication (signup, login)
- Ledger management (create, delete)
- Ledger operations (add/edit/delete entries via SQL statements)
- Sequence-based synchronization for collaborative editing
- Ledger sharing between users

## Tech Stack

- Go (Golang)
- Gin Web Framework
- PostgreSQL Database
- JWT Authentication

## Project Structure

```
/
├── cmd/
│   └── server/           # Main application entry point
├── internal/
│   ├── api/              # HTTP handlers and middleware
│   ├── config/           # Configuration management
│   ├── models/           # Data models
│   ├── repository/       # Database operations
│   ├── service/          # Business logic
│   └── utils/            # Utility functions
├── Dockerfile            # Docker configuration
├── go.mod                # Go modules
└── .env.example          # Environment variables template
```

## Getting Started

### Prerequisites

- Go 1.20 or higher
- PostgreSQL database

### Configuration

1. Copy the `.env.example` file to `.env` and update the values:

```bash
cp .env.example .env
```

2. Update the environment variables in the `.env` file with your configuration.

### Running locally

1. Install dependencies:

```bash
go mod download
```

2. Build and run the application:

```bash
go run cmd/server/main.go
```

### Running Tests

To run the test suite:

```bash
# Make sure PostgreSQL is running
./scripts/run_tests.sh
```

This will:
1. Set up a test database (billapp_test)
2. Create the necessary tables
3. Run all tests for the API endpoints

### Running with Docker

1. Build the Docker image:

```bash
docker build -t mobile-bill-app-server .
```

2. Run the container:

```bash
docker run -p 8080:8080 --env-file .env mobile-bill-app-server
```

## API Documentation

### Authentication Endpoints

#### 1. Sign Up

**Endpoint:** `/api/auth/signup`  
**Method:** POST  

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123",
  "name": "User Name"
}
```

#### 2. Login

**Endpoint:** `/api/auth/login`  
**Method:** POST  

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

### Ledger Management Endpoints

#### 3. Create Ledger

**Endpoint:** `/api/ledgers`  
**Method:** POST  
**Authentication:** Required  

**Request Body:**
```json
{
  "name": "Household Expenses",
  "description": "Monthly household bills and expenses",
  "currency": "USD"
}
```

#### 4. Delete Ledger

**Endpoint:** `/api/ledgers/{ledgerId}`  
**Method:** DELETE  
**Authentication:** Required  

### Ledger Operations Endpoint

#### 5. Submit Ledger Change

**Endpoint:** `/api/ledgers/{ledgerId}/changes`  
**Method:** POST  
**Authentication:** Required  

**Request Body:**
```json
{
  "sqlStatement": "INSERT INTO entries (id, amount, description, category, date) VALUES ('entry123', 50.25, 'Grocery Shopping', 'Food', '2025-09-13')"
}
```

#### 6. Get Ledger Changes

**Endpoint:** `/api/ledgers/{ledgerId}/changes`  
**Method:** GET  
**Authentication:** Required  

**Query Parameters:**
- `fromSequence` (required): Starting sequence number (inclusive)
- `toSequence` (optional): Ending sequence number (inclusive)

#### 7. Get Latest Sequence Number

**Endpoint:** `/api/ledgers/{ledgerId}/sequence`  
**Method:** GET  
**Authentication:** Required  

**Response (200 OK):**
```json
{
  "status": "success",
  "ledgerId": "ledger123",
  "latestSequenceNumber": 42
}
```

#### 8. Add User to Ledger

**Endpoint:** `/api/ledgers/{ledgerId}/users`  
**Method:** POST  
**Authentication:** Required  

**Request Body:**
```json
{
  "email": "friend@example.com",
  "permissions": "write" // "read" or "write"
}
```

## License

[MIT](LICENSE)
