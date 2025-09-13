#!/bin/bash

# Script to set up and run the API tests

# Color constants
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up test environment...${NC}"

# Check for required Go packages
echo -e "Checking dependencies..."
go get -t -v github.com/stretchr/testify/assert
go get -t -v github.com/gin-gonic/gin

# Create test database if it doesn't exist
echo -e "Setting up test database..."
PGPASSWORD=password psql -h localhost -U postgres -c "CREATE DATABASE billapp_test;" || true
PGPASSWORD=password psql -h localhost -U postgres -d billapp_test -c "DROP TABLE IF EXISTS ledger_changes, ledger_users, ledgers, users CASCADE;"

# Run the database initialization script on test DB
PGPASSWORD=password psql -h localhost -U postgres -d billapp_test -f scripts/db_init_test.sql

# Run the tests
echo -e "${GREEN}Running tests...${NC}"
# Run tests individually to avoid package conflicts
go test -v ./internal/api/tests/auth_test.go
go test -v ./internal/api/tests/ledger_test.go
go test -v ./internal/api/tests/ledger_changes_test.go
go test -v ./internal/api/tests/ledger_sharing_test.go

# Check if tests passed
if [ $? -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
