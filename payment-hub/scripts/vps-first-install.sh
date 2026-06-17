#!/usr/bin/env bash
# First-time UPIPays install on VPS (run as root from repo root)
# Usage:
#   cd /www/wwwroot/upays.in
#   bash payment-hub/scripts/vps-first-install.sh admin@upays.in 'YourAdminPassword'
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HUB="$ROOT/payment-hub"
ADMIN_EMAIL="${1:-}"
ADMIN_PASS="${2:-}"

cd "$ROOT"

if [[ ! -f "$HUB/.env" ]]; then
  echo "Missing $HUB/.env — copy env.production.template and edit first."
  exit 1
fi

# shellcheck disable=SC1091
set -a
source "$HUB/.env"
set +a

echo "==> 1/6 Optional: copy IMAP from old buyahref hub"
if [[ -f "$HUB/scripts/copy-imap-from-old-hub.sh" ]]; then
  bash "$HUB/scripts/copy-imap-from-old-hub.sh" "$HUB/.env" || echo "(skip — set IMAP_PASSWORD manually in .env)"
  set -a; source "$HUB/.env"; set +a
fi

echo "==> 2/6 MySQL tables"
mysql -h "${DB_HOST:-127.0.0.1}" -P "${DB_PORT:-3306}" -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
  < "$HUB/migrations/install_upipays_full.sql"
echo "Tables OK"

echo "==> 3/6 Admin user"
if [[ -z "$ADMIN_EMAIL" || -z "$ADMIN_PASS" ]]; then
  echo "Skip seedadmin — run manually:"
  echo "  cd $HUB && go run ./cmd/seedadmin admin@upays.in 'YourPassword'"
else
  cd "$HUB"
  go run ./cmd/seedadmin "$ADMIN_EMAIL" "$ADMIN_PASS"
fi

echo "==> 4/6 Node + admin build"
if ! command -v node &>/dev/null; then
  echo "WARN: node not found — install Node 18+ or build admin locally and git pull"
else
  cd "$ROOT/payment-hub-admin"
  npm ci
  npm run build
fi

echo "==> 5/6 Go build"
cd "$HUB"
go build -o bin/upipays ./cmd/server

echo "==> 6/6 Start server"
mkdir -p logs
pkill -f "bin/upipays" 2>/dev/null || true
sleep 1
nohup ./bin/upipays > logs/app.log 2>&1 &
sleep 2

if curl -sf "http://127.0.0.1:${APP_PORT:-8091}/health" >/dev/null; then
  echo "Health OK on port ${APP_PORT:-8091}"
else
  echo "WARN: health check failed — tail -f $HUB/logs/app.log"
fi

cat <<EOF

========================================
Next steps (aaPanel):
1. Site upays.in → Reverse proxy ALL / → http://127.0.0.1:${APP_PORT:-8091}
   (see payment-hub/deploy/upays.in.conf)
2. SSL certificate for upays.in
3. Open https://upays.in — homepage
4. Admin: https://upays.in/admin/login
5. Add UPI Profile in admin (UPI ID + Gmail app password) if not bootstrapped
========================================
EOF
