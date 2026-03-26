#!/bin/sh
# wait-for.sh
# Wait for a TCP service to be available
# Usage: wait-for.sh host:port [-- command args]
#
# This script is used to wait for services like PostgreSQL to be ready
# before starting the application. Useful in Docker Compose and Kubernetes
# init containers.
#
# Examples:
#   wait-for.sh postgres:5432 -- /app/main
#   wait-for.sh localhost:8080
#
# Exit codes:
#   0 - Service became available (or optional command succeeded)
#   1 - Service did not become available or command failed

set -e

# Get host:port argument
host="$1"
shift
cmd="$@"

# Validation
if [ -z "$host" ]; then
  echo "Usage: $0 host:port [-- command args]"
  echo ""
  echo "Examples:"
  echo "  $0 postgres:5432 -- /app/main"
  echo "  $0 localhost:8080"
  exit 1
fi

# Parse host and port
port="${host##*:}"
host="${host%:*}"

# Validate port is numeric
if ! echo "$port" | grep -qE '^[0-9]+$'; then
  echo "Error: Invalid port '$port'. Port must be numeric."
  exit 1
fi

echo "Waiting for $host:$port to become available..."

max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
  # Try to connect using nc (netcat)
  if nc -z "$host" "$port" 2>/dev/null; then
    echo "✓ $host:$port is available"
    echo ""

    # If a command was provided, run it
    if [ -n "$cmd" ]; then
      echo "Executing: $cmd"
      exec $cmd
    fi
    exit 0
  fi

  attempt=$((attempt + 1))
  echo "  Attempt $attempt/$max_attempts failed. Retrying in 2 seconds..."
  sleep 2
done

echo ""
echo "✗ Error: $host:$port did not become available after $((max_attempts * 2)) seconds"
exit 1
