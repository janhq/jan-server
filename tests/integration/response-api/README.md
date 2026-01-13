# Response API Integration Tests

This directory contains integration tests for the Response API's plan and artifact features.

## Prerequisites

1. PostgreSQL database running with `response_api` schema
2. All migrations applied
3. Environment variables configured

## Environment Variables

```bash
export TEST_DATABASE_URL="postgres://user:password@localhost:5432/jan?search_path=response_api"
export TEST_RESPONSE_API_URL="http://localhost:8082"
```

## Running Tests

```bash
# From repository root
cd tests/integration/response-api

# Run all integration tests
go test -v ./...

# Run specific test
go test -v -run TestPlanAPI ./...

# Run with coverage
go test -v -coverprofile=coverage.out ./...
```

## Test Structure

- `plan_test.go` - Plan API integration tests
- `artifact_test.go` - Artifact API integration tests
- `helpers_test.go` - Common test utilities
