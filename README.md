# UPIPays — upays.in

Affordable UPI Dynamic QR payment service for merchants who don't have traditional payment gateway access.

**Brand:** UPIPays  
**Domain:** [upays.in](https://upays.in)  
**GitHub:** [sagartiwari-net/upays.in](https://github.com/sagartiwari-net/upays.in)

---

## What we're building

Merchants sign up → add their UPI ID + Gmail (IMAP) → start accepting payments with **0% transaction fee**. We charge an affordable **monthly subscription** (similar model to [UPIGateway](https://upigateway.com/) and [UPI Gateway.dev](https://www.upigateway.dev/)).

Built on top of the existing **Payment Hub** core (Go + React): UPI QR checkout, IMAP auto-verification, HMAC API, webhooks.

---

## Project docs

| File | Purpose |
|------|---------|
| [PLAN.md](./PLAN.md) | Full phased roadmap — what to build and in what order |
| [tracker.md](./tracker.md) | Live progress tracker with % completion per phase |
| `LOCAL-SETUP.md` | Server paths, DB credentials, GitHub notes (**local only — not on GitHub**) |

---

## Repo structure (current)

```
upays.in/
├── PLAN.md                    ← phased roadmap
├── tracker.md                 ← progress %
├── LOCAL-SETUP.md             ← credentials (local only, gitignored)
├── payment-hub/               ← Go API + marketing static + admin build
│   ├── web/public/            ← Homepage, pricing, FAQ, etc.
│   ├── web/admin/             ← Built super-admin React app
│   ├── scripts/vps-deploy-upays.sh
│   └── deploy/upays.in.conf   ← Reverse proxy config
└── payment-hub-merchant/        ← Merchant dashboard source (build → payment-hub/web/merchant)
```

---

## Quick start (when coding begins)

1. Read `LOCAL-SETUP.md` on your machine for VPS path, DB, and deploy commands.
2. Check [tracker.md](./tracker.md) for current phase and what's done.
3. Follow [PLAN.md](./PLAN.md) for the active phase only — don't skip ahead.

---

## Legal note (product)

UPIPays provides **Dynamic QR generating + payment verification** service. It is **not** a licensed payment aggregator under RBI. Merchants receive payments directly in their own UPI-linked bank account.
