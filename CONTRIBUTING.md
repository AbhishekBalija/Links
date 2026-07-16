# Contributing to LINKS

Thanks for your interest in contributing! LINKS is the primary campus hub for MITT students.

## How to Contribute

1. **Fork** the repository (if you're an external contributor).
2. **Create a branch** off `master` using the naming convention below.
3. **Make your changes** with atomic, well-scoped commits.
4. **Open a pull request** into `master`.
5. **Wait for CodeRabbit** — an automated reviewer will check your PR. Address its comments before requesting a merge.

## Branch Naming

| Type | Prefix | Example |
|------|--------|---------|
| Feature | `feat/` | `feat/auth-system` |
| Bug fix | `fix/` | `fix/login-redirect` |
| Docs/tooling/config | `chore/` | `chore/ci-setup` |

## Commit Messages

- Use imperative mood: "Add auth middleware" not "Added auth middleware".
- Include the *why* in the body when the reason isn't obvious.
- Keep commits atomic — one logical change per commit.

## Code Standards

- **Backend**: See [docs/backend-standards.md](docs/backend-standards.md) for Go conventions, layering rules, and testing requirements.
- **Frontend**: See [docs/frontend-ux-ui.md](docs/frontend-ux-ui.md) for UI/UX guidelines.
- **Security**: See [docs/security.md](docs/security.md) for security requirements and privacy rules.

## Reporting Issues

Open a GitHub issue with a clear description of the problem, steps to reproduce, and expected behavior.

## Questions?

Check the [docs/](docs/) directory for architecture, API specs, and product requirements.
