# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed
- The API failed to start on Vercel (every request returned 500): on each
  cold start it checked every migration in its own transaction, and with
  the database in another region those round trips passed the startup
  deadline. Startup now reads the applied migrations in one query and only
  locks and applies the missing ones.

### Added
- The event feed carries each Event's answer counts and your own answer,
  lists upcoming, going or past Events (`show`), and keeps a cancelled Event
  in view, until it ends, for the people who answered it.
- Event proposals (Phase 3): staff and student coordinators save Event drafts
  for their Department (the principal and admins for any or none, the
  placement officer training only), with type, place, times, capacity,
  faculty mentor and Audience, edit them, and list their own. Submitted
  events go to their Department's HOD (or the principal or an admin when there
  is none) and then to the principal or an admin for final approval; reviewers
  approve, ask for changes or reject with a note, and see earlier notes. HODs'
  own events and training events skip the HOD stage; the principal's and
  admins' events publish straight away. Members see upcoming published
  events for their Audience, filtered by date, Department and type, and open
  one; review notes stay with the proposer and reviewers. Members answer
  going, interested or not going; going stops at the event's capacity, and
  organisers see who answered and can export them as CSV (each export is
  audited). After publishing, organisers can change the description, place,
  times and capacity (each change audited), or cancel with a reason; answers
  are kept.
- People, profile and department screens (#12). People opens on your own
  Department, A to Z under letter headings, with search that forgives typos
  and filters for Department, role and Batch (selects on desktop, chips and
  one filter sheet on phones). Opening someone shows who they are, what they
  wrote, the contact details they share (with a copy button) and their
  Department. Profile shows your own page as others see it, with Edit
  profile, which now returns there after saving. A Department page lists
  its HOD, faculty and student coordinators with student numbers by Batch.
  Home's department code links to it. Phones keep at most five tabs, so an
  HOD's Profile moves to the avatar on Home.
- The directory says how many members match (`meta.total`), and a signed-in
  member opening a profile sees that member's roles, Department and Batch.
  Anonymous visitors still see only the profile itself.
- Member directory (#12): signed-in members can list active members
  alphabetically, filtered by Department, role and Batch, with roles,
  Department, Batch for students, and contact details only where the member
  opted in. Hidden, suspended, pending and former members never appear.
  Searching by name, username or headline tolerates typos and ranks the
  best matches first. Each Department has an overview with its HOD, staff,
  and student counts per Batch.
- Role management (#22): the principal and admins can list, grant and end a
  user's staff roles, with Department scopes, start and end dates, one HOD per
  Department, and an audit log. Only an admin can grant or end the admin role,
  and the last admin can't be removed.
- Bulk student import (#17): admins, the principal and HODs (for their own
  Department) upload a CSV of up to 200 students with email, full name and
  USN. Each valid row becomes a verified student with an activation email,
  sent in batches once every row is saved, and the response reports every
  row's result.

### Changed
- Department codes come from the database (#18): a Department an admin adds
  works for access requests and appears in the sign-up form without a code
  change. A new public endpoint lists Department codes and names. A USN whose
  Department differs from the chosen one is now rejected.

### Security
- Refresh and logout refuse requests from other sites (Origin or Referer
  check), every API response carries security headers, and the web app
  sends a report-only Content-Security-Policy whose violations are logged
  by the API (#19). The API refuses to start with an insecure cookie setup.
- Ending a role, suspending or rejecting a user signs them out everywhere by
  revoking their refresh tokens (#22).

## [0.3.0] - 2026-09-27

LINKS gets its look and its screens: everyone can read notices, staff can
write them, and approvers can review them, on desktop and phone.

### Added
- Home summary endpoint (#39): the signed-in user's latest notices, plus
  what's waiting for approvers and the state of an author's own
  announcements. Announcements gain a detail endpoint, a category filter, a
  publishing preview and approver names on pending items.
- The LINKS look and app shell (#40): the Gazette design tokens and fonts, a
  sidebar on desktop and bottom navigation on phones, a Home page from the
  new summary, and a Notices feed that loads more as you scroll, filters by
  category and opens each notice. Every screen has loading, empty and error
  states.
- Writing announcements (#41): a composer with one-tap audience picks (or
  custom groups by department, batch and role) and how many people they
  reach, a preview, an optional expiry, and a line saying whether it
  publishes now or which approver it goes to. My announcements puts
  anything sent back first, with the note, and tabs for drafts, waiting,
  live and ended; opening one gives Edit and Withdraw (with a
  confirmation). Leaving the composer with unsaved text offers to keep it
  as a draft.
- The approval queue (#42): HODs, the principal and admins review what's
  waiting for them, oldest first, with the full text, who it goes to and
  how many people it reaches. Approving asks once more (Enter confirms);
  sending back needs a note. Edits to live notices can be compared with the
  live version, items that expired while waiting can only be sent back, and
  if someone else acts first the queue says so and refreshes. Home lists the
  oldest waiting items, and the sidebar shows the count.

### Fixed
- Students who signed up through the access request form had no Batch, so
  announcements for a batch (e.g. "CS students, batch 2023") never reached
  them (#45). The Batch now comes from the USN, and existing students are
  fixed by migration 015.

### Changed
- The first visit downloads about 42% less (743 kB to 409 kB): each page,
  and Sentry's session replay, load only when needed (#52).
- Only Vercel production builds upload source maps to Sentry; local builds
  no longer do (#51).
- Strict TypeScript, gofmt checked in CI, Vitest unit tests for the client's
  logic (#53) and unit tests for sign-in, activation and profile privacy
  (#8). ADR 0020 records that migrations only go forward.
- The composer and the auth service are split into smaller files (#54,
  #56), with no change in behaviour.

## [0.2.0] - 2026-09-26

Phase 2 begins: department management and targeted announcements with
approval, so staff can reach exactly the students a notice concerns.

### Added
- Targeted announcements (#27): HODs (own Department), the
  principal, admins and the placement officer (placement notices) publish
  Announcements to an Audience of Departments, batch years and roles, or the
  whole college. Readers get a paginated feed of what targets them, and
  expired notices drop out.
- Announcement approval (#28): faculty, student coordinators and anyone
  posting outside their own authority submit Announcements for approval. The
  Department's HOD approves single-Department ones, the principal or an admin
  everything else. Approvers get a queue, rejections carry a note, authors can
  save drafts, fix rejected ones and resubmit, and see all their own
  Announcements with their status.
- Editing and withdrawing announcements (#29): authors with publishing
  authority edit published announcements directly; other authors' edits wait
  for approval while readers keep seeing the approved text. Authors and
  approvers can withdraw a published announcement.
- API tests run the real router against an isolated Postgres schema (#26).
- Department management (#10): any signed-in user can list and read
  departments, and admins can create, update and delete them. Codes can't
  change, a department still in use can't be deleted, and every change is
  audited. Migration 011 seeds the pilot college's current B.E.
  departments (CS, AD, AI, CV, EC, ME).

### Fixed
- Roles count only while they are in effect: an ended or future-dated role
  no longer grants permissions at login, refresh or HOD assignment.
- Approving a user with a department scope now checks the department
  exists, and can't race with deleting that department (#9).
- Two admins creating the same department code at once get 409, not 500.

### Changed
- LINKS is described as a hub for any college, with MITT as the pilot college
  (ADR 0018). Pilot-college reference notes moved out of the public repo.
- The access request form and USN validator use `AI` for CSE (AI and ML)
  instead of the unconfirmed `CI` code.

## [0.1.0] - 2026-08-29

The first release: Phase 0 (foundation) and Phase 1 (identity and access),
shipped in PR #6 and verified in production on 2026-08-30. The changelog was
started after this release, so these entries are written from the phase
records.

### Added
- Go modular-monolith API with validated config, request IDs, structured
  logs, health and readiness checks, and SQL migrations that run on startup
  and are embedded in the binary.
- React client (Vite, TypeScript, Tailwind CSS, shadcn/ui), deployed with the
  API as one Vercel project.
- Access requests with Gmail, USN and department, validated against the college's
  USN format.
- HOD/admin review queue and approval, which assigns the student role and
  emails a single-use activation link through Resend.
- Account activation: the user sets their password from the emailed link.
- Login with short-lived JWT access tokens and rotating refresh tokens in an
  HTTP-only cookie; refresh tokens are stored hashed. Silent refresh in the
  client, including after a page reload.
- Scoped role assignments and a permission policy (RBAC).
- Profiles: an edit-profile page with privacy toggles for email and phone, and a public profile API.
- Audit logs for approvals and status changes.
- Security tests for the auth surface and a Playwright e2e suite run in CI.

[Unreleased]: https://github.com/AbhishekBalija/Links/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/AbhishekBalija/Links/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/AbhishekBalija/Links/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/AbhishekBalija/Links/releases/tag/v0.1.0
