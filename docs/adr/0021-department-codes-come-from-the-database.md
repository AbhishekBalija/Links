# Department codes come from the database

The USN validator (`server/internal/auth/usn.go`) and the Access request form each kept their own list of Department codes. A Department an admin created through the API was saved, but its students couldn't request access: the USN check rejected the code and the form didn't offer it (#18). ADR 0016 also made every Department read require a token, which the sign-up form doesn't have.

## Decision

- The `departments` table is the only list of Department codes. `ValidateUSNFormat` checks the USN's shape and joining year only, and request-access then looks the code up in the table.
- When a request carries both a USN and a `department_code`, they must name the same Department. The USN wins as the identity key (ADR 0003), so a mismatch is `400`.
- `GET /api/v1/public/departments` returns each Department's `code` and `name` without a token, with `Cache-Control: public, max-age=300`. This narrows ADR 0016's "reads require authentication": the full records (IDs, HOD, description) still need a token.

## Considered options

- **Keep the lists hardcoded and document that a new Department needs a code change:** rejected. Department management through the API would then only half work, and the two lists had already drifted from the seed once.

## Consequences

- A new Department reaches the sign-up form within five minutes (the cache time) with no deploy.
- The public list shows which Departments exist, which the college's own website already shows.
