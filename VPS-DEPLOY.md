# UPIPays VPS Deploy — upays.in

Path: `/www/wwwroot/upays.in`

## Aapne ab tak kiya

```bash
cd /www/wwwroot/upays.in
git clone https://github.com/sagartiwari-net/upays.in.git .
cp payment-hub/env.production.template payment-hub/.env
# .env edit kiya (DB, JWT, UPI, etc.)
```

---

## Purana Gmail App Password kaise dekhein

App password **plain text** sirf `.env` file mein hota hai (database mein encrypted hai).

### Option 1 — Old buyahref hub ki `.env` (sabse aasaan)

VPS par SSH:

```bash
grep IMAP_PASSWORD /www/wwwroot/buyahref.com/payment/payment-hub/.env
```

Agar path alag ho:

```bash
find /www/wwwroot -path '*/payment-hub/.env' 2>/dev/null
cat /path/to/payment-hub/.env | grep IMAP_
```

### Option 2 — Auto copy script

```bash
cd /www/wwwroot/upays.in
bash payment-hub/scripts/copy-imap-from-old-hub.sh
```

Ye purani `.env` se `IMAP_PASSWORD`, `IMAP_USER`, `UPI_ID` nayi `.env` mein copy karega.

### Option 3 — Gmail se naya app password

Agar purani file nahi mili:

1. https://myaccount.google.com/apppasswords
2. 2-Step Verification ON honi chahiye
3. App: **Mail**, Device: **Other (UPIPays)**
4. 16-character password copy → `.env` mein `IMAP_PASSWORD=xxxx xxxx xxxx xxxx` (spaces hata sakte ho)

---

## .env checklist

File: `/www/wwwroot/upays.in/payment-hub/.env`

| Key | Aapka value |
|-----|-------------|
| DB_USER | `upipays` |
| DB_NAME | `upipays` |
| APP_PORT | `8091` |
| APP_URL | `https://upays.in` |
| IMAP_PASSWORD | old hub se copy ya naya app password |

`IMAP_PASSWORD=your_gmail_app_password` placeholder **replace** karna zaroori hai.

---

## One-command install

```bash
cd /www/wwwroot/upays.in
git pull origin main

# Pehle IMAP copy (optional)
bash payment-hub/scripts/copy-imap-from-old-hub.sh

# Full install + admin user
bash payment-hub/scripts/vps-first-install.sh admin@upays.in 'ApnaStrongPassword123'
```

---

## Manual steps (agar script fail ho)

```bash
cd /www/wwwroot/upays.in/payment-hub

# Tables
mysql -u upipays -p upipays < migrations/install_upipays_full.sql

# Admin
go run ./cmd/seedadmin admin@upays.in 'YourPassword'

# Build
cd ../payment-hub-admin && npm ci && npm run build
cd ../payment-hub && go build -o bin/upipays ./cmd/server

# Run
pkill -f bin/upipays || true
nohup ./bin/upipays > logs/app.log 2>&1 &

curl http://127.0.0.1:8091/health
```

---

## aaPanel reverse proxy

Go app port **8091** par chalti hai. Site **upays.in** ke liye:

**Nginx** (recommended):

```nginx
location / {
    proxy_pass http://127.0.0.1:8091;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**Apache** — see `payment-hub/deploy/upays.in.conf`

SSL: Let's Encrypt on upays.in

---

## Test

| URL | Expected |
|-----|----------|
| https://upays.in | Homepage |
| https://upays.in/pricing | Pricing |
| https://upays.in/admin/login | Admin login |
| https://upays.in/health | JSON ok |

Admin login ke baad: **UPI Profiles** → Add profile (UPI + Gmail) → **Add Website** → test payment.

---

## Logs

```bash
tail -f /www/wwwroot/upays.in/payment-hub/logs/app.log
```
