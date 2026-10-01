# Getting in without passwords: Google sign-in and an email code, the class list decides

LINKS signed people in with a password chosen when requesting access and again at Activation, had no way to recover a forgotten one, and stored password hashes. The owner decided (spec #129) that LINKS, built for any college, signs people in with **Google** or a **one-time email code**, with no password form, and that a Department's class list (the import, #17), or a staff invite, decides who may get in; anyone not on a list can still send an Access request (ADR 0025).

## Considered options

- **Keep passwords and add a reset flow:** rejected. It keeps the double password, the reset flow, the stored hashes and password reuse, for no gain over Google and email codes.
- **Google only:** rejected. A product for any college meets colleges on Microsoft 365 and students without a Google account; the email code covers them with the same rules.
- **Trust a college email domain:** rejected as the only check. A domain also lets in staff, alumni and anyone else with an address there; the class list decides, the domain at most helps match.
- **Email link instead of a code:** rejected. On phones a link often opens in a different browser from the one signing in; a code is typed where the sign-in started.

## Consequences

- The principal and admins sign in with Google only, so their accounts get Google's 2-step protection. The first admin of a college is created from the command line.
- Passwords, Activation emails and tokens are removed once the new screens are live (#136).
- The e2e suite signs in through a test-only endpoint (`ENABLE_TEST_SIGN_IN`, refused unless `APP_ENV=local`), so it doesn't depend on how people really sign in (#130).
- Vercel preview URLs can't be registered with Google (no wildcards), so previews use the email code.
