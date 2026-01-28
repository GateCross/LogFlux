#!/bin/sh

set -e

echo "Starting LogFlux Application..."

# Start backend in background
echo "Starting backend API..."
/app/logflux-api -f /app/etc/config.yaml &
BACKEND_PID=$!

# Wait for backend to be ready
echo "Waiting for backend to be ready..."
sleep 5

# Start Caddy in foreground
echo "Starting Caddy..."
exec caddy run --config /etc/caddy/Caddyfile
