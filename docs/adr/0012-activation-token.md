# Account Activation via Cryptographically Random Token

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
