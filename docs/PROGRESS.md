# LINKS Project Progress Tracker

**Last Updated:** 2026-07-23 (audit: Tier 0-1 complete, migration 009 added, CI e2e job wired)
**Current Phase:** Phase 1 — Identity and Access

---

## 📋 Current Phase Overview

| Phase                              | Status         | Start Date | Expected End |
| ---------------------------------- | -------------- | ---------- | ------------ |
| **Phase 0: Foundation**            | ✅ Complete     | 2026-06-19 | 2026-07-16   |
| **Phase 1: Identity and Access**   | 🔄 In Progress | 2026-07-18 | TBD          |

> Phase numbering follows `roadmap.md` and `implementation.md`.

### Phase Goals (per `roadmap.md` Phase 1 / `implementation.md` Step 5)

- [x] 5.1 Identity schema migrations (`users`, `profiles`, `student_identities`, `departments`, `role_assignments`, `audit_logs`, `account_activation_tokens`)
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
- [x] 5.13 Frontend: Access Request + Login screens
- [x] 5.14 Frontend: auth context, silent refresh, protected routes
- [x] 5.15 Full e2e test suite (11 tests, all passing in isolated Neon schema)
- [x] ✅ Phase 1 Definition of Done: Sensitive-field leakage check (8 user-data endpoints + JWT payloads) — PASS. Cross-department isolation — N/A (no endpoint returns department-scoped data yet).

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
| 5.15 Full e2e test suite | 11 Playwright tests covering auth guards, full student onboarding flow (request → admin approve → login → edit profile → refresh → persist), silent refresh with token rotation, and zero-role user guard | `client/e2e/playwright.config.ts`, `client/e2e/globalSetup.ts`, `client/e2e/globalTeardown.ts`, `client/e2e/helpers/db.ts`, `client/e2e/helpers/auth.ts`, `client/e2e/scenarios/auth-guards.spec.ts`, `client/e2e/scenarios/full-flow.spec.ts`, `client/e2e/scenarios/silent-refresh.spec.ts`, `client/vite.config.ts`, `client/src/shared/api/client.ts`, `client/src/features/auth/store.ts`, `client/src/features/auth/pages/AccountPending.tsx` | All 11 tests pass in isolated Neon schema. Tests use pooled/unpooled URL split for Neon compatibility. Vite proxy configured. `initializeAuth` now tries cookie-based refresh first so page reloads preserve sessions. Silent refresh test validates token rotation via exposed `window.__apiRequest` (dev-only). Run: `cd client && npx playwright test --config=e2e/playwright.config.ts --project=chromium`. |
| 5.1 Identity schema migrations | 6 SQL migration files creating `departments`, `users`, `profiles`, `student_identities`, `role_assignments`, `audit_logs` with all indexes and CHECK constraints from `database-design.md` | `server/migrations/001_create_departments.up.sql` through `006_create_audit_logs.up.sql` | Tables follow `database-design.md` verbatim. Status CHECK on `users` matches `auth.md` state machine. Role CHECK on `role_assignments` matches `auth.md` roles. `departments.hod_user_id` FK deferred to migration 002 to avoid circular dep. All indexes from "Important Indexes" section included. Run server locally — migration runner auto-applies. Verify with `\dt` and `\d <table>`. |
| 5.2 Password hashing (Argon2id) | Argon2id hash/verify + password strength validation (min 8 chars, uppercase, lowercase, digit) | `server/internal/auth/password.go`, `password_test.go`, `doc.go` | Argon2id params: 32MB memory, 1 iteration, 2 threads. PHC-format output. Constant-time comparison. All 7 tests pass. |
| 5.6 Auth middleware + actor extraction | `RequireAuth`/`OptionalAuth` middleware, `Actor` type, `GetActor` helper, `/api/v1/me` endpoint | `server/internal/auth/middleware.go`, `server/internal/auth/handler.go` (Me handler), `server/internal/app/server.go` | Middleware extracts JWT claims into `Actor` struct, sets it in Gin context. `GetActor(c)` retrieves it. Protected `/me` route wired in server.go. E2E test passes: request-access → activate via DB → login → /me returns user data → 401 without token. |
| 5.7 RBAC policy layer | `Policy` struct with 11 `Permission` constants, role-to-permission grant map, `Authorize(actor, permission)`, `AuthorizeActor(c, policy, permission)` convenience | `server/internal/auth/policy.go`, `test/unit/auth/policy_test.go` | Permission constants from `auth.md` permission table. Grants map defined in `NewPolicy()`. `allRoles()` helper for public permissions. 5 unit tests pass: happy path, wrong role, unknown permission, nil actor, all helpers. |
| 5.8 Admin/HOD verification + access-approval flow | Review queue, approve (verify + role assignment), status change (suspend/restore/reject), audit logging | `server/internal/auth/admin_handler.go`, `server/internal/auth/model.go` (AuditLog, interface additions), `server/internal/auth/repository.go` (FindPendingUsers, CreateRoleAssignment, GormAuditLogRepository), `server/internal/auth/service.go` (ReviewQueue, VerifyUser, UpdateUserStatus), `server/internal/auth/dto.go` (admin DTOs), `server/internal/app/server.go` (wiring) | Three endpoints under `/api/v1/admin/users/`. Protected by `PermissionManageUsersAndRoles` (admin + principal only). `GET /review-queue` lists pending users with profile and student identity preloaded. `PATCH /:id/verify` creates a `student` role assignment, flips status to `active`, records audit log. `PATCH /:id/status` changes status (reject/suspend/restore), validates transitions (cannot activate rejected user), records audit log. `GormAuditLogRepository` in `repository.go` handles JSON metadata serialization. Service validates: user existence, pending status for verify, no-op status change. Build + vet + existing tests pass. |
| 5.10 Profile CRUD | Public profile by username, update own profile (headline, bio, avatar, social links, privacy toggles), expanded /api/v1/me with full profile data | `server/internal/profiles/` (model, repository, service, handler, dto), `server/internal/auth/model.go` (expanded Profile struct), `server/internal/auth/repository.go` (FindByID with Preload, FindEmailByUserID, FindPhoneByUserID), `server/internal/auth/dto.go` (MeResponse expanded with profile + student_identity), `server/internal/auth/service.go` (GetMe returns full profile), `server/internal/app/server.go` (wiring) | Two endpoints: `GET /api/v1/profiles/:username` (public, OptionalAuth — respects privacy toggles, returns 404 for disabled profiles), `PATCH /api/v1/me/profile` (authenticated — updates headline, bio, avatar_url, show_email, show_phone, linkedin_url, github_url, portfolio_url). Profile module follows architecture.md as `internal/profiles/`. `UserReader` interface decouples profiles service from auth package. `/api/v1/me` now includes profile and student_identity via GORM Preload. Build + vet + existing tests pass. E2E verified: profile read, update, privacy. |
| 5.11 USN validation + department code map + seed migration | USN format validation (VTU 4MN<year><dept><roll>), confirmed department code map (CS, AD, CI, CV, ME, EC), wired into request-access flow, year-range validated dynamically (2005 to nowYear+2), seed migration inserts 6 departments | `server/internal/auth/usn.go`, `test/unit/auth/usn_test.go`, `server/internal/auth/service.go`, `server/migrations/008_seed_departments.up.sql` | Year-range computed against time.Now().Year() — never needs manual updates. CI provisional (flagged). MBA/MCA excluded. FK auto-resolve returns explicit error if department code structually valid but not in DB. Seed migration uses ON CONFLICT DO NOTHING for idempotency. 7 unit tests pass. |
| 5.12 Security test cases (auth-surface) | USN hardening (SQL injection, null bytes, unicode, overrun, casing, boundary length), enumeration resistance, input validation (missing fields, garbage JSON, oversized payload, wrong content-type), timing check | `test/unit/auth/usn_security_test.go`, `test/e2e/security_test.go` | USN hardening: 6 tests (27 payloads) — pure function tests, no DB needed. Enumeration/input validation: 5 test functions (14 cases) — e2e-gated, hit real HTTP handler. Timing check logs warnings, doesn't fail. |
| 5.9 Activation email + Resend mailer | Resend HTTP API mailer, /activate and /resend-activation endpoints, activation token creation hooked into RequestAccess flow, rate-limited resend (5 min cooldown), ADR-014 | `server/internal/mailer/mailer.go`, `server/internal/auth/handler.go` (2 new routes), `server/internal/auth/service.go` (ResendActivation, sendActivationEmail, generateActivationToken), `server/internal/auth/repository.go` (Create, FindLatestByUserID, RevokeAllUnusedByUserID), `server/internal/auth/dto.go` (ActivateInput, ResendActivationInput), `server/internal/auth/model.go` (interface additions), `server/internal/app/server.go` (mailer wiring), `server/pkg/config/config.go` (MailerConfig), `server/migrations/*` (no new migration — token table already exists) | Resend chosen over SendGrid/Mailgun/raw SMTP. Synchronous sending per ADR-014. NoopMailer when RESEND_API_KEY is empty (safe for local dev without creds). Activation token: 32 bytes crypto/rand, base64.RawURLEncoding, SHA-256 hash, 7-day expiry. Resend rate-limited: reject if last unused token < 5 min old. Revokes all prior unused tokens before issuing replacement. Build + vet + unit tests pass. |
| 5.13 Frontend: Access Request + Login screens | React 19 + Vite + TypeScript. Access Request form, Login form, Activation flow. Auth via Zustand store (replaced React Context), Tailwind v4 + shadcn/ui styling, guest/protected route guards. | `client/src/features/auth/store.ts`, `api.ts`, `types.ts`, `pages/AccessRequest.tsx`, `pages/Login.tsx`, `components/ProtectedRoute.tsx`, `features/dashboard/pages/Dashboard.tsx`, `shared/api/client.ts`, `app/providers.tsx`, `app/router.tsx` | Stack: React 19 + Vite + Zustand + TanStack Query + Tailwind v4 + shadcn/ui (base-nova). Auth state migrated from React Context to Zustand. Tailwind v4 single `@import "tailwindcss"` in `index.css`. ProtectedRoute/GuestRoute guards based on isAuthenticated. USN validation, department dropdown, password strength check. |
| 5.14 Frontend: silent refresh + zero-roles guard | Silent token refresh on 401 with request queue to avoid race conditions. Zero-roles guard centralized in ProtectedRoute redirecting to `/account-pending`. | `client/src/shared/api/client.ts`, `client/src/features/auth/api.ts`, `client/src/features/auth/store.ts`, `client/src/features/auth/components/ProtectedRoute.tsx`, `client/src/features/auth/pages/AccountPending.tsx` | Refresh endpoint returns `{ access_token, expires_in }` via HTTP-only cookie. `apiRequest` intercepts 401 (excluding `/auth/refresh`), queues concurrent 401s, attempts refresh, retries queued requests with new token. On refresh failure: clears auth state, throws session-expired error. Zero-roles check in ProtectedRoute redirects to `/account-pending` — enforced centrally, not per-page. Backend: removed `PermissionViewTargetedNotices` from `/me` (ADR-015). |

---

## 🔄 In Progress

_Ready for dev verification._

---

## ⚠️ Blocked / Issues

| Issue    | Description | Root Cause | Status |
| -------- | ----------- | ---------- | ------ |
| None yet | -           | -          | -      |

## 📋 Open Audit Items

| Tier | Item | Priority | Status |
|------|------|----------|--------|
| 0 | Hardcoded Neon creds in `client/e2e/helpers/db.ts` — never committed, fixed to use env var | High | ✅ Resolved |
| 1.1 | `server_test.go` panics on empty CORS | High | ✅ Resolved |
| 1.2 | 11 ESLint errors in client code | High | ✅ Resolved |
| 1.3 | Missing e2e job in CI workflow | High | ✅ Resolved |
| 2 | Missing migration 009 for `account_activation_tokens` | High | ✅ Resolved |
| 2 | Phone field: `users.phone` exists in DB and appears in `/me` response but has no update endpoint | Medium | 📝 Documented — backend gap; the profile update PATCH doesn't set phone. Needs a `phone` field in UpdateProfileInput or a separate endpoint. |
| 2 | Product questions (enumeration, USN optionality): ADR-013 accepts enumeration risk for MVP. USN is mandatory per ADR-003. | Low | ✅ Decided |
| 3 | PROGRESS.md reconciliation with all changes | Low | 📝 In progress |
| Deferred | CSRF, CSP headers not configured | Low | 📝 Documented — not scoped for Phase 1 |
| Deferred | CSV bulk import (Phase 2) + frontend activation UI deferred | Low | 📝 Documented — Phase 2 scope per roadmap |
| Deferred | TanStack Query migration (API client currently uses raw fetch) | Low | 📝 Documented — post-MVP refactor |
| Deferred | `account_activation_tokens` no longer needs manual `createTestSchema()` workaround (migration 009 exists) | Low | ✅ Resolved |

---

## 📋 Tracked Follow-ups (Non-Blocking)

| Item | Description | Reference | Priority | Target Phase |
|------|-------------|-----------|----------|--------------|
| Refresh token hashing: bcrypt → SHA-256 | Refresh tokens currently use bcrypt (auth.md). Same DoS concern as activation tokens. Migrate to SHA-256 for fast, deterministic hashing of high-entropy tokens. | ADR-012 open follow-up | Low | Post-Phase 1 |

---

## 🤖 AI Agent Checklist: What to Verify After Implementation

**Before marking any feature as complete, the AI Agent MUST:**

### Post-Implementation Verification

- [x] Code follows Go standards (handler fix: `errors.Is(err, io.EOF)` pattern in `admin_handler.go`)
- [x] Database migrations: `009_create_account_activation_tokens.up.sql` created. The e2e workaround in `createTestSchema()` is now removed — the backend migration runner handles this table like all others.
- [x] Error handling: `VerifyUser` now tolerates empty JSON body via `io.EOF` check
- [x] Authorization checks: `AuthorizeActor` check already present on `VerifyUser` handler
- [x] Audit logs: already recorded by `VerifyUser` service method
- [x] No global mutable state: client-side `accessToken` is module-scoped, not global
- [x] Constructor-based DI: server-side uses explicit constructors throughout
- [x] Code compiles: `go build ./cmd/api` passes, Vite dev server boots successfully
- [x] Tests: 11 e2e tests pass (3 spec files, isolated Neon schema, verified no leftover schemas)
- [x] Documentation: `docs/PROGRESS.md` updated with bug findings and fix log

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
| **Verification Date**   | 2026-07-23                                                 |
| **Everything Working?** | ✅ Yes                                                      |
| **Notes**               | Audit cleanup: Tier 0 (hardcoded creds — never committed), Tier 1.1 (server_test.go CORS panic fixed), Tier 1.2 (11 ESLint errors fixed), Tier 1.3 (CI e2e job added). Migration 009 created for account_activation_tokens. Phase 1 DoD: sensitive-field leakage ✅ PASS, cross-department isolation N/A. |
| **Issues Found**        | None (all audit Tiers 0-1 resolved)                        |

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
- **2026-07-22 e2e suite finding:** The automated e2e tests caught a real production bug that manual testing missed: page reload (F5) silently logged the user out because the access token was in-memory only and `initializeAuth` had no fallback. This was fixed by adding cookie-based session restore at boot. **Lesson: manual testing rarely triggers page reloads mid-session — automated e2e is essential for this class of bug.**
- **2026-07-22 manual testing finding — Activation token not retrievable with NoopMailer:** The activation flow requires the raw token (32-byte crypto/rand, base64url-encoded), but NoopMailer discards it silently. The DB only stores `token_hash = SHA256(raw_token)`, which is irreversible. **Running `SELECT token_hash` from the DB and using it as the token value in `/activate` will always fail with `UNAUTHENTICATED / invalid or expired activation token`.** Two ways to test activation flow locally: (1) uncomment RESEND_API_KEY in `.env.local` and check email inbox, or (2) add `fmt.Println("activation link:", activationLink)` to `NoopMailer.SendActivationEmail` in `server/internal/mailer/mailer.go:76` — the raw token appears in the server stdout. The e2e tests bypass this entirely by calling admin `/verify` directly instead of the user `/activate` flow.
- **2026-07-23 — `account_activation_tokens` migration (009) created.** The e2e `db.ts` no longer creates this table manually — the backend's migration runner handles it via `009_create_account_activation_tokens.up.sql`. The `createTestSchema()` function now only creates the schema namespace.

### Fix Log (Cross-Cutting)

| Date | Fix | Root Cause | Files Changed |
|------|-----|------------|---------------|
| 2026-07-22 | `/me` no longer gates on `PermissionViewTargetedNotices` | Original permission check was copy-pasted from admin handler pattern (commit `bc8403f`) without considering zero-role state. ADR-015 documents the principle. | `server/internal/auth/handler.go`, `client/src/features/auth/components/ProtectedRoute.tsx` (added zero-roles redirect), `client/src/features/auth/pages/AccountPending.tsx` (new) |
| 2026-07-22 | 401 intercept + silent refresh in `apiRequest` | Access token expired silently with no fallback — user would see 401 with no refresh attempt. Implemented retry queue to avoid race conditions on concurrent 401s. Guard changed from path-based exclusion (`path !== '/auth/refresh'`) to token-based (`accessToken !== null`) to naturally exclude unauthenticated requests (login, request-access). | `client/src/shared/api/client.ts`, `client/src/features/auth/api.ts`, `client/src/features/auth/store.ts` |
| 2026-07-22 | `adminApproveUser` e2e helper missing request body | `VerifyUser` handler expects JSON body (`VerifyUserInput` with optional fields). Empty body caused `EOF` → 400 VALIDATION_ERROR. Fix: send `data: {}` in PATCH. | `client/e2e/helpers/auth.ts` |
| 2026-07-22 | AccountPending logout doesn't redirect to /login | `useAuthStore.logout()` clears auth state but doesn't navigate. AccountPending page is outside ProtectedRoute, so no auto-redirect occurs. Fix: added `useNavigate()` + `replace: true` to `/login` after logout. | `client/src/features/auth/pages/AccountPending.tsx` |
| 2026-07-22 | Silent refresh test used raw `fetch()` bypassing app interceptor | `page.evaluate(() => fetch(...))` didn't have the `Authorization` header set by the app's `apiRequest`. Fix: expose `apiRequest` and `getAccessToken` on `window` in dev mode; test calls through app's own API client. | `client/src/shared/api/client.ts`, `client/e2e/scenarios/silent-refresh.spec.ts` |
| 2026-07-22 | Page reload loses in-memory access token ⚠️ **Most significant finding** | `page.reload()` or `page.goto('/')` clears JS memory, losing the access token. `initializeAuth` only called `fetchCurrentUser()` which failed without a token. **This meant any browser refresh would silently log the user out** — a real production bug that manual testing never caught because testers don't typically F5 after login. Fix: `initializeAuth` now attempts cookie-based refresh first (`attemptRefresh()`) before falling back to unauthenticated state. | `client/src/features/auth/store.ts`, `client/src/shared/api/client.ts` |
| 2026-07-22 | Frontend `CurrentUser.roles` type mismatch — backend sends `string[]`, frontend expected `{role,scope_type,scope_id}[]` | Backend `MeResponse.Roles` is `[]string` (flat role names). Frontend `CurrentUser.roles` was typed as `RoleAssignment[]` (objects). This caused Dashboard to show "no role" because `r.role` on a plain string returns `undefined`. **Root cause of the "no role" display.** The e2e test that should have caught this was previously weakened: original `text=student` assertion was silently downgraded to `strong`.toBeVisible() — which checked element existence only, not content — to make CI pass while the bug was live. Lesson: a weakened assertion that passes despite broken rendering is worse than a failing test; it gives false confidence and halts further investigation. | `client/src/features/auth/types.ts`, `client/src/features/dashboard/pages/Dashboard.tsx`, `client/e2e/scenarios/full-flow.spec.ts` |
| 2026-07-23 | `account_activation_tokens` now has SQL migration (009) | Table was created inline in e2e `db.ts` instead of a proper migration. Migration tracks table via `schema_migrations` like all others. | `server/migrations/009_create_account_activation_tokens.up.sql` — new; `client/e2e/helpers/db.ts` — removed manual table creation from `createTestSchema()` |
| 2026-07-23 | `server_test.go` panics on empty `CORS.AllowedOrigins` | Test config didn't set `CORS`, causing nil-slice in gin cors middleware. | `server/internal/app/server_test.go` — added `CORS: config.CORSConfig{AllowedOrigins: []string{"*"}}` |
| 2026-07-23 | 11 ESLint errors across 5 files (e2e tests + api client) | Unused imports, `as any` type casts, unused params. Fixed: removed unused `loginViaAPI` imports, removed unused `adminToken`/`request` vars, typed `window.__apiRequest` via `declare global` interface augmentation (`client.ts` + `silent-refresh.spec.ts`), used `void _dbURL` for intentionally-unused bootstrapAdmin param. | `client/src/shared/api/client.ts`, `client/e2e/helpers/auth.ts`, `client/e2e/scenarios/auth-guards.spec.ts`, `client/e2e/scenarios/full-flow.spec.ts`, `client/e2e/scenarios/silent-refresh.spec.ts` |
| 2026-07-23 | CI missing e2e test job | Only `server` (go vet/test) and `client` (lint/build) jobs existed. Added `e2e` job with PostgreSQL service container, Go + Bun, Playwright install, and `playwright test` with `E2E_NEON_URL` env pointing to service. | `.github/workflows/ci.yml` — added `e2e` job |

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
