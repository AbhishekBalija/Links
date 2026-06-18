# Decision Log

## ADR-001: Use Modular Monolith First

Decision: Use one Go REST API with internal modules.

Reason:

- The domain is connected.
- Transactions matter.
- Deployment stays simple.
- The team can move faster.

Trade-off:

- Requires discipline to maintain module boundaries.

## ADR-002: Use PostgreSQL as Source of Truth

Decision: Use PostgreSQL for all core data.

Reason:

- Strong relational workflows
- Transactions
- Foreign keys
- JSONB eligibility rules
- Good reporting support

## ADR-003: Use USN as Student Identity Key

Decision: USN is the primary student/alumni identity key when available.

Reason:

- MITT does not provide official student emails.
- USN includes joining year, branch, and sequence.
- USN helps avoid duplicate accounts.

## ADR-004: Use Gmail Invites Plus Manual Approval

Decision: Use Gmail IDs already shared with the college when available; otherwise HOD/admin can manually approve access.

Reason:

- This matches the current college reality.
- It avoids depending on official college email.

## ADR-005: Use Scoped RBAC

Decision: Use role assignments with scopes.

Reason:

- A student can also be a coordinator.
- A coordinator can be scoped to a department or club.
- HOD permissions should be department-scoped.

## ADR-006: Require Event Approval

Decision: Student coordinator events require HOD review and principal/admin final approval.

Reason:

- Events need official records.
- Department authority and college authority both matter.

## ADR-007: Defer Redis and Workers

Decision: Do not introduce Redis or background workers in MVP unless the need becomes real.

Reason:

- PostgreSQL and synchronous APIs are enough for the first version.
- Extra infrastructure increases operational burden.

## ADR-008: Keep Applicant Data Sensitive

Decision: Placement applicant data is restricted to the student, placement officer, principal, and admin. HODs see department summaries unless explicitly allowed.

Reason:

- Application and shortlisting status is sensitive.
- Least privilege matters.

## ADR-009: Use Multi-Channel Notifications

Decision: Use in-app notifications as the permanent record, email for important official fallback, and web push for opted-in users when the app is closed.

Reason:

- Students may not keep LINKS open.
- Placement links and shortlisting updates are time-sensitive.
- Web push depends on browser/device permission and is not guaranteed.
- Email is more reliable for official communication.

Trade-off:

- Requires notification preferences, delivery tracking, and a worker/outbox system.
