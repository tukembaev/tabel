#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DSN="${DATABASE_URL:-postgres://tabel:tabel@localhost:5432/tabel?sslmode=disable}"
if command -v goose >/dev/null 2>&1; then
  goose -dir "$ROOT/migrations" postgres "$DSN" up
else
  echo "goose not found; applying SQL via psql"
  for f in "$ROOT/migrations/"*.sql; do
    echo "applying $f"
    sed '/^-- +goose Down/,$$d;/^-- +goose/d' "$f" | psql "$DSN" -v ON_ERROR_STOP=1
  done
fi
