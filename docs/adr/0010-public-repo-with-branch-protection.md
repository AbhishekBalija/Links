# Make Repo Public with Automated Review and Branch Protection

**Date:** 2026-07-16

Decision: Make the LINKS repository public, add CodeRabbit as an automated PR reviewer, and enforce a GitHub ruleset on `master` that blocks direct pushes — all changes must go through a pull request.

Reason:

- Public visibility invites community contributions and external review.
- CodeRabbit provides fast, consistent automated feedback on every PR, reducing manual review burden.
- Branch protection on `master` prevents accidental force-pushes and ensures every change is reviewed before merging.

Trade-off:

- Requires discipline to keep PRs small and well-described.
- CodeRabbit is an external service — review quality depends on its configuration and availability.
