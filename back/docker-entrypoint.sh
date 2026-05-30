#!/bin/sh
set -e

DATABASE_PATH="${DATABASE_PATH:-/app/db/linktic.db}"
JWT_SECRET="${JWT_SECRET:-dev-secret-change-me}"
PORT="${PORT:-8080}"
SEED="${SEED:-false}"

export DATABASE_PATH JWT_SECRET

cat > /app/.env <<EOF
DATABASE_PATH=${DATABASE_PATH}
JWT_SECRET=${JWT_SECRET}
EOF

mkdir -p "$(dirname "$DATABASE_PATH")"
touch "$DATABASE_PATH"

echo "[entrypoint] running migrations..."
/app/migrate -db="$DATABASE_PATH" -m=/app/migrations up || true

if [ "$SEED" = "true" ] && [ ! -f "${DATABASE_PATH}.seeded" ]; then
  echo "[entrypoint] seeding database..."
  sqlite3 "$DATABASE_PATH" < /app/seed.sql
  touch "${DATABASE_PATH}.seeded"
fi

echo "[entrypoint] starting app on port ${PORT}..."
exec /app/app -env /app/.env -p "$PORT"
