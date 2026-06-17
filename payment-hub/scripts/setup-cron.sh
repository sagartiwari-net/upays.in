#!/usr/bin/env bash
# Install daily backup + uptime cron for UPIPays (idempotent).
set -euo pipefail

HUB="${DEPLOY_ROOT:-/www/wwwroot/upays.in/payment-hub}"
MARKER="# upipays-cron"

mkdir -p "$HUB/logs" "$HUB/backups"
chmod +x "$HUB/scripts/mysql-backup.sh" "$HUB/scripts/uptime-check.sh"

current="$(crontab -l 2>/dev/null || true)"
if echo "$current" | grep -qF "$MARKER"; then
  echo "Cron already configured for UPIPays"
  crontab -l | grep -F upipays || true
  exit 0
fi

{
  [[ -n "$current" ]] && echo "$current"
  echo "$MARKER"
  echo "0 3 * * * $HUB/scripts/mysql-backup.sh >> $HUB/logs/backup.log 2>&1"
  echo "*/5 * * * * $HUB/scripts/uptime-check.sh >> $HUB/logs/uptime.log 2>&1"
} | crontab -

echo "Cron installed:"
crontab -l | grep -F upipays
