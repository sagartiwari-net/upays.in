#!/usr/bin/env bash
# Uptime check — use with cron or external monitor (UptimeRobot, etc.)
# Exit 0 = healthy, 1 = unhealthy
set -euo pipefail

URL="${UPIPAYS_HEALTH_URL:-https://upays.in/health}"
TIMEOUT="${UPIPAYS_HEALTH_TIMEOUT:-10}"

if curl -sf --max-time "$TIMEOUT" "$URL" | grep -q '"status":"ok"'; then
  echo "OK $URL"
  exit 0
fi

echo "FAIL $URL"
exit 1
