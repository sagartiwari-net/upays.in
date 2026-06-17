#!/usr/bin/env bash
# Daily MySQL backup for UPIPays — add to cron:
#   0 3 * * * /www/wwwroot/upays.in/payment-hub/scripts/mysql-backup.sh >> /www/wwwroot/upays.in/payment-hub/logs/backup.log 2>&1
set -euo pipefail

ROOT="${DEPLOY_ROOT:-/www/wwwroot/upays.in/payment-hub}"
BACKUP_DIR="${BACKUP_DIR:-$ROOT/backups}"
KEEP_DAYS="${KEEP_DAYS:-7}"

cd "$ROOT"
if [[ ! -f .env ]]; then
  echo "ERROR: .env not found in $ROOT"
  exit 1
fi

set -a
source .env
set +a

mkdir -p "$BACKUP_DIR"
STAMP="$(date -u +%Y%m%d-%H%M%S)"
OUT="$BACKUP_DIR/upipays-${STAMP}.sql.gz"

mysqldump \
  -h "${DB_HOST:-127.0.0.1}" \
  -P "${DB_PORT:-3306}" \
  -u "$DB_USER" \
  -p"$DB_PASSWORD" \
  --single-transaction \
  --routines \
  --triggers \
  "$DB_NAME" | gzip > "$OUT"

echo "Backup saved: $OUT ($(du -h "$OUT" | awk '{print $1}'))"

find "$BACKUP_DIR" -name 'upipays-*.sql.gz' -mtime +"$KEEP_DAYS" -delete 2>/dev/null || true
