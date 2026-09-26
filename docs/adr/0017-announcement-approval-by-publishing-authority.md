# Announcement approval follows the author's publishing authority

An announcement publishes straight away only when its author has publishing authority over its whole Audience; otherwise it waits for Announcement approval. We decide by who is posting to whom, not by announcement type, because LINKS is the official record: a notice reaching a whole department should have been posted or approved by someone who answers for that department.

## Decision

- **Publishing authority:**
  - An HOD has it for their own Department.
  - The principal and admins have it for every Audience.
  - The placement officer has it for placement announcements, the only kind they can post (`docs/auth.md`).
  - Faculty and student coordinators have none. They can post, but their announcements always need approval.
- **Who approves:**
  - When the Audience is a single Department, that Department's HOD approves.
  - When it's several Departments or the whole college, the principal or an admin approves.
  - The principal and admins can approve any announcement, which also covers Departments that have no HOD linked yet.
- **Edits:** when an author without publishing authority edits an approved announcement, the edit goes back to approval. The approved version stays visible to students until the edit is approved, so students never see unreviewed text. Authors with publishing authority edit directly.
- Club audiences are left for Phase 5, when clubs exist.

## Considered Options

- **Department-only announcements skip approval, college-wide ones need it:** simpler to explain, but any faculty member or student coordinator could post unreviewed to an entire department.
- **No approval in the MVP:** least work, but it contradicts the "Limited" posting rights for faculty and coordinators in `docs/auth.md`.

## Consequences

- An announcement needs to store a pending revision separately from its published content, so an edit can wait for approval while the approved version stays visible.
- Approval requests notify the approver: the HOD for a single Department, principals and admins otherwise (`docs/notifications.md`).
- Every approval, rejection and direct publish gets an audit log entry.
