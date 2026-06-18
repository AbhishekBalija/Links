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

Never log:

- Passwords
- Refresh tokens
- Full authorization headers
- Sensitive applicant notes

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

## Health Checks

Endpoints:

```text
GET /health
GET /ready
```

`/health` checks process liveness.

`/ready` checks database connectivity and required dependencies.

## Alerts

Alert on:

- API 5xx spike
- Login failure spike
- Database connection errors
- High DB latency
- Export failures
- Email failures
- Unusual applicant data access

## Tracing

Distributed tracing is not required in MVP.

Use request IDs from day one. Add OpenTelemetry later when the system grows.

