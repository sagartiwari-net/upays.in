#!/usr/bin/env bash
# Deploy UPIPays on VPS (aaPanel / BT Panel)
# Web root: /www/wwwroot/upays.in
set -euo pipefail

ROOT="${DEPLOY_ROOT:-/www/wwwroot/upays.in}"
PORT="${UPIPAYS_PORT:-8091}"

echo "==> Deploying UPIPays to $ROOT"

cd "$ROOT"
git pull origin main

echo "==> Building admin UI"
cd payment-hub-admin
npm ci
npm run build

echo "==> Building Go server"
cd ../payment-hub
go build -o bin/upipays ./cmd/server

echo "==> Running migrations (if migrate tool available)"
if command -v migrate &>/dev/null && [[ -f .env ]]; then
  source .env 2>/dev/null || true
  migrate -path migrations -database "mysql://${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST:-127.0.0.1}:${DB_PORT:-3306})/${DB_NAME}" up || echo "Migration skipped or failed — check manually"
fi

echo "==> Restarting server on port $PORT"
pkill -f "bin/upipays" 2>/dev/null || true
mkdir -p logs
nohup ./bin/upipays > logs/app.log 2>&1 &
sleep 1
curl -sf "http://127.0.0.1:${PORT}/health" && echo " Health OK" || echo " WARN: health check failed — check logs/app.log"

echo "==> Done. Site: https://upays.in"
