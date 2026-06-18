# Roadmap

## Phase 0: Foundation

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

## Phase 1: Identity and Access

Goal: Build secure user identity and role access.

Deliverables:

- User table
- Student identities with unique USN
- Profiles
- Password hashing
- Login and refresh token flow
- Gmail invite/access request flow
- Admin/HOD verification
- Scoped role assignments
- Auth middleware
- RBAC policy checks

## Phase 2: Campus Hub

Goal: Make LINKS useful as a daily information hub.

Deliverables:

- Role-based dashboard endpoints
- Campus directory
- Department pages
- Public profiles
- Targeted announcements
- Smart notice board
- Search basics

## Phase 3: Events

Goal: Support official event proposal, approval, and participation tracking.

Deliverables:

- Event drafts
- Student coordinator proposal flow
- HOD review
- Principal/admin final approval
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
