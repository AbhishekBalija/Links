# LINKS Project Progress Tracker

**Last Updated:** 2026-09-02 (focused unit tests for auth service, middleware, and profile privacy)
**Current Phase:** Phase 1 — Identity and Access

---

## 📋 Current Phase Overview

| Phase                              | Status         | Start Date | Expected End |
| ---------------------------------- | -------------- | ---------- | ------------ |
| **Phase 0: Foundation**            | ✅ Complete     | 2026-06-19 | 2026-07-16   |
| **Phase 1: Identity and Access**   | 🔄 In Progress | 2026-07-18 | TBD          |

> Phase numbering follows `roadmap.md` and `implementation.md`.

### Phase Goals (per `roadmap.md` Phase 1 / `implementation.md` Step 5)

> Checked items below are implemented in the Phase 1 PR. They remain in
> **Ready for Dev Verification** until the developer signs off and the PR is merged.

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
- [x] 5.11 MITT USN format + provisionally seeded department codes (CS, AD, CV, ME, EC; CI awaits MITT confirmation)
- [x] 5.12 Security test cases (auth-surface: USN hardening, enumeration, input validation)
- [x] 5.13 Frontend: Access Request + Login screens
- [x] 5.14 Frontend: auth context, silent refresh, protected routes
- [x] 5.15 Full e2e test suite (12 tests, all passing in an isolated schema, including activation-link UI and refresh rotation)
- [ ] Phase 1 Definition of Done: implementation checks pass; developer verification, PR merge, and post-merge production verification are still pending.

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
| Phase 1 release-candidate hardening | Transactional access requests, approval, activation, refresh rotation, status updates, and privacy audits; safe JWT/config validation; activation-link UI; SPA deep-link routing; resilient E2E cleanup | `server/internal/auth/`, `server/internal/profiles/`, `server/pkg/config/`, `server/migrations/003…010`, `client/src/features/auth/`, `client/src/shared/api/client.ts`, `client/e2e/`, `client/vite.config.ts`, `vercel.json` | AI verification passes: all Go tests, vet, API build, client lint/build, and all 12 Playwright tests. Still requires developer verification on the Vercel preview before merge. |
| 5.15 Full e2e test suite | 12 Playwright tests covering auth guards, the real student onboarding and activation-link UI, profile persistence, session restore, silent refresh with token rotation, and zero-role handling | `client/e2e/playwright.config.ts`, `client/e2e/globalSetup.ts`, `client/e2e/globalTeardown.ts`, `client/e2e/helpers/db.ts`, `client/e2e/helpers/auth.ts`, `client/e2e/scenarios/auth-guards.spec.ts`, `client/e2e/scenarios/full-flow.spec.ts`, `client/e2e/scenarios/silent-refresh.spec.ts`, `client/vite.config.ts`, `client/src/shared/api/client.ts`, `client/src/features/auth/store.ts`, `client/src/features/auth/pages/ActivateAccount.tsx`, `client/src/features/auth/pages/AccountPending.tsx` | Ready for developer verification. Run from `client` with the documented E2E database environment: `bunx playwright test --config e2e/playwright.config.ts`. The harness creates and removes an isolated schema. |
| 5.1 Identity schema migrations | 6 SQL migration files creating `departments`, `users`, `profiles`, `student_identities`, `role_assignments`, `audit_logs` with all indexes and CHECK constraints from `database-design.md` | `server/migrations/001_create_departments.up.sql` through `006_create_audit_logs.up.sql` | Tables follow `database-design.md` verbatim. Status CHECK on `users` matches `auth.md` state machine. Role CHECK on `role_assignments` matches `auth.md` roles. `departments.hod_user_id` FK deferred to migration 002 to avoid circular dep. All indexes from "Important Indexes" section included. Run server locally — migration runner auto-applies. Verify with `\dt` and `\d <table>`. |
| 5.2 Password hashing (Argon2id) | Argon2id hash/verify + password strength validation (min 8 chars, uppercase, lowercase, digit) | `server/internal/auth/password.go`, `password_test.go`, `doc.go` | Argon2id params: 32MB memory, 1 iteration, 2 threads. PHC-format output. Constant-time comparison. All 7 tests pass. |
| 5.6 Auth middleware + actor extraction | `RequireAuth`/`OptionalAuth` middleware, `Actor` type, `GetActor` helper, `/api/v1/me` endpoint | `server/internal/auth/middleware.go`, `server/internal/auth/handler.go` (Me handler), `server/internal/app/server.go` | Middleware extracts JWT claims into `Actor` struct, sets it in Gin context. `GetActor(c)` retrieves it. Protected `/me` route wired in server.go. E2E test passes: request-access → activate via DB → login → /me returns user data → 401 without token. |
| 5.7 RBAC policy layer | `Policy` struct with 11 `Permission` constants, role-to-permission grant map, `Authorize(actor, permission)`, `AuthorizeActor(c, policy, permission)` convenience | `server/internal/auth/policy.go`, `test/unit/auth/policy_test.go` | Permission constants from `auth.md` permission table. Grants map defined in `NewPolicy()`. `allRoles()` helper for public permissions. 5 unit tests pass: happy path, wrong role, unknown permission, nil actor, all helpers. |
| 5.8 Admin/HOD verification + access-approval flow | Review queue, approve (verify + role assignment), status change (suspend/restore/reject), audit logging | `server/internal/auth/admin_handler.go`, `server/internal/auth/model.go`, `server/internal/auth/repository.go`, `server/internal/auth/service.go`, `server/internal/auth/dto.go`, `server/internal/app/server.go` | Three endpoints under `/api/v1/admin/users/`, protected by `PermissionManageUsersAndRoles`. Approval atomically creates the student role, marks the pending user verified, records the audit log, and creates the activation token. The user remains `pending` until the single-use activation link sets the password and moves the account to `active`. Status changes and their audit records are also atomic. |
| 5.10 Profile CRUD | Public profile by username, update own profile (headline, bio, avatar, social links, privacy toggles), expanded /api/v1/me with full profile data | `server/internal/profiles/` (model, repository, service, handler, dto), `server/internal/auth/model.go` (expanded Profile struct), `server/internal/auth/repository.go` (FindByID with Preload, FindEmailByUserID, FindPhoneByUserID), `server/internal/auth/dto.go` (MeResponse expanded with profile + student_identity), `server/internal/auth/service.go` (GetMe returns full profile), `server/internal/app/server.go` (wiring) | Two endpoints: `GET /api/v1/profiles/:username` (public, OptionalAuth — respects privacy toggles, returns 404 for disabled profiles), `PATCH /api/v1/me/profile` (authenticated — updates headline, bio, avatar_url, show_email, show_phone, linkedin_url, github_url, portfolio_url). Profile module follows architecture.md as `internal/profiles/`. `UserReader` interface decouples profiles service from auth package. `/api/v1/me` now includes profile and student_identity via GORM Preload. Build + vet + existing tests pass. E2E verified: profile read, update, privacy. |
| 5.11 USN validation + department code map + seed migration | USN format validation (VTU 4MN<year><dept><roll>), provisionally seeded department code map (CS, AD, CV, ME, EC), wired into request-access flow, and dynamically validated year range | `server/internal/auth/usn.go`, `test/unit/auth/usn_test.go`, `server/internal/auth/service.go`, `server/migrations/008_seed_departments.up.sql` | CI is intentionally not accepted or seeded until MITT confirms an actual 4MNxxCIxxx sample. MBA/MCA excluded. |
| 5.12 Security test cases (auth-surface) | USN hardening (SQL injection, null bytes, unicode, overrun, casing, boundary length), enumeration resistance, input validation (missing fields, garbage JSON, oversized payload, wrong content-type), timing check | `test/unit/auth/usn_security_test.go`, `test/e2e/security_test.go` | USN hardening: 6 tests (27 payloads) — pure function tests, no DB needed. Enumeration/input validation: 5 test functions (14 cases) — e2e-gated, hit real HTTP handler. Timing check logs warnings, doesn't fail. |
| 5.9 Activation email + Resend mailer | Resend HTTP API mailer, `/activate` and `/resend-activation` endpoints, approval-triggered activation token, rate-limited resend, and deployed activation-link UI | `server/internal/mailer/mailer.go`, `server/internal/auth/`, `client/src/features/auth/pages/ActivateAccount.tsx`, `client/src/app/router.tsx`, `vercel.json` | Approval commits role, verification, audit, and activation token before sending email. Failed delivery invalidates the new token so resend can recover. The emailed `/activate?token=…` deep link now opens a real password-setting screen; Vercel serves SPA deep links through the web service. |
| 5.13 Frontend: Access Request + Login screens | React 19 + Vite + TypeScript. Access Request form, Login form, Activation flow. Auth via Zustand store (replaced React Context), Tailwind v4 + shadcn/ui styling, guest/protected route guards. | `client/src/features/auth/store.ts`, `api.ts`, `types.ts`, `pages/AccessRequest.tsx`, `pages/Login.tsx`, `components/ProtectedRoute.tsx`, `features/dashboard/pages/Dashboard.tsx`, `shared/api/client.ts`, `app/providers.tsx`, `app/router.tsx` | Stack: React 19 + Vite + Zustand + TanStack Query + Tailwind v4 + shadcn/ui (base-nova). Auth state migrated from React Context to Zustand. Tailwind v4 single `@import "tailwindcss"` in `index.css`. ProtectedRoute/GuestRoute guards based on isAuthenticated. USN validation, department dropdown, password strength check. |
| 5.14 Frontend: silent refresh + zero-roles guard | Silent token refresh on 401 with request queue to avoid race conditions. Zero-roles guard centralized in ProtectedRoute redirecting to `/account-pending`. | `client/src/shared/api/client.ts`, `client/src/features/auth/api.ts`, `client/src/features/auth/store.ts`, `client/src/features/auth/components/ProtectedRoute.tsx`, `client/src/features/auth/pages/AccountPending.tsx` | Refresh endpoint returns `{ access_token, expires_in }` via HTTP-only cookie. `apiRequest` intercepts 401 (excluding `/auth/refresh`), queues concurrent 401s, attempts refresh, retries queued requests with new token. On refresh failure: clears auth state, throws session-expired error. Zero-roles check in ProtectedRoute redirects to `/account-pending` — enforced centrally, not per-page. Backend: removed `PermissionViewTargetedNotices` from `/me` (ADR-015). |
| Focused unit tests for riskiest auth/profile paths | Refresh-token rotation + replay protection, activation-token consumption, approval/verification flow, request-access conflict and username-collision handling, status-transition guards, resend rate limiting, auth middleware bearer/expiry/actor extraction, profile privacy gating and privacy-change audit | `server/internal/auth/service_test.go`, `server/internal/auth/middleware_test.go`, `server/internal/profiles/service_test.go` | Fake-repository unit tests (no DB). Covers the release-candidate hardening paths that previously had no unit coverage: conditional token consumption (`RevokeIfActive`/`MarkUsed`), replay rejection without a new token, email-failure token invalidation, USN→department derivation, and profile privacy/audit behavior. |

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
| 2 | Access-request phone was collected by the UI but not persisted | Medium | ✅ Resolved — request access trims and stores the optional phone value; later profile phone editing remains a separate product decision. |
| 2 | Product questions (enumeration, USN optionality): ADR-013 accepts enumeration risk for MVP. USN is mandatory per ADR-003. | Low | ✅ Decided |
| 3 | PROGRESS.md reconciliation with all changes | Low | ✅ Resolved |
| 4 | `.env.prod.example` missing JWT secrets, weak warnings | High | ✅ Resolved |
| 5 | E2e test unused imports | Low | ✅ Resolved |
| Deferred | CSRF, CSP headers not configured | Low | 📝 Documented — not scoped for Phase 1 |
| Deferred | CSV bulk import (Phase 2) + frontend activation UI deferred | Low | 📝 Documented — Phase 2 scope per roadmap |
| Deferred | TanStack Query migration (API client currently uses raw fetch) | Low | 📝 Documented — post-MVP refactor |
| Deferred | `account_activation_tokens` no longer needs manual `createTestSchema()` workaround (migration 009 exists) | Low | ✅ Resolved |

---

## 📋 Tracked Follow-ups (Non-Blocking)

| Item | Description | Reference | Priority | Target Phase |
|------|-------------|-----------|----------|--------------|
| Refresh token hashing: bcrypt → SHA-256 | Implemented: high-entropy refresh tokens are SHA-256 hashed, and rotation now conditionally consumes the old token and creates the replacement in one transaction. | ADR-012 | Low | ✅ Resolved in Phase 1 PR |

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
- [x] Tests: 12 e2e tests pass (3 spec files, isolated schema, verified teardown); all Go tests, vet, builds, and client lint pass
- [x] Documentation: `docs/PROGRESS.md` reconciled with the current release-candidate behavior and remaining verification gate

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
| **Everything Working?** | ✅ Yes at the 2026-07-23 developer verification snapshot   |
| **Notes**               | This is the last developer sign-off. The 2026-08-28 PR hardening is AI-verified but must not be treated as developer-verified until the preview is tested and signed off. |
| **Issues Found**        | None in the prior developer snapshot                       |

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
- **2026-07-22 manual testing finding — Activation token not retrievable with NoopMailer:** The activation flow requires the raw token, while the database stores only its SHA-256 hash. For local testing, use RESEND_API_KEY and inspect the inbox, or configure a test-only mailer capture in code that exposes the link to the test process without logging the bearer token. Never print activation links or raw tokens to logs.
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
| 2026-07-23 | `.env.prod.example` missing `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`; weak warnings on `FRONTEND_URL`, `FROM_EMAIL` | Prod deploy would use default/localhost values for `FRONTEND_URL` and `@resend.dev` sandbox sender for `FROM_EMAIL`. Added commented-out JWT secret vars with generation command, + explicit warnings that FRONTEND_URL and FROM_EMAIL must be overridden for prod. | `server/.env.prod.example` |
| 2026-07-23 | Migration embed fix for Vercel (migrations not in binary artifact) | Filesystem migrations are unavailable in a Vercel Go artifact. SQL migrations are now embedded and applied through `fs.FS`. | `server/migrations/embed.go`, `server/pkg/db/migrations.go`, `server/pkg/db/postgres.go`, `server/cmd/api/main.go`, related tests |
| 2026-08-28 | Phase 1 release-candidate integrity and preview hardening | Review found partial multi-write failures, replay/concurrent refresh risks, missing activation UI, SPA deep-link 404s, and fragile E2E cleanup. Added transaction-scoped repositories, conditional token consumption, refresh deduplication, activation UI, service-level SPA rewrite, and process/schema cleanup. | `server/internal/auth/`, `server/internal/profiles/`, `server/pkg/config/`, `server/migrations/`, `client/src/features/auth/`, `client/src/shared/api/client.ts`, `client/e2e/`, `vercel.json` |

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
