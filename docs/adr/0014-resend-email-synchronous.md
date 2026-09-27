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

**Notes (2026-09-27), bulk student import (#17):**
- The import sends its Activation emails after every row is saved, in batches of up to 100 through Resend's batch endpoint (`POST /emails/batch`, all or nothing per batch), instead of one request per row. A failed batch leaves its rows created with a note, and their tokens invalidated, so the student uses Resend activation.
- A file holds at most 200 rows, so a whole import fits inside the API's 30 second `WriteTimeout`.
- Expected time per row: each row is one transaction of about 11 database round trips (begin, email and USN checks, Department lock, user, profile, Student identity, role, token, audit log, commit). Against a local Postgres that is about 3 ms per row, so 200 rows take under a second (`TestAFullImportFinishesWellInsideTheWriteTimeout` logs the figure). The email batches add two Resend requests. The cost grows with the round trip to the database: at the roughly 230 ms seen from Vercel to Neon in ap-southeast-1 (#77), a row is about 2.5 s and 200 rows would not fit. The deployed API has to sit next to its database for this limit to hold; otherwise the import needs fewer round trips per row or to move off the request.

**Deferred (not MVP):**
- Async email queue / worker
- Delivery status tracking
- Email open/click tracking
