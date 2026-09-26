# Email Delivery via Resend, Synchronous in MVP

**Date:** 2026-07-21

**Decision:** Use Resend's HTTP API for transactional email delivery, sending synchronously from the request handler. No queue, no worker, no Redis dependency.

**Details:**
- Package: `server/internal/mailer` wraps Resend's REST API (`POST /emails`) with a simple `Mailer` interface (`SendActivationEmail`)
- Config: `RESEND_API_KEY` env var, `FROM_EMAIL` env var
- Sandbox mode (onboarding@resend.dev) for local dev; verified domain for production
- When `RESEND_API_KEY` is empty, `NoopMailer` is used (no emails sent) — safe for local dev without credentials

**Rationale:**
- Consistent with ADR-007 (Redis/workers deferred) — no queue infrastructure needed yet
- Consistent with ADR-012's own note that synchronous sending is MVP-acceptable
- Resend's plain HTTP API fits the existing Go handler pattern better than raw SMTP
- Volume is low (per-user activation emails, not bulk blast) — no queue needed yet
- If bulk-CSV import throughput becomes an issue later, revisit with basic goroutine-limited concurrency before reaching for a worker/queue

**What changed from ADR-012's plan:**
- ADR-012 planned the activation token schema and endpoint but deferred email delivery
- Now wired: `/activate` and `/resend-activation` endpoints + mailer integration
- Resend rate limit: DB query (`account_activation_tokens` ordered by `created_at desc`), reject if last token < 5 minutes old. Resend transactionally revokes all prior unused tokens before issuing a replacement. No new table/Redis needed.
- Activation token creation hooked into `RequestAccess` flow — users get the email immediately on sign-up

**Env vars added:**
- `RESEND_API_KEY` — Resend API key (different per environment, not committed)
- `FROM_EMAIL` — sender address (onboarding@resend.dev local, verified domain prod)
- `FRONTEND_URL` — base URL for building activation links (already existed for CORS)

**Deferred (not MVP):**
- Async email queue / worker
- Delivery status tracking
- Email open/click tracking
