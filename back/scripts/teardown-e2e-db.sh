#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DB_PATH="$PROJECT_ROOT/db/linktic-e2e.db"

echo "=== E2E Database Teardown ==="

rm -f "$DB_PATH" "$DB_PATH-wal" "$DB_PATH-shm"

echo "=== E2E database removed ==="
