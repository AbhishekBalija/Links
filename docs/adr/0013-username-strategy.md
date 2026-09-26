# Username Strategy — Auto-Generated with Collision Retry, Planned Real-Time Check

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
