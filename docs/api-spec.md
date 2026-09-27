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

Built so far (#39): `GET /api/v1/dashboard` returns the signed-in user's Home
summary. Sections a user doesn't need are left out, and later features add
sections without changing these:

```json
{
  "user": { "full_name": "...", "roles": ["faculty"], "department": { "id": "...", "code": "CS", "name": "..." } },
  "notices": { "items": [/* newest five from the feed */], "has_more": true },
  "approvals": { "pending_count": 2, "oldest_submitted_at": "..." },
  "my_announcements": { "draft": 1, "pending": 1, "rejected": 1, "edits_waiting": 1 }
}
```

- `approvals` appears only for HODs, the principal and admins.
- `my_announcements` appears only for users who can post.
- `department` is the Student identity's Department, otherwise the first
  Department-scoped role, otherwise `null`.

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

Also built: `GET /api/v1/announcements/mine`, `GET /api/v1/announcements/approvals`
and `POST /api/v1/announcements/:id/withdraw`.

`POST /api/v1/announcements` (roles: student coordinator, faculty, HOD,
placement officer, principal, admin):

```json
{
  "title": "CS lab closed on Friday",
  "body": "The CS labs are closed for maintenance this Friday.",
  "category": "department",
  "audience": [{ "department_id": "<uuid>", "batch_year": 2022, "role": "student" }],
  "expires_at": "2026-10-01T00:00:00Z",
  "draft": false
}
```

- `category` is `official`, `department` or `placement`. The placement officer
  can only post `placement`.
- `audience` is a list of rules. The fields in one rule must all match a
  reader; matching any rule is enough. An empty or missing list means the
  whole college. Each rule needs at least one field.
- The author's current roles are read from the database (ADR 0017). With
  Publishing authority over the whole Audience, it returns `201` with status
  `published`. Otherwise status is `pending` and it waits for Announcement
  approval. `"draft": true` saves it as `draft` without submitting.
- `400` for an unknown Department, a role that isn't a LINKS role, an empty
  rule, a batch year outside 2000 to 2100, or an expiry in the past.

`PATCH /api/v1/announcements/:id` (author only) edits an Announcement with the
same fields as create. Someone else's Announcement is `404`.

- `draft` or `rejected`: the content is replaced.
- `published`: an author with Publishing authority over the new Audience edits
  it directly. Anyone else's edit waits for approval in the same queue, while
  readers keep seeing the approved version until it's approved. A rejected edit
  shows its note in `/mine`, and editing again resubmits it. `409` while another
  edit is already waiting.
- `pending` or `withdrawn`: `409`.

`POST /api/v1/announcements/:id/withdraw` takes a published Announcement down
for everyone. Allowed for its author and for whoever would approve it (the
Department's HOD for a single-Department Audience, the principal or an admin);
`403` for anyone else, `409` if it isn't published. An edit still waiting for
approval is closed.

`POST /api/v1/announcements/:id/submit-for-approval` (author only) submits a
`draft` or `rejected` Announcement. With Publishing authority it publishes
straight away; otherwise it becomes `pending`.

`GET /api/v1/announcements/approvals` (roles: HOD, principal, admin) lists what
the caller may approve, oldest first: single-Department submissions for the
Departments they are HOD of, or everything for the principal and admins. Their
own submissions are never listed.

Each item carries `kind` (`new`, or `edit` for a change to a published
Announcement), `submitted_at`, and for an edit `live: {title, body}`, the
text readers see now, so the approver can compare.

`PATCH /api/v1/announcements/:id/approval` with
`{"decision": "approve" | "reject", "note": "..."}`. Rejecting needs a note,
which the author sees in `/mine` as `review_note`. `403` for anyone who isn't
this Announcement's approver or who submitted it; `409` if it isn't waiting
for approval, including when another approver acted first.

`GET /api/v1/announcements/:id` returns one Announcement to its author, to an
approver it's waiting for, or to a reader whose feed includes it (published
only). Anyone else gets `404`, so unpublished Announcements stay private.

`POST /api/v1/announcements/preview` with `{"category": "...", "audience": [...]}`
returns `{"publishes_directly": true, "reach": 376}` or
`{"publishes_directly": false, "approver": "CS HOD", "reach": 128}`, using the
same rule as posting. `reach` counts the active users the Audience matches
right now, matched the same way as the feed. The composer uses it to say what
will happen, and to whom, before posting.

Pending and sent-back items carry `approver` ("CS HOD" or "Principal or
admin"): who it waits for, or who sent it back. For their
author, a published Announcement with a waiting or rejected edit carries
`edit: {status, review_note, approver}`. The feed accepts `category` to show
one category only.

`GET /api/v1/announcements/mine` (any signed-in user, so former authors keep
seeing their history) lists the caller's own Announcements in any
status, newest first. `status` narrows it to what needs the author:

- `attention`: sent back, either a rejected Announcement or a live one whose
  edit was rejected
- `draft`, `waiting` (pending approval)
- `live`: readers can see it now, including ones with an edit waiting
- `ended`: withdrawn or expired

Any other value is `400`. A waiting or rejected edit's `edit` object carries
its own `title`, `body`, `category`, `audience` and `expires_at`, so the
author can fix it instead of starting again from the live text.

`GET /api/v1/announcements?limit=20&cursor=...` returns the reader's feed:
published, unexpired Announcements whose Audience includes them, newest first.
Each of the reader's roles counts only for its own Department: a
Department-scoped role for its scope, any other role for the Department of the
reader's Student identity. A rule matches when one role satisfies its role and
Department together, so a CS student who is also EC faculty doesn't match "CS
faculty". Batch year comes from the Student identity, which takes it from the
joining year in the USN at sign-up (`4MN23CS001` is batch 2023; any
`batch_year` the client sends is ignored).
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

The workflow and its rules are in ADR 0023. Built so far: creating drafts,
editing them, the proposer's list, submission, both review stages, the feed
and the detail.

`GET /api/v1/events` (any signed-in member, `view_targeted_notices`) lists
published Events whose Audience includes the reader, matched exactly as the
Announcement feed matches, soonest first, cursor-paginated (`limit` up to 50,
`meta.next_cursor`). Without `from` only Events not over yet are listed.
Filters: `from` and `to` (RFC 3339, on `starts_at`), `department` (a code),
`event_type`. A bad value is `400` with the field in `details`. Feed items
carry no review notes.

`GET /api/v1/events/:id` returns one Event to its proposer (any status); to
the principal, admins and the HOD of its Department once it has left draft,
with its `reviews`; and to a reader in its Audience once it has been
published (including after it is cancelled), without review notes. Anyone
else gets `404`, so drafts and proposals stay private.

`POST /api/v1/events/:id/rsvp` with `{"status": "going" | "interested" | "not_going"}`
records the caller's answer (one per person, changeable). The Event must be
published and in the caller's Audience (`404` otherwise), not cancelled and
not started (`409`). `going` when the Event is at capacity is
`409 CONFLICT` ("the event is full"); the Event row is locked while counting,
so two people can't take the last seat. Returns the counts and `my_status`.

`GET /api/v1/events/:id/rsvps` returns `{"counts": {"going", "interested",
"not_going"}, "my_status"}` to anyone who can see the Event. Its organisers
(the proposer, the Department's HOD, the principal and admins) also get
`people` (`user_id`, `full_name`, `username`, `status`, `responded_at`),
earliest answer first, cursor-paginated with `meta.next_cursor`.

`GET /api/v1/events/:id/export` gives the same organisers a CSV download
(`text/csv`, `Content-Disposition: attachment`, `Cache-Control: no-store`) of
everyone who answered: `full_name, email, usn, batch_year, department,
rsvp_status, responded_at`, going first, then by name. USN and Batch appear
only for students. Cells a spreadsheet would run as a formula get a leading
`'`. Every export writes an `event_participants_exported` audit log in the
same transaction. `403` for others who can see the Event, `404` for anyone
else. No passwords, tokens, phone numbers or applicant data are included.

`POST /api/v1/events` (roles with `propose_event`: student coordinator,
faculty, HOD, placement officer, principal, admin) submits a new Event for
review, or saves it as a draft with `"draft": true`:

```json
{
  "title": "Intro to embedded systems",
  "description": "A hands-on session with microcontrollers.",
  "event_type": "workshop",
  "department_id": "<uuid or null for college-wide>",
  "faculty_mentor_id": "<uuid, optional>",
  "location": "CS Lab 3",
  "starts_at": "2026-10-04T09:00:00Z",
  "ends_at": "2026-10-04T11:00:00Z",
  "capacity": 40,
  "audience": [{ "department_id": "<uuid>", "batch_year": 2023 }]
}
```

- `event_type` is `talk`, `workshop`, `competition`, `cultural`, `sports`,
  `training` or `other`. `title` 3 to 200 characters, `description` up to
  5000, `location` 1 to 200, `ends_at` after `starts_at`, `capacity` 1 to
  100000 or left out for no limit, a faculty mentor must hold a faculty role
  in effect. The Audience works as for Announcements; empty means the whole
  college. A draft may have past dates; submitting will need future ones.
- Who may propose what (roles read from the database): faculty and student
  coordinators only for their own Department, an HOD for their own Department,
  the principal and admins for any Department or none, the placement officer
  `training` events only. Anything else is `400` naming `department_id` or
  `event_type`; `403` without `propose_event`.
- Returns `201` with the Event: `id`, `title`, `description`, `event_type`,
  `status`, `proposer_id`, `proposer_name`, `department {id, code}`,
  `faculty_mentor {user_id, full_name}`, `location`, `starts_at`, `ends_at`,
  `capacity`, `audience` (with `department_code`), timestamps.

`PATCH /api/v1/events/:id` (proposer only; anyone else gets `404`) changes
only the fields sent. `department_id`, `faculty_mentor_id` and `capacity` can
be cleared with `null`. The same rules apply to the result. Allowed while the
Event is `draft`, `hod_changes_requested` or `final_changes_requested`;
otherwise `409`.

`POST /api/v1/events/:id/submit-for-approval` (proposer only, `404` for
others) submits a `draft` or an Event sent back for changes; anything else is
`409`. `starts_at` must be in the future (`400`), and the proposer's roles are
checked again. Where it goes:

- principal or admin: `published` straight away;
- after `final_changes_requested`: back to `hod_approved` (final approval);
- the HOD of the Event's Department, or the placement officer with a
  `training` Event: `hod_approved`, skipping the HOD stage;
- anyone else: `submitted`, waiting for HOD review.

`PATCH /api/v1/events/:id/hod-review` (roles with `review_branch_event`) and
`PATCH /api/v1/events/:id/final-approval` (principal, admin) take
`{"decision": "approve" | "request_changes" | "reject", "note": "..."}`. A note
is required unless approving (`400`).

- HOD review: by the HOD of the Event's Department, or by the principal or an
  admin when that Department has no HOD. Approve moves it to `hod_approved`,
  otherwise `hod_changes_requested` or `hod_rejected`.
- Final approval: approve publishes it (`409` if it has already started),
  otherwise `final_changes_requested` or `final_rejected`.
- `403` for anyone who isn't this Event's reviewer at that stage, and for the
  proposer reviewing their own Event. `409` when the Event isn't waiting for
  that stage, including when another reviewer decided first.
- Every decision is kept; Events carry `reviews` (stage, decision, note,
  reviewer name, time), oldest first.

`GET /api/v1/events/reviews` (roles with `review_branch_event`) lists what
waits for the caller, oldest submission first, cursor-paginated: the HOD stage
for the Departments they are HOD of (and, for the principal and admins, for
Departments without an HOD), and every final approval for the principal and
admins. Each item carries `stage` (`hod` or `final`) and the earlier
`reviews`. The caller's own Events are never listed.

`GET /api/v1/events/mine` (any signed-in user) lists the caller's Events,
newest first, cursor-paginated (`limit` up to 50, `meta.next_cursor`).
`status` narrows it: `draft`, `waiting` (at either review stage), `attention`
(changes requested or rejected), `live` (published, not over) or `ended`
(cancelled or over). Any other value is `400`.

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
returns `409 CONFLICT` when student identities, scoped role assignments,
Events or audience rules still reference the department.

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
- Every list endpoint must paginate, except small reference lists that only admins can grow and that clients need whole, such as `GET /api/v1/departments` (one row per college department).
- Every CSV export must be audited.
- Every request body must have a DTO.
- Never return password hashes, refresh tokens, or private applicant notes to unauthorized users.
