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
A person's request to join LINKS with their Gmail, USN and Department, waiting for Access approval or rejection by an HOD or admin.
_Avoid_: signup, registration

**Access approval**:
An HOD or admin accepting an Access request: the user becomes verified, gets the student role, and is sent an Activation link.
_Avoid_: verification (the `is_verified` flag is its result, not the step), plain "approval"

**Activation**:
The user setting their password through a single-use emailed link, which moves the account from `pending` to `active`.

**Account status**:
Where an account is in its lifecycle: `pending`, `active`, `rejected` or `suspended`. Only `active` accounts can use protected features.

**Role assignment**:
A Role granted to a user with a Scope. A user can hold several.
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
A Student who can propose events and post limited announcements within their Scope.

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
