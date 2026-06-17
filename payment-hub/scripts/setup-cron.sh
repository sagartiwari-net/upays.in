#!/usr/bin/env bash
# Install daily backup + uptime cron for UPIPays (idempotent — safe to re-run).
set -euo pipefail

HUB="${DEPLOY_ROOT:-/www/wwwroot/upays.in/payment-hub}"
MARKER="# upipays-cron"

mkdir -p "$HUB/logs" "$HUB/backups"
chmod +x "$HUB/scripts/mysql-backup.sh" "$HUB/scripts/uptime-check.sh"

current="$(crontab -l 2>/dev/null || true)"
cleaned="$(printf '%s\n' "$current" \
  | grep -vF "$MARKER" \
  | grep -v 'payment-hub/scripts/mysql-backup.sh' \
  | grep -v 'payment-hub/scripts/uptime-check.sh' \
  | sed '/^[[:space:]]*$/d' || true)"

{
  [[ -n "$cleaned" ]] && printf '%s\n' "$cleaned"
  echo "$MARKER"
  echo "0 3 * * * $HUB/scripts/mysql-backup.sh >> $HUB/logs/backup.log 2>&1"
  echo "*/5 * * * * $HUB/scripts/uptime-check.sh >> $HUB/logs/uptime.log 2>&1"
} | crontab -

echo "Cron configured (deduped):"
crontab -l | grep -F upipays
