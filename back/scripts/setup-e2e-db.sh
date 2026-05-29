#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DB_PATH="$PROJECT_ROOT/db/linktic-e2e.db"
MIGRATIONS_DIR="$PROJECT_ROOT/migrations"
SEED_FILE="$PROJECT_ROOT/db/seed.sql"

echo "=== E2E Database Setup ==="

echo "[1/3] Creating test database file..."
mkdir -p "$PROJECT_ROOT/db"
touch "$DB_PATH"

echo "[2/3] Running migrations..."
cd "$PROJECT_ROOT"
go run cmd/migrate/main.go -db="$DB_PATH" -m="$MIGRATIONS_DIR" up

echo "[3/3] Seeding data..."
sqlite3 "$DB_PATH" < "$SEED_FILE"

echo "=== E2E database ready at $DB_PATH ==="
