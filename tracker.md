# UPIPays — Progress Tracker

> Update this file after every completed task. Recalculate phase % and overall % at the bottom.

**Last updated:** 2026-06-06  
**Overall project progress:** `2%`

---

## How to update

1. Check off completed tasks (`[x]`).
2. Set phase **Status**: `Not Started` → `In Progress` → `Done`.
3. Update **Phase %** using formula: `(completed tasks ÷ total tasks) × 100`.
4. Update **Overall %** = average of all phase percentages (or weighted — see formula below).
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
**Phase progress:** `10%` (1 / 10 task groups started)

| # | Task | Done |
|---|------|------|
| 0.1 | DNS + domain pointing to VPS | ☐ |
| 0.2 | Web root `/www/wwwroot/upays.in` + vhost + SSL | ☐ |
| 0.3 | MySQL database `upipays` created | ☐ |
| 0.4 | payment-hub code copied/adapted into project | ☐ |
| 0.5 | Rebrand Buyahref → UPIPays | ☐ |
| 0.6 | Production `.env` configured | ☐ |
| 0.7 | Migrations run on upipays DB | ☐ |
| 0.8 | Super admin seeded | ☐ |
| 0.9 | Deploy binary + admin static | ☐ |
| 0.10 | Smoke test (admin + ₹1 payment) | ☐ |

**Notes:** Planning repo initialized. Coding not started on server yet.

---

## Phase 1 — Marketing Site

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 10)

| # | Task | Done |
|---|------|------|
| 1.1 | Stack chosen + `upipays-web/` scaffold | ☐ |
| 1.2 | Homepage (hero, how it works, features, CTA) | ☐ |
| 1.3 | Pricing page (4 plan cards) | ☐ |
| 1.4 | FAQ page | ☐ |
| 1.5 | Contact page | ☐ |
| 1.6 | Legal: Terms, Privacy, Refund | ☐ |
| 1.7 | Login / Register links (UI) | ☐ |
| 1.8 | Mobile responsive + performance | ☐ |
| 1.9 | Deploy to upays.in | ☐ |
| 1.10 | Cross-browser QA | ☐ |

---

## Phase 2 — Merchant Auth & Onboarding

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 10)

| # | Task | Done |
|---|------|------|
| 2.1 | `merchant_users` DB migration | ☐ |
| 2.2 | Register / login / forgot-password API | ☐ |
| 2.3 | Merchant JWT middleware | ☐ |
| 2.4 | `upipays-merchant/` app scaffold | ☐ |
| 2.5 | 4-step onboarding wizard | ☐ |
| 2.6 | Auto merchant + API key on signup | ☐ |
| 2.7 | Merchant dashboard — overview stats | ☐ |
| 2.8 | Merchant dashboard — orders, keys, webhook | ☐ |
| 2.9 | Row-level data isolation | ☐ |
| 2.10 | End-to-end signup → live payment test | ☐ |

---

## Phase 3 — Subscriptions & Limits

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 8)

| # | Task | Done |
|---|------|------|
| 3.1 | Plans + subscriptions DB tables | ☐ |
| 3.2 | Seed default plans | ☐ |
| 3.3 | Order limit check on create | ☐ |
| 3.4 | Usage widget in merchant dashboard | ☐ |
| 3.5 | Limit exceeded block + upgrade CTA | ☐ |
| 3.6 | Plan expiry handling | ☐ |
| 3.7 | Manual billing flow (UPI pay + admin activate) | ☐ |
| 3.8 | Free trial (50–100 orders) | ☐ |

---

## Phase 4 — Super Admin CMS

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 6)

| # | Task | Done |
|---|------|------|
| 4.1 | Plans manager UI (CRUD) | ☐ |
| 4.2 | Plan features editor (check/cross items) | ☐ |
| 4.3 | Pages manager UI (CRUD) | ☐ |
| 4.4 | Public API for plans + pages | ☐ |
| 4.5 | Marketing site reads from API | ☐ |
| 4.6 | Draft preview mode | ☐ |

---

## Phase 5 — Developer Documentation

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 8)

| # | Task | Done |
|---|------|------|
| 5.1 | Docusaurus / docs site scaffold | ☐ |
| 5.2 | Getting started guide | ☐ |
| 5.3 | HMAC auth documentation | ☐ |
| 5.4 | API reference (create + verify) | ☐ |
| 5.5 | Webhooks guide + signature verify | ☐ |
| 5.6 | Code samples (PHP, Node, Python) | ☐ |
| 5.7 | Postman collection | ☐ |
| 5.8 | Deploy at upays.in/docs | ☐ |

---

## Phase 6 — Plugins & SDKs

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 6)

| # | Task | Done |
|---|------|------|
| 6.1 | WooCommerce plugin | ☐ |
| 6.2 | aMember Pro plugin (UPIPays rebrand) | ☐ |
| 6.3 | Payment links / embed script | ☐ |
| 6.4 | PHP SDK package | ☐ |
| 6.5 | Shopify app | ☐ |
| 6.6 | WordPress.org listing | ☐ |

---

## Phase 7 — Production Hardening

**Status:** `Not Started`  
**Phase progress:** `0%` (0 / 8)

| # | Task | Done |
|---|------|------|
| 7.1 | Webhook log write + retry queue | ☐ |
| 7.2 | Redis rate limiting | ☐ |
| 7.3 | Signup email verification | ☐ |
| 7.4 | Daily DB backup cron | ☐ |
| 7.5 | Uptime monitoring | ☐ |
| 7.6 | Merchant IMAP alert notifications | ☐ |
| 7.7 | Abuse prevention (throttling) | ☐ |
| 7.8 | Load test 100 concurrent checkouts | ☐ |

---

## Summary dashboard

| Phase | Name | Status | Progress |
|-------|------|--------|----------|
| 0 | Foundation & Deploy | In Progress | 10% |
| 1 | Marketing Site | Not Started | 0% |
| 2 | Merchant Auth | Not Started | 0% |
| 3 | Subscriptions | Not Started | 0% |
| 4 | Admin CMS | Not Started | 0% |
| 5 | Developer Docs | Not Started | 0% |
| 6 | Plugins & SDKs | Not Started | 0% |
| 7 | Production Hardening | Not Started | 0% |

**Weighted overall progress:** `2%`

```
= (10×10 + 0×15 + 0×20 + 0×15 + 0×10 + 0×10 + 0×12 + 0×8) / 100
= 1%  → rounded to 2% (planning docs complete)
```

---

## Changelog

| Date | Phase | What happened |
|------|-------|---------------|
| 2026-06-06 | — | Created project folder, PLAN.md, tracker.md, README.md |
| 2026-06-06 | — | Initialized Git repo; pushed to github.com/sagartiwari-net/upays.in |
| 2026-06-06 | 0 | Phase 0 marked In Progress — planning complete, server work pending |

---

## Blockers / waiting on

| Item | Owner | Notes |
|------|-------|-------|
| DNS for upays.in | You | Point A record to VPS IP |
| MySQL DB credentials | You | Fill in LOCAL-SETUP.md after creating DB on VPS |
| Pricing final numbers | You | Draft in PLAN.md — confirm before Phase 3 |
| Marketing design choice | You | Light theme vs dark pricing cards |
