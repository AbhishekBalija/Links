# Seed Current B.E. Departments and Restrict Department Management to Admins

**Date:** 2026-08-30

**Decision:** Seed the six current MITT B.E. departments using VTU course codes:
`CS`, `AD`, `AI`, `CV`, `EC`, and `ME`. MBA and MCA are excluded from this
slice because they use a different student identity format. Department reads
require authentication, and create, update, and delete operations require the
`admin` role.

**HOD mapping:** The official MITT department pages provide current HOD names,
but `departments.hod_user_id` references a LINKS user UUID. Seeded rows therefore
leave this field null. An admin can assign it after that person has a LINKS
account with an HOD role assignment.

**Safety rules:**

- Department codes are immutable because USN parsing and audience targeting use them.
- A department cannot be deleted while a student identity or department-scoped role references it.
- Every department mutation and its audit log entry run in one database transaction.

**Sources:** MITT's current UG course and department pages, plus VTU's current
course-code equivalence list. These are checked again before changing the seed
data because college staff and course offerings can change.
