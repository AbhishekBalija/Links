# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Targeted announcements, first slice (#27): HODs (own Department), the
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
- API tests run the real router against an isolated Postgres schema (#26).
- Department management (#10): any signed-in user can list and read
  departments, and admins can create, update and delete them. Codes can't
  change, a department still in use can't be deleted, and every change is
  audited. Migration 011 seeds the pilot college's current B.E. departments (CS, AD,
  AI, CV, EC, ME).

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

[Unreleased]: https://github.com/AbhishekBalija/Links/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/AbhishekBalija/Links/releases/tag/v0.1.0
