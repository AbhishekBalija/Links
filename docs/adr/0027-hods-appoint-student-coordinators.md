# HODs appoint Student coordinators for their own Department

Role management used to belong to the principal and admins only (`manage_users_and_roles`), and `docs/auth.md` said HODs don't manage roles. A Student coordinator works inside one Department, and its HOD is the one who knows which students to trust with it, so the owner decided (#143) that the HOD grants and ends `student_coordinator` for students of their own Department. Each role now has a set of people who normally appoint it:

| Role | Granted and ended by |
|---|---|
| Student coordinator | Their Department's HOD, or an admin |
| Faculty, HOD, placement officer | The principal or an admin |
| Principal, admin | An admin only |

The service checks this from roles read from the database, not the token, the same way Access approval checks the Department (ADR 0025). An HOD asking about a Role assignment outside what they may manage gets `403`; one outside their Department, for a student they can't see, behaves as if it doesn't exist.

## Considered options

- **Principal and admins appoint coordinators** (as built): rejected. The principal doesn't know each Department's students, and every appointment would queue at the top.
- **The principal keeps every non-admin role:** rejected. The principal appoints senior staff; coordinators are a Department matter, and granting the principal role is an operator's job.

## Consequences

- An admin can still appoint coordinators, as the fallback; the screens keep that quiet (the "can vs normally does" principle).
- The principal can no longer grant or end `student_coordinator` or `principal`.
