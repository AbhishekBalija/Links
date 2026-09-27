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

## Phase 2: Campus Hub (in progress)

Goal: Make LINKS useful as a daily information hub. Milestone "Phase 2: Campus Hub".

Deliverables:

- [x] Department management API (#10, v0.2.0); department pages come with #15
- [ ] Role-based dashboard endpoints (#11); the Home summary is built (#39, v0.3.0)
- [ ] Campus directory and search basics (#12); the directory API with filters and typo-tolerant search is built
- [x] Targeted announcements API with approval, editing and withdrawal (#13, spec #25, v0.2.0)
- [x] Home and announcement screens: design system and app shell, reading notices, composer and My announcements, approval queue (spec #38: #39–#42, v0.3.0)
- [ ] Public profiles in the directory and department pages (#12, #15); the Department overview API is built
- [ ] Frontend for all of the above (#15), then a manual UX pass (#16)
- [ ] Bulk CSV import endpoint for admin/HOD (#17)

## Phase 3: Events

Goal: Support official event proposal, approval, and participation tracking.

Deliverables:

- Event drafts (API built)
- Student coordinator proposal flow (API built)
- HOD review (API built)
- Principal/admin final approval (API built)
- RSVP/interest tracking
- Participant export
- Event cancellation flow

## Phase 4: Placement

Goal: Build the placement workflow end to end.

Deliverables:

- Placement officer dashboard
- Job/internship/training posts
- Eligibility targeting
- Internal application flow
- External application tracking
- Applicant list
- Shortlisting
- Status updates
- CSV export

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
