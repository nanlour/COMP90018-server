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

### Client Authentication Flow

#### Common Authentication Errors

```json
// 401 Unauthorized - Missing token
{
    "status": "error",
    "code": "UNAUTHORIZED",
    "message": "Authentication required"
}

// 401 Unauthorized - Invalid token
{
    "status": "error",
    "code": "UNAUTHORIZED",
    "message": "Invalid token"
}

// 401 Unauthorized - Expired token
{
    "status": "error",
    "code": "UNAUTHORIZED",
    "message": "Invalid token"
}

// 401 Unauthorized - Wrong format
{
    "status": "error",
    "code": "UNAUTHORIZED",
    "message": "Invalid token format"
}
```

1. **Initial Authentication:**
   - New users must first sign up using the `/api/auth/signup` endpoint
   - Existing users can log in using the `/api/auth/login` endpoint
   - Upon successful login, the server returns a JWT token in the response

2. **Using the JWT Token:**
   - For all authenticated requests, clients must include the JWT token in the HTTP Authorization header
   - Format: `Authorization: Bearer <your-jwt-token>`
   - Example:
     ```http
     GET /api/ledgers/123/changes HTTP/1.1
     Host: your-server.com
     Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ...
     ```

3. **Token Expiration:**
   - JWT tokens are valid for 24 hours from issuance
   - The `expiresIn` field in the login response indicates the token validity period in seconds
   - Clients should track token expiration and request a new token by logging in again before expiry

4. **Error Handling:**
   - If a token is missing or invalid, the server returns 401 Unauthorized
   - If a token is expired, the server returns 401 Unauthorized
   - In both cases, the client should redirect to the login flow

Example Client Authentication Implementation:
```swift
// Swift example
class APIClient {
    private var token: String?
    private var tokenExpiration: Date?
    
    func login(email: String, password: String) async throws {
        let response = try await post("/api/auth/login", body: [
            "email": email,
            "password": password
        ])
        
        // Store the token
        self.token = response.token
        self.tokenExpiration = Date().addingTimeInterval(Double(response.expiresIn))
    }
    
    func authenticatedRequest(_ endpoint: String) async throws -> Response {
        // Check if token is expired
        if let expiration = tokenExpiration, expiration <= Date() {
            // Token expired, need to login again
            throw AuthError.tokenExpired
        }
        
        // Add token to headers
        var headers = ["Authorization": "Bearer \(token ?? "")"]
        return try await makeRequest(endpoint, headers: headers)
    }
}
```

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

**Response (201 Created):**
```json
{
  "status": "success",
  "userId": "uuid-string", 
  "email": "user@example.com",
  "name": "User Name"
}
```

**Error Response (409 Conflict):**
```json
{
  "status": "error",
  "code": "CONFLICT",
  "message": "user with this email already exists"
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

**Response (200 OK):**
```json
{
  "status": "success",
  "userId": "uuid-string",
  "token": "jwt-token-string",
  "expiresIn": 86400
}
```

**Error Response (401 Unauthorized):**
```json
{
  "status": "error",
  "code": "UNAUTHORIZED",
  "message": "Invalid email or password"
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

**Response (201 Created):**
```json
{
  "status": "success",
  "ledgerId": "uuid-string",
  "name": "Household Expenses",
  "createdAt": "2025-09-14T10:30:00Z",
  "initialSequenceNumber": 0
}
```

#### 4. Delete Ledger

**Endpoint:** `/api/ledgers/{ledgerId}`  
**Method:** DELETE  
**Authentication:** Required  

**Response (200 OK):**
```json
{
  "status": "success",
  "message": "Ledger deleted successfully"
}
```

**Error Responses:**
```json
// 403 Forbidden
{
  "status": "error",
  "code": "FORBIDDEN",
  "message": "you don't have permission to delete this ledger"
}

// 404 Not Found
{
  "status": "error",
  "code": "NOT_FOUND",
  "message": "ledger not found"
}
```

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

**Response (200 OK):**
```json
{
  "status": "success",
  "assignedSequenceNumber": 42,
  "timestamp": "2025-09-14T10:30:00Z"
}
```

**Error Responses:**
```json
// 403 Forbidden
{
  "status": "error",
  "code": "FORBIDDEN",
  "message": "you don't have write permission for this ledger"
}

// 409 Conflict
{
  "status": "error",
  "code": "CONFLICT",
  "message": "Sequence number conflict. Please fetch latest changes and retry."
}
```

#### 6. Get Ledger Changes

**Endpoint:** `/api/ledgers/{ledgerId}/changes`  
**Method:** GET  
**Authentication:** Required  

**Query Parameters:**
- `fromSequence` (required): Starting sequence number (inclusive)
- `toSequence` (optional): Ending sequence number (inclusive)

**Response (200 OK):**
```json
{
  "status": "success",
  "ledgerId": "ledger-uuid",
  "changes": [
    {
      "id": "change-uuid",
      "ledgerId": "ledger-uuid",
      "userId": "user-uuid",
      "sequenceNumber": 42,
      "sqlStatement": "INSERT INTO entries ...",
      "timestamp": "2025-09-14T10:30:00Z",
      "baseSequenceNumber": 41
    }
  ],
  "latestSequenceNumber": 42
}
```

**Error Response (403 Forbidden):**
```json
{
  "status": "error",
  "code": "FORBIDDEN",
  "message": "you don't have access to this ledger"
}
```

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

**Response (200 OK):**
```json
{
  "status": "success",
  "message": "User added to ledger successfully",
  "userId": "user-uuid",
  "email": "friend@example.com",
  "permissions": "write"
}
```

**Error Responses:**
```json
// 403 Forbidden
{
  "status": "error",
  "code": "FORBIDDEN",
  "message": "you don't have permission to add users to this ledger"
}

// 404 Not Found
{
  "status": "error",
  "code": "NOT_FOUND", 
  "message": "user not found"
}
```

## Client-Side Synchronization Guide

### Sequence Number Handling

The server uses sequence numbers to maintain consistency across all clients. Here's how to properly handle sequence numbers in your client implementation:

1. **Track Local Sequence Number:**
   - Store the last known sequence number locally
   - Update it after each successful change submission
   - Initialize it to 0 when first connecting to a ledger

2. **Submitting Changes:**
   ```go
   // Example pseudo-code for submitting changes
   type Change struct {
       SQLStatement string
       ExpectedSeq  int64  // Your local sequence number + 1
   }
   ```

3. **Handling Sequence Gaps:**
   - Before applying any change, verify the received sequence number
   - If there's a gap, fetch missing changes from the server
   ```go
   if receivedSeq > localSeq + 1 {
       // Fetch missing changes
       changes := fetchChanges(ledgerID, localSeq + 1, receivedSeq - 1)
       // Apply missing changes first
       applyChanges(changes)
   }
   ```

## License

[MIT](LICENSE)
