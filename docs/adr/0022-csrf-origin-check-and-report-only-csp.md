# CSRF origin check, API security headers, and a report-only CSP for the web app

The refresh token lives in an `HttpOnly` cookie, so `POST /api/v1/auth/refresh` and `/logout` act on whatever cookie the browser sends. `SameSite=Lax` already keeps it off cross-site POSTs in current browsers, but that is one setting away from failing, and no Content-Security-Policy was set anywhere (#19).

## Decision

- **Origin check on cookie endpoints.** Refresh and logout accept a request only when its `Origin` (or, without one, its `Referer`) is `FRONTEND_URL`, one of `CORS_ALLOWED_ORIGINS`, or the host the request was sent to (`Host` or `X-Forwarded-Host`, for the same-domain Vercel deployment). Anything else, including `Origin: null`, is `403`. A request with neither header comes from a non-browser client, which can't be made to send someone else's cookie, so it passes. No CSRF token: the check needs no client change and covers the only two cookie-authenticated endpoints. Every other write uses the bearer token, which a cross-site page can't read.
- **Cookie settings are enforced at startup.** `COOKIE_SAME_SITE` must be `lax` or `strict` (`none` is refused). `COOKIE_SECURE` must be true unless `APP_ENV=local`.
- **API headers.** Every API response carries `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `X-Frame-Options: DENY` and `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, plus HSTS when `APP_ENV=production`.
- **Web app CSP, report-only first.** `vercel.json` sends `Content-Security-Policy-Report-Only` allowing only the app's own origin (the API is same-origin), `data:` fonts and images (Vite inlines small ones), `blob:` workers (Sentry Replay), and Sentry's ingest hosts. The ingest hosts come from the DSN format (`o<org>.ingest[.<region>].sentry.io`), not from a DSN, so no key is in the file. No third-party fonts (they are self-hosted) and no inline scripts. Violations are posted to `POST /api/csp-report`, which logs them.
- **Enforce after a week of clean reports.** Once the logs show a week with no violations from real use, rename the header to `Content-Security-Policy` in a follow-up PR.

## Considered options

- **Synchronizer or double-submit CSRF token:** rejected for now. Only two endpoints use the cookie, and the Origin check protects them without new client code or a second cookie.
- **Enforcing the CSP straight away:** rejected. A missed source (an avatar host, a Sentry region) would break pages for real users, and report-only shows what would break first.

## Consequences

- Avatar images from other hosts will show up as reported violations. That is the first thing to decide before enforcing: allow those hosts, or proxy or upload avatars.
- A new deployment domain has to be same-origin with the API or listed in `FRONTEND_URL`/`CORS_ALLOWED_ORIGINS`, or refresh returns `403`.
