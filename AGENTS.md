# Agent Instructions for LINKS

Before making code, architecture, product, database, API, frontend, deployment, or security changes, read the project context first.

## Required Context

Always start by reading:

1. `My_Plan.md`
2. `docs/product-requirements.md`
3. `docs/architecture.md`
4. The specific docs related to the task

## Task-to-Doc Map

Use this map before working:

| Task Type | Read These Files |
|---|---|
| Product planning | `My_Plan.md`, `docs/product-requirements.md`, `docs/roadmap.md`, `docs/decision-log.md` |
| Backend architecture | `docs/architecture.md`, `docs/backend-standards.md`, `docs/decision-log.md` |
| Database/schema/migrations | `docs/database-design.md`, `docs/backend-standards.md`, `docs/security.md` |
| API changes | `docs/api-spec.md`, `docs/auth.md`, `docs/frontend-contract.md` |
| Authentication/RBAC | `docs/auth.md`, `docs/security.md`, `docs/database-design.md` |
| Frontend work | `docs/frontend-ux-ui.md`, `docs/frontend-contract.md`, `docs/product-requirements.md`, `docs/api-spec.md` |
| Notifications | `docs/notifications.md`, `docs/product-requirements.md`, `docs/frontend-ux-ui.md`, `docs/backend-standards.md` |
| Deployment/config | `docs/deployment.md`, `docs/environment.md`, `docs/monitoring.md` |
| Security/privacy | `docs/security.md`, `docs/auth.md`, `docs/backend-standards.md` |
| Scaling/performance | `docs/scaling.md`, `docs/monitoring.md`, `docs/database-design.md` |

## Project Direction

LINKS is the primary MITT campus hub, not a chat platform or casual social network.

Core workflows:

- USN-based student identity
- Gmail invite or HOD/admin approval
- Public verified profiles with private contact protection
- Targeted announcements by department, batch, year, role, and eligibility
- In-app, email, and web push notifications for important updates
- Student coordinator event proposals
- HOD review and principal/admin final event approval
- Placement opportunities with internal/external application tracking
- Shortlisting, status updates, and CSV exports

## Backend Expectations

- Use Go with a clean modular monolith structure.
- Keep handlers thin.
- Put business logic in services.
- Put database access in repositories.
- Enforce authorization in service/policy layers.
- Avoid global mutable state such as package-level database access.
- Avoid `init()` side effects for config, database, and routes.
- Use explicit constructors and dependency injection.
- Use PostgreSQL with explicit SQL migrations.
- Add audit logs for sensitive actions.
- Never expose password hashes, refresh tokens, or private applicant data.

## Frontend Expectations

- Read `docs/frontend-ux-ui.md` before designing or changing UI.
- Build role-aware product surfaces, not generic dashboards.
- Optimize student workflows for mobile.
- Optimize admin, HOD, and placement workflows for desktop efficiency.
- Keep UI calm, official, readable, and workflow-first.
- Use cards only where they represent discrete repeated items or true interactions.
- Design empty, loading, error, and permission states for every screen.
- Backend remains the source of truth for authorization.

## Documentation Rules

When a decision changes, update the relevant doc in `docs/`.

If a major product or architecture decision is made, also update:

- `docs/decision-log.md`

Do not modify `My_Plan.md` unless the user explicitly asks to update the main plan.
