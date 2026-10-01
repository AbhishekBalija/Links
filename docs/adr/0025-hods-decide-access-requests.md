# HODs decide Access requests for their own Department

Access requests used to be decided only by the principal and admins (`manage_users_and_roles`), while `CONTEXT.md` said an HOD or an admin. The owner decided (spec #121) that an HOD decides Access requests for their own Department, and the principal and admins for any. The HOD knows their students, and spreading the work across Departments gets students in faster than one central queue. A new permission, `approve_access` (HOD, principal, admin), covers the review queue, approving and rejecting. The service checks the Department, the same way the bulk import does (#17): an HOD's queue holds only requests whose USN Department is theirs, and a request outside it answers `404`, as if it didn't exist.

## Considered options

- **Principal and admins only** (as built): rejected. One queue at the top is slow during admissions, and the people deciding don't know the students.
- **Any HOD decides any request:** rejected. An HOD has no basis to judge another Department's students.

## Consequences

- Rejecting is part of deciding, so an HOD may set a pending, unapproved user to `rejected` through `PATCH /admin/users/:id/status`. Suspending, reactivating and rejecting anyone else stay with the principal and admins (`manage_users_and_roles`).
- A request whose USN Department has no HOD waits for the principal or an admin, who see every request.
