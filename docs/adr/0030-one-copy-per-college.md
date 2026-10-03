# One copy of LINKS per college

LINKS serves one college per deployment: each college gets its own Vercel project, its own Neon database and its own address, and the developer adds its first admin once with `add-admin`. The owner chose this on 2026-10-04 when planning how to hand LINKS to a second college. A college's data stays entirely its own, nothing in the code has to learn which college a request belongs to, and a mistake in one copy can't show one college's students to another.

## Considered options

- **One app for many colleges** (a college on every row, people choosing their college, a developer screen that creates colleges and their first admin): rejected for now. It touches every table, query and permission rule, and is worth it only when running many copies becomes the chore. Moving later means adding a college to every row and merging databases, so this decision is costly to undo.

## Consequences

- What differs between colleges must come from settings for that copy, not from code: the USN format (today `4MN…` is fixed in `auth/usn.go`) and the Departments (today created by migrations 008 and 011).
- Each copy is deployed, migrated and monitored on its own. A release goes to every copy.
- The words in `CONTEXT.md` keep meaning "the college": there is never more than one.
