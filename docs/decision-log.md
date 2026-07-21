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
