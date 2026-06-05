# Security Policy

Push Booster is currently alpha software. Do not deploy it with production secrets or public internet exposure without an independent security review.

## Reporting a Vulnerability

Please report vulnerabilities privately to the maintainers. If this repository is mirrored to GitHub, use private security advisories when available. Otherwise contact the repository owner directly.

Do not publish exploit details until maintainers have had a reasonable opportunity to investigate and prepare a fix.

## Supported Versions

Only the latest `main` branch is currently considered for security fixes.

## Security Expectations

- Rotate all local credentials before any real deployment.
- Set a strong `AUTH_JWT_SECRET`.
- Use real VAPID keys and keep private keys out of client-side code.
- Restrict PostgreSQL, ClickHouse, Redis and Redpanda network exposure.
- Configure HTTPS and trusted origins for public SDK, subscribe and service-worker endpoints.
- Review provider sync URLs before enabling private or internal network fetches.
