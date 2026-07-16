# Backend Standards

## Go Standards

- Keep code simple and explicit.
- Prefer small interfaces at package boundaries.
- Avoid global mutable state.
- Avoid `init()` for config, DB connections, and route setup.
- Use constructors that return errors.
- Use `context.Context` for request-scoped work.
- Handle errors first with early returns.
- Do not panic for expected failures.

## Layering Rules

- Handler: HTTP and DTO concerns only.
- Service: business rules and transactions.
- Repository: database operations.
- Policy: permission checks.
- DTO: API payloads.
- Model: persistence/domain representation.

## Constructor Pattern

Use plain constructors.

```go
func NewUserService(repo UserRepository, audit AuditService) (*UserService, error) {
    if repo == nil {
        return nil, errors.New("user repository is required")
    }
    return &UserService{repo: repo, audit: audit}, nil
}
```

Use functional options only when a constructor is expected to grow.

## Error Handling

Use typed application errors:

```text
VALIDATION_ERROR
UNAUTHENTICATED
FORBIDDEN
NOT_FOUND
CONFLICT
RATE_LIMITED
INTERNAL_ERROR
```

Rules:

- Do not leak raw DB errors.
- Log internal errors with request ID.
- Return stable error codes.
- Wrap errors with context using `%w`.

## GORM Rules

- Do not expose GORM models directly in API responses.
- Do not put business logic in GORM hooks.
- Use explicit transactions for workflows.
- Use scoped `Preload` and `Select` to avoid overfetching.
- Avoid package-level global `db.DB`.
- Use raw SQL for complex reports when clearer.

## Transactions

Use transactions for:

- User creation with profile/student identity
- Access approval with audit log
- Role assignment with audit log
- Event approval
- Placement application status changes
- CSV export audit records

Recommended interface:

```go
type TxManager interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

## Domain State

Use typed statuses.

```go
type UserStatus string

const (
    UserStatusUnknown   UserStatus = ""
    UserStatusPending   UserStatus = "pending"
    UserStatusActive    UserStatus = "active"
    UserStatusSuspended UserStatus = "suspended"
    UserStatusRejected  UserStatus = "rejected"
)
```

Rules:

- Include an unknown zero value.
- Validate transitions in services.
- Mirror allowed values with DB check constraints.

## Resource Management

- Defer `Close()` immediately after opening resources.
- Close HTTP response bodies.
- Close SQL rows.
- Bound database pools.
- Bound request body sizes.
- Stream large exports.
- Add timeouts to external calls.

## Git Workflow

All changes must go through a pull request into `master`. Direct pushes are blocked by a GitHub ruleset.

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
5. If a task is ambiguous about whether it's "one feature," default to smaller/more branches.

### Commit Messages

- Use imperative mood: "Add auth middleware" not "Added auth middleware".
- Include the *why* in the body when the reason isn't obvious from the subject.
- Keep commits atomic — one logical change per commit.

## Testing Requirements

Unit tests:

- USN parsing
- Audience matching
- Eligibility checks
- RBAC policies
- Event state transitions
- Application state transitions

Integration tests:

- Login/refresh
- Access approval
- Targeted announcements
- Event approval end to end
- Placement application and shortlisting
- Export authorization

Definition of done:

- API contract documented
- Validation implemented
- Authorization enforced
- Business logic tested
- Migration added
- Audit log added for sensitive actions
- No sensitive fields leaked

