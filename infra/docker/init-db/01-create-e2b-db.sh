#!/bin/sh
# Create sandbox user and e2b_sandbox database for E2B service
# This script runs automatically when PostgreSQL container starts for the first time

set -e

# Create the sandbox user
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'sandbox') THEN
            CREATE USER sandbox WITH PASSWORD 'sandbox';
        END IF;
    END
    \$\$;
EOSQL

# Create the e2b_sandbox database if it doesn't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'Database e2b_sandbox already exists' WHERE EXISTS (SELECT FROM pg_database WHERE datname = 'e2b_sandbox');
EOSQL

if ! psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -lqt | cut -d \| -f 1 | grep -qw e2b_sandbox; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        CREATE DATABASE e2b_sandbox OWNER sandbox;
EOSQL
    echo "Created database e2b_sandbox"
else
    echo "Database e2b_sandbox already exists"
fi

# Grant privileges
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    GRANT ALL PRIVILEGES ON DATABASE e2b_sandbox TO sandbox;
EOSQL

echo "E2B sandbox database setup complete"
