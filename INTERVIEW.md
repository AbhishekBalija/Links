# LINKS Interview Notes

## Department Management

### Why use a service, repository, and handler instead of putting everything in the Gin handler?

The handler owns HTTP concerns such as JSON parsing and status codes. The
service owns validation, authorization-sensitive rules, and transaction flow.
The repository owns database queries. This keeps business rules testable without
starting a server or connecting to PostgreSQL.

### Why are department codes immutable?

VTU course codes are used when parsing a student's USN and will later be used by
department filters and audience rules. Changing a code would silently break
those references. Editable names and descriptions allow display text to evolve
without changing the stable identifier.

### Why write the department mutation and audit log in one transaction?

A transaction treats both writes as one operation. If the department update
succeeds but the audit insert fails, PostgreSQL rolls back the update. This
prevents sensitive admin changes from existing without a matching audit record.

### Why validate the HOD role before saving `hod_user_id`?

A foreign key can prove that a user exists, but it cannot prove that the user is
an HOD. The service checks for an HOD role assignment first, while the database
foreign key still protects referential integrity. The role scope must match the
same department, preventing an HOD from another department being linked by
mistake.

### Why block deletion of referenced departments instead of cascading?

Cascade deletion could erase or detach student and authorization data after one
admin request. Returning `409 CONFLICT` makes the dependency explicit and keeps
cleanup or reassignment as a deliberate separate operation.

## API Test Harness

### Why test the API against a real Postgres instead of mocking the repositories?

Much of the risk in LINKS is in SQL: which announcements match a student's
department and batch, and what row locks do when two admins act at once. A fake
repository only checks the logic you wrote into the fake. Running the real
router against real Postgres tests the handler, service and query together, the
way a user hits them.

### How do the tests avoid touching each other's data or the real database?

Each test creates its own Postgres schema, runs every migration into it, and
points the connection pool at it with `search_path`. When the test ends the
schema is dropped with `CASCADE`, even if the test failed. Tests only run when
`TEST_DATABASE_URL` is set, and that must be a throwaway database, never the
Neon dev or production branch.

### Why seed users directly in SQL but call the API for everything else?

Seeding is setup, not the thing under test. Creating an HOD through the real
access-request, approval and activation flow would make every test slow and
fragile. The behaviour being tested always goes through the public HTTP API, so
tests keep passing when the internals are refactored.
