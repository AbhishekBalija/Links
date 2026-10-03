# Ending a role withdraws or hands over the person's unfinished work

Ending a Role assignment used to keep its history and sign the person out, and nothing else: an Announcement or Event proposal they had sent for approval could still be approved and published afterwards, because approval checks only the approver. The owner decided (#143) that when a role ends and the person can no longer author that kind of work for that Department under any other role they hold:

- What they already published stays, under their name, and becomes read-only for them.
- Announcements waiting for approval or sent back are withdrawn. Event proposals waiting for review or sent back return to private drafts. Drafts stay private and can't be submitted.
- Upcoming Events they organise move to a new Organiser, picked by the person ending the role (by default the Department's HOD). RSVPs stay.
- All of it happens in the same transaction as ending the role, and is audited there.

An Event gets an Organiser separate from its proposer, so handing an Event over doesn't rewrite who proposed it. The Organiser starts as the proposer.

The rule depends on lost authoring power, not on which role ended, so it applies to coordinators, faculty and HODs alike, and Graduation (#46) reuses it.

## Considered options

- **Hand everything to another coordinator the HOD picks:** rejected. Drafts and pending work are the person's own judgement; someone else shouldn't inherit them silently.
- **Return everything to the HOD:** rejected. It fills the HOD's lists with drafts they never wrote.
- **Change the Event's proposer instead of adding an Organiser:** rejected. The Event would claim the new person proposed it, and only the audit log would keep the truth.
- **Cancel pending Events, or add a `withdrawn` Event status:** rejected. "Cancelled" is for Events people were told about, and a new status touches every status rule; returning to draft already means "not under review".

## Consequences

- Ending a role whose holder organises upcoming Events needs a new Organiser before it can go ahead; the confirmation lists every consequence first.
- A person holding another role that still lets them author the work (for example a former HOD who is still faculty in that Department) keeps it untouched.
