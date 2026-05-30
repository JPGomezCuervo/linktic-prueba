#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DB_PATH="$PROJECT_ROOT/db/linktic-e2e.db"
MIGRATIONS_DIR="$PROJECT_ROOT/migrations"
SEED_FILE="$PROJECT_ROOT/db/seed.sql"

echo "=== E2E Database Setup ==="

echo "[1/4] Creating test database file..."
mkdir -p "$PROJECT_ROOT/db"
touch "$DB_PATH"

echo "[2/4] Running migrations..."
cd "$PROJECT_ROOT"
go run cmd/migrate/main.go -db="$DB_PATH" -m="$MIGRATIONS_DIR" up

echo "[3/4] Seeding data..."
sqlite3 "$DB_PATH" < "$SEED_FILE"

echo "[4/4] Seeding data..."
echo -e "DATABASE_PATH=db/linktic-e2e.db\nJWT_SECRET=YEC1q8P+ezIahZXrjbjW50U8HMSCKhX4RzyGKHd6tmY=\n" > "$PROJECT_ROOT/.test.env.e2e"

echo "=== E2E database ready at $DB_PATH ==="
