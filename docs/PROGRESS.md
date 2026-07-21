# LINKS Project Progress Tracker

**Last Updated:** 2026-07-21
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
- [ ] 5.6 Auth middleware + actor extraction
- [ ] 5.7 RBAC policy layer (scoped role_assignments per ADR-005)
- [ ] 5.8 Admin/HOD verification + access-approval flow
- [ ] 5.9 Gmail invite mechanics (open decision)
- [ ] 5.10 Profile CRUD
- [ ] 5.11 Real MITT USN format, department codes, batch-year ranges (dev supplies)
- [ ] 5.12 Security test cases
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

---

## 🔄 In Progress

| Feature               | Description                                     | Status      | Implementation Started | Expected Completion |
| --------------------- | ----------------------------------------------- | ----------- | ---------------------- | ------------------- |
| 5.6 Auth middleware   | Auth middleware + actor extraction               | 📋 Next     | -                      | TBD                 |

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
