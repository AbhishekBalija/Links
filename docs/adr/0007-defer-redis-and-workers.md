# Defer Redis and Workers

Decision: Do not introduce Redis or background workers in MVP unless the need becomes real.

Reason:

- PostgreSQL and synchronous APIs are enough for the first version.
- Extra infrastructure increases operational burden.
