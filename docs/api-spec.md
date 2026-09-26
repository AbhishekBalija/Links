# API Specification

## API Standards

- Prefix routes with `/api/v1`.
- Use JSON request and response bodies.
- Use stable error codes.
- Use cursor pagination for feeds and large lists.
- Enforce authorization in services/policies.
- Do not expose database models directly.

Health endpoints are an unauthenticated infrastructure exception and remain under `/api` rather than `/api/v1`.

## Response Envelope

Success:

```json
{
  "data": {},
  "meta": {}
}
```

Error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request payload",
    "details": {}
  }
}
```

## Common Error Codes

| Code | HTTP |
|---|---:|
| `VALIDATION_ERROR` | 400 |
| `UNAUTHENTICATED` | 401 |
| `FORBIDDEN` | 403 |
| `NOT_FOUND` | 404 |
| `CONFLICT` | 409 |
| `RATE_LIMITED` | 429 |
| `INTERNAL_ERROR` | 500 |

## Health Checks

These endpoints are intentionally unauthenticated so hosting platforms and uptime monitors can call them.

`/api/health` and `/api/ready` return raw JSON as exceptions to the global `{data, meta}` success envelope.

```text
GET /api/health
GET /api/ready
```

| Endpoint     | Success | Failure              |
| ------------ | ------- | -------------------- |
| `/api/health` | `200`  | n/a                  |
| `/api/ready`  | `200`  | `503 Service Unavailable` |

Expected `/api/health` response (`200`):

```json
{
  "service": "links-api",
  "status": "ok"
}
```

Expected `/api/ready` success response (`200`):

```json
{
  "database": "connected",
  "status": "ok"
}
```

Expected `/api/ready` failure response (`503`):

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "database unavailable"
  }
}
```

## Pagination

```text
GET /api/v1/events?limit=20&cursor=...
```

```json
{
  "data": [],
  "meta": {
    "next_cursor": "..."
  }
}
```

## Auth

```text
POST /api/v1/auth/request-access
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
POST /api/v1/auth/activate
POST /api/v1/auth/resend-activation
```

## Current User and Profiles

```text
GET   /api/v1/me
PATCH /api/v1/me/profile
GET   /api/v1/profiles/:username
GET   /api/v1/users
GET   /api/v1/users/:id
```

`username` is a unique, immutable profile handle. It is distinct from the user's UUID.

### `GET /api/v1/me`

Returns the authenticated user's identity, profile, and student identity.

**Auth:** Requires valid bearer token (`RequireAuth` middleware). Does NOT gate on any business permission — returns `200` for any authenticated user regardless of role count. This is intentional (see ADR-015).

**Responses:**

| HTTP | Code | When |
|------|------|------|
| 200 | — | Authenticated — returns user payload |
| 401 | `UNAUTHENTICATED` | No bearer token or invalid/expired token |

**Example 200 response:**

```json
{
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "status": "active",
    "roles": ["student"],
    "profile": { ... },
    "student_identity": { ... }
  }
}
```

## Dashboards, Search, and Reports

```text
GET /api/v1/dashboard
GET /api/v1/search?q=...
GET /api/v1/reports/placement
GET /api/v1/reports/events
```

## Admin Users

```text
GET    /api/v1/admin/users/review-queue
POST   /api/v1/admin/users
POST   /api/v1/admin/users/import
PATCH  /api/v1/admin/users/:id/verify
PATCH  /api/v1/admin/users/:id/status
POST   /api/v1/admin/users/:id/roles
DELETE /api/v1/admin/users/:id/roles/:roleAssignmentId
```

`PATCH /api/v1/admin/users/:id/verify` accepts an optional `scope_type` and
`scope_id` for the student role (global when omitted). A `department` scope
must carry an existing department's ID, otherwise it returns
`400 VALIDATION_ERROR`. The department row is share-locked while the role is
created, so a concurrent department delete either waits and returns `409` or
runs first and the approval returns `400`.

## Announcements

```text
GET   /api/v1/announcements
POST  /api/v1/announcements
GET   /api/v1/announcements/:id
PATCH /api/v1/announcements/:id
POST  /api/v1/announcements/:id/submit-for-approval
PATCH /api/v1/announcements/:id/approval
```

Built so far (#27): `GET /api/v1/announcements` and `POST /api/v1/announcements`.
The approval endpoints arrive with #28 and #29.

`POST /api/v1/announcements` (roles: student coordinator, faculty, HOD,
placement officer, principal, admin):

```json
{
  "title": "CS lab closed on Friday",
  "body": "The CS labs are closed for maintenance this Friday.",
  "category": "department",
  "audience": [{ "department_id": "<uuid>", "batch_year": 2022, "role": "student" }],
  "expires_at": "2026-10-01T00:00:00Z"
}
```

- `category` is `official`, `department` or `placement`. The placement officer
  can only post `placement`.
- `audience` is a list of rules. The fields in one rule must all match a
  reader; matching any rule is enough. An empty or missing list means the
  whole college. Each rule needs at least one field.
- The author's current roles are read from the database. With Publishing
  authority over the whole Audience (ADR 0017), it returns `201` with the
  published Announcement. Without it, it returns `403` until Announcement
  approval is built (#28).
- `400` for an unknown Department, a role that isn't a LINKS role, an empty
  rule, a batch year outside 2000 to 2100, or an expiry in the past.

`GET /api/v1/announcements?limit=20&cursor=...` returns the reader's feed:
published, unexpired Announcements whose Audience includes them, newest first.
The reader's Departments come from their Student identity and their
Department-scoped roles, and their batch year from their Student identity.
`limit` defaults to 20 (max 50). `meta.next_cursor` is present when there is
another page.

## Events

```text
GET   /api/v1/events
POST  /api/v1/events
GET   /api/v1/events/:id
PATCH /api/v1/events/:id
POST  /api/v1/events/:id/submit-for-approval
PATCH /api/v1/events/:id/hod-review
PATCH /api/v1/events/:id/final-approval
POST  /api/v1/events/:id/rsvp
GET   /api/v1/events/:id/rsvps
GET   /api/v1/events/:id/export
```

## Opportunities and Applications

```text
GET   /api/v1/opportunities
POST  /api/v1/opportunities
GET   /api/v1/opportunities/:id
PATCH /api/v1/opportunities/:id
POST  /api/v1/opportunities/:id/apply
GET   /api/v1/opportunities/:id/applications
PATCH /api/v1/opportunity-applications/:id/status
POST  /api/v1/opportunities/:id/save
DELETE /api/v1/opportunities/:id/save
GET   /api/v1/opportunities/:id/export
```

## Departments

```text
GET /api/v1/departments
GET /api/v1/departments/:code
POST /api/v1/admin/departments
PUT /api/v1/admin/departments/:code
DELETE /api/v1/admin/departments/:code
GET /api/v1/departments/:code/announcements
GET /api/v1/departments/:code/events
GET /api/v1/departments/:code/reports
```

The list and detail routes require authentication. Department mutations require
the `admin` role. Department codes are immutable uppercase VTU course codes.

Create request:

```json
{
  "code": "AI",
  "name": "Computer Science and Engineering (AI and ML)",
  "description": null,
  "hodUserId": null
}
```

Update replaces the editable fields for the department identified by `:code`:

```json
{
  "name": "Computer Science and Engineering (AI and ML)",
  "description": null,
  "hodUserId": null
}
```

Create the department before assigning its HOD. On update, `hodUserId` must
identify a user with an existing HOD role scoped to that same department. Delete
returns `409 CONFLICT` when student identities or scoped role assignments still
reference the department.

## Clubs

```text
GET  /api/v1/clubs
POST /api/v1/clubs
GET  /api/v1/clubs/:slug
PUT  /api/v1/clubs/:id
POST /api/v1/clubs/:id/interests
```

## Mentorship

```text
GET   /api/v1/mentors
POST  /api/v1/mentorship-requests
PATCH /api/v1/mentorship-requests/:id
```

## Notifications

```text
GET    /api/v1/notifications
PATCH  /api/v1/notifications/:id/read
PATCH  /api/v1/notifications/read-all
GET    /api/v1/notification-preferences
PATCH  /api/v1/notification-preferences
POST   /api/v1/push-subscriptions
DELETE /api/v1/push-subscriptions/:id
```

## API Contract Rules

- Every write endpoint must authenticate.
- Every protected endpoint must authorize resource access.
- Every list endpoint must paginate, except small reference lists that only admins can grow and that clients need whole, such as `GET /api/v1/departments` (one row per MITT department).
- Every CSV export must be audited.
- Every request body must have a DTO.
- Never return password hashes, refresh tokens, or private applicant notes to unauthorized users.
