# Decision Log

## ADR-001: Use Modular Monolith First

Decision: Use one Go REST API with internal modules.

Reason:

- The domain is connected.
- Transactions matter.
- Deployment stays simple.
- The team can move faster.

Trade-off:

- Requires discipline to maintain module boundaries.

## ADR-002: Use PostgreSQL as Source of Truth

Decision: Use PostgreSQL for all core data.

Reason:

- Strong relational workflows
- Transactions
- Foreign keys
- JSONB eligibility rules
- Good reporting support

## ADR-003: Use USN as Student Identity Key

Decision: USN is the primary student/alumni identity key when available.

Reason:

- MITT does not provide official student emails.
- USN includes joining year, branch, and sequence.
- USN helps avoid duplicate accounts.

## ADR-004: Use Gmail Invites Plus Manual Approval

Decision: Use Gmail IDs already shared with the college when available; otherwise HOD/admin can manually approve access.

Reason:

- This matches the current college reality.
- It avoids depending on official college email.

## ADR-005: Use Scoped RBAC

Decision: Use role assignments with scopes.

Reason:

- A student can also be a coordinator.
- A coordinator can be scoped to a department or club.
- HOD permissions should be department-scoped.

## ADR-006: Require Event Approval

Decision: Student coordinator events require HOD review and principal/admin final approval.

Reason:

- Events need official records.
- Department authority and college authority both matter.

## ADR-007: Defer Redis and Workers

Decision: Do not introduce Redis or background workers in MVP unless the need becomes real.

Reason:

- PostgreSQL and synchronous APIs are enough for the first version.
- Extra infrastructure increases operational burden.

## ADR-008: Keep Applicant Data Sensitive

Decision: Placement applicant data is restricted to the student, placement officer, principal, and admin. HODs see department summaries unless explicitly allowed.

Reason:

- Application and shortlisting status is sensitive.
- Least privilege matters.

## ADR-010: Make Repo Public with Automated Review and Branch Protection

**Date:** 2026-07-16

Decision: Make the LINKS repository public, add CodeRabbit as an automated PR reviewer, and enforce a GitHub ruleset on `master` that blocks direct pushes — all changes must go through a pull request.

Reason:

- Public visibility invites community contributions and external review.
- CodeRabbit provides fast, consistent automated feedback on every PR, reducing manual review burden.
- Branch protection on `master` prevents accidental force-pushes and ensures every change is reviewed before merging.

Trade-off:

- Requires discipline to keep PRs small and well-described.
- CodeRabbit is an external service — review quality depends on its configuration and availability.

## ADR-009: Use Multi-Channel Notifications

Decision: Use in-app notifications as the permanent record, email for important official fallback, and web push for opted-in users when the app is closed.

Reason:

- Students may not keep LINKS open.
- Placement links and shortlisting updates are time-sensitive.
- Web push depends on browser/device permission and is not guaranteed.
- Email is more reliable for official communication.

Trade-off:

- Requires notification preferences, delivery tracking, and a worker/outbox system.

## ADR-011: Use GORM with Explicit SQL Migrations

Decision: Use GORM as the Go persistence library for the initial modular monolith, while keeping schema changes in reviewed SQL migration files.

Reason:

- The existing Go scaffold already uses GORM.
- GORM supports ordinary CRUD without exposing persistence models at the API boundary.
- Explicit SQL keeps PostgreSQL constraints, indexes, and workflow-critical schema changes reviewable.

Trade-off:

- Complex reports and targeting queries need deliberate SQL or carefully reviewed GORM queries.
- The project may adopt `sqlc` later if query complexity makes generated typed SQL more valuable.

## ADR-012: Account Activation via Cryptographically Random Token

**Date:** 2026-07-18

**Decision:** Account activation uses a cryptographically random, single-use token emailed to the user's Gmail, valid 7 days, used to set a password and move status from `pending` to `active`.

**Context:** Two entry paths converge on the same activation step:

1. **Admin/HOD bulk CSV import** — for users already on the college's Gmail/USN list. The import creates `pending` users and issues activation emails synchronously (MVP), reporting per-row success/failure.
2. **Student self-service access request** — for everyone else. The request is reviewed by HOD/Admin per ADR-004, then an activation email is issued.

Both paths use the identical token flow: generate a 32-byte random token (crypto/rand, base64.RawURLEncoding), hash it with SHA-256 for storage, email the raw token via a magic link, user submits token + password, server verifies SHA-256 hash, marks token used, hashes password (Argon2id/bcrypt), flips status to `active`. All state changes (token consumed + user updated) happen in a single transaction with an affected-rows check on the conditional consume to prevent replay.

**Rationale:**

- Email ownership proves identity without requiring an official college email domain.
- Single-use + 7-day expiry limits blast radius of leaked links.
- **SHA-256 for token_hash** (not bcrypt/argon2) — tokens are high-entropy random strings, so a fast hash is sufficient and avoids the DoS vector of slow hashes on verification endpoints.
- Converging both entry paths on one activation step keeps the state machine simple.

**Trade-offs:**

- Requires email delivery (Resend/SMTP) before Phase 1 is fully usable.
- Bulk CSV import is synchronous for MVP; per-row success/failure reported in response.
- Rate limiting on resend-activation: query `account_activation_tokens` by `user_id` ordered by `created_at desc`, reject if last token < 5 minutes old. Resend transactionally revokes or marks all prior unused tokens before issuing a replacement. No new table/Redis needed.

**Open Questions Resolved:**

- Token format: 32 bytes crypto/rand, base64.RawURLEncoding (not UUIDv4).
- Hashing: SHA-256 for activation and refresh tokens (fast hash for high-entropy random strings).
- Resend rate-limit: DB query on existing table, no new infra.
- Bulk CSV: synchronous, per-row status in response.

## ADR-013: Username Strategy — Auto-Generated with Collision Retry, Planned Real-Time Check

**Date:** 2026-07-21

**Decision:** For MVP, usernames are auto-generated from the user's full name during
request-access, with a collision-aware retry loop. A future phase will add a
user-chosen username with a real-time availability check endpoint.

**Phase 1 (current):** Auto-generate username at signup from `full_name`:
lowercase, spaces → dots, strip non-alphanumeric, append `UnixMilli() % 10000`.
On unique constraint collision, regenerate suffix (new timestamp or increment)
and retry up to 5 times. This is dead code once the real-time check ships.

**Phase 2 (future):**
- `GET /api/v1/profiles/check-username?u=<value>` — public endpoint
- Request validation: 3–20 chars, `[a-z0-9._]`, no start/end with `.` or `_`
- Reserved usernames list checked before DB query
- DB: `SELECT 1 FROM profiles WHERE lower(username) = $1 LIMIT 1`
- Suggestion generation when taken: `base+1`, `base+2`, `base+3`
- Per-IP in-memory rate limiter (~20 req / 10s), no Redis per ADR-007
- Frontend: `useDebounce` (350ms) + `useUsernameCheck` hooks
- Wire into request-access form with status indicator and suggestion picker

**Rationale:**
- Usernames are unique and case-insensitive via existing `idx_profiles_username` index
- No citext extension — follow existing pattern (normalize in Go, `lower()` in query)
- Auto-generate is temporary; real-time check gives users control over their handle
- In-memory rate limiter avoids Redis dependency per ADR-007
- Frontend debounce prevents spamming the public endpoint on keystroke

**Trade-offs:**
- Auto-generated usernames are ugly (e.g. `johndoe4382`) — acceptable for MVP
- Collision is rare but possible; retry loop prevents 500 errors
- Reserved list is manual — needs maintenance as roles evolve
- No rate limiting on request-access itself (enumeration risk accepted for MVP)

## ADR-014: Email Delivery via Resend, Synchronous in MVP

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
