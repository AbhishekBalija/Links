# Security policy

## Reporting a vulnerability

Please do not open a public issue for security problems. Report them
privately through GitHub: on this repository, go to **Security > Report a
vulnerability**. You will get a response as soon as possible.

Include what you found, how to reproduce it, and its impact if you know it.

## Scope

LINKS stores student personal data: names, Gmail addresses, phone numbers,
USNs, and later placement applications. Anything that exposes this data to
the wrong role, bypasses account approval or activation, or leaks password
hashes, refresh tokens or activation tokens is in scope.

Secrets (database URL, JWT secrets, Resend key, Sentry DSN) live only in
git-ignored `.env.*` files and in the hosting provider's environment
settings. See `docs/security.md` for the project's security rules.
