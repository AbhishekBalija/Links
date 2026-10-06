# Environment Configuration

## Required Environment Variables

For the current health-check-only backend, production needs:

```text
DATABASE_URL
APP_ENV=production
GIN_MODE=release
```

`PORT` is optional: the host may provide it. `APP_PORT` is the local fallback.

The full product will later need:

```text
APP_ENV
APP_PORT
DATABASE_URL
JWT_ACCESS_SECRET
JWT_REFRESH_SECRET
ACCESS_TOKEN_TTL
REFRESH_TOKEN_TTL
CORS_ALLOWED_ORIGINS
COOKIE_SECURE      # must be true unless APP_ENV=local
COOKIE_SAME_SITE   # lax (default) or strict; none is refused
ENABLE_TEST_SIGN_IN # true only for the e2e suite; refused unless APP_ENV=local.
                    # Adds POST /api/v1/test/sign-in and GET /api/v1/test/sign-in-code?email=
                    # (the last code emailed to that address)
MAIL_PROVIDER      # resend (default, production) or smtp
SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD
                    # for MAIL_PROVIDER=smtp: a testing inbox (Mailtrap:
                    # sandbox.smtp.mailtrap.io, 2525) or Gmail with an app
                    # password (smtp.gmail.com, 587). See docs/deployment.md
EMAIL_CODE_FOR_EVERY_ROLE # true on a test copy: the principal and admins may sign in with
                    # an email code too, so testers can use throwaway inboxes.
                    # Refused with APP_ENV=production (Google only there, ADR 0026)
NOT_ON_LIST_CODES_PER_DAY # sign-in codes a day to emails on no list, whole copy
                    # (default 50). Raise it for orientation day if the email
                    # service's quota allows; must be above 0
GOOGLE_CLIENT_ID   # Google sign-in's OAuth client ID (public); Google sign-in is off while empty
VITE_GOOGLE_CLIENT_ID # the same client ID for the web app; the Google button is hidden while empty
STORAGE_PROVIDER
STORAGE_BUCKET
STORAGE_API_KEY
EMAIL_PROVIDER
EMAIL_API_KEY
RATE_LIMIT_ENABLED
VAPID_PUBLIC_KEY
VAPID_PRIVATE_KEY
```

## Optional Environment Variables

```text
LOG_LEVEL
DB_MAX_OPEN_CONNS
DB_MAX_IDLE_CONNS
DB_CONN_MAX_LIFETIME
DB_CONN_MAX_IDLE_TIME
REQUEST_BODY_LIMIT
CSV_IMPORT_LIMIT
EXPORT_ROW_LIMIT
```

## Rules

- No secrets in Git.
- Validate config on startup.
- Fail fast when required config is missing.
- Use `server/.env.local` only for local development.
- Use different secrets per environment. Vercel Production and Preview have separate `DATABASE_URL` and JWT secrets (see `deployment.md`).
- Tests and previews never use the production database.
- Never log secret values.
- Rotate JWT and cookie secrets using a planned process.

## Local Development

Use `server/.env.local` for developer machines.

Example:

```text
APP_ENV=local
APP_PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/linksdb?sslmode=disable
CORS_ALLOWED_ORIGINS=http://localhost:5173
COOKIE_SECURE=false
COOKIE_SAME_SITE=lax
```

## Contributor Tooling

### CodeRabbit

CodeRabbit is configured as an automated PR reviewer on this repository. It runs automatically on every pull request — no local setup or API key is required from contributors. If CodeRabbit flags issues on your PR, address them before requesting a merge.

## Config Loading

Use explicit config loading:

```go
cfg, err := config.Load()
if err != nil {
    return err
}
```

Do not load config through `init()` side effects.
