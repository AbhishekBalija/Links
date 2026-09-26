# LINKS Interview Notes

## Department Management

### Why use a service, repository, and handler instead of putting everything in the Gin handler?

The handler owns HTTP concerns such as JSON parsing and status codes. The
service owns validation, authorization-sensitive rules, and transaction flow.
The repository owns database queries. This keeps business rules testable without
starting a server or connecting to PostgreSQL.

### Why are department codes immutable?

VTU course codes are used when parsing a student's USN and will later be used by
department filters and audience rules. Changing a code would silently break
those references. Editable names and descriptions allow display text to evolve
without changing the stable identifier.

### Why write the department mutation and audit log in one transaction?

A transaction treats both writes as one operation. If the department update
succeeds but the audit insert fails, PostgreSQL rolls back the update. This
prevents sensitive admin changes from existing without a matching audit record.

### Why validate the HOD role before saving `hod_user_id`?

A foreign key can prove that a user exists, but it cannot prove that the user is
an HOD. The service checks for an HOD role assignment first, while the database
foreign key still protects referential integrity. The role scope must match the
same department, preventing an HOD from another department being linked by
mistake.

### Why block deletion of referenced departments instead of cascading?

Cascade deletion could erase or detach student and authorization data after one
admin request. Returning `409 CONFLICT` makes the dependency explicit and keeps
cleanup or reassignment as a deliberate separate operation.
