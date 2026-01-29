#!/bin/bash
set -e

# Development helper script
# Usage: ./scripts/dev.sh [command]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

case "${1:-help}" in
    run)
        # Run the backend service locally
        echo "Starting backend service..."
        cd apps/backend
        go run cmd/server/main.go
        ;;

    build)
        # Build the backend binary
        echo "Building backend..."
        cd apps/backend
        go build -o bin/jan-server cmd/server/main.go
        echo "Binary created at apps/backend/bin/jan-server"
        ;;

    test)
        # Run tests
        echo "Running tests..."
        cd apps/backend
        go test ./... -v
        ;;

    test-coverage)
        # Run tests with coverage
        echo "Running tests with coverage..."
        cd apps/backend
        go test ./... -coverprofile=coverage.out
        go tool cover -html=coverage.out -o coverage.html
        echo "Coverage report generated at apps/backend/coverage.html"
        ;;

    lint)
        # Run linter
        echo "Running linter..."
        cd apps/backend
        if command -v golangci-lint &> /dev/null; then
            golangci-lint run ./...
        else
            echo "golangci-lint not installed. Install with:"
            echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
            exit 1
        fi
        ;;

    fmt)
        # Format code
        echo "Formatting code..."
        cd apps/backend
        go fmt ./...
        echo "Code formatted."
        ;;

    tidy)
        # Tidy modules
        echo "Tidying Go modules..."
        cd packages/go-common && go mod tidy
        cd "$ROOT_DIR"
        cd apps/backend && go mod tidy
        echo "Modules tidied."
        ;;

    db-console)
        # Open database console
        source .env 2>/dev/null || true
        docker compose exec postgres psql -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-jan}
        ;;

    db-reset)
        # Reset database
        echo "Resetting database..."
        docker compose down postgres
        docker volume rm mono_postgres_data 2>/dev/null || true
        docker compose up -d postgres
        echo "Database reset complete."
        ;;

    logs)
        # Show logs
        docker compose logs -f ${2:-}
        ;;

    help|*)
        echo "Jan Server Development Helper"
        echo ""
        echo "Usage: ./scripts/dev.sh [command]"
        echo ""
        echo "Commands:"
        echo "  run           Run the backend service locally"
        echo "  build         Build the backend binary"
        echo "  test          Run tests"
        echo "  test-coverage Run tests with coverage report"
        echo "  lint          Run golangci-lint"
        echo "  fmt           Format Go code"
        echo "  tidy          Tidy Go modules"
        echo "  db-console    Open PostgreSQL console"
        echo "  db-reset      Reset the database"
        echo "  logs [svc]    Show logs (optionally for specific service)"
        echo "  help          Show this help message"
        ;;
esac
