#!/usr/bin/env bash
# Copy Gmail IMAP app password from buyahref Payment Hub .env → UPIPays .env
set -euo pipefail

OLD_ENV="${OLD_HUB_ENV:-/www/wwwroot/buyahref.com/payment/payment-hub/.env}"
NEW_ENV="${1:-/www/wwwroot/upays.in/payment-hub/.env}"

if [[ ! -f "$OLD_ENV" ]]; then
  echo "Old hub .env not found: $OLD_ENV"
  echo ""
  echo "Find it with:"
  echo "  find /www/wwwroot -path '*/payment-hub/.env' 2>/dev/null"
  exit 1
fi

if [[ ! -f "$NEW_ENV" ]]; then
  echo "UPIPays .env not found: $NEW_ENV"
  exit 1
fi

echo "Reading from: $OLD_ENV"

IMAP_PW=$(grep -E '^IMAP_PASSWORD=' "$OLD_ENV" | head -1 | cut -d= -f2- | tr -d '\r"')
IMAP_USER=$(grep -E '^IMAP_USER=' "$OLD_ENV" | head -1 | cut -d= -f2- | tr -d '\r"')
UPI_ID=$(grep -E '^UPI_ID=' "$OLD_ENV" | head -1 | cut -d= -f2- | tr -d '\r"')

if [[ -z "$IMAP_PW" || "$IMAP_PW" == *your* ]]; then
  echo "IMAP_PASSWORD missing or placeholder in old .env"
  echo "Create new: https://myaccount.google.com/apppasswords"
  exit 1
fi

cp "$NEW_ENV" "${NEW_ENV}.bak.$(date +%s)"

python3 - "$NEW_ENV" "$IMAP_USER" "$IMAP_PW" "$UPI_ID" <<'PY'
import re, sys
path, imap_user, imap_pw, upi_id = sys.argv[1:5]
text = open(path).read()

def set_key(text, key, val):
    if not val:
        return text
    line = f"{key}={val}"
    if re.search(rf"^{re.escape(key)}=", text, re.M):
        return re.sub(rf"^{re.escape(key)}=.*", line, text, flags=re.M)
    return text + "\n" + line + "\n"

text = set_key(text, "IMAP_USER", imap_user)
text = set_key(text, "IMAP_PASSWORD", imap_pw)
text = set_key(text, "UPI_ID", upi_id)
open(path, "w").write(text)
print("Updated IMAP_USER, IMAP_PASSWORD, UPI_ID in", path)
PY

echo "Done. Verify: grep IMAP_ $NEW_ENV | sed 's/IMAP_PASSWORD=.*/IMAP_PASSWORD=***hidden***/'"
