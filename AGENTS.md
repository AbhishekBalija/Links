## Agent skills

### Issue tracker

Issues live in GitHub Issues via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five canonical labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context (`CONTEXT.md` + `docs/adr/`). See `docs/agents/domain.md`.

# LINKS

LINKS is a campus hub for colleges, not a chat platform or casual social network. Domain words (USN, Access request, Access approval, Activation, Role assignment, Scope, Audience) are defined in `CONTEXT.md`; use them in code, issues and docs.

## Where things are

- **Current work**: GitHub Issues and milestones (one milestone per roadmap phase). Check open issues for the current phase before starting, so work isn't duplicated.
- **Plan**: `docs/roadmap.md` (phases, done items, issue links) and `docs/implementation.md` (step-by-step build guide).
- **Decisions**: `docs/adr/`, one file per ADR. If a change contradicts an ADR, say so and write a new ADR rather than silently diverging.
- **Vision**: `My_Plan.md`. Do not modify it unless the user explicitly asks. Where it disagrees with `docs/`, `docs/` wins.

## Task-to-doc map

| Task type | Read these files |
|---|---|
| Product planning | `My_Plan.md`, `docs/product-requirements.md`, `docs/roadmap.md`, `docs/adr/` |
| Backend architecture | `docs/architecture.md`, `docs/backend-standards.md`, `docs/adr/` |
| Database/schema/migrations | `docs/database-design.md`, `docs/backend-standards.md`, `docs/security.md`, `docs/local/college-info.md` (local only) |
| API changes | `docs/api-spec.md`, `docs/auth.md`, `docs/frontend-contract.md` |
| Authentication/RBAC | `docs/auth.md`, `docs/security.md`, `docs/database-design.md`, `docs/local/college-info.md` (local only) |
| Frontend work | `docs/frontend-ux-ui.md`, `docs/frontend-contract.md`, `docs/product-requirements.md`, `docs/api-spec.md` |
| Notifications | `docs/notifications.md`, `docs/product-requirements.md`, `docs/frontend-ux-ui.md`, `docs/backend-standards.md` |
| Deployment/config | `docs/deployment.md`, `docs/environment.md`, `docs/monitoring.md` |
| Security/privacy | `docs/security.md`, `docs/auth.md`, `docs/backend-standards.md` |
| Scaling/performance | `docs/scaling.md`, `docs/monitoring.md`, `docs/database-design.md` |

## Git workflow

Direct pushes to `master` are blocked by a GitHub ruleset; every change goes through a pull request.

- Branch off `master`: `feat/...`, `fix/...`, `chore/...`, `docs/...`. Prefer smaller branches over one giant one.
- Commits follow Conventional Commits (`feat(server): ...`, `fix(client): ...`, `docs: ...`), imperative mood, with the why in the body when it isn't obvious.
- User-visible changes get a line in `CHANGELOG.md` under Unreleased.
- PR description: what changed, why (with `Closes #N`), how to test, screenshots for UI changes. Let CodeRabbit review; address or dismiss its comments before merging.
- The developer tests the "How to test" steps before merging; merging closes the issue.

## Working agreement

How the owner and agents have agreed to work, beyond the rules above. It applies to every session, local or cloud.

**Always**
- Only the owner merges, or tells an agent to merge a specific PR. Open the PR, get CI green, and stop.
- One concern per PR. Behaviour changes and pure moves or refactors never share a PR.
- Test first at the agreed seam: a failing API test in `server/test/integration/` (or a Vitest unit test for client logic), then the code.
- Bun only, never npm. Propose any new dependency with a reason and wait for the owner's OK.
- No em dashes in code, docs, commits or PRs.
- Judge outside or agent-written PRs on merit. Closing one with a short explanation is fine.

**UI work**
- Design before code, on the one "LINKS screens" canvas (a Claude Design artifact; the owner has the link), in the approved Gazette style from `docs/frontend-ux-ui.md` and the tokens in `client/src/index.css`.
- Draw every state, not just the main view: desktop and phone, loading, empty or first-time, error, validation, confirmations, "leaving with changes", each role's variant, and each status's action bar.
- While designing, ask of every screen: what does the user do first? Is the common case the fewest steps? Are the actions in one predictable place (one primary action per area, few button styles)? Could a first-time student or faculty member finish without help?
- Wait for the owner's approval before writing UI code. When the owner says they like a design, critique it first: a numbered list of weaknesses, button placement and ease of use first, each with a concrete fix.
- UI PRs include screenshots of every state on desktop (1440 wide) and phone (390 wide), committed under `docs/screenshots/<issue>-<name>/` and linked in the PR.

**Refactors**
- Split by responsibility, not line count. Move code unchanged, diff the function or declaration list before and after, and keep all tests passing.
- Refactor just before changing an area, not as ongoing churn.

**Unattended sessions (cloud, overnight)**
- Don't merge, don't close issues, and don't start work that needs a design approval. Stop at the approval point and leave a comment on the issue saying what is ready for review.
- Start Postgres first with `service postgresql start`. Postgres 16 is installed and `TEST_DATABASE_URL` is already set, but only the Postgres files are cached between sessions, not the running server, so every new session has to start it. If the `links` role or `links_test` database is missing from the cache, create them as the `postgres` user to match `TEST_DATABASE_URL` (CI's `links` role is a superuser, so give the local one the same).
- If the session still can't reach a database, say so in the PR. GitHub CI runs the API and e2e tests against Postgres, and a green CI is required.
- End with a comment on the issue: what was done, what's open, and any question for the owner.

## Done means

- `go vet ./...` and `go test ./...` pass in `server/`, with `TEST_DATABASE_URL` pointing at a throwaway local database so the API tests in `server/test/integration/` run; `bun run lint`, `bun run test` and `bun run build` pass in `client/`; the Playwright e2e suite passes for auth or flow changes.
- `CONTEXT.md`, the ADRs, `docs/roadmap.md` and the related docs match the code.
- `.env.*.example` files list any new env vars.

## Backend expectations

- Go modular monolith. Handlers stay thin, business logic lives in services, database access in repositories.
- Enforce authorization in service/policy layers.
- No global mutable state (no package-level database access) and no `init()` side effects for config, database or routes.
- Explicit constructors and dependency injection.
- PostgreSQL with explicit SQL migrations.
- Audit logs for sensitive actions, written in the same transaction as the change.
- Never expose password hashes, refresh tokens, activation tokens or private applicant data.
- Tests that write data use an isolated database or schema, never the shared Neon dev branch.

## Frontend expectations

- Read `docs/frontend-ux-ui.md` before designing or changing UI.
- Build role-aware product surfaces, not generic dashboards.
- Optimize student workflows for mobile; optimize admin, HOD and placement workflows for desktop efficiency.
- Keep UI calm, official, readable and workflow-first. Use cards only for discrete repeated items or true interactions.
- Design empty, loading, error and permission states for every screen.
- The backend remains the source of truth for authorization.
