# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- An address LINKS doesn't have now says "This page isn't here" with a way
  back, instead of quietly landing on Home. On phones, log out moves from an
  unlabelled icon on Home to the end of your Profile, named with your email,
  and asks first (#212).
- "Too many codes" now says which limit was hit (`details.limit`: `network`
  or `email`), and the daily cap on codes to emails on no list is a setting,
  `NOT_ON_LIST_CODES_PER_DAY` (default 50), for orientation day (#202).
- Signing in with an email on no list now asks "I'm a student here" or "I
  work here" first (#200). Staff are told who adds them and get their email
  to send, instead of a USN form; staff first sign-in no longer mentions a
  class list.
- Only admins and HODs add people (ADR 0029): the principal can no longer
  import students or add staff. HODs can now add faculty to their own
  Department, and an HOD who is also the principal imports only their own
  Department's students.
- Ending someone's role now handles their unfinished work (ADR 0028, #143):
  announcements waiting for approval or sent back are withdrawn, event
  proposals under review return to drafts, and upcoming events they organise
  move to the Department's HOD or someone picked. A preview endpoint lists
  this before anything happens.
- HODs appoint and remove Student coordinators for their own Department
  (ADR 0027, #143). The principal now grants and ends faculty, HOD and
  placement officer roles only; granting the principal or Student coordinator
  role moves to admins (and HODs for coordinators).

### Removed
- Passwords and activation (#136): password login, the password Request
  access form, Activation emails and links, and Resend activation are gone,
  with their API routes. Everyone signs in with Google or an email code.
  Migration 023 drops `users.password_hash` and `account_activation_tokens`.

### Fixed
- A link opened while signed out (a job, event or notice shared on
  WhatsApp) now lands there after signing in, even through the first
  sign-in step, which a reload no longer loses (#212). A session that ends
  on its own says so on the sign-in screen, and a pasted "Your code:
  123456" fills the code boxes.
- A campus on one Wi-Fi could ask for only 60 sign-in codes every 15
  minutes, so a class signing in together on day one got stuck. The default
  is now 600 and a setting, `CODES_PER_NETWORK_PER_15_MIN`; the deployment
  guide has a checklist for a college's first day (email quota first).
- The code screen promised a code that sometimes never comes (#202). After
  the minute's wait, "No code yet?" lists every reason one might not, the
  same for every email; "too many codes from this network" (a campus on one
  Wi-Fi) is now a warning that points to mobile data, not "this email".
- Dates and times followed the device's time zone (#211). Event times, job
  deadlines, notice expiry, date tiles and Home's date are now always the
  college's, so a phone set to another zone can't create an event at the
  wrong time. A role starting later reads "from 1 Nov 2026", not a raw date.
- Someone on the same campus Wi-Fi could still stop a student getting
  sign-in codes. A browser that has signed in before is now remembered for
  that account (an httpOnly cookie; migration 026 adds `known_devices`), and
  its code requests count only against its own allowance.
- Student coordinators could post any category to any audience, the whole
  college included (#209). They now post department notices to their own
  department's students, or one batch of them; the composer offers only
  that.
- Someone else asking for a member's sign-in codes, or guessing wrong, no
  longer stops the member getting codes on their own phone (#178): the limits
  count per email and address, with a ceiling of 30 codes a day per email.
- Your own published announcement didn't show in your Notices when it was
  sent only to others (an HOD posting to their students); it now does, marked
  "You". Events gets a "Propose an event" button, as Notices has "New
  announcement".
- A role given an end date ended on that date with none of the person's
  drafts or events handed over (#204). Roles no longer take an end date; they
  end by hand, which hands work over.
- After a role ended and its event was handed to a new Organiser, the new
  Organiser had no tools on the event and couldn't find it in My posts, and
  the page still named the proposer (#205). The event page now follows the
  Organiser, and My posts lists events you organise.
- The principal or an admin could press "Not you?" on their first sign-in
  and lock the college out, since only an admin could review it (#201). They
  no longer can, and the button is hidden for them. A reported staff row,
  once approved, gets its staff role back instead of a student role, and the
  review says it is a staff invite.
- Waiting times said "waiting 1 hour" for something sent seconds ago, and
  rounded 90 minutes up to 2 hours. They now count minutes under an hour
  and whole hours or days after that.
- Rejecting an Access request burned its email and USN for good, so a
  corrected class list could never add the person (#203). Rejecting now
  removes the request (audited), and only requests can be rejected; members
  are suspended. Reactivating a suspended account no longer skips approval or
  a first sign-in.
- Home said "Good morning" after midnight; it now says good evening until
  5 AM, and a Home left open updates its greeting and date on its own.
- A current student could be given a staff role (faculty, HOD, placement
  officer, principal). That is now refused; a graduate can still join the
  staff.
- Server errors now reach the logs and Sentry: a 500's cause is logged with
  the request ID and reported, and Sentry covers every route (it was added
  after the routes, so it covered none).
- After a role ends and an event is handed to a new Organiser (ADR 0028),
  the former proposer no longer sees who answered or exports the
  participant list (names, emails, USNs); both follow the Organiser.
- Suspending is guarded (#177): nobody suspends themselves, only an admin
  suspends an admin or the principal, and the last active admin can't be
  suspended.
- Suspending or rejecting an account takes effect at once (#175): its
  access tokens stop working on the next request instead of up to 15
  minutes later.
- Profiles of accounts that haven't signed in yet, or are suspended or
  rejected, are no longer readable by others, including anonymous visitors
  (#176). Admins and the principal still see them.
- Importing a class list no longer fails a row when two students share a
  name (#166). Usernames get a random suffix instead of the clock's, and a
  username clash is retried in a savepoint instead of aborting the row. The
  same applies to staff invites and approved Access requests.
- Updating a Department with a different `code` in the body changed its name
  and quietly kept the old code; it is now refused, since codes never change
  (ADR 0021). A new Department's code must be two letters, as in a USN.
- Approving a class list row reported with "Not you?" no longer gives the
  student a second student role.
- The "Continue with Google" button was cut off on the sign-in screen.

### Added
- A student made a coordinator is welcomed the first time they open LINKS
  after: who made them one, what they can now post and propose, and that
  approval comes first. Shown once per account, on any device (a new
  `welcomed_at` column on role assignments, migration 025).
- Faculty have their own Home (#142): what was sent back to them with the
  reviewer's note, what waits on a reviewer and for how long, and New for an
  announcement or an event. Student coordinators keep the student Home and
  get New, plus their sent back and waiting posts when they have any.
- Authors are emailed when a reviewer decides: their announcement or event is
  published, or sent back with the reviewer's note, or not approved. An HOD
  approving an event on its way to the principal sends nothing.
- Email can go through any SMTP server (`MAIL_PROVIDER=smtp`): a testing
  inbox such as Mailtrap for a test copy, or Gmail for a small pilot before
  the college has a verified domain. Production stays on Resend.
- Notification emails, first slice (#206): someone whose Access request is
  decided is told (with the reviewer's note when it isn't approved);
  everyone going to or interested in an event is told when it is cancelled
  or its date, time or place changes; an applicant is told when they are
  shortlisted, selected or not taken further. The request-sent screen says an
  email will come, and the reject note says the person gets it.
- Not signed in screen (#127, #174): admins (Admin, Not signed in) and HODs
  (from Home, "See who") see who hasn't signed in yet, filter it, copy the
  emails for a reminder, and fix a row's email or remove the row from its
  "…" menu. A reported row's email can be fixed in Access requests.
- Fixing a wrong class-list row, API (#174): `PATCH /admin/users/:id/email`
  corrects the email of a row nobody has signed into (or one reported with
  "Not you?"), and `DELETE /admin/users/:id` removes such a row so its USN
  and email can be added again. Admins anywhere, HODs for their department.
- Add staff screens (#189): admins add the principal, HODs, faculty,
  placement officers and admins from Admin, Add staff; HODs add faculty from
  their Home. Home's "No HOD" rows open the form with the department chosen,
  and after each HOD the next department without one is picked.
- Adding staff (#189) emails the person who added them, as what, and how to
  sign in (Google only for the principal and admins), and a refused email or
  department says why: a current student, a student request, a member
  (with their username) or an HOD already there.
- Not signed in list API (#127): `GET /api/v1/admin/users/not-signed-in`
  with department and student/staff filters, paging and who added each
  person, and `/not-signed-in/emails` for copying every matching email.
- Import students screen (#126): admins import a class list for any
  department from the Admin workspace, HODs for their own at `/import`.
  Every row is checked first (nothing saved), grouped by the department and
  batch in each USN, then imported; rows that weren't added can be
  downloaded, fixed and uploaded again.
- Homes for the HOD, the principal and admins (#127). An HOD sees what waits
  for them and their Department (students by batch, who hasn't signed in,
  what's coming up, links to the Department page and Import students). The
  principal sees the college: every Department with its students, staff and
  HOD (or "Assign HOD"), and what waits for their approval. An admin sees a
  to-do list (access requests, departments with no HOD, who hasn't signed
  in, the approval queue) and recent imports.
- The student import can check a file before saving it (#126): send
  `dry_run=true` to see every row's result and the rows grouped by the
  Department and Batch read from each USN, with nothing saved. An optional
  `department` field ties the import to one Department and flags rows
  outside it; an HOD's import is held to their own Department.
- Departments API for the admin screen (#141): `GET /admin/departments` lists
  each Department with its HOD (or none) and student and staff counts, and
  `PATCH /admin/departments/:code` renames one without touching anything
  else.
- Roles on a profile (#125): the principal and admins see a person's roles
  (active, scheduled, ended), grant a role with dates and end or cancel one.
  An HOD makes their own students student coordinators and ends that role.
  Ending a role first shows what happens to the person's unfinished work and
  who takes over their upcoming events.
- Events have an Organiser who runs them (edits once published, cancels,
  exports participants), starting as the proposer (ADR 0028, #143).
  Migration 024 adds `events.organiser_id`.
- Home data for HODs, the principal and admins (#127): `GET /api/v1/dashboard`
  gets `lists` (who the class lists and staff invites let in who hasn't
  signed in yet, with the list, and the latest five imports with who ran them
  and how many students each created per Department and Batch) and
  `college.departments_without_hod`. HODs see their own Departments only.
  Imports now record their Department and Batch counts in the audit log.
- Home for authors (#142): `GET /api/v1/dashboard` gets `my_work` with the
  caller's Announcements and Events a reviewer sent back (with the note and who
  sent it) and those waiting on a reviewer (who, and since when). Faculty and
  student coordinators get the same section.
- Access request screens (#124): HODs decide their Department's requests in
  a new tab of the Approval queue, and admins in a new Admin workspace. Each
  request says why it's there (not on a class list, a Department with no HOD,
  or a row reported with "Not you?"); approving asks once, rejecting needs a
  note, and a request someone else decided first says so.

- Add a college's first admin from the command line: `go run ./cmd/add-admin
  -email ... -name ...` (ADR 0026). They sign in with Google; the command
  refuses once the college has an admin.
- New sign-in screens (#135): Continue with Google or get a 6-digit code by
  email, with no password. Someone on no class list sends a request to their
  HOD with just their USN, read back as Department and Batch. The first
  sign-in shows who you signed in as, with "Not you?" to report a wrong row.
  They replace Log in, Request access, Account pending and Activate; old
  links lead to the new sign-in screen.
- The class list and staff invites wait for a first sign-in (#133): an
  imported row gets no Activation email; signing in with its email, by Google
  or a code, makes it active and shows who it is, with "Not you?" to report a
  wrong row. Admins and the principal add staff by email and role. Someone
  on no list proves their email and sends an Access request with just their
  USN, no password.
- Sign in with Google (#132): the API verifies a Google ID token (audience,
  issuer, expiry, verified email and a nonce against login CSRF), matches the
  member by email the first time and by Google account from then on, and
  answers an email on no list with `NOT_ON_LIST` so the screen can offer an
  Access request. Set `GOOGLE_CLIENT_ID` to turn it on.
- Sign in with a one-time email code (#131): the API emails a 6-digit code
  that works once, for 10 minutes, in the browser that asked for it, with
  limits on tries and requests. The reply is the same whether or not the
  email is known. The principal and admins can't use codes. The screens
  come later (#135).
- HODs decide Access requests for their own Department (#122, ADR 0025):
  their review queue holds only their Department's requests, and they approve
  or reject them; the principal and admins still see every request. Home's
  data gains an HOD's Department (students by batch, staff, upcoming events),
  the college for the principal and admins (each Department with its HOD),
  and the Access requests waiting.

### Fixed
- The Access request queue no longer lists students who were already
  approved but haven't activated yet.

## [0.5.1] - 2026-10-01

Two fixes: cancelled requests no longer show up as server errors, and a
saved profile appears in People straight away.

### Fixed
- A request the browser cancels (for example by navigating away) is no
  longer answered and logged as a server error (#115). It is logged as 499,
  "client closed request", and the cut-short database query is not printed
  as an error, so real server errors stand out.
- After saving your profile, People and your Department page show the new
  headline straight away instead of up to 30 seconds later (#109).

## [0.5.0] - 2026-10-01

Placement works end to end: the placement office posts jobs, internships
and training, students find and apply to the ones open to them, and the
office shortlists and exports applicants, with both sides on Home.

### Fixed
- Home and the Approval queue badge count waiting event proposals too
  (#108). An HOD or the principal with only proposals waiting was told
  "Nothing is waiting for you"; Home now lists the oldest waiting items of
  either kind, each opening in the queue.

### Added
- Placement on Home (#106): students see the next jobs open to them, with
  their deadlines; placement staff see the open drives with their
  applicants, how many applications wait for review, and a shortcut to the
  drive with the most waiting. The principal sees approvals first, then
  placement.
- The applicant list for placement staff (#105): every applicant to an
  Opportunity with their USN, department, batch and date, by status (To
  review, Shortlisted, Selected, Rejected, Withdrawn) with counts, searchable
  by name, USN or email and filtered by department and batch. Each row's
  status changes in place with an Undo; if someone else changed it first, the
  row reloads and says so instead of overwriting. Export CSV downloads
  everyone or one status. The Opportunity page opens the list and exports
  too.
- Placement for the placement office (#104): every Opportunity under Open,
  Drafts and Closed with its applicant counts; a form for a job, internship
  or training that says who will see it ("Shows in Jobs for AD and CS
  students, batch 2023"), keeps drafts and asks before leaving with changes;
  and each Opportunity's page, where a draft is published and an open one is
  closed early, each after a confirmation. Who can apply now reads as one
  sentence on students' job pages too.
- Jobs for students (#103): a Jobs tab with the jobs, internships and
  training open to them, soonest deadline first, plus what they applied to
  and closed ones. Each opportunity's page applies in LINKS after one
  confirmation, or opens the company's site and asks on return whether they
  applied, so the placement office still counts it. Their status (applied,
  shortlisted, selected, not selected, withdrawn) shows on the page and in
  the list, and they can withdraw while it is still only applied. On phones
  Jobs takes People's tab; People stays in the desktop sidebar.
- Placement staff see each Opportunity's applicant counts by status on
  their list and on the Opportunity, and search its applicants by name,
  email or USN (#101). Home gains the next open Opportunities for members
  who are eligible, and a placement summary (open drives, applicant counts,
  Applications waiting for review) for placement staff.

## [0.4.0] - 2026-10-01

Events arrive end to end (proposing, review, answering and organiser
tools), the campus directory and profiles get their screens, and the
placement workflow is built on the API side.

### Fixed
- Your own notices say "You" again, and signing in as someone else on the
  same tab clears the previous person's cached data. The client read the
  signed-in user's id from a field `/me` never sends.
- The API failed to start on Vercel (every request returned 500): on each
  cold start it checked every migration in its own transaction, and with
  the database in another region those round trips passed the startup
  deadline. Startup now reads the applied migrations in one query and only
  locks and applies the missing ones.

### Added
- Event proposals in the Approval queue, oldest first alongside
  announcements. An HOD approves or sends back; at final approval the
  principal sees who approved it first and confirms before publishing.
  Sending back asks for changes or rejects, always with a note.
- Organiser tools on a published event, for its proposer, its Department's
  HOD, the principal and admins: who's coming (counts, the people who
  answered, and a full list by answer), a CSV export, Edit details for the
  place, time, seats and description (a limit can't drop below those
  going), and Cancel event with a reason everyone invited sees.
- Proposing events from the app: a proposal form (kind, title, when, where,
  who's invited and a seat limit) that says who reviews it before you send
  it, keeps drafts, and asks before you leave with changes. My announcements
  becomes **My posts**: announcements and events together under Needs you,
  Drafts, Waiting, Live and Ended, with one New button. Each proposal has its
  own page with its history and the actions its status allows: edit and
  resubmit, delete a draft, or cancel with a reason.
- Placement staff download an Opportunity's applicants as CSV, all or one
  status only, with spreadsheet formulas neutralised and no phone numbers.
  Every download is audited.
- Placement staff see each Opportunity's applicants (name, email, USN,
  Department, Batch, never phone numbers), filtered by status, Department
  and Batch, and move Applications between applied, shortlisted, rejected and
  selected. A change based on a status someone else already changed is
  refused, and opening the list and every change are audited.
- Students apply to an open internal Opportunity they are eligible for, once,
  and withdraw while it is still only applied; for an external one they mark
  that they applied on the company's site so it is tracked. Each Opportunity
  shows the student their own application only, and an "applied" view lists
  everything they applied to.
- Placement staff publish Opportunities and close them early; members see
  the open ones they are eligible for, soonest deadline first, and the closed
  ones apart, filtered by type and Department. A published Opportunity keeps
  its application mode and its deadline ahead.
- Placement Opportunities (Phase 4, ADR 0024): the placement officer, the
  principal and admins save jobs, internships and training posts as drafts,
  with company, role, description, location, stipend or CTC, apply-by date,
  an internal application or an external link, and Eligibility by
  Department, Batch and role. Any of them can edit any draft, and they list
  every Opportunity by status.
- Members can make their profile private or public again
  (`public_profile_enabled` on `PATCH /api/v1/me/profile`, #81). A private
  member leaves People and their profile opens only for them; department
  counts still include them. Each change is audited.
- Home's approvals summary also counts the Event proposals waiting for the
  reviewer (`events_pending_count` and `oldest_event_submitted_at`), apart from
  Announcements, with the same scope as the event review queue.
- Events screens: an Events tab listing what's coming up for you by week
  (Upcoming, Going and Past, and by type), each with your answer and the
  seats left; an event page with one Going / Interested / Can't go control
  (in a bar at the bottom on phones) that respects the seat limit and closes
  when the event starts; the cancellation reason on a cancelled event; and
  "Coming up" on Home. Phones keep five tabs, so an HOD's People tab moves
  to Home's department link.
- Proposers can delete their own event drafts. Anything already submitted is
  cancelled instead, so its history stays.
- The event feed carries each Event's answer counts and your own answer,
  lists upcoming, going or past Events (`show`), and keeps a cancelled Event
  in view, until it ends, for the people who answered it.
- Event proposals (Phase 3): staff and student coordinators save Event drafts
  for their Department (the principal and admins for any or none, the
  placement officer training only), with type, place, times, capacity,
  faculty mentor and Audience, edit them, and list their own. Submitted
  events go to their Department's HOD (or the principal or an admin when there
  is none) and then to the principal or an admin for final approval; reviewers
  approve, ask for changes or reject with a note, and see earlier notes. HODs'
  own events and training events skip the HOD stage; the principal's and
  admins' events publish straight away. Members see upcoming published
  events for their Audience, filtered by date, Department and type, and open
  one; review notes stay with the proposer and reviewers. Members answer
  going, interested or not going; going stops at the event's capacity, and
  organisers see who answered and can export them as CSV (each export is
  audited). After publishing, organisers can change the description, place,
  times and capacity (each change audited), or cancel with a reason; answers
  are kept.
- People, profile and department screens (#12). People opens on your own
  Department, A to Z under letter headings, with search that forgives typos
  and filters for Department, role and Batch (selects on desktop, chips and
  one filter sheet on phones). Opening someone shows who they are, what they
  wrote, the contact details they share (with a copy button) and their
  Department. Profile shows your own page as others see it, with Edit
  profile, which now returns there after saving. A Department page lists
  its HOD, faculty and student coordinators with student numbers by Batch.
  Home's department code links to it. Phones keep at most five tabs, so an
  HOD's Profile moves to the avatar on Home.
- The directory says how many members match (`meta.total`), and a signed-in
  member opening a profile sees that member's roles, Department and Batch.
  Anonymous visitors still see only the profile itself.
- Member directory (#12): signed-in members can list active members
  alphabetically, filtered by Department, role and Batch, with roles,
  Department, Batch for students, and contact details only where the member
  opted in. Hidden, suspended, pending and former members never appear.
  Searching by name, username or headline tolerates typos and ranks the
  best matches first. Each Department has an overview with its HOD, staff,
  and student counts per Batch.
- Role management (#22): the principal and admins can list, grant and end a
  user's staff roles, with Department scopes, start and end dates, one HOD per
  Department, and an audit log. Only an admin can grant or end the admin role,
  and the last admin can't be removed.
- Bulk student import (#17): admins, the principal and HODs (for their own
  Department) upload a CSV of up to 200 students with email, full name and
  USN. Each valid row becomes a verified student with an activation email,
  sent in batches once every row is saved, and the response reports every
  row's result.

### Changed
- Department codes come from the database (#18): a Department an admin adds
  works for access requests and appears in the sign-up form without a code
  change. A new public endpoint lists Department codes and names. A USN whose
  Department differs from the chosen one is now rejected.

### Security
- Refresh and logout refuse requests from other sites (Origin or Referer
  check), every API response carries security headers, and the web app
  sends a report-only Content-Security-Policy whose violations are logged
  by the API (#19). The API refuses to start with an insecure cookie setup.
- Ending a role, suspending or rejecting a user signs them out everywhere by
  revoking their refresh tokens (#22).

## [0.3.0] - 2026-09-27

LINKS gets its look and its screens: everyone can read notices, staff can
write them, and approvers can review them, on desktop and phone.

### Added
- Home summary endpoint (#39): the signed-in user's latest notices, plus
  what's waiting for approvers and the state of an author's own
  announcements. Announcements gain a detail endpoint, a category filter, a
  publishing preview and approver names on pending items.
- The LINKS look and app shell (#40): the Gazette design tokens and fonts, a
  sidebar on desktop and bottom navigation on phones, a Home page from the
  new summary, and a Notices feed that loads more as you scroll, filters by
  category and opens each notice. Every screen has loading, empty and error
  states.
- Writing announcements (#41): a composer with one-tap audience picks (or
  custom groups by department, batch and role) and how many people they
  reach, a preview, an optional expiry, and a line saying whether it
  publishes now or which approver it goes to. My announcements puts
  anything sent back first, with the note, and tabs for drafts, waiting,
  live and ended; opening one gives Edit and Withdraw (with a
  confirmation). Leaving the composer with unsaved text offers to keep it
  as a draft.
- The approval queue (#42): HODs, the principal and admins review what's
  waiting for them, oldest first, with the full text, who it goes to and
  how many people it reaches. Approving asks once more (Enter confirms);
  sending back needs a note. Edits to live notices can be compared with the
  live version, items that expired while waiting can only be sent back, and
  if someone else acts first the queue says so and refreshes. Home lists the
  oldest waiting items, and the sidebar shows the count.

### Fixed
- Students who signed up through the access request form had no Batch, so
  announcements for a batch (e.g. "CS students, batch 2023") never reached
  them (#45). The Batch now comes from the USN, and existing students are
  fixed by migration 015.

### Changed
- The first visit downloads about 42% less (743 kB to 409 kB): each page,
  and Sentry's session replay, load only when needed (#52).
- Only Vercel production builds upload source maps to Sentry; local builds
  no longer do (#51).
- Strict TypeScript, gofmt checked in CI, Vitest unit tests for the client's
  logic (#53) and unit tests for sign-in, activation and profile privacy
  (#8). ADR 0020 records that migrations only go forward.
- The composer and the auth service are split into smaller files (#54,
  #56), with no change in behaviour.

## [0.2.0] - 2026-09-26

Phase 2 begins: department management and targeted announcements with
approval, so staff can reach exactly the students a notice concerns.

### Added
- Targeted announcements (#27): HODs (own Department), the
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
- Editing and withdrawing announcements (#29): authors with publishing
  authority edit published announcements directly; other authors' edits wait
  for approval while readers keep seeing the approved text. Authors and
  approvers can withdraw a published announcement.
- API tests run the real router against an isolated Postgres schema (#26).
- Department management (#10): any signed-in user can list and read
  departments, and admins can create, update and delete them. Codes can't
  change, a department still in use can't be deleted, and every change is
  audited. Migration 011 seeds the pilot college's current B.E.
  departments (CS, AD, AI, CV, EC, ME).

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

[Unreleased]: https://github.com/AbhishekBalija/Links/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/AbhishekBalija/Links/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/AbhishekBalija/Links/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/AbhishekBalija/Links/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/AbhishekBalija/Links/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/AbhishekBalija/Links/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/AbhishekBalija/Links/releases/tag/v0.1.0
