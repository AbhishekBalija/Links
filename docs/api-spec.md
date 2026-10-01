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
POST /api/v1/auth/code
POST /api/v1/auth/code/verify
GET  /api/v1/auth/google/nonce
POST /api/v1/auth/google
POST /api/v1/auth/access-request
POST /api/v1/auth/not-me
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
POST /api/v1/auth/activate
POST /api/v1/auth/resend-activation
```

`POST /api/v1/auth/request-access` reads the Department from the USN
(`4MN24IS001` is `IS`) and checks it exists in the `departments` table, so a
Department an admin adds works straight away. `department_code`, when sent,
must be the same Department. `400` for a malformed USN, a joining year out of
range, a code with no Department, or a mismatch.

### Email code

Signing in with a one-time code sent by email (spec #129, ADR 0026).

`POST /api/v1/auth/code` with `{"email": "..."}` always answers `200` the same
way, whether or not the email belongs to anyone:

```json
{
  "data": {
    "challenge_id": "5f0c...",
    "message": "If this email can use LINKS, a code is on its way."
  }
}
```

The browser keeps `challenge_id`; the code works only with it. A 6-digit
code is emailed to any email except one whose account is the principal's or
an admin's (they sign in with Google only), or is suspended or rejected. An
email on no list gets a code too, so its owner can prove it and send an
Access request. Spaces around the
email and its case don't matter. `400` for something that isn't an email.
`429 RATE_LIMITED` after 3 requests for one email, or 60 from one IP address,
within 15 minutes; the limits count every request, known email or not. The
reply takes at least a second, so its timing doesn't show whether a code was
sent.

Both sign-in endpoints accept an account waiting for its first sign-in (an
imported row, a staff invite, or an approved Access request) and make it
active. That reply carries who the account is, for "Not you?":

```json
{
  "data": {
    "access_token": "...",
    "expires_in": 900,
    "first_sign_in": {
      "full_name": "Asha Rao",
      "email": "asha.rao@gmail.com",
      "usn": "4MN23CS042",
      "department_code": "CS",
      "department_name": "Computer Science and Engineering",
      "batch_year": 2023,
      "roles": ["student"]
    }
  }
}
```

Staff have no `usn` or `batch_year`; their Department is the first one a
role is scoped to. Audited as `auth.first_sign_in`.

`POST /api/v1/auth/code/verify` with
`{"challenge_id": "...", "email": "...", "code": "123456"}` signs in like
login: `200` with `access_token` and `expires_in`, and the refresh cookie
(with `first_sign_in` on a first sign-in, see below). `email` is the address
the code was asked for; it is needed only for an email on no list, which gets
`403 NOT_ON_LIST` with a `request_token` (as for Google). A member whose
Access request is waiting or was refused gets `403 ACCOUNT_NOT_ACTIVE`. A code works once, for 10 minutes; 5 wrong tries kill it.
A wrong, used, expired or killed code, an unknown challenge, or an account
that can no longer sign in all get the same
`401 UNAUTHENTICATED` ("the code is wrong or has expired"). Every sign-in is
audited (`auth.signed_in`, method `email_code`). Both code endpoints check
`Origin` like refresh and logout (ADR 0022), so another site can't sign a
browser into someone's account.

### Google sign-in

Signing in with Google Identity Services (spec #129, ADR 0026). Both routes
exist only when `GOOGLE_CLIENT_ID` is set.

`GET /api/v1/auth/google/nonce` returns `{"nonce": "..."}` and sets the same
value in an httpOnly `google_nonce` cookie (path `/api/v1/auth/google`, 10
minutes). The screen passes the nonce to Google Identity Services, which puts
it in the ID token.

`POST /api/v1/auth/google` with `{"credential": "<Google ID token>"}` checks
the token's signature (Google's keys), audience (`GOOGLE_CLIENT_ID`), issuer
(`accounts.google.com`), expiry and `email_verified`, and that its `nonce`
matches the cookie. The nonce cookie is cleared on every try. It finds the
member by Google account ID (`sub`); on their first Google sign-in, by the
verified email (case-insensitive), and then stores the Google account ID.

- `200`: signed in like login, with `access_token`, `expires_in` and the
  refresh cookie. Audited as `auth.signed_in` (method `google`), plus
  `auth.google_linked` the first time.
- `401 UNAUTHENTICATED` ("Google sign-in failed; try again"): a token that
  fails any check, a missing or different nonce, or an email whose member is
  already linked to a different Google account.
- `403 NOT_ON_LIST`: the verified email isn't on any list. No account is
  created. The details prefill an Access request, and `request_token`
  (30 minutes) sends it:

```json
{
  "error": {
    "code": "NOT_ON_LIST",
    "message": "this email isn't on any list for LINKS yet",
    "details": {
      "email": "asha.rao@gmail.com",
      "full_name": "Asha Rao",
      "request_token": "..."
    }
  }
}
```

- `403 ACCOUNT_NOT_ACTIVE` with `details.status` (`pending`, `suspended` or
  `rejected`): the member exists but can't sign in now.

The principal and admins sign in this way; they can't use email codes.
The POST checks `Origin` like refresh and logout (ADR 0022).

### Not on the list

`POST /api/v1/auth/access-request` sends an Access request without a
password, for an email proven by a Google sign-in or an email code:

```json
{ "request_token": "<from NOT_ON_LIST>", "usn": "4MN23CS077", "full_name": "Kiran S" }
```

The Department and Batch come from the USN. The request joins the
Department's review queue (ADR 0025). `201` with
`{"user_id": "...", "status": "pending"}`. `401` for a missing, expired or
forged token; `400` for a malformed USN, a Department code with no
Department, or an empty name; `409` for an email or USN already registered.
Audited as `access_requested`. Once approved, the person signs in with Google
or a code; there is nothing to activate, and no Activation email is sent.

### Not you?

`POST /api/v1/auth/not-me` (signed in) is "Not you?" on a first sign-in: the
list row isn't the person who signed in. Within an hour of the first sign-in
it signs the account out everywhere (every refresh token revoked, the cookie
cleared), unlinks any Google account, and returns the account to waiting for
its first sign-in, so an admin can fix the row. Audited as `auth.not_me`.
`409` for an account not signed into for the first time in the last hour.
An access token already issued keeps working until it expires (15 minutes).

`GET /api/v1/public/departments` needs no token. It returns only what the
Access request form shows, ordered by name, with
`Cache-Control: public, max-age=300` (ADR 0021):

```json
{
  "data": [
    { "code": "CS", "name": "Computer Science and Engineering" },
    { "code": "IS", "name": "Information Science and Engineering" }
  ]
}
```

`POST /api/v1/auth/refresh` and `POST /api/v1/auth/logout` use the refresh
cookie, so they also check where the request came from (ADR 0022). A request
whose `Origin` (or, without one, `Referer`) isn't `FRONTEND_URL`, one of
`CORS_ALLOWED_ORIGINS` or the API's own host gets `403 FORBIDDEN`
("request origin not allowed"). Requests with neither header, from
non-browser clients, pass.

Every API response carries `X-Content-Type-Options: nosniff`,
`Referrer-Policy: strict-origin-when-cross-origin`, `X-Frame-Options: DENY`
and `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`
(plus `Strict-Transport-Security` in production).

`POST /api/csp-report` takes a browser's Content-Security-Policy violation
report (`{"csp-report": {...}}`) without a token, logs it and returns `204`.
It exists for the web app's report-only policy.

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

### `GET /api/v1/profiles/:username`

Works without a token. A public profile is visible to everyone, a private one
only to its owner (`404` for anyone else). Email and phone appear only for the
owner or when the owner opted in.

Signed-in viewers also get who the member is at the college, described exactly
as a directory entry describes them: `roles` (in effect, most senior first),
`department` (`{code, name}`) and, for students, `batch_year`. Anonymous
visitors never get these fields.

### `PATCH /api/v1/me/profile`

Updates the caller's own profile. Every field is optional; a missing field is
left as it is, and an empty string clears a text field.

```json
{
  "headline": "...", "bio": "...", "avatar_url": "https://...",
  "linkedin_url": "https://...", "github_url": "https://...", "portfolio_url": "https://...",
  "public_profile_enabled": false, "show_email": true, "show_phone": false
}
```

- `public_profile_enabled` (default `true`): off, the member leaves
  `GET /api/v1/directory` and `GET /api/v1/profiles/:username` returns `404`
  to everyone but them. Department overview counts still include them. A
  change is audited as `profile_visibility_changed` with the new value, in the
  same transaction.
- `show_email` and `show_phone`: a change is audited as
  `profile_privacy_updated`.
- URLs must be `http` or `https`. Returns the updated profile as the owner
  sees it.

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
  "approvals": {
    "pending_count": 2, "oldest_submitted_at": "...",
    "events_pending_count": 1, "oldest_event_submitted_at": "..."
  },
  "my_announcements": { "draft": 1, "pending": 1, "rejected": 1, "edits_waiting": 1 },
  "opportunities": { "items": [/* next three open ones, as in the feed */], "has_more": true },
  "placement": {
    "open_count": 4, "awaiting_review_count": 2,
    "drives": [{ "id": "...", "opportunity_type": "job", "title": "...", "company": "...", "apply_by": "...",
      "applicant_counts": { "total": 2, "applied": 1, "shortlisted": 1, "rejected": 0, "selected": 0, "withdrawn": 0 } }]
  }
}
```

- `approvals` appears only for HODs, the principal and admins.
  `pending_count` and `oldest_submitted_at` count Announcements and edits
  waiting for Announcement approval; `events_pending_count` and
  `oldest_event_submitted_at` count Event proposals waiting for the caller,
  with the same scope as `GET /api/v1/events/reviews` (the HOD stage of their
  Departments, and for the principal and admins the HOD stage of Departments
  without an HOD and every final approval; never the caller's own). An
  `oldest_*` field is `null` when nothing waits.
- `my_announcements` appears only for users who can post.
- `department` appears for HODs: their Department's `code`, `name`, number of
  `students` and `staff`, `students_by_batch` (`batch_year`, `count`) and
  `upcoming_events`, the next three published Events of the Department
  (`id`, `title`, `event_type`, `location`, `starts_at`), whoever they are for.
- `college` appears for the principal and admins: `departments`, each with
  `code`, `name`, `students`, `staff` and `hod` (`full_name`, `username`, or
  `null` when there is none).
- `access_requests` appears for whoever decides Access requests (the
  principal, admins and HODs, scoped as the review queue): `pending_count` and
  `oldest_requested_at`.
- `opportunities` appears when the caller is eligible for an open
  Opportunity: the first three from `GET /api/v1/opportunities`, soonest
  deadline first, each with the caller's own Application.
- `placement` appears only for placement staff: how many Opportunities are
  open, how many Applications to published or closed ones are still
  `applied`, and up to five open drives, soonest deadline first, with their
  `applicant_counts` (as on the manage list).
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

`GET /api/v1/admin/users/review-queue` lists the Access requests still
waiting for approval (pending and not yet approved), oldest first: every
request for the principal and admins, and only their own Department's for an
HOD (`approve_access`, ADR 0025). An approved student who hasn't activated yet
is no longer in it.

Approving (`verify`) and rejecting (`status` with `rejected`) follow the same
scope: an HOD gets `404` for a request outside their Department. Suspending
and reactivating need `manage_users_and_roles` (principal and admin); an HOD
gets `403`.

`PATCH /api/v1/admin/users/:id/verify` accepts an optional `scope_type` and
`scope_id` for the student role (global when omitted). A `department` scope
must carry an existing department's ID, otherwise it returns
`400 VALIDATION_ERROR`. The department row is share-locked while the role is
created, so a concurrent department delete either waits and returns `409` or
runs first and the approval returns `400`.

`PATCH /api/v1/admin/users/:id/status` with `{"status": "suspended" | "rejected" | "active", "note": "..."}`.
Moving a user to `suspended` or `rejected` also revokes all their refresh
tokens in the same transaction, so every signed-in device is signed out at its
next refresh.

### Staff invites

`POST /api/v1/admin/users` (principal and admin, `manage_users_and_roles`)
adds a staff member by email and role:

```json
{
  "email": "meera@college.example",
  "full_name": "Meera Iyer",
  "role": "faculty",
  "scope_type": "department",
  "scope_id": "<department id>",
  "note": "optional"
}
```

The account waits for its first sign-in, like an imported row. The role
follows role management's rules (below): only an admin invites an admin,
a Department has one HOD at a time, and the Scope must fit the role. `201`
with `{"user_id": "...", "status": "pending"}`. `400` for an invalid email,
an empty name or a role and Scope that don't fit; `409` for an email already
registered or a role that clashes. Nothing is created on any error. Audited
as `user_invited` and `role_granted`.

### Student import

`POST /api/v1/admin/users/import` (admin, principal, or an HOD for their own
Department) takes a multipart upload with the CSV in the field `file`:

```csv
email,full_name,usn
asha.rao@gmail.com,Asha Rao,4MN23CS101
ravi.k@gmail.com,"Kumar, Ravi",4MN24EC102
```

- The header must have exactly `email`, `full_name` and `usn`, in any order.
  Excel's UTF-8 byte order mark and CRLF line ends are fine.
- At most 200 rows and 1 MB. An empty file, a wrong header, a malformed CSV,
  too many rows or too large a file is `400` and nothing is imported.
- Each row is its own transaction, so rows succeed or fail on their own. A
  created row is a `pending`, verified user with a Student identity (Department
  and Batch from the USN) and the `student` role, waiting for its first
  sign-in (#133). No email is sent: the student signs in with Google or an
  email code, and that first sign-in makes the account active.
- A row fails for: an invalid email, an empty or overlong name, a missing or
  malformed USN, a Department code with no Department, an email or USN
  already registered or earlier in the same file, or (for an HOD) a
  Department other than theirs.

Response `200`:

```json
{
  "data": {
    "created": 1,
    "failed": 1,
    "rows": [
      { "row": 2, "email": "asha.rao@gmail.com", "status": "created", "user_id": "uuid" },
      { "row": 3, "email": "ravi.k@gmail.com", "status": "failed", "error": "the USN is already registered" }
    ]
  }
}
```

`row` is the spreadsheet row (the header is row 1). The import writes one `students_imported` audit log
with the counts and one `user_imported` per created user.

### Role management

Principal and admin (`manage_users_and_roles`). Only an admin may grant or end
the `admin` role; the principal gets `403` for it. The caller's roles for that
check are read from the database, not the token.

`GET /api/v1/admin/users/:id/roles` lists all of a user's Role assignments,
newest start first: in effect now (`active`), starting later (`scheduled`) and
`ended`. It is not paginated; a user holds a handful of assignments.

```json
{
  "data": [
    {
      "id": "uuid",
      "role": "faculty",
      "scope_type": "department",
      "scope_id": "uuid",
      "department": { "id": "uuid", "code": "CS", "name": "Computer Science and Engineering" },
      "assigned_by": "uuid",
      "starts_at": "2026-09-27T10:00:00Z",
      "ends_at": null,
      "state": "active",
      "created_at": "2026-09-27T10:00:00Z"
    }
  ]
}
```

`POST /api/v1/admin/users/:id/roles` grants a staff role and returns `201` with
the assignment in the shape above:

```json
{
  "role": "hod",
  "scope_type": "department",
  "scope_id": "<department uuid>",
  "starts_at": "2026-10-01T00:00:00Z",
  "ends_at": null,
  "note": "Takes over from Dr. Rao"
}
```

- `role` is one of `faculty`, `hod`, `student_coordinator` (these need
  `scope_type: "department"` and a Department ID) or `placement_officer`,
  `principal`, `admin` (these need `scope_type: "global"` and no `scope_id`).
  `student` and `alumni` come from Access approval and Graduation, and club
  roles wait for clubs, so they are `400` here.
- `starts_at` defaults to now and can't be in the past. `ends_at` is optional
  and must be after `starts_at` and in the future.
- A `student_coordinator` must be a current Student (student role in effect)
  whose Student identity is in that Department.
- `400 VALIDATION_ERROR` for the rules above or an unknown Department (with
  field `details`); `404` for an unknown user.
- `409 CONFLICT` when the user already holds the same role and Scope for an
  overlapping time, when the Department already has another HOD for an
  overlapping time, or when the user is `rejected`.

`DELETE /api/v1/admin/users/:id/roles/:roleAssignmentId` ends the assignment
now and returns it with `state: "ended"`. The row is kept as history. A
scheduled assignment is ended at its start, so it never takes effect. In the
same transaction it revokes all the user's refresh tokens, clears the
Department's named HOD when an HOD role ends, and writes the audit log. `404`
when the assignment isn't this user's, `409` when it has already ended or when
it is the last admin role in effect.

Both grant and end write an audit log (`role_granted`, `role_ended`) with the
role, Scope, dates and optional note.

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
editing them, the proposer's list, submission, both review stages, the feed,
the detail, RSVPs, the export, logistics edits and cancelling. The full list
of Event endpoints adds `GET /api/v1/events/mine`, `GET /api/v1/events/reviews`
and `POST /api/v1/events/:id/cancel`.

`GET /api/v1/events` (any signed-in member, `view_targeted_notices`) lists
published Events whose Audience includes the reader, matched exactly as the
Announcement feed matches, soonest first, cursor-paginated (`limit` up to 50,
`meta.next_cursor`). Without `from` only Events not over yet are listed.
Filters: `from` and `to` (RFC 3339, on `starts_at`), `department` (a code),
`event_type`, and `show`:

- `upcoming` (the default): Events not over yet, soonest first. A cancelled
  Event stays listed, with `status: "cancelled"`, for readers who answered
  it, until it ends, so the people planning to come find out.
- `going`: upcoming Events the reader answered `going` to.
- `past`: Events that have ended, most recent first; `next_cursor` pages
  backwards in time.

A bad value is `400` with the field in `details`. Feed items carry no review
notes. Each carries `rsvp: {counts: {going, interested, not_going},
my_status}`, read for the whole page at once, so a list needs no call per
Event.

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

A `published` Event that isn't over takes logistics edits through
`PATCH /api/v1/events/:id` from its organisers (the proposer, the Department's
HOD, the principal, admins; `404` for anyone else): `description`,
`location`, `starts_at` (in the future), `ends_at` and `capacity` (not below
the number already going; `null` removes the limit). Sending `title`,
`event_type`, `department_id`, `faculty_mentor_id` or `audience` is `400`:
those need the Event cancelled and proposed again. Each edit writes an
`event_logistics_updated` audit log. An Event that is over is `409`.

`DELETE /api/v1/events/:id` deletes the caller's own draft and its Audience
rules, audited as `event_draft_deleted`, and returns `204`. Someone else's
draft is `404`. An Event that has been submitted is `409`: reviewers have seen
it, so it is cancelled instead and its history kept.

`POST /api/v1/events/:id/cancel` with `{"reason": "..."}` (required, up to 500
characters) cancels an Event that isn't over, rejected or already cancelled
(`409`). Allowed for its proposer and its reviewers (Department HOD, principal,
admins); `403` for others who can see it, `404` for anyone else. RSVPs are
kept; the Event leaves the feed but its Audience can still open it and see
`cancel_reason`. Audited as `event_cancelled`.

`GET /api/v1/events/mine` (any signed-in user) lists the caller's Events,
newest first, cursor-paginated (`limit` up to 50, `meta.next_cursor`).
`status` narrows it: `draft`, `waiting` (at either review stage), `attention`
(changes requested or rejected), `live` (published, not over) or `ended`
(cancelled or over). Any other value is `400`.

## Opportunities and Applications

The placement workflow is ADR 0024: drafts, publishing, the feed,
Applications, the applicant list with status updates, and the export.

```text
GET   /api/v1/opportunities
POST  /api/v1/opportunities
GET   /api/v1/opportunities/manage
GET   /api/v1/opportunities/:id
PATCH /api/v1/opportunities/:id
POST  /api/v1/opportunities/:id/publish
POST  /api/v1/opportunities/:id/close
POST  /api/v1/opportunities/:id/apply
POST  /api/v1/opportunities/:id/withdraw
GET   /api/v1/opportunities/:id/applications
PATCH /api/v1/opportunity-applications/:id/status
GET   /api/v1/opportunities/:id/export
```

### Opportunities

Placement staff (the placement officer, the principal and admins,
`post_opportunity`) work as one office: any of them can create, edit and list
any Opportunity, drafts included. Their roles are read from the database.
Anyone else gets `403` on these routes.

`POST /api/v1/opportunities` saves a draft (`201`):

```json
{
  "opportunity_type": "job",
  "title": "Graduate Engineer Trainee",
  "company": "Acme Systems",
  "description": "...",
  "location": "Mysuru",
  "compensation": "4.5 LPA",
  "apply_by": "2026-10-15T18:30:00Z",
  "application_mode": "internal",
  "external_url": null,
  "eligibility": [{ "department_id": "<uuid>", "batch_year": 2023, "role": "student" }]
}
```

- `opportunity_type`: `job`, `internship` or `training`.
- `title` (the role) 3 to 200 characters; `company` 1 to 200; `description`
  up to 10,000; `location` and `compensation` optional, up to 200.
  `compensation` is free text for a stipend or CTC.
- `apply_by` is required.
- `application_mode`: `internal` (Students apply in LINKS) or `external`
  (Students apply on the company's site). `external_url` is required for
  `external`, must be an `http` or `https` link, and is refused for
  `internal`.
- `eligibility` is an Audience, matched as for Announcements (a rule's fields
  must all match; any rule is enough; empty means everyone), up to 20 rules.
  An unknown Department is `400`.

`PATCH /api/v1/opportunities/:id` changes only the fields sent; `location`,
`compensation` and `external_url` can be cleared with `null`, and
`eligibility` replaces the whole list. The result is checked as on create
(`400`); an unknown ID is `404`. A published Opportunity keeps its
`application_mode` (`409` otherwise) and a changed `apply_by` must be in the
future (`400`); a closed one can't be edited (`409`).

`POST /api/v1/opportunities/:id/publish` opens a draft to its Eligibility
(`200`). Its `apply_by` must be in the future (`400`); anything but a draft is
`409`. `POST /api/v1/opportunities/:id/close` closes a published Opportunity
early (`200`); anything else is `409`. Closing is final. Both are placement
staff only and audited (`opportunity_published`, `opportunity_closed`).

`GET /api/v1/opportunities` (any signed-in member) lists published and closed
Opportunities the caller is eligible for, matched like an Announcement's
Audience:

- `state`: `open` (default: published and `apply_by` ahead, soonest deadline
  first), `closed` (closed early or past `apply_by`, latest deadline first) or
  `applied` (what the caller applied to, in any status, eligible now or not,
  latest deadline first).
- `type`: `job`, `internship` or `training`.
- `department`: a Department code (any case); lists Opportunities whose
  Eligibility names that Department.
- `cursor` and `limit` (1 to 50, default 20); the next page is
  `meta.next_cursor`.

An unknown `state`, `type` or `department`, or a bad cursor, is `400`.

`GET /api/v1/opportunities/manage` lists every Opportunity for placement
staff, newest first: `status` (`draft`, `published`, `closed`), `cursor` and
`limit` (1 to 50, default 20); the next page is `meta.next_cursor`. Each
carries `applicant_counts`: `total` and the count in each status, with
`withdrawn` counted apart and left out of `total`. Placement staff also get
`applicant_counts` on `GET /api/v1/opportunities/:id`; nobody else ever does.

`GET /api/v1/opportunities/:id` returns one Opportunity: any of them to
placement staff, a published or closed one to a member in its Eligibility or
who applied to it, and `404` to anyone else, so drafts stay private.

Each Opportunity:

```json
{
  "id": "uuid",
  "opportunity_type": "job",
  "title": "...", "company": "...", "description": "...",
  "location": "Mysuru", "compensation": "4.5 LPA",
  "apply_by": "...",
  "application_mode": "internal", "external_url": null,
  "eligibility": [{ "department_id": "<uuid>", "department_code": "CS", "batch_year": 2023, "role": "student" }],
  "status": "published",
  "open": true,
  "my_application": { "id": "uuid", "opportunity_id": "uuid", "mode": "internal", "status": "applied", "applied_at": "...", "withdrawn_at": null },
  "posted_by": { "user_id": "uuid", "full_name": "..." },
  "published_at": null, "closed_at": null,
  "created_at": "...", "updated_at": "..."
}
```

`open` is `true` while the Opportunity is published and `apply_by` is ahead.
`my_application` is the caller's own Application (or `null`); nobody sees
anyone else's here.
Creating and editing are audited (`opportunity_created`,
`opportunity_updated`) in the same transaction.

### Applications

`POST /api/v1/opportunities/:id/apply` (`201`) records the caller's
Application. For an `internal` Opportunity this is their application; for an
`external` one it is their note that they applied on the company's site
(`mode: "external"`), so it shows in their list and in placement reports.

- The Opportunity must be published or closed and the caller in its
  Eligibility, otherwise `404`.
- Only a Student applies (the `student` role in effect and a Student
  identity): anyone else gets `403`.
- It must be open (published, `apply_by` ahead), otherwise `409`.
- Once per Student per Opportunity: a second application, even after
  withdrawing, is `409`.

`POST /api/v1/opportunities/:id/withdraw` (`200`) withdraws the caller's
Application while it is still `applied`; once shortlisted, rejected or
selected it is `409`, and without an Application `404`. Withdrawing is final.

Both return the caller's Application (`my_application` above) and are audited
(`application_submitted`, `application_withdrawn`) in the same transaction.

### Applicant list and status updates

`GET /api/v1/opportunities/:id/applications` (placement staff,
`view_applicant_data`; `403` for anyone else, `404` for an unknown
Opportunity) lists its Applications in the order they were made:

```json
{
  "data": [{
    "id": "uuid",
    "opportunity_id": "uuid",
    "student": {
      "user_id": "uuid", "full_name": "...", "username": "...",
      "email": "...", "usn": "4MN23CS001", "department_code": "CS", "batch_year": 2023
    },
    "mode": "internal",
    "status": "applied",
    "applied_at": "...", "withdrawn_at": null, "status_changed_at": null
  }],
  "meta": { "next_cursor": "..." }
}
```

- Filters: `q` (part of the Student's name, username, email or USN, any
  case, up to 100 characters), `status` (`applied`, `shortlisted`,
  `rejected`, `selected`, `withdrawn`), `department` (a Department code, any
  case) and `batch` (the Student's Batch); `cursor` and `limit` (1 to 50, default 20). A bad filter
  or cursor is `400`.
- Never a phone number. Opening the list (a request without a cursor) is
  audited as `applicants_viewed` with the filters.

`PATCH /api/v1/opportunity-applications/:id/status` (placement staff,
`shortlist_applicants`) with `{"from": "applied", "status": "shortlisted"}`
moves an Application among `applied`, `shortlisted`, `rejected` and
`selected`, in any direction so a mistake can be undone. `from` is the status
the caller saw: the Application row is locked, and if its status has moved on
the change is `409` rather than an overwrite. A withdrawn Application is the
Student's decision (`409`); `withdrawn`, an unknown status or the current one
as the target is `400`; an unknown ID is `404`. Returns the Application as in
the list, and is audited as `application_status_changed` with `from` and `to`.

`GET /api/v1/opportunities/:id/export` (placement staff, `view_applicant_data`)
downloads the applicants as `text/csv` (`Content-Disposition: attachment`,
`Cache-Control: no-store`), in the order they applied, optionally one
`status` only:

```csv
full_name,email,usn,department,batch_year,mode,status,applied_at,status_changed_at
```

Cells that could run as a spreadsheet formula (starting with `=`, `+`, `-`,
`@`, a tab or a carriage return) get a leading `'`. No phone numbers. Every
export is audited as `applicants_exported` with the row count and status, in
the same transaction as reading the rows. `403` for anyone else, `404` for an
unknown Opportunity, `400` for an unknown status.

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

The list and detail routes require authentication (the code-and-name list for
the sign-up form is `GET /api/v1/public/departments`, above). Department mutations require
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

`GET /api/v1/departments/:code/overview` (any signed-in member; `401` without
a token, `404` for an unknown code, the code is case-insensitive) is the
Department's page:

```json
{
  "data": {
    "department": { "code": "CS", "name": "Computer Science and Engineering", "description": null },
    "hod": { "username": "hema.h", "full_name": "Hema H", "roles": ["hod", "faculty"], "department": { "code": "CS", "name": "..." } },
    "counts": {
      "students": 3,
      "faculty": 4,
      "students_by_batch": [ { "batch_year": 2023, "count": 2 }, { "batch_year": 2024, "count": 1 } ]
    },
    "staff": [ /* directory entries */ ]
  }
}
```

- `hod` is the Department's HOD as a directory entry, or `null` when there is
  none or the directory wouldn't list them (hidden profile, suspended).
- `counts` include every active member with the role in effect, hidden
  profiles too, since a number reveals no one: `students` (student role, Student
  identity in this Department) by Batch, oldest first, and `faculty` (faculty
  role scoped here, including an HOD who also teaches).
- `staff` lists the members the directory would show who hold an HOD, placement
  officer or faculty role scoped to this Department, most senior role first,
  then by name. Entries have the same shape and privacy as the directory.

Create the department before assigning its HOD. On update, `hodUserId` must
identify a user with an existing HOD role scoped to that same department. Delete
returns `409 CONFLICT` when student identities, scoped role assignments,
Events or audience rules still reference the department.

## Directory

```text
GET /api/v1/directory
```

Any signed-in member (`view_public_profiles`); `401` without a token. Lists
members alphabetically by `full_name` (case-insensitive), then by user ID,
cursor-paginated: `limit` 1 to 50 (default 20), `cursor` from
`meta.next_cursor`. `meta.total` is how many members match the filters in all,
the same on every page.

Who appears: active, verified accounts with a public profile and at least one
role in effect now. Suspended, pending and rejected accounts, hidden profiles
(even the viewer's own) and people whose roles have all ended never appear.

Filters, combined with AND:

- `department`: a Department code (`CS`). Matches a member whose Student
  identity is in it or who holds a Department-scoped role in effect there.
- `role`: a LINKS role (`faculty`), in effect now.
- `batch`: the Student identity's Batch (`2023`), 2000 to 2100.

An unknown Department, a role that isn't a LINKS role, a batch out of range,
or a bad `limit` or `cursor` is `400 VALIDATION_ERROR` with the field in
`details`.

Search: `q` matches `full_name`, `username` and `headline`, case-insensitively
and with typos (`ash`, `ASHA`, `ahsa` and `asha rau` all find "Asha Rao"). It
combines with the filters. With `q` the best matches come first (ties by
name), up to 50 in one response with no `next_cursor`; `limit` is ignored and
sending `cursor` with `q` is `400`. `meta.total` is the number of matches
returned. A `q` shorter than 2 characters is ignored
(the normal list comes back); longer than 100 is `400`. The USN is never
searched.

A result matches when the text contains `q`, when a word is close enough by
trigram similarity (`pg_trgm` `word_similarity` of at least 0.3), or, for a
one-word `q`, when a name word has the same letters (a swapped-letter typo).

```json
{
  "data": [
    {
      "username": "asha.rao",
      "full_name": "Asha Rao",
      "headline": "Networks and systems",
      "avatar_url": null,
      "roles": ["hod", "faculty"],
      "department": { "code": "CS", "name": "Computer Science and Engineering" },
      "email": "asha.rao@gmail.com"
    },
    {
      "username": "bala.k",
      "full_name": "Bala Krishna",
      "headline": null,
      "avatar_url": null,
      "roles": ["student_coordinator", "student"],
      "department": { "code": "CS", "name": "Computer Science and Engineering" },
      "batch_year": 2023
    }
  ],
  "meta": { "next_cursor": "...", "total": 412 }
}
```

- `roles` are the roles in effect, once each, most senior first: admin,
  principal, hod, placement_officer, faculty, student_coordinator,
  club_organizer, student, alumni.
- `department` is the most senior Department-scoped role's Department,
  otherwise the Student identity's, otherwise `null`.
- `batch_year` is present only for students.
- `email` and `phone` are present only when the member chose to show them,
  by the same rule as `GET /api/v1/profiles/:username`.
- The USN is never returned.

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
