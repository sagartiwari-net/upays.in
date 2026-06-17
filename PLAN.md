# UPIPays — Phased Development Plan

> **Rule:** Complete one phase before starting the next. Update [tracker.md](./tracker.md) after every meaningful milestone.

---

## Overview

| Phase | Name | Goal | Est. duration |
|-------|------|------|---------------|
| 0 | Foundation & Deploy | upays.in live, rebranded hub, separate DB | 1 week |
| 1 | Marketing Site | Homepage, pricing, FAQ, legal pages | 2 weeks |
| 2 | Merchant Auth & Onboarding | Self-signup, UPI setup wizard, merchant dashboard | 2–3 weeks |
| 3 | Subscriptions & Limits | Plans, order quotas, billing (manual first) | 2 weeks |
| 4 | Admin CMS | Manage pricing + dynamic pages from super admin | 1–2 weeks |
| 5 | Developer Docs | API reference, guides, Postman collection | 2 weeks |
| 6 | Plugins & SDKs | WooCommerce, aMember, Shopify, payment links | Ongoing |
| 7 | Production Hardening | Webhook retry, monitoring, abuse prevention | Ongoing |

---

## Phase 0 — Foundation & Deploy

**Goal:** UPIPays runs on `upays.in` with its own environment, separate from buyahref.com hub.

### Tasks

- [ ] DNS: `upays.in` → VPS IP (+ optional `api.upays.in`)
- [ ] Create web root: `/www/wwwroot/upays.in`
- [ ] Nginx/Apache vhost + SSL (Let's Encrypt)
- [ ] Create MySQL database `upipays` (see `LOCAL-SETUP.md`)
- [ ] Copy/adapt `payment-hub` from existing repo into this project
- [ ] Rebrand: Buyahref → **UPIPays** (admin UI, checkout page, emails)
- [ ] `.env` for production: `APP_NAME=UPIPays`, new DB credentials
- [ ] Run all migrations on `upipays` DB
- [ ] Seed super admin account
- [ ] Deploy Go binary + built admin static files
- [ ] Smoke test: admin login, create test merchant, test ₹1 UPI flow

### Deliverables

- `https://upays.in/admin` — super admin panel working
- `https://upays.in/pay/:token` — checkout page with UPIPays branding
- Separate DB — no shared data with buyahref.com hub

### Depends on

- VPS access, domain DNS, DB created on server

---

## Phase 1 — Marketing Site

**Goal:** Professional public website that converts visitors to signups.

### Tasks

- [ ] Choose stack: **Next.js** or **Astro** (recommended for SEO)
- [ ] Create `upipays-web/` folder in repo
- [ ] **Homepage**
  - Hero: "Accept UPI. Pay Zero Fees."
  - How it works (4 steps)
  - Features grid
  - Supported UPI apps logos
  - Testimonials section
  - CTA → Register
- [ ] **Pricing page** — 4 plan cards (Starter / Growth / Business / Pro)
  - Initially static JSON; Phase 4 makes it DB-driven
- [ ] **FAQ page**
- [ ] **Contact page** (WhatsApp link + email)
- [ ] **Legal pages:** Terms, Privacy, Refund Policy
- [ ] **Login / Register** links (UI only until Phase 2)
- [ ] Mobile responsive, fast Lighthouse score
- [ ] Deploy at `https://upays.in` (marketing) + hub at `/admin`, `/pay`, `/api`

### Deliverables

- Public site live at `upays.in`
- Pricing visible; "Get Started" leads to register (Phase 2)

### Design reference

- Light professional theme (current admin purple accent)
- Pricing cards similar to UPIGateway style — checkmarks / crosses per feature

---

## Phase 2 — Merchant Auth & Self-Service Onboarding

**Goal:** Any user can register, add UPI + Gmail, and accept payments without admin help.

### Tasks

- [ ] DB: `merchant_users` table (email, password_hash, merchant_id)
- [ ] API: `POST /auth/register`, `POST /auth/login`, `POST /auth/forgot-password`
- [ ] Merchant JWT (separate from super admin JWT)
- [ ] Create `upipays-merchant/` React app
- [ ] **Onboarding wizard (4 steps)**
  1. Business name + website domain
  2. UPI ID + payee name
  3. Gmail IMAP + app password + bank parser
  4. Test ₹1 payment → confirm success before "Go Live"
- [ ] **Merchant dashboard**
  - Overview: today revenue, orders, success rate
  - Orders list (own merchant only)
  - API keys: view + regenerate
  - Webhook URL config
  - UPI profile edit
  - Test payment button
- [ ] Auto-create `merchants` row + API key/secret on signup
- [ ] Row-level security: merchant sees only their data

### Deliverables

- End-to-end: register → UPI setup → API keys → website integration → payment success

---

## Phase 3 — Subscriptions & Plan Limits

**Goal:** Monetize via affordable monthly plans; enforce QR/order limits.

### Tasks

- [ ] DB: `subscription_plans`, `merchant_subscriptions`, `plan_features`
- [ ] Seed default plans (see pricing table in tracker.md notes)
- [ ] On order create: check `orders_used < plan_limit` for billing cycle
- [ ] Dashboard widget: "4,230 / 5,000 orders used"
- [ ] Block new orders when limit exceeded + upgrade CTA
- [ ] Plan expiry handling + grace period (optional 2–3 days)
- [ ] **Billing v1 (manual):**
  - Merchant pays via UPI to UPIPays UPI ID
  - Admin activates plan from super admin panel
- [ ] **Billing v2 (later):** Razorpay subscriptions or self-dogfood UPIPays

### Suggested launch pricing (editable in Phase 4 CMS)

| Plan | Price | Validity | Order/QR limit |
|------|-------|----------|----------------|
| Starter | ₹499 | 28 days | 5,000 |
| Growth | ₹999 | 28 days | 10,000 |
| Business | ₹1,999 | 28 days | 15,000 |
| Pro | ₹4,999 | 28 days | 25,000 |

**Free trial:** 20 QR requests, 7 days

### Deliverables

- Paid plans enforced; free trial works; manual activation flow for admin

---

## Phase 4 — Super Admin CMS

**Goal:** Change pricing and add pages without code deploy.

### Tasks

- [ ] Admin UI: **Plans manager** — CRUD plans, features, prices, limits, "Recommended" badge
- [ ] Admin UI: **Pages manager** — slug, title, meta, HTML/Markdown, publish/draft, nav visibility
- [ ] Public API: `GET /public/plans`, `GET /public/pages/:slug`
- [ ] Marketing site fetches pricing + dynamic pages from API
- [ ] Preview mode for draft pages (admin only)

### Deliverables

- Edit plan price in admin → reflects on `upays.in/pricing` within minutes
- Add `/about` or `/blog/post-1` from admin without deploy

---

## Phase 5 — Developer Documentation

**Goal:** Developers integrate without contacting support.

### Tasks

- [ ] Create `upipays-docs/` (Docusaurus recommended)
- [ ] Sections:
  - Getting started
  - Authentication (HMAC signing with examples)
  - API reference: `POST /orders/create`, `GET /orders/:id/verify`
  - Webhooks (payload + signature verification)
  - Checkout flow diagram
  - Error codes
  - SDKs: PHP, Node.js, Python
  - Plugin install guides
- [ ] Postman collection downloadable from docs
- [ ] Sandbox/test API keys for developers
- [ ] Deploy at `upays.in/docs`

### Deliverables

- Complete docs site; a developer can go live in under 30 minutes

---

## Phase 6 — Plugins & SDKs

**Goal:** One-click integration on popular platforms.

### Priority order

| # | Platform | Folder | Priority |
|---|----------|--------|----------|
| 1 | WordPress / WooCommerce | `plugins/woocommerce/` | High |
| 2 | aMember Pro | `plugins/amember/` | High (adapt existing) |
| 3 | Payment links / embed | `plugins/payment-links/` | High |
| 4 | Shopify | `plugins/shopify/` | Medium |
| 5 | WHMCS / OpenCart | `plugins/whmcs/` | Medium |
| 6 | Android SDK | `sdks/android/` | Low |

### Each plugin must include

- Hub URL setting: `https://upays.in`
- API Key + Secret fields
- Webhook URL auto-suggest
- Test payment button in settings
- README with install steps

---

## Phase 7 — Production Hardening

**Goal:** Reliable, secure, scalable SaaS.

### Tasks

- [x] Webhook outbound: write to `webhook_logs` + retry worker (DB queue, 5 retries)
- [ ] Rate limiting per merchant API key (Redis) — in-memory limiter live
- [ ] Email verification on signup
- [ ] Optional domain verification for merchants
- [x] Daily MySQL backup cron (`scripts/mysql-backup.sh` + `setup-cron.sh`)
- [x] Uptime monitoring (`scripts/uptime-check.sh` + `/health`)
- [ ] IMAP health alerts → email/WhatsApp to merchant
- [x] Abuse: signup throttling (5/hour per IP on register)
- [ ] Load test: 100 concurrent checkouts

---

## Architecture (target state)

```
upays.in
├── /                    → Marketing site (Phase 1)
├── /pricing             → Plans (Phase 1 static → Phase 4 dynamic)
├── /docs                → Developer docs (Phase 5)
├── /register            → Merchant signup (Phase 2)
├── /dashboard           → Merchant portal (Phase 2)
├── /admin               → Super admin (Phase 0)
├── /api/v1/*            → Merchant payment API (existing)
├── /pay/:token          → Customer checkout (existing)
└── /admin/api/*         → Platform admin API (existing + CMS Phase 4)
```

---

## Decisions log

Record final decisions here when confirmed:

| Decision | Choice | Date |
|----------|--------|------|
| Marketing site stack | _TBD: Next.js vs Astro_ | — |
| Free trial orders | 20 QR / 7 days | — |
| Billing v1 | Manual UPI + admin activate | — |
| buyahref.com hub | Separate — keep running independently | — |
| Design theme | Light + purple accent (match admin) | — |
| Launch pricing | Starter ₹499 / Growth ₹999 / Business ₹1999 / Pro ₹4999 | — |

---

## Related repos

| Repo | Purpose |
|------|---------|
| [sagartiwari-net/upays.in](https://github.com/sagartiwari-net/upays.in) | This project (UPIPays SaaS) |
| [sagartiwari-net/payment](https://github.com/sagartiwari-net/payment) | Original Payment Hub (buyahref.com) — source to fork from |
