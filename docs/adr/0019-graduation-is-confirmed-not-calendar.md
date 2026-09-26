# Graduation is confirmed by the college, not inferred from the calendar

A student doesn't become alumni just because four years have passed since the joining year in their USN. Students with backlogs finish the course without a degree, year-back students move to a later batch, lateral entry students finish in three years, and some students leave. So LINKS records each student's **Academic status** and **Batch**, and a student becomes alumni only in a **Graduation** step that an admin or HOD confirms from the university results.

## Decision

- **Academic status** on the Student identity: `studying`, `completed_with_backlogs`, `graduated` or `left`.
- **Batch** is the batch a student currently belongs to. It starts as the joining year in the USN and an admin can change it for a year-back student. Audience rules match on it, never on the USN directly.
- **Assumption, not yet confirmed with the college:** lateral-entry students (diploma holders joining second year, roll numbers 400 and up) get a USN carrying the year of the batch they join, so the USN year is their Batch too. If that turns out wrong, they would start one batch late, and an admin can move them.
- **Graduation**: once a year, an admin or HOD opens the final-year list with everyone selected, unticks students with backlogs (or uploads the results list), and confirms. For each confirmed student, in one transaction with an audit log:
  - the student Role assignment ends (`ends_at`, kept for history);
  - they get the alumni Role;
  - an alumni profile records the actual graduation year;
  - Academic status becomes `graduated`.
- Students left out stay students with status `completed_with_backlogs`. They keep receiving notices (backlog exams, results) and graduate in a later run once they clear.
- **Students who leave** (dropout, transfer) get status `left` and their student Role assignment ends. They do not become alumni.
- Nothing is deleted. Ending a Role assignment should also end the user's sessions (#22).

## Considered options

- **Automatic conversion four years after joining**: rejected. It would turn students with backlogs and year-back students into alumni, cut them off from exam notices, and miss lateral entry students.
- **Alumni with a "backlogs pending" flag**: rejected. Until they clear, these students still act as students (exams, results, college notices), and alumni status is a claim the college makes about a degree.

## Consequences

- Graduation needs a screen and a bulk path, which ties into CSV import (#17) and is tracked in #46.
- Placement eligibility rules (Phase 3) can read Academic status instead of guessing from the batch.
- Batch has to be filled in reliably at sign-up; today it is saved as `0` (#45).
