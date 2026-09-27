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

### Why read Department codes from the database instead of a list in code?

A list in code and the `departments` table can disagree, and they did: an admin
could create a Department that nobody could sign up to, because the USN check
and the sign-up form each had their own hardcoded list. With the table as the
only list, the USN check just checks the shape, and the Department lookup
decides whether the code is real.

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

## Targeted Announcements

### Why is the publishing rule a pure function instead of living in the handler or SQL?

"Does this author publish directly, and if not, who approves?" depends only on
the author's roles and the Audience. As a pure function it can be tested with a
table of every role and Audience combination in milliseconds, without a server
or database. The service gathers the inputs (roles from the database) and acts
on the answer.

### Why read the author's roles from the database when the JWT already has them?

The token only carries role names, not which Department an HOD is scoped to,
and it keeps working for up to 15 minutes after a role is removed. Publishing
authority needs the scope and must stop the moment a role ends, so the service
loads the roles that are in effect right now.

### How does the feed decide who sees an Announcement?

Each Announcement has a list of Audience rules. Within one rule, every field
it sets (Department, batch year, role) must match the reader: that's AND.
Across rules, matching any one is enough: that's OR. No rules means the whole
college. This lets one notice target "CS final years or CS faculty" without a
special case. A role and its Department are matched as a pair,
not separately: a CS student who also teaches in EC must not match "CS
faculty", which is what happens if you check "is in CS" and "is faculty" on
their own.

### Why cursor pagination instead of page numbers?

With `OFFSET`, a notice published while a student is scrolling shifts every row
down, so they see one item twice or miss one. The cursor remembers the last
item seen (its published time and ID) and asks for older ones, so pages stay
stable. It's also faster, because Postgres can seek straight to that point in
the index instead of counting past skipped rows.

## Announcement Approval

### Why store content waiting for approval in a separate revisions table?

An approved Announcement that gets edited must keep showing the approved text
until the edit is approved. If the edit overwrote the Announcement row, students
would see unreviewed text immediately. Keeping the pending content in a
revision, and copying it onto the Announcement only when approved, keeps the
public version and the proposed version apart. A new Announcement's first
version goes through the same path, so there is one approval flow, not two.

### How do you stop two approvers from both acting on the same submission?

Approving, rejecting and resubmitting all start by locking the Announcement row
with `SELECT ... FOR UPDATE`. The second approver waits for the first to
commit, then sees the revision is no longer pending and gets `409 Conflict`. A
partial unique index also guarantees at most one open revision per
Announcement.

### Why return 404 instead of 403 when someone edits another person's announcement?

A 403 confirms the Announcement exists, which leaks drafts and pending notices
the caller shouldn't know about. Treating someone else's Announcement as not
found gives nothing away.

### Why does withdrawing close a pending edit instead of leaving it?

A withdrawn Announcement is off every feed. If its pending edit stayed in the
queue, an approver could approve it and bring the notice back without anyone
deciding to republish it. Closing the edit, in the same transaction as the
withdrawal, keeps the queue showing only things that can still go live.

## Events

### Why one status column for events instead of a status and an approval status?

With two columns, "published but HOD rejected" is a state the database allows
and the code must never produce. One column with nine values lists every state
that can exist, a CHECK constraint enforces it, and each workflow step is a
move from one value to another. The history of who decided what lives in a
separate `event_reviews` table, so the status doesn't have to remember it.

### How does an event sent back for changes know which stage to return to?

The status says so. `hod_changes_requested` means the HOD asked, so
resubmitting goes back to `submitted` (the HOD's queue);
`final_changes_requested` means the principal or an admin asked, so it goes to
`hod_approved` (their queue) and the HOD isn't asked again. No extra column is
needed, and a unit test pins the whole table of moves.

### How does the capacity limit hold when two people RSVP at the same moment?

Counting "going" answers and then inserting one is a check-then-act race: two
requests can both count 39 of 40 and both insert. The RSVP transaction first
locks the Event row with `SELECT ... FOR UPDATE`, so the second request waits
until the first commits, then counts 40 and gets `409 event is full`. A test
fires six requests at a one-seat event at once and checks that exactly one
gets in.

### Why prefix some CSV cells with a quote?

Spreadsheets run a cell that starts with `=`, `+`, `-` or `@` as a formula. A
member could set their name to a formula that fetches a URL with the other
cells' data when an organiser opens the export ("CSV injection"). Prefixing
such cells with `'` makes the spreadsheet show them as text.

### Why can only logistics change after an event is published?

Reviewers approved a specific event: its title, type, Department and Audience.
If those could change afterwards, an HOD-approved CS workshop could quietly
become a college-wide cultural event nobody reviewed. Room, time and capacity
changes happen all the time and don't change what was approved, so organisers
can make them (each is audited). Anything else means cancelling and proposing
again.

### How does a PATCH tell "leave this field alone" from "clear it"?

In Go, a missing JSON field and `null` both leave a pointer nil. The event
update uses a small `Optional[T]` type whose `UnmarshalJSON` runs whenever the
key is present, even for `null`, so it records "was sent" separately from the
value. Leaving `capacity` out keeps it; sending `"capacity": null` removes the
limit.
## Role Management

### Why end a role by setting `ends_at` instead of deleting the row?

The row is the history of who held which role and who granted it, which an
audit needs ("who was CS HOD when this event was approved?"). Every query that
checks roles already asks for roles in effect now (`starts_at <= now()` and
`ends_at` null or later), so an ended row simply stops counting.

### If roles are in the JWT, how does removing a role take effect?

The access token keeps its roles until it expires, at most 15 minutes. Ending
a role revokes the user's refresh tokens in the same transaction, so their next
refresh fails and they sign in again with fresh roles. The window in between
is short, and the checks that matter most (who may publish or approve) read
roles from the database, not the token.

## Testing

### Why have three kinds of tests instead of just end-to-end ones?

Each layer catches different bugs at a different cost (the "test pyramid").
Vitest unit tests check pure logic, like how an expiry date reads ("tomorrow"
vs "in 2 days") or whether a group can have a batch. They run in Node in
about 200 ms, so every edge case can have its own test. The Go API tests hit a
real Postgres in an isolated schema, which is where rules like "who may
approve this" live. Playwright runs a few whole journeys in a browser (a
faculty member submits, the HOD approves, a student sees it). Those are slow
and break more easily, so they cover the paths that matter, not every branch.
