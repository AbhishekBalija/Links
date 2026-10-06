# Monitoring

## Goals

Monitoring should answer:

- Is the API healthy?
- Are users able to log in?
- Are DB queries slow?
- Are approvals and applications working?
- Are exports failing?
- Are unauthorized attempts increasing?

## Logging

Use structured JSON logs in production.

Required fields:

- `request_id`
- `user_id`
- `role`
- `method`
- `path`
- `status_code`
- `latency_ms`
- `error_code`

A `status_code` of **499** means the client gave up before the answer was
ready (usually a browser navigating away). It is not a server error and is
left out of the error rate and 5xx alerts; the cancelled database query is
not logged either.

The database logger (`server/pkg/db/logger.go`) prints slow queries and
errors with `?` in place of every value, so emails, names, USNs and tokens
stay out of the logs.

Never log:

- Passwords
- Refresh tokens
- Full authorization headers
- Sensitive applicant notes
- Values inside SQL queries (the database logger hides them)

## Metrics

Track:

- HTTP request count
- HTTP latency
- HTTP error rate
- DB query latency
- Login failures
- Access requests
- Event approvals
- Opportunity applications
- Shortlisting changes
- CSV exports
- File upload failures
- Notification outbox backlog
- Email delivery failures
- Web push delivery failures

## Health Checks

Endpoints:

```text
GET /api/health
GET /api/ready
```

`/api/health` checks process liveness.

`/api/ready` checks database connectivity only, unless the handler is explicitly extended to check additional required dependencies.

## Alerts

Alert on:

- API 5xx spike
- Login failure spike
- Database connection errors
- High DB latency
- Export failures
- Email failures
- Notification worker failures
- Large pending notification backlog
- Unusual applicant data access

## Tracing

Distributed tracing is not required in MVP.

Use request IDs from day one. Add OpenTelemetry later when the system grows.
