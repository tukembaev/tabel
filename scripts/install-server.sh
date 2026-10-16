#!/usr/bin/env bash
# Install binaries + systemd units on a Linux host (run as root on the server).
# Usage: sudo ./scripts/install-server.sh
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/tabel}"
ENV_FILE="${ENV_FILE:-/etc/tabel/tabel.env}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if [[ ! -f "$ROOT/bin/tabel-server" || ! -f "$ROOT/bin/tabel-worker" ]]; then
  echo "binaries missing. Build first: ./scripts/build-linux.sh" >&2
  exit 1
fi

id -u tabel &>/dev/null || useradd --system --home "$APP_DIR" --shell /usr/sbin/nologin tabel

mkdir -p "$APP_DIR/bin" "$APP_DIR/migrations" /etc/tabel /var/log/tabel
cp -f "$ROOT/bin/tabel-server" "$ROOT/bin/tabel-worker" "$APP_DIR/bin/"
cp -a "$ROOT/migrations/." "$APP_DIR/migrations/"
chmod 755 "$APP_DIR/bin/tabel-server" "$APP_DIR/bin/tabel-worker"
chown -R tabel:tabel "$APP_DIR" /var/log/tabel

if [[ ! -f "$ENV_FILE" ]]; then
  cp "$ROOT/.env.example" "$ENV_FILE"
  chmod 640 "$ENV_FILE"
  chown root:tabel "$ENV_FILE"
  echo "Created $ENV_FILE — edit JWT_SECRET, DATABASE_URL, REDIS_URL before start."
fi

cp -f "$ROOT/deploy/systemd/tabel-api.service" /etc/systemd/system/
cp -f "$ROOT/deploy/systemd/tabel-worker.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable tabel-api tabel-worker

echo "Installed. Next:"
echo "  1) Edit $ENV_FILE"
echo "  2) goose -dir $APP_DIR/migrations postgres \"\$DATABASE_URL\" up"
echo "  3) systemctl start tabel-api tabel-worker"
echo "  4) curl http://127.0.0.1:8080/healthz"
echo "API listens on HTTP_ADDR from env (default :8080). Point existing reverse proxy to it, or open the port directly — no Docker/nginx-for-containers needed."
