# LINKS

The official campus hub for a college: verified identities, targeted announcements, event approvals and placement workflows. Not a chat platform or social network.

## Identity and access

**USN**:
The VTU University Seat Number, `4MN<joining year><department code><roll>` (e.g. `4MN22CS001`), the primary identity key for students and alumni.
_Avoid_: roll number, student ID

**Student identity**:
The record that ties a user to their USN, Department, Batch and Academic status.

**Batch**:
The batch a student currently belongs to, named by its joining year (`2023`). It starts as the joining year in the USN; an admin can change it for a year-back student. Audiences target Batch, not the USN.
_Avoid_: year (ambiguous with year of study), USN year

**Academic status**:
Where a student stands: `studying`, `completed_with_backlogs` (course finished, degree not yet), `graduated` or `left` (dropped out or transferred) (ADR 0019).

**Graduation**:
The yearly step where an admin or HOD confirms which final-year students have their degree: their student Role assignment ends and they become Alumni. Never automatic from the calendar (ADR 0019).
_Avoid_: pass-out (a student can finish the course without graduating)

**Access request**:
A request to join LINKS from someone whose email is on no list: they prove their email (Google or a Sign-in code) and give their USN, which names their Department. It waits for Access approval or rejection by their Department's HOD, the principal or an admin.
_Avoid_: signup, registration

**Access approval**:
Their Department's HOD, the principal or an admin accepting an Access request: the user becomes verified, gets the student role, and waits for their First sign-in.
_Avoid_: verification (the `is_verified` flag is its result, not the step), plain "approval"

**First sign-in**:
The first time someone signs into an account waiting for it (a class list row, a staff invite or an approved Access request), by Google or a Sign-in code with its email. It makes the account `active` and shows who they signed in as, with "Not you?" to report a wrong row.
_Avoid_: activation, onboarding

**Staff invite**:
An admin adding a staff member by email and role, or an HOD adding faculty to their own Department. LINKS emails them that they were added and how to sign in; the account waits for its First sign-in.

**Sign-in code**:
A 6-digit code emailed to someone signing in without Google. It works once, for 10 minutes, in the browser that asked for it. The principal and admins can't use one.
_Avoid_: OTP, magic code, PIN

**Account status**:
Where an account is in its lifecycle: `pending`, `active`, `rejected` or `suspended`. Only `active` accounts can use protected features.

**Role assignment**:
A Role granted to a user with a Scope. A user can hold several. Ending one withdraws the person's unfinished work they can no longer author and hands their upcoming Events to a new Organiser (ADR 0028). A current Student can't hold a staff role (faculty, HOD, placement officer, principal); a graduate can.
_Avoid_: user role, user type

**Scope**:
Where a Role assignment applies: global, one Department, or one club.

**Public profile**:
A user's profile that others can view, with contact details hidden unless the user chooses to show them. Public by default; a member can make it private, visible only to them.

**Directory**:
The list of members any signed-in member can browse and filter by Department, role and Batch. It shows only active, verified members with a Public profile and a role in effect.
_Avoid_: people search, user list

## Roles

**Student**:
A current B.E. student of the college with a Student identity.

**Alumni**:
A former student whose Graduation was confirmed. Students who left without graduating are not Alumni.

**Student coordinator**:
A Student who can propose events and post Department notices to their own Department's students. Appointed and removed by their Department's HOD (ADR 0027).

**HOD**:
Head of a Department. Reviews that Department's access requests and events, and sees only summaries of its students' placement applications.
_Avoid_: head, department admin

**Placement officer**:
The staff member who posts Opportunities and manages Applications and Shortlists.

**Principal**:
College head with college-wide summaries and final Event approval.

**Admin**:
Operator of LINKS with full user, role and Department management.

## College structure

**Department**:
An academic B.E. department of the college, identified by its Department code, with at most one HOD.
_Avoid_: branch

**Department code**:
The two-letter VTU course code of a Department (`CS`, `AD`, `AI`, `CV`, `EC`, `ME`), also part of every USN in that Department. Never changes once created. The `departments` table is the only list of codes (ADR 0021).

## Campus hub

**Announcement**:
An official notice sent to an Audience.
_Avoid_: post, message

**Audience**:
The users an Announcement or Opportunity targets, described by Department, batch, year, Role and eligibility rules.

**Publishing authority**:
The right to publish an Announcement to an Audience without approval: an HOD for their own Department, the principal and admins everywhere, the Placement officer for placement announcements (ADR 0017).

**Announcement approval**:
An HOD, the principal or an admin accepting an Announcement (or an edit to one) from an author without Publishing authority over its Audience.
_Avoid_: plain "approval", moderation

**Event**:
An official campus event (talk, workshop, competition, cultural, sports, training) with a date, place, Department (or the whole college) and Audience.
_Avoid_: activity, programme

**Event proposal**:
An Event before it is published: drafted by a Student coordinator, faculty member, HOD, placement officer (training only), principal or admin, and submitted for review (ADR 0023).

**HOD review**:
The first review of a Student coordinator's or faculty member's Event proposal, by the HOD of its Department (the principal or an admin when the Department has no HOD): approve, request changes or reject.
_Avoid_: department approval

**Organiser**:
The person who runs an Event: edits it, cancels it and exports its participants. Starts as its proposer and moves to someone else when their role ends.
_Avoid_: owner, host

**Final approval**:
The principal's or an admin's review of an Event proposal after HOD review (or straight away for an HOD's own Event or a training Event). Approving publishes it.

**RSVP**:
A reader's answer to a published Event: going, interested or not going. Going counts against the Event's capacity.
_Avoid_: registration, sign-up

**Opportunity**:
A job, internship or training posted by placement staff (the Placement officer, the principal or an admin) with an Eligibility. Open while published and before its apply-by date.
_Avoid_: job post, drive

**Eligibility**:
The Audience of an Opportunity: who may see it and apply, by Department, Batch and role.
_Avoid_: criteria, target

**Application**:
A Student's application to an Opportunity, made inside LINKS (internal) or their record that they applied on the company's site (external). One per Student per Opportunity; withdrawing is final. Visible only to the Student, the Placement officer, the Principal and admins.

**Shortlist**:
The Applications placement staff move forward for an Opportunity: those with the status shortlisted, before selection or rejection.
