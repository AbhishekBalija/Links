# Contributing to LINKS

Thanks for helping. This guide covers setup, the workflow, and what a good
pull request looks like.

## Setup

Follow [Run locally](README.md#run-locally) in the README. Use your own local
database or Neon branch for development, never a shared one for tests.

## Before you start

- Read [`CONTEXT.md`](CONTEXT.md) for the domain words (USN, Access request,
  Access approval, Activation, Role assignment, Scope) and use them in code and docs.
- Architecture decisions live in [`docs/adr/`](docs/adr). If your change
  alters one, add a new ADR instead of silently diverging.
- Check [`docs/roadmap.md`](docs/roadmap.md) and the open issues; comment on
  an issue before starting larger work.

## Workflow

- Branch from `master`: `feat/...`, `fix/...`, `chore/...`, `docs/...`.
- Never push to `master`; open a pull request. CI must pass, and CodeRabbit
  reviews every PR: address or dismiss its comments before merging.
- Commits follow [Conventional Commits](https://www.conventionalcommits.org/)
  (`feat(server): ...`, `fix(client): ...`), and user-visible changes get a
  line in [CHANGELOG.md](CHANGELOG.md) under Unreleased.

## Checks

```sh
cd server && go vet ./... && go test ./...
cd client && bun run lint && bun run build
```

The Playwright e2e suite (`client/e2e/`) creates and drops its own schema; CI
runs it against a throwaway Postgres. Code standards: Go in
[docs/backend-standards.md](docs/backend-standards.md), UI in
[docs/frontend-ux-ui.md](docs/frontend-ux-ui.md), security in
[docs/security.md](docs/security.md).

## Pull requests

Describe what changed, why, and how to test it, and add screenshots for UI
changes (the template asks for these). Keep pull requests focused on one
change.

## Releases

LINKS uses [Semantic Versioning](https://semver.org/) and stays on 0.x while
it is pre-1.0: a minor bump for a finished phase or a notable set of
features, a patch bump for fixes only. A release is a `chore/release-X.Y.Z`
pull request that moves the Unreleased entries in CHANGELOG.md into a dated
section. After it merges, the merge commit is tagged `vX.Y.Z` and published as
a GitHub release with that changelog section as its notes.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md).
