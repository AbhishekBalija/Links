# Deployment

## Deployment Architecture

LINKS is deployed as a single Vercel project using
[Vercel Services](https://vercel.com/docs/services). The monorepo contains:

| Service     | Root Directory | Framework    | Public Route |
| ----------- | -------------- | ------------ | ------------ |
| Frontend    | `client/`      | Vite + React | `/`          |
| Backend API | `server/`      | Go + Gin     | `/api/*`     |

The database remains on **Neon PostgreSQL**, with two branches:

| Neon branch  | Used by                                    |
| ------------ | ------------------------------------------ |
| `production` | Vercel Production only                     |
| `dev`        | Vercel Preview deployments and local development |

Preview deployments run their PR's migrations on startup, so they must never
point at `production`: an unmerged migration would change real data. Never use
Neon's "Reset from parent" on `dev`, because it copies real student data from
`production`. Both frontend and backend
share the same Vercel project domain, so the browser calls the API using
relative paths (`/api/health`) with no CORS.

## Why Vercel Services?

- One project, one deployment, one domain
- No CORS between frontend and backend
- No forced 15-minute sleep like Render free tier
- Frontend and backend deploy together on every git push

## Root `vercel.json`

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "services": {
    "web": {
      "root": "client/",
      "framework": "vite"
    },
    "api": {
      "root": "server/",
      "framework": "go",
      "buildCommand": "go build -o bin/api ./cmd/api",
      "outputDirectory": "bin"
    }
  },
  "rewrites": [
    { "source": "/api/(.*)", "destination": { "service": "api" } },
    { "source": "/(.*)", "destination": { "service": "web" } }
  ]
}
```

## Environment Variables

Set these in the Vercel project dashboard. They apply to both services.

| Variable             | Environment                         | Description                          |
| -------------------- | ----------------------------------- | ------------------------------------ |
| `DATABASE_URL`       | Production and Preview, separately  | `production` and `dev` Neon branches |
| `JWT_ACCESS_SECRET`  | Production and Preview, separately  | Different values, so preview tokens don't work in production |
| `JWT_REFRESH_SECRET` | Production and Preview, separately  | Different values, as above           |
| `APP_ENV`            | Preview, Production                 | Runtime environment name             |
| `GIN_MODE`           | Preview, Production                 | `release`                            |
| `RESEND_API_KEY`, `FROM_EMAIL` | Preview, Production       | Email delivery                       |
| `SENTRY_DSN`, `VITE_SENTRY_DSN`, `SENTRY_AUTH_TOKEN` | Preview, Production | Error tracking            |
| `GOOGLE_CLIENT_ID`, `VITE_GOOGLE_CLIENT_ID` | Production | Google sign-in (the same public client ID). Previews leave it unset and sign in with an email code, since Google can't list preview URLs as origins |

"Separately" means two Vercel variables with the same name, one scoped to
Production and one to Preview.

No `VITE_API_URL` or `FRONTEND_URL` is needed because both services share the
same domain.

## Smoke Test

After deploying, verify:

```text
GET https://<project>.vercel.app/
GET https://<project>.vercel.app/api/health
GET https://<project>.vercel.app/api/ready
```

## Local Development

Run all services together with Vercel CLI:

```bash
vercel dev
```

Or run them separately:

```bash
# Terminal 1: backend
cd server
go run ./cmd/api

# Terminal 2: frontend
cd client
bun run dev
```

## Release Process

> `master` is protected by a GitHub ruleset that blocks direct pushes. All
> changes must go through a pull request.

1. Create a feature/fix/chore branch off `master`.
2. Open a PR into `master`.
3. Vercel creates a preview deployment.
4. CodeRabbit runs automated review — address comments before merging.
5. Smoke test preview URLs.
6. Merge to `master`.
7. Vercel deploys to production.
8. Run production smoke tests.

## Rollback

- Redeploy a previous Vercel deployment from the dashboard.
- Avoid destructive migrations.
- Prefer forward fixes for production database issues.

## Docker

Not required for Vercel Services. For local parity or future self-hosting, a
multi-stage Dockerfile should:

- Use the official Go image
- Run as a non-root user
- Copy only required files
- Expose the configured port
- Use environment variables

## Kubernetes

Kubernetes is not needed for MVP.

Consider Kubernetes only when:

- Multiple workers/services exist beyond the Vercel project.
- Traffic requires horizontal scaling.
- The team can operate Kubernetes safely.

## Previous Deployment Notes

Earlier deployments used a separate Render backend (`links-d1b5.onrender.com`)
and a separate Vercel frontend (`links-campus.vercel.app`). This was migrated to
Vercel Services to avoid Render's free-tier sleep behavior and to simplify the
deployment surface.
