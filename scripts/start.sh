#!/bin/bash
set -e

echo "=========================================="
echo "eSTOCK Backend - Container Startup"
echo "=========================================="
echo ""

# Verify required environment variables
echo "Step 1: Verifying environment variables..."
required_vars="DATABASE_URL JWT_SECRET SERVER_ADDRESS ENVIRONMENT MIGRATION_URL"

missing_vars=""
for var in $required_vars; do
  value=$(eval echo \$$var)
  if [ -z "$value" ]; then
    missing_vars="$missing_vars $var"
  fi
done

if [ -n "$missing_vars" ]; then
  echo "✗ ERROR: The following required environment variables are not set:$missing_vars"
  echo ""
  echo "Required variables:"
  echo "  - DATABASE_URL: PostgreSQL connection string"
  echo "  - JWT_SECRET: JWT signing secret (min 32 chars)"
  echo "  - SERVER_ADDRESS: Server address (e.g., :8080)"
  echo "  - ENVIRONMENT: Environment name (development/release/test)"
  echo "  - MIGRATION_URL: Path to migrations (e.g., file://db/migrations)"
  echo ""
  exit 1
fi

echo "✓ All required environment variables are set"
echo ""

# Run database migrations
echo "Step 2: Checking database migrations..."
echo "  DATABASE_URL: postgresql://***:***@***:***/**"
echo "  MIGRATION_URL: $MIGRATION_URL"
echo ""

# Check if golang-migrate CLI is available
if command -v migrate &> /dev/null; then
  echo "  Using golang-migrate CLI..."

  # Wait for database to be ready
  max_attempts=30
  attempt=0

  while [ $attempt -lt $max_attempts ]; do
    if migrate -path "$MIGRATION_URL" -database "$DATABASE_URL" version 2>/dev/null; then
      echo "  ✓ Database is accessible"
      break
    fi

    attempt=$((attempt + 1))
    echo "  Attempt $attempt/$max_attempts: Database not ready yet. Retrying in 2 seconds..."
    sleep 2
  done

  if [ $attempt -eq $max_attempts ]; then
    echo "  ✗ ERROR: Could not connect to database after $((max_attempts * 2)) seconds"
    exit 1
  fi

  # Run pending migrations
  echo "  Running pending migrations..."
  migrate -path "$MIGRATION_URL" -database "$DATABASE_URL" up

  echo "  ✓ Migrations completed successfully"
else
  echo "  ℹ golang-migrate CLI not found in PATH"
  echo "  Assuming migrations are handled by the application on startup"
  echo "  (If using embedded migrations in Go code, the app will handle them)"
fi

echo ""

# Start the server
echo "Step 3: Starting eSTOCK Backend Server..."
echo "  SERVER_ADDRESS: $SERVER_ADDRESS"
echo "  ENVIRONMENT: $ENVIRONMENT"
echo "  Version: $Version"
echo ""
echo "=========================================="
echo ""

# Execute the main application
# Pass all remaining arguments to the main process
exec /app/main "$@"
