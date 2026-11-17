# Security policy

## Supported versions

Security fixes are applied to the latest release on the `main` branch. Users
should upgrade to the newest available version before reporting an issue.

## Reporting a vulnerability

Please do not disclose security vulnerabilities in a public issue. Report them
privately through GitHub Security Advisories for this repository. Include:

- the affected version or commit;
- a clear description of the impact;
- reproducible steps or a minimal proof of concept; and
- any suggested mitigation.

Allow reasonable time for investigation and a fix before public disclosure.

## Scope

Security reports are especially useful for token forgery, replay protection,
answer leakage, cross-request challenge confusion, secret handling, and
verification-state races. CAPTCHA-solving research and abuse-resistance
observations are also welcome, but Gotcha is not intended to replace rate
limiting, MFA, account protection, or server-side authorization.
