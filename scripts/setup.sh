#!/usr/bin/env bash
set -euo pipefail

echo "→ Setting up Alexandria..."

# Go dependencies
echo "→ Downloading Go dependencies..."
go mod download

# Node dependencies
echo "→ Installing frontend dependencies..."
cd client && npm install && cd ..

# Create data dir
mkdir -p data

echo ""
echo "✓ Setup complete! Run 'make dev' to start development."
