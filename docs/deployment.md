# Deployment

## MVP Deployment Targets

| Component | Recommended |
|---|---|
| Frontend | Vercel |
| Backend | Render |
| Database | Neon PostgreSQL |
| Object storage | Cloudinary or S3-compatible storage |
| Email | Resend |

## Environments

- Local
- Staging
- Production

Production should require:

- Passing CI
- Migration review
- Environment validation
- Rollback plan

## Docker

Backend Dockerfile should:

- Use multi-stage build
- Run as non-root user
- Copy only required files
- Expose configured port
- Use env vars

## Startup Order

```mermaid
flowchart LR
    Config[Load Config] --> Logger[Init Logger]
    Logger --> DB[Connect DB]
    DB --> Migrate[Run Migrations]
    Migrate --> Services[Build Services]
    Services --> Router[Build Router]
    Router --> Server[Start HTTP Server]
```

## Release Process

1. Merge to main after CI passes.
2. Deploy backend to staging.
3. Run migrations on staging.
4. Smoke test auth, health, DB connection.
5. Deploy frontend to staging.
6. Promote to production.
7. Run production smoke test.

## Rollback

Backend rollback:

- Redeploy previous backend image.
- Avoid destructive migrations.
- Use backwards-compatible DB changes where possible.

Database rollback:

- Prefer forward fixes for production.
- Use down migrations only when safe and tested.

## Kubernetes

Kubernetes is not needed for MVP.

Consider Kubernetes only when:

- Multiple workers/services exist.
- Traffic requires horizontal scaling.
- The team can operate Kubernetes safely.

