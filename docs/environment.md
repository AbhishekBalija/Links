# Environment Configuration

## Required Environment Variables

```text
APP_ENV
APP_PORT
DATABASE_URL
JWT_ACCESS_SECRET
JWT_REFRESH_SECRET
ACCESS_TOKEN_TTL
REFRESH_TOKEN_TTL
CORS_ALLOWED_ORIGINS
COOKIE_SECURE
COOKIE_SAME_SITE
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
- Use `.env.local` only for local development.
- Use different secrets per environment.
- Never log secret values.
- Rotate JWT and cookie secrets using a planned process.

## Local Development

Use local `.env.local` for developer machines.

Example:

```text
APP_ENV=local
APP_PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/linksdb?sslmode=disable
CORS_ALLOWED_ORIGINS=http://localhost:5173
COOKIE_SECURE=false
COOKIE_SAME_SITE=lax
```

## Config Loading

Use explicit config loading:

```go
cfg, err := config.Load()
if err != nil {
    return err
}
```

Do not load config through `init()` side effects.
