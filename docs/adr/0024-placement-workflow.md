# Placement workflow: Opportunities, Eligibility and Applications

Phase 4 needs placement staff to post Opportunities and track Applications without leaking one Student's application to another (ADR 0008). This records the whole workflow so it can be built in small PRs, and where it departs from the sketches in `docs/database-design.md` and `docs/api-spec.md`.

## Decision

**Who posts** (`post_opportunity`): the Placement officer, the principal and admins. They work as one placement office: any of them can edit, publish or close any Opportunity, drafts included. Nobody else sees drafts.

**Opportunity fields:** `opportunity_type` (`job`, `internship`, `training`), `title` (the role), `company`, `description`, `location`, `compensation` (free text for a stipend or CTC, since offers are written many ways: "4.5 LPA", "15k per month", "unpaid"), `apply_by`, `application_mode` (`internal` or `external`) with `external_url` required for `external`, and an Eligibility.

**Eligibility is an Audience, stored in `audience_rules`** with `target_type = 'opportunity'`, not the `eligibility jsonb` column the schema sketch had. It matches exactly as an Announcement's Audience (Department, Batch and role; each rule's fields must all match, any rule is enough, no rules means everyone), so one matcher serves notices, Events and Opportunities, and the Department filter and future reports can use indexed columns. Richer rules (CGPA, backlogs) can be added as columns on `audience_rules` when there is data for them.

**Opportunity status:** one `status` column: `draft`, `published`, `closed`. "Open" is not stored: an Opportunity is open while it is `published` and `apply_by` is in the future; closing it early sets `closed`. A published Opportunity can still be edited (every edit audited) except its `application_mode`, so a typo or a later `apply_by` doesn't need a new post.

**Who sees an Opportunity:** placement staff see all of them. A member sees a published or closed Opportunity only when they are in its Eligibility; anyone else gets `404`.

**Applications:** one per Student per Opportunity (`unique (opportunity_id, student_id)`), in `opportunity_applications`.
- Internal: a Student (the `student` role in effect, with a Student identity) in the Eligibility applies while the Opportunity is open. They can withdraw while it is still `applied`; withdrawing is final, so "applies once" holds.
- External: the Student applies on the company's site and marks "I applied" in LINKS, which records an Application with `mode = external`, so their own list and the placement reports include it. The same rules apply (eligible, open, once).
- Statuses: `applied`, `shortlisted`, `rejected`, `selected`, and `withdrawn` (set only by the Student). Placement staff move an Application between the first four in any direction (a wrong click can be undone), sending the status they saw; the row is locked (`FOR UPDATE`) and a status that has moved on is `409`, so two staff can't overwrite each other. Every change is audited with the old and new status.

**Who sees Applications** (ADR 0008): a Student sees only their own. Placement staff (`view_applicant_data`) see the applicant list with name, email, USN, Department, Batch, mode, status and dates, never phone numbers, and can export it as CSV with formula-injection escaping. Opening the list (its first page) and every export are audited. HODs get no applicant list here; Department summaries are a later reporting feature.

## Considered options

- **`eligibility jsonb` on `opportunities`**, as sketched: rejected. It would need a second matcher that had to agree with the Audience one, and JSON can't be indexed for the Department filter as simply.
- **Placement staff edit only their own Opportunities**, like Event proposals: rejected. The placement office is a small team that covers for each other; per-author drafts would strand a draft when its author is away.
- **Re-applying after withdrawing:** rejected for now. Withdrawing is rare and deliberate, and "once" keeps the applicant list honest. `withdrawn` is the Student's decision, so staff can't change it either; a Student who withdrew by mistake asks the placement office.

## Consequences

- `audience_rules.target_type` gains `opportunity`.
- A Department in any Eligibility can't be deleted, as for Events.
- `docs/database-design.md` and `docs/api-spec.md` are updated to this design as each part lands.
