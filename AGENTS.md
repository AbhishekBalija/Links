# Agent Instructions for LINKS

## Git Workflow (Standing Rule)

All changes must go through a pull request. Direct pushes to `master` are blocked by a GitHub ruleset.

### Branch Naming

| Type | Prefix | Example |
|------|--------|---------|
| Feature | `feat/` | `feat/auth-system` |
| Bug fix | `fix/` | `fix/login-redirect` |
| Docs/tooling/config | `chore/` | `chore/ci-setup` |

### Process

1. Create a dedicated branch off `master` before writing any code.
2. Make atomic, logically scoped commits with clear messages (what + why).
3. Push the branch and open a PR into `master`.
4. Let CodeRabbit's automated review run — do not merge until its comments are addressed or explicitly dismissed.
5. If a task is ambiguous about whether it's "one feature," default to smaller/more branches over one giant one.

### Commit Messages

- Use imperative mood: "Add auth middleware" not "Added auth middleware".
- Include the *why* in the body when the reason isn't obvious from the subject.

Before making code, architecture, product, database, API, frontend, deployment, or security changes, read the project context first.

## Required Context

Always start by reading:

1. `My_Plan.md`
2. `docs/product-requirements.md`
3. `docs/architecture.md`
4. `docs/PROGRESS.md` ⚠️ **MANDATORY** - Check current phase and what's in progress before starting
5. The specific docs related to the task

## ⚠️ CRITICAL: Before Implementing Anything

**You MUST read and follow this workflow:**

1. **Check Phase Status** - Read `docs/PROGRESS.md` to understand:
   - What phase the project is in
   - What's already completed ✅
   - What's currently in progress 🔄
   - What's blocked ⚠️

2. **Verify You're Not Duplicating Work** - Check if the feature you're about to build is already in progress or completed

3. **Read Relevant Docs** - Use the "Task-to-Doc Map" below to read all context for your specific task

4. **Implementation Phase**:
   - Build the feature following backend/frontend expectations
   - Run the "AI Agent Checklist" from `docs/PROGRESS.md` BEFORE marking complete
   - Verify all checks pass (code standards, tests, error handling, auth, etc.)

5. **Update PROGRESS.md** - Before submitting code:
   - Move feature from section to "Ready for Dev Verification" in `docs/PROGRESS.md`
   - List what files were updated
   - Include implementation notes for the dev to understand what to test

6. **Dev Must Verify** - Only the dev can mark a feature as ✅ complete in `docs/PROGRESS.md`
   - The dev tests locally
   - The dev confirms it works as described
   - The dev adds verification date and sign-off

**If you skip reading `docs/PROGRESS.md` first, you risk duplicating work or missing critical context.**

## Task-to-Doc Map

Use this map before working:

| Task Type | Read These Files |
|---|---|
| Product planning | `My_Plan.md`, `docs/product-requirements.md`, `docs/roadmap.md`, `docs/decision-log.md` |
| Backend architecture | `docs/architecture.md`, `docs/backend-standards.md`, `docs/decision-log.md` |
| Database/schema/migrations | `docs/database-design.md`, `docs/backend-standards.md`, `docs/security.md`, `docs/college-info.md` |
| API changes | `docs/api-spec.md`, `docs/auth.md`, `docs/frontend-contract.md` |
| Authentication/RBAC | `docs/auth.md`, `docs/security.md`, `docs/database-design.md`, `docs/college-info.md` |
| Frontend work | `My_Plan.md`, `docs/frontend-ux-ui.md`, `docs/frontend-contract.md`, `docs/product-requirements.md`, `docs/api-spec.md` |
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
