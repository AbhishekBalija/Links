# Roadmap

The build plan, one phase per GitHub milestone. Issues are filed per phase
when the phase starts, not all up front; open issues are the source of truth
for what is in progress. `docs/implementation.md` has the step-by-step guide
for each phase. Terms follow `CONTEXT.md`; decisions live in `docs/adr/`.

## Phase 0: Foundation (done)

Verified 2026-07-16.

Goal: Prepare the codebase for serious backend development.

Deliverables:

- Backend app container
- Config validation
- Logger and request ID middleware
- Database connection and pool settings
- Migration tracking
- Standard error/response format
- Health check endpoint
- Basic CI

## Phase 1: Identity and Access (done)

Shipped in PR #6, verified in production 2026-08-30.

Goal: Build secure user identity and role access.

Deliverables:

- User table
- Student identities with unique USN
- Profiles
- Password hashing
- Login and refresh token flow
- Account activation token flow (cryptographically random, single-use, 7-day expiry, hashed storage)
- Gmail invite/access request flow
- Admin/HOD verification
- Scoped role assignments
- Auth middleware
- RBAC policy checks

Follow-up, spec #129 "Getting in" (passwordless, the class list decides):

- [x] Test-only sign-in for the e2e suite (#130)
- [x] Email code, Google sign-in, first sign-in and requests from people on no list (#131, #132, #133)
- [x] Sign-in screens (#135), designed on the canvas (#134)
- [x] Passwords removed (#136); the first admin is added from the command line

## Phase 2: Campus Hub (in progress)

Goal: Make LINKS useful as a daily information hub. Milestone "Phase 2: Campus Hub".

Deliverables:

- [x] Department management API (#10, v0.2.0); department pages come with #15; the admin list with HODs and counts, and renaming, for the Departments screen (#141, API only)
- [ ] Role-based dashboard endpoints (#11); the Home summary is built (#39, v0.3.0) and counts waiting Event reviews
- [x] Campus directory and search (#12): the People screen with filters and typo-tolerant search (v0.4.0)
- [x] Targeted announcements API with approval, editing and withdrawal (#13, spec #25, v0.2.0)
- [x] Home and announcement screens: design system and app shell, reading notices, composer and My announcements, approval queue (spec #38: #39–#42, v0.3.0)
- [x] Public profiles and department pages (#12): Profile, your own profile and the Department page (v0.4.0)
- [ ] Frontend for all of the above (#15), then a manual UX pass (#16)
- [x] Bulk CSV import endpoint for admin/HOD (#17, v0.4.0)

## Phase 3: Events

Goal: Support official event proposal, approval, and participation tracking.
Everything below is built, API and screens, as of v0.4.0.

Deliverables:

- Event drafts (API built)
- Student coordinator proposal flow (API built; the proposal form and My posts are built)
- HOD review (API built; events are in the Approval queue)
- Principal/admin final approval (API built; events are in the Approval queue)
- RSVP/interest tracking (API built; the Events list, event page and answering are built)
- Participant export (API built; the organiser's event page exports it)
- Event cancellation flow (API built; organisers cancel from the event page)

## Phase 4: Placement (done)

Goal: Build the placement workflow end to end.
The API below shipped in v0.4.0 and the screens (spec #100) in v0.5.0.

Deliverables:

- Placement officer dashboard (API built: the Home placement summary and applicant counts, #101)
- Placement screens (spec #100, #101 to #106): students' Jobs, the office's Placement list, form and applicant table, and Home sections are built
- Job/internship/training posts (drafts, publishing and closing API built; ADR 0024)
- Eligibility targeting (API built: the feed matches Eligibility like an Audience)
- Internal application flow (API built: apply once while open, withdraw before shortlisting)
- External application tracking (API built: "I applied" records an external Application)
- Applicant list (API built: filters by status, Department and Batch; audited)
- Shortlisting (API built)
- Status updates (API built: row-locked, audited)
- CSV export (API built: audited, formula-safe)

## Phase 5: Clubs, Alumni, and Polish

Goal: Extend the campus graph and improve usability.

Deliverables:

- Club pages
- Club interest forms
- Alumni profiles
- Alumni USN verification
- Mentorship requests
- In-app, email, and web push notifications
- Analytics dashboard
- Accessibility pass
- Mobile responsive polish

## Recommended First Build

1. Admin/HOD-managed user access
2. USN-based student records
3. Gmail invite or manual approval flow
4. Public verified profiles
5. Role-based dashboard
6. Campus directory
7. Targeted announcements
8. Event proposal and approval
9. Placement applications and shortlisting
10. CSV export
