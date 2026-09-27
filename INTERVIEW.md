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

## Placement

### Why is an Opportunity's eligibility stored as Audience rules instead of a JSON column?

The first sketch had an `eligibility jsonb` column. But "CS students of batch
2023" is exactly what an Announcement's Audience already says, and the feed
already knows how to match it. Two matchers would drift: a student could see
an Opportunity in one place and be told they're not eligible in another.
Reusing `audience_rules` with `target_type = 'opportunity'` gives one rule
set, indexed columns for the Department filter, and a foreign key that stops a
Department in use from being deleted.

### Why isn't "open" a status of an Opportunity?

An Opportunity stops taking applications when its apply-by date passes, and
nothing runs at that moment to change a status (there are no background jobs
yet, ADR 0007). Storing `open` would need one, or it would lie after the
deadline. So the only stored statuses are the ones a person sets (`draft`,
`published`, `closed`), and "open" is computed in the query: published and
`apply_by > now()`. The closed view is the reverse: closed early, or past the
deadline.

### Why does an external application get a row in the database at all?

LINKS can't see what a student does on a company's careers site. But the
placement office still wants to know who applied where, and the student wants
one list of everything they applied to. So "I applied" creates an Application
with `mode = external`: the same one-per-student rule, the same statuses the
office can update when the company replies, and the same place in reports.

### How do two placement staff avoid overwriting each other's shortlisting?

Each status change sends the status the caller saw (`from`) with the one they
want. The server locks the Application row (`SELECT ... FOR UPDATE`), and if
the status is no longer `from`, it answers `409` instead of writing. So if
one officer shortlists a student while another, looking at a stale page,
rejects them, the second one is told it changed and reloads. The lock makes
the check and the write one step; the `from` makes a stale screen visible.

### Why can any placement staff member edit any Opportunity?

Events belong to their proposer, but the placement office is a small team that
covers for each other during a recruitment drive. If only the author could
edit a draft, a job post would be stuck the day its author is away. Every
edit is audited, so who changed what is still on record.

## Member Directory

### Why share the profile privacy rules through a type instead of repeating them in the directory's SQL?

Two copies of "who may see this email" drift apart: someone fixes one and
forgets the other, and the directory starts leaking what the profile page
hides. The `profiles.Privacy` type holds the rules once, and both the profile
endpoint and the directory call it. The directory's SQL only does the part
that must happen in the database to paginate correctly (leave out hidden,
suspended or role-less members).

### Why paginate the directory by name with a (name, id) cursor?

People look for someone by name, so alphabetical order is what they expect.
Names aren't unique, so the cursor carries the user ID too, and the query asks
for rows after `(lower(full_name), id)`. Every member appears exactly once
across pages even when two share a name, and new sign-ups don't shift pages
the way an offset would.
## Role Management

### Why give each CSV row its own transaction instead of importing all or nothing?

Admins import class lists of hundreds of students, and a few rows are usually
wrong: a typo in an email, a student who already signed up. With one
transaction per row the good rows go in and the response lists exactly which
rows failed and why, so the admin fixes those and uploads just them. Problems
with the file itself (wrong header, not a CSV, too big) are still checked
first and reject the whole upload before anything is written.

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

### How does the directory search tolerate typos?

Postgres's `pg_trgm` splits text into three-letter pieces and scores how many
pieces a search shares with a name, so "asha rau" still finds "Asha Rao". GIN
trigram indexes keep the `LIKE '%ash%'` part fast. Trigrams are weak on short
words with swapped letters: "ahsa" shares almost no three-letter pieces with
"asha", and scores the same as any name starting with "a". So a one-word search
is also compared letter by letter: a name word with exactly the same letters,
in any order, counts as a match. Results are ranked by score, then by name.

### Why does the Department overview count hidden profiles but not list them?

Hiding a profile means "don't show me to people", and a count of 120 students
in Batch 2023 shows no one. Leaving hidden members out of the counts would make
the numbers wrong for no privacy gain. The lists (HOD, staff) name people, so
they go through the same rules as the directory.

### How does the profile page show roles without profiles importing the directory?

The directory package already imports profiles (for the privacy rules), so
profiles can't import the directory back: Go forbids import cycles. Instead
profiles declares the small interface it needs, `MembershipReader`, and the
directory's service happens to satisfy it. `app/server.go` passes one to the
other. This is dependency inversion: the package that uses a capability owns
its interface, and the wiring code connects them. It also keeps one copy of
the rule for "which Department does this member belong to".

## Frontend

### My posts shows announcements and events from two paged lists. How do you merge them without putting a post in the wrong place?

Both lists come newest first, a page at a time, so this is a k-way merge of
streams you can't see the end of. After loading a page from each, you can't
just sort everything loaded: if the events list has more pages, its next page
could hold an event newer than the oldest announcement you already have. So
the page only shows items at or after a cutoff: the newest "last loaded item"
among the lists that still have more pages. Everything older waits until
"Show older" loads further. `mergeNewestFirst` in `features/posts/merge.ts`
does this, and its unit tests check the held-back case.

## Web Security

### How does LINKS stop CSRF when the refresh token is a cookie?

Only two endpoints read the cookie, refresh and logout. Both check the
`Origin` header (or `Referer` when there's no Origin) and refuse anything that
isn't the web app's own origin. A forged form on another site can make the
browser send the cookie, but it can't fake the Origin. `SameSite=Lax` already
stops most of these, so the Origin check is a second lock, not the only one.
Every other write needs the bearer access token, which another site can't read.

### Why ship the Content-Security-Policy as report-only first?

A CSP that's too strict breaks pages silently for real users: an avatar from
another host, a Sentry region you didn't list. Report-only lets browsers tell
you what they would have blocked while nothing breaks. After a week with no
reports from real use, the same policy is switched to enforcing.

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
