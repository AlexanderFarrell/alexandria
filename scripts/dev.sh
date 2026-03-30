#!/usr/bin/env bash
set -euo pipefail

# Start Go server and Vite dev server in parallel.
# Both processes are killed when this script exits (Ctrl+C).

trap 'kill $(jobs -p) 2>/dev/null; exit' INT TERM EXIT

mkdir -p data

echo "→ Starting Alexandria dev servers..."
echo "   API  : http://localhost:8080"
echo "   App  : http://localhost:5173  (proxies /api → :8080)"
echo ""

# Go server
go run ./cmd/server &

# Vite dev server
cd client && npm run dev &

wait
