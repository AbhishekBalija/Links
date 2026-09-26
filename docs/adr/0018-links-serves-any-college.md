# LINKS serves any college, with MITT as the pilot

LINKS started as MITT's campus hub, but most colleges share the same structure: departments, HODs, a principal, a placement office, and USN-based student identities. So the product, docs and domain language describe "the college" rather than one institution, and MITT is the pilot college the first deployment is built and tested with.

## Consequences

- Some pilot-college data is still built in, on purpose, as the working example: the VTU college code `4MN` in USN validation, and the B.E. department list seeded by migrations 008 and 011. ADRs 0003 and 0016 describe the decisions behind them. Making these per-college settings is future work, to do when a second college is onboarded.
- Reference notes about the pilot college, including staff contacts, are kept locally in `docs/local/` (git-ignored) instead of in the public repo.
