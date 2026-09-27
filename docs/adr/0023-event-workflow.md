# Event workflow: who proposes, who reviews, and one status column

ADR 0006 says student coordinator events need HOD review and then principal or admin final approval. This fills in the rest of the Event workflow so it can be built (Phase 3).

## Decision

**Who proposes** (`propose_event`):
- Student coordinators and faculty: for their own Department only.
- An HOD: for their own Department.
- The principal and admins: for any Department, or none (college-wide).
- The placement officer: `training` events only, for any Department or none.

**Event fields:** title, description, `event_type` (`talk`, `workshop`, `competition`, `cultural`, `sports`, `training`, `other`), Department (none means college-wide), an optional faculty mentor (someone with a faculty role in effect), location, `starts_at`, `ends_at` (after `starts_at`, and in the future when submitted), an optional positive capacity, and an Audience in `audience_rules` with `target_type = 'event'` (empty means the whole college).

**Status:** one `status` column, not the `status` plus `approval_status` pair sketched earlier in `docs/database-design.md`. One column makes every state explicit and checkable with a CHECK constraint, and the two columns could disagree. The statuses are `draft`, `submitted`, `hod_changes_requested`, `hod_rejected`, `hod_approved`, `final_changes_requested`, `final_rejected`, `published` and `cancelled`.

**Workflow:**
- Student coordinator or faculty event: `submitted`, then HOD review by the HOD of the event's Department, then `hod_approved`, then final approval by the principal or an admin, then `published`.
- An HOD's own event skips its own review: submitting makes it `hod_approved`, waiting for final approval.
- A principal or admin event publishes when submitted.
- A placement officer's training event goes straight to final approval (`hod_approved`), with no Department HOD stage.
- A Department with no HOD: the principal or an admin does the HOD stage.
- Reviewers approve, request changes (note required) or reject (note required). After changes are requested the proposer edits and resubmits, and the Event returns to the stage that asked (HOD or final). Every decision and note is kept in `event_reviews`, so reviewers see earlier notes.
- Nobody reviews their own proposal. Every transition locks the Event row (`FOR UPDATE`); a decision on an Event that has already moved on is `409`.
- A published Event: the proposer or any approver (the Department's HOD, the principal, an admin) may edit only its logistics (description, location, times, capacity), and each edit is audited. Other changes need a cancel and a new proposal.
- Cancelling needs a reason and is done by the proposer or an approver. RSVPs are kept, so people can see what they signed up for was cancelled.

**RSVP and export:** readers in the Audience answer `going`, `interested` or `not_going` on a published Event that hasn't started. `going` beyond capacity is `409`. Everyone who can see the Event sees the counts. The proposer, the Department's HOD, the principal and admins see who answered and can export them as CSV (name, email, USN and Batch for students, Department, answer, time). Every export is audited.

## Considered options

- **`status` plus `approval_status`:** rejected as above.
- **Let the principal or an admin do the HOD stage for any Department:** rejected. The HOD answers for their Department (ADR 0017's reasoning); the principal steps in only when there is no HOD.

## Consequences

- Review queues are queries on `status`: `submitted` for HODs (and principal or admin for Departments without an HOD), `hod_approved` for the final stage.
- A Department with Events can't be deleted (they reference it).
