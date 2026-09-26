# LINKS

[![CI](https://github.com/AbhishekBalija/Links/actions/workflows/ci.yml/badge.svg)](https://github.com/AbhishekBalija/Links/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

**The official campus hub for MITT.** Verified student identities, targeted
announcements, event approvals and placement tracking in one place. Not a chat
app or a social network.

## Status

- **Done:** Phase 0 (foundation) and Phase 1 (identity and access).
  Students request access with their Gmail and USN, an HOD or admin approves
  them, and they activate their account from an emailed link. Login uses
  short-lived access tokens with rotating refresh tokens, and permissions come
  from scoped role assignments.
- **Now:** Phase 2, the campus hub: departments, dashboards, directory and
  targeted announcements. See the
  [milestone](https://github.com/AbhishekBalija/Links/milestone/1) and
  [roadmap](docs/roadmap.md).

## Stack

- **Server:** Go modular monolith (Gin, GORM), PostgreSQL on Neon with plain SQL migrations
- **Client:** React 19, Vite, TypeScript, Tailwind CSS v4, shadcn/ui, Zustand, TanStack Query
- **Email:** Resend. **Errors:** Sentry. **Hosting:** Vercel.
- **Tests:** Go tests, Playwright e2e

## Run locally

Prerequisites: Go 1.26+, [Bun](https://bun.sh) and a PostgreSQL database
(local, or a Neon branch).

```sh
cp server/.env.local.example server/.env.local   # set DATABASE_URL and the JWT secrets
cp client/.env.local.example client/.env.local   # set VITE_API_URL

cd server && go run ./cmd/api                    # applies migrations on start
cd client && bun install && bun run dev          # in a second terminal
```

Without `RESEND_API_KEY`, no emails are sent. See
[docs/environment.md](docs/environment.md) for every variable.

## Docs

- [CONTEXT.md](CONTEXT.md): the domain words (USN, Access request, Activation, Role assignment)
- [docs/architecture.md](docs/architecture.md) and [docs/adr/](docs/adr): how it's built and why
- [docs/api-spec.md](docs/api-spec.md): the REST API
- [docs/roadmap.md](docs/roadmap.md): phases and what's next

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Report security problems privately, as
described in [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE)
