# LINKS Project Progress Tracker

**Last Updated:** 2026-07-22
**Current Phase:** Phase 1 — Identity and Access

---

## 📋 Current Phase Overview

| Phase                              | Status         | Start Date | Expected End |
| ---------------------------------- | -------------- | ---------- | ------------ |
| **Phase 0: Foundation**            | ✅ Complete     | 2026-06-19 | 2026-07-16   |
| **Phase 1: Identity and Access**   | 🔄 In Progress | 2026-07-18 | TBD          |

> Phase numbering follows `roadmap.md` and `implementation.md`.

### Phase Goals (per `roadmap.md` Phase 1 / `implementation.md` Step 5)

- [x] 5.1 Identity schema migrations (`users`, `profiles`, `student_identities`, `departments`, `role_assignments`, `audit_logs`)
- [x] 5.2 Password hashing (Argon2id)
- [x] 5.3 Auth service: login, refresh, logout, request-access
- [x] 5.4 JWT issuing + rotating, hashed refresh-token storage
- [x] 5.5 Cookie strategy
- [x] 5.6 Auth middleware + actor extraction
- [x] 5.7 RBAC policy layer (scoped role_assignments per ADR-005)
- [x] 5.8 Admin/HOD verification + access-approval flow
- [x] 5.9 Activation email via Resend (synchronous, MVP)
- [x] 5.10 Profile CRUD
- [x] 5.11 Real MITT USN format + department codes (VTU-confirmed: CS, AD, CI, CV, ME, EC)
- [x] 5.12 Security test cases (auth-surface: USN hardening, enumeration, input validation)
- [ ] 5.13 Frontend: Access Request + Login screens
- [ ] 5.14 Frontend: auth context, silent refresh, protected routes
- [ ] 5.15 Full manual run-through

---

## ✅ Completed Items

> "Completed" here means implemented in code and pending dev verification. Dev sign-off
> moves an item to verified status.

| Feature             | Details                                                | Dev Verified | Verified Date | Verified By | Notes                       |
| ------------------- | ------------------------------------------------------ | ------------ | ------------- | ----------- | --------------------------- |
| Phase 0 foundation | App container, validated config, pool settings, migration history, logs, request IDs, health/readiness routes, and CI | ✅ Verified | 2026-07-16 | Abhishek Balija | Verified locally against Neon dev branch — all 11 test-plan steps passed |
| Phase 1 doc prep | ADR-012 (activation token), account_activation_tokens table, /activate & /resend-activation endpoints, auth.md token details, roadmap.md deliverables updated | ✅ Verified | 2026-07-18 | Abhishek Balija | Docs-only PR; no code shipped; ready for Phase 1 implementation |
| 5.3–5.5 Auth service | Login, refresh, logout, request-access endpoints with JWT issuing/rotation, hashed refresh-token storage, and HTTP-only cookie strategy | ✅ Verified | 2026-07-21 | Abhishek Balija | All four endpoints verified locally. Includes Argon2id password hashing (5.2), JWT access tokens (15min TTL), SHA-256 hashed refresh tokens (7d rotation), secure HTTP-only cookies |

---

## 📦 Ready for Dev Verification

| Feature | Description | Files Changed | Implementation Notes |
| ------- | ----------- | ------------- | -------------------- |
| 5.1 Identity schema migrations | 6 SQL migration files creating `departments`, `users`, `profiles`, `student_identities`, `role_assignments`, `audit_logs` with all indexes and CHECK constraints from `database-design.md` | `server/migrations/001_create_departments.up.sql` through `006_create_audit_logs.up.sql` | Tables follow `database-design.md` verbatim. Status CHECK on `users` matches `auth.md` state machine. Role CHECK on `role_assignments` matches `auth.md` roles. `departments.hod_user_id` FK deferred to migration 002 to avoid circular dep. All indexes from "Important Indexes" section included. Run server locally — migration runner auto-applies. Verify with `\dt` and `\d <table>`. |
| 5.2 Password hashing (Argon2id) | Argon2id hash/verify + password strength validation (min 8 chars, uppercase, lowercase, digit) | `server/internal/auth/password.go`, `password_test.go`, `doc.go` | Argon2id params: 32MB memory, 1 iteration, 2 threads. PHC-format output. Constant-time comparison. All 7 tests pass. |
| 5.6 Auth middleware + actor extraction | `RequireAuth`/`OptionalAuth` middleware, `Actor` type, `GetActor` helper, `/api/v1/me` endpoint | `server/internal/auth/middleware.go`, `server/internal/auth/handler.go` (Me handler), `server/internal/app/server.go` | Middleware extracts JWT claims into `Actor` struct, sets it in Gin context. `GetActor(c)` retrieves it. Protected `/me` route wired in server.go. E2E test passes: request-access → activate via DB → login → /me returns user data → 401 without token. |
| 5.7 RBAC policy layer | `Policy` struct with 11 `Permission` constants, role-to-permission grant map, `Authorize(actor, permission)`, `AuthorizeActor(c, policy, permission)` convenience | `server/internal/auth/policy.go`, `test/unit/auth/policy_test.go` | Permission constants from `auth.md` permission table. Grants map defined in `NewPolicy()`. `allRoles()` helper for public permissions. 5 unit tests pass: happy path, wrong role, unknown permission, nil actor, all helpers. |
| 5.8 Admin/HOD verification + access-approval flow | Review queue, approve (verify + role assignment), status change (suspend/restore/reject), audit logging | `server/internal/auth/admin_handler.go`, `server/internal/auth/model.go` (AuditLog, interface additions), `server/internal/auth/repository.go` (FindPendingUsers, CreateRoleAssignment, GormAuditLogRepository), `server/internal/auth/service.go` (ReviewQueue, VerifyUser, UpdateUserStatus), `server/internal/auth/dto.go` (admin DTOs), `server/internal/app/server.go` (wiring) | Three endpoints under `/api/v1/admin/users/`. Protected by `PermissionManageUsersAndRoles` (admin + principal only). `GET /review-queue` lists pending users with profile and student identity preloaded. `PATCH /:id/verify` creates a `student` role assignment, flips status to `active`, records audit log. `PATCH /:id/status` changes status (reject/suspend/restore), validates transitions (cannot activate rejected user), records audit log. `GormAuditLogRepository` in `repository.go` handles JSON metadata serialization. Service validates: user existence, pending status for verify, no-op status change. Build + vet + existing tests pass. |
| 5.10 Profile CRUD | Public profile by username, update own profile (headline, bio, avatar, social links, privacy toggles), expanded /api/v1/me with full profile data | `server/internal/profiles/` (model, repository, service, handler, dto), `server/internal/auth/model.go` (expanded Profile struct), `server/internal/auth/repository.go` (FindByID with Preload, FindEmailByUserID, FindPhoneByUserID), `server/internal/auth/dto.go` (MeResponse expanded with profile + student_identity), `server/internal/auth/service.go` (GetMe returns full profile), `server/internal/app/server.go` (wiring) | Two endpoints: `GET /api/v1/profiles/:username` (public, OptionalAuth — respects privacy toggles, returns 404 for disabled profiles), `PATCH /api/v1/me/profile` (authenticated — updates headline, bio, avatar_url, show_email, show_phone, linkedin_url, github_url, portfolio_url). Profile module follows architecture.md as `internal/profiles/`. `UserReader` interface decouples profiles service from auth package. `/api/v1/me` now includes profile and student_identity via GORM Preload. Build + vet + existing tests pass. E2E verified: profile read, update, privacy. |
| 5.11 USN validation + department code map + seed migration | USN format validation (VTU 4MN<year><dept><roll>), confirmed department code map (CS, AD, CI, CV, ME, EC), wired into request-access flow, year-range validated dynamically (2005 to nowYear+2), seed migration inserts 6 departments | `server/internal/auth/usn.go`, `test/unit/auth/usn_test.go`, `server/internal/auth/service.go`, `server/migrations/008_seed_departments.up.sql` | Year-range computed against time.Now().Year() — never needs manual updates. CI provisional (flagged). MBA/MCA excluded. FK auto-resolve returns explicit error if department code structually valid but not in DB. Seed migration uses ON CONFLICT DO NOTHING for idempotency. 7 unit tests pass. |
| 5.12 Security test cases (auth-surface) | USN hardening (SQL injection, null bytes, unicode, overrun, casing, boundary length), enumeration resistance, input validation (missing fields, garbage JSON, oversized payload, wrong content-type), timing check | `test/unit/auth/usn_security_test.go`, `test/e2e/security_test.go` | USN hardening: 6 tests (27 payloads) — pure function tests, no DB needed. Enumeration/input validation: 5 test functions (14 cases) — e2e-gated, hit real HTTP handler. Timing check logs warnings, doesn't fail. |
| 5.9 Activation email + Resend mailer | Resend HTTP API mailer, /activate and /resend-activation endpoints, activation token creation hooked into RequestAccess flow, rate-limited resend (5 min cooldown), ADR-014 | `server/internal/mailer/mailer.go`, `server/internal/auth/handler.go` (2 new routes), `server/internal/auth/service.go` (ResendActivation, sendActivationEmail, generateActivationToken), `server/internal/auth/repository.go` (Create, FindLatestByUserID, RevokeAllUnusedByUserID), `server/internal/auth/dto.go` (ActivateInput, ResendActivationInput), `server/internal/auth/model.go` (interface additions), `server/internal/app/server.go` (mailer wiring), `server/pkg/config/config.go` (MailerConfig), `server/migrations/*` (no new migration — token table already exists) | Resend chosen over SendGrid/Mailgun/raw SMTP. Synchronous sending per ADR-014. NoopMailer when RESEND_API_KEY is empty (safe for local dev without creds). Activation token: 32 bytes crypto/rand, base64.RawURLEncoding, SHA-256 hash, 7-day expiry. Resend rate-limited: reject if last unused token < 5 min old. Revokes all prior unused tokens before issuing replacement. Build + vet + unit tests pass. |

---

## 🔄 In Progress

_Nothing currently in progress. Next: 5.13 Frontend: Access Request + Login screens._

---

## ⚠️ Blocked / Issues

| Issue    | Description | Root Cause | Status |
| -------- | ----------- | ---------- | ------ |
| None yet | -           | -          | -      |

---

## 📋 Tracked Follow-ups (Non-Blocking)

| Item | Description | Reference | Priority | Target Phase |
|------|-------------|-----------|----------|--------------|
| Refresh token hashing: bcrypt → SHA-256 | Refresh tokens currently use bcrypt (auth.md). Same DoS concern as activation tokens. Migrate to SHA-256 for fast, deterministic hashing of high-entropy tokens. | ADR-012 open follow-up | Low | Post-Phase 1 |

---

## 🤖 AI Agent Checklist: What to Verify After Implementation

**Before marking any feature as complete, the AI Agent MUST:**

### Post-Implementation Verification

- [ ] Code follows Go standards from `docs/backend-standards.md`
- [ ] Database migrations are created (if applicable)
- [ ] Error handling is implemented
- [ ] Authorization checks are in place (service/policy layer)
- [ ] Audit logs added for sensitive actions
- [ ] No global mutable state used
- [ ] Constructor-based dependency injection used
- [ ] Code compiles without errors: `go build ./cmd/api`
- [ ] Tests written (if applicable)
- [ ] Documentation updated in relevant `docs/` files

### Files to Update After Each Feature

1. **This file** (`docs/PROGRESS.md`) - Move feature from "In Progress" to "Ready for Dev Verification"
2. **`docs/decision-log.md`** - If a significant decision was made
3. **`docs/implementation.md`** - Add implementation details
4. **`docs/api-spec.md`** - If new endpoints added
5. **Relevant service/module docs** - If architecture changed

### Status Update Workflow

1. ✍️ AI implements feature
2. ✅ AI runs verification checklist above
3. 📝 AI moves feature to "Ready for Dev Verification" section
4. 👨‍💻 Dev tests the feature locally
5. ✔️ Dev adds verification to this file (moves to "Completed Items")
6. 📊 Status becomes "Live"

---

## 👨‍💻 Dev Verification & Confirmation

### Latest Verification

| Field                   | Value                                                      |
| ----------------------- | ---------------------------------------------------------- |
| **Last Verified By**    | Abhishek Balija                                            |
| **Verification Date**   | 2026-07-21                                                 |
| **Everything Working?** | ✅ Yes                                                      |
| **Notes**               | Phase 1 doc prep (2026-07-18) + Phase 1 auth login/refresh/logout/request-access (2026-07-21) verified locally against Neon dev branch |
| **Issues Found**        | None                                                        |

### Verification Process

When dev verifies a completed feature:

1. Test locally with the instructions provided
2. Confirm it works as described in the AI agent's implementation notes
3. Update this section with date, your name, and any issues
4. Mark feature as ✅ in "Completed Items" table

---

## 📌 Important Notes

- **All features must be verified by dev before moving to "Completed"**
- **AI Agent: Always consult `AGENTS.md` before making changes**
- **AI Agent: Read relevant `docs/` files before implementing**
- **Dev: Update this file after each verification - it's your sign-off**
- **Keep this file in sync with actual project state**

---

## 🔗 Related Documentation

- [Project Requirements](product-requirements.md)
- [Architecture](architecture.md)
- [Backend Standards](backend-standards.md)
- [Database Design](database-design.md)
- [API Specification](api-spec.md)
- [Authentication](auth.md)
- [Decision Log](decision-log.md)
- [Postman Test Guide](postman-test-guide.md)
