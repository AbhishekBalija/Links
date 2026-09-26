# Remove Permission Gate from /me — Identity Endpoint Should Not Check Business Permissions

**Date:** 2026-07-22

**Decision:** Remove `AuthorizeActor(c, h.policy, PermissionViewTargetedNotices)` from the `GET /api/v1/me` handler.

**Rationale:**
- Identity endpoints (whoami) should only verify that the caller is authenticated — not whether they hold a specific business permission
- The original permission check was an unintentional copy-paste from the admin handler pattern (introduced in commit `bc8403f`), not a deliberate design decision
- Any authenticated user with zero role assignments (e.g., after `RequestAccess` + `Activate` but before admin assigns roles) would receive an indistinguishable `403 FORBIDDEN` — the same error as an unauthorized user trying to access an admin endpoint
- This made the zero-role state unresolvable because the frontend couldn't distinguish "authenticated with no roles" from "not authorized for this action"

**Consequence:**
- `GET /api/v1/me` now returns `200` with user identity + profile data for any valid JWT, regardless of role count
- Authentication is still enforced at the `RequireAuth` middleware layer (returns `401 UNAUTHENTICATED` with no bearer token) — removing the authorization check did not weaken authentication
- All other endpoints continue to gate on `AuthorizeActor` as before
- The frontend `ProtectedRoute` component now detects `user.roles.length === 0` and redirects to `/account-pending`, centralizing zero-role handling at the routing layer
- ADR-015 documents that identity endpoints must never depend on business permissions — this is a standing principle going forward
