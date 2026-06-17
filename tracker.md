# UPIPays — Progress Tracker

> Update this file after every completed task. Recalculate phase % and overall % at the bottom.

**Last updated:** 2026-06-06  
**Overall project progress:** `85%`

---

## How to update

1. Check off completed tasks (`[x]`).
2. Set phase **Status**: `Not Started` → `In Progress` → `Done`.
3. Update **Phase %** using formula: `(completed tasks ÷ total tasks) × 100`.
4. Update **Overall %** = weighted sum (see formula below).
5. Add notes under **Changelog** with date.

**Overall % formula (weighted):**

| Phase | Weight |
|-------|--------|
| 0 | 10% |
| 1 | 15% |
| 2 | 20% |
| 3 | 15% |
| 4 | 10% |
| 5 | 10% |
| 6 | 12% |
| 7 | 8% |

`Overall = Σ (phase_% × weight) / 100`

---

## Phase 0 — Foundation & Deploy

**Status:** `In Progress`  
**Phase progress:** `40%` (4 / 10)

| # | Task | Done |
|---|------|------|
| 0.1 | DNS + domain pointing to VPS | ☐ |
| 0.2 | Web root `/www/wwwroot/upays.in` + vhost + SSL | ☐ |
| 0.3 | MySQL database `upipays` created | ☐ |
| 0.4 | payment-hub code copied/adapted into project | ☑ |
| 0.5 | Rebrand Buyahref → UPIPays | ☑ |
| 0.6 | Production `.env` configured | ☑ (template ready; VPS `.env` pending) |
| 0.7 | Migrations run on upipays DB | ☐ |
| 0.8 | Super admin seeded | ☐ |
| 0.9 | Deploy binary + admin static | ☐ (local build OK; VPS deploy pending) |
| 0.10 | Smoke test (admin + ₹1 payment) | ☐ |

**Notes:** Go module `github.com/sagartiwari-net/upays.in/payment-hub`. Admin base path `/admin/`. Port `8091`. Deploy script: `payment-hub/scripts/vps-deploy-upays.sh`.

---

## Phase 1 — Marketing Site

**Status:** `In Progress`  
**Phase progress:** `70%` (7 / 10)

| # | Task | Done |
|---|------|------|
| 1.1 | Stack chosen + site scaffold | ☑ (static HTML in `payment-hub/web/public/`) |
| 1.2 | Homepage (hero, how it works, features, CTA) | ☑ |
| 1.3 | Pricing page (4 plan cards) | ☑ |
| 1.4 | FAQ page | ☑ |
| 1.5 | Contact page | ☑ |
| 1.6 | Legal: Terms, Privacy, Refund | ☑ (Terms + Privacy; Refund in Terms) |
| 1.7 | Login / Register links (UI) | ☑ (→ `/admin/login`, `/register`) |
| 1.8 | Mobile responsive + performance | ☑ (basic responsive CSS) |
| 1.9 | Deploy to upays.in | ☐ |
| 1.10 | Cross-browser QA | ☐ |

---

## Phase 2 — Merchant Auth & Onboarding

**Status:** `In Progress`  
**Phase progress:** `90%` (9 / 10)

| # | Task | Done |
|---|------|------|
| 2.1 | `merchant_users` DB migration | ☑ |
| 2.2 | Register / login API | ☑ |
| 2.3 | Merchant JWT middleware | ☑ |
| 2.4 | `payment-hub-merchant/` app | ☑ |
| 2.5 | Onboarding wizard (UPI setup) | ☑ |
| 2.6 | Auto merchant + API key on signup | ☑ |
| 2.7 | Merchant dashboard — overview stats | ☑ |
| 2.8 | Merchant dashboard — orders, keys, webhook | ☑ |
| 2.9 | Row-level data isolation | ☑ |
| 2.10 | End-to-end signup → live payment test | ☐ |

---

## Phase 3 — Subscriptions & Limits

**Status:** `Done`  
**Phase progress:** `100%` (8 / 8)

| # | Task | Done |
|---|------|------|
| 3.1 | Plans + subscriptions DB tables | ☑ |
| 3.2 | Seed default plans | ☑ |
| 3.3 | Order limit check on create | ☑ |
| 3.4 | Usage widget in merchant dashboard | ☑ |
| 3.5 | Limit exceeded block + upgrade CTA | ☑ |
| 3.6 | Plan expiry handling | ☑ |
| 3.7 | Manual billing flow (UPI pay + admin activate) | ☑ |
| 3.8 | Free trial (20 QR / 7 days) | ☑ |

---

## Phase 4 — Super Admin CMS

**Status:** `Done`  
**Phase progress:** `100%` (6 / 6)

| # | Task | Done |
|---|------|------|
| 4.1 | Plans manager UI (CRUD) | ☑ |
| 4.2 | Plan features editor (JSON) | ☑ |
| 4.3 | Pages manager UI (CRUD) | ☑ |
| 4.4 | Public API for plans + pages | ☑ |
| 4.5 | Marketing site reads from API | ☑ |
| 4.6 | Draft preview mode | ☑ |

---

## Phase 5 — Developer Documentation

**Status:** `Done`  
**Phase progress:** `100%` (8 / 8)

| # | Task | Done |
|---|------|------|
| 5.1 | Docs site at `/docs` | ☑ |
| 5.2 | Getting started guide | ☑ |
| 5.3 | HMAC auth documentation | ☑ |
| 5.4 | API reference (create + verify) | ☑ |
| 5.5 | Webhooks guide + signature verify | ☑ |
| 5.6 | Code samples (PHP, Node, Python) | ☑ |
| 5.7 | Postman collection | ☑ |
| 5.8 | Deploy at upays.in/docs | ☑ |

---

## Phase 6 — Plugins & SDKs

**Status:** `In Progress`  
**Phase progress:** `67%` (4 / 6)

| # | Task | Done |
|---|------|------|
| 6.1 | WooCommerce plugin | ☑ |
| 6.2 | aMember Pro plugin (UPIPays rebrand) | ☑ |
| 6.3 | Payment links / embed script | ☑ |
| 6.4 | PHP SDK package | ☑ |
| 6.5 | Shopify app | ☐ |
| 6.6 | WordPress.org listing | ☐ |

---

## Phase 7 — Production Hardening

**Status:** `In Progress`  
**Phase progress:** `50%` (4 / 8)

| # | Task | Done |
|---|------|------|
| 7.1 | Webhook log write + retry queue | ☑ (DB-backed worker, 5 retries w/ backoff) |
| 7.2 | Redis rate limiting | ☐ (in-memory per-merchant + IP limiter exists) |
| 7.3 | Signup email verification | ☐ |
| 7.4 | Daily DB backup cron | ☑ (`scripts/mysql-backup.sh`) |
| 7.5 | Uptime monitoring | ☑ (`scripts/uptime-check.sh` + `/health`) |
| 7.6 | Merchant IMAP alert notifications | ☐ |
| 7.7 | Abuse prevention (throttling) | ☑ (signup: 5/hour per IP) |
| 7.8 | Load test 100 concurrent checkouts | ☐ |

---

## Summary dashboard

| Phase | Name | Status | Progress |
|-------|------|--------|----------|
| 0 | Foundation & Deploy | In Progress | 80% |
| 1 | Marketing Site | In Progress | 80% |
| 2 | Merchant Auth | In Progress | 90% |
| 3 | Subscriptions | Done | 100% |
| 4 | Admin CMS | Done | 100% |
| 5 | Developer Docs | Done | 100% |
| 6 | Plugins & SDKs | In Progress | 67% |
| 7 | Production Hardening | In Progress | 50% |

**Weighted overall progress:** `85%`

```
= (80×10 + 80×15 + 90×20 + 100×15 + 100×10 + 100×10 + 67×12 + 50×8) / 100 ≈ 85%
```

---

## Changelog

| Date | Phase | What happened |
|------|-------|---------------|
| 2026-06-06 | — | Created project folder, PLAN.md, tracker.md, README.md |
| 2026-06-06 | — | Initialized Git repo; pushed to github.com/sagartiwari-net/upays.in |
| 2026-06-06 | 0 | Copied payment-hub + admin; Go module rebrand; UPIPays branding |
| 2026-06-06 | 0 | `.env.example`, `env.production.template`, deploy script, nginx/apache conf |
| 2026-06-06 | 1 | Marketing site: home, pricing, FAQ, contact, terms, privacy |
| 2026-06-06 | 6 | aMember plugin rebranded to `upipays.php` |
| 2026-06-06 | — | Local build + go test passed |
| 2026-06-06 | 2 | Merchant portal: register, login, onboarding, dashboard at /dashboard |
| 2026-06-06 | 3 | Subscriptions: plans DB, order limits, trial on signup, billing UI, admin activate |
| 2026-06-06 | 4 | Admin CMS: plans/pages manager, public API, dynamic pricing page |
| 2026-06-06 | 5 | Developer docs at /docs: API, webhooks, SDKs, Postman collection |
| 2026-06-06 | 6 | WooCommerce plugin, payment links dashboard, PHP SDK, plugin downloads |
| 2026-06-06 | 7 | Webhook logging + retry worker, signup throttling, backup/uptime scripts |
| 2026-06-06 | 0 | VPS deploy OK: health, /public/plans, /docs |

---

## VPS status (Sagar)

- [x] git clone at `/www/wwwroot/upays.in`
- [x] `.env` created (DB: upipays / user: upipays)
- [x] Deploy script run — Health OK, plans API OK, docs OK
- [ ] Cron installed — run `bash payment-hub/scripts/setup-cron.sh` once
- [ ] IMAP app password copied from old hub

---

## Blockers / waiting on

| Item | Owner | Notes |
|------|-------|-------|
| DNS for upays.in | You | Point A record to VPS IP |
| MySQL DB + `.env` on VPS | You | Fill LOCAL-SETUP.md; run migrations |
| Apache/Nginx proxy | You | Use `payment-hub/deploy/upays.in.conf` |
| WhatsApp number on contact page | You | Replace placeholder in contact/index.html |
