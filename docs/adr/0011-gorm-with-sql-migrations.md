# Use GORM with Explicit SQL Migrations

Decision: Use GORM as the Go persistence library for the initial modular monolith, while keeping schema changes in reviewed SQL migration files.

Reason:

- The existing Go scaffold already uses GORM.
- GORM supports ordinary CRUD without exposing persistence models at the API boundary.
- Explicit SQL keeps PostgreSQL constraints, indexes, and workflow-critical schema changes reviewable.

Trade-off:

- Complex reports and targeting queries need deliberate SQL or carefully reviewed GORM queries.
- The project may adopt `sqlc` later if query complexity makes generated typed SQL more valuable.
