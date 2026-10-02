# Admins and HODs add people, not the principal

Importing students (`import_students`) was open to admins, the principal and HODs, and staff invites (`POST /admin/users`) to the principal and admins through `manage_users_and_roles`. The owner decided in the admin screens design round (#123) that the principal no longer imports students or adds staff: admins and HODs do, and an HOD only for their own Department. The principal's job is oversight and final approvals, not data entry, and the people who know who belongs in a Department are its HOD and the operator who sets LINKS up.

| Action | Admin | HOD | Principal |
|---|---|---|---|
| Import students | Any Department | Their own Department | No |
| Add staff (staff invite) | Any role | Faculty of their own Department | No |

A new permission, `invite_staff` (HOD, admin), covers staff invites, so `manage_users_and_roles` no longer decides them. The service reads the actor's roles from the database, as for Access approval (ADR 0025): an admin acts anywhere, an HOD only in the Departments they head, and being the principal never widens that, so a principal who is also an HOD imports only their own Department's students.

## Considered options

- **Keep the principal as a quiet fallback** (the "can vs normally does" rule from #123): rejected for these two actions. Admins already cover every Department, so a third fallback adds a path nobody's screens use.
- **HODs invite any Department-scoped role:** rejected. An HOD can't appoint another HOD, and a Student coordinator must already be a student, so faculty is the only role an HOD adds by invite.

## Consequences

- The principal gets `403` from `POST /admin/users` and `POST /admin/users/import`. Other parts of `manage_users_and_roles` (granting and ending roles, suspending accounts) are unchanged here.
- This narrows `docs/auth.md`'s earlier rule and the Staff invite definition in `CONTEXT.md`.
