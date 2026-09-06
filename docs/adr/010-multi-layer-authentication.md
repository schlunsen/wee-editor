# ADR 010: Multi-Layered Authentication Strategy

## Status

Accepted

## Date

2024-01-01

## Context

Wee operates in diverse environments:

- **Local development**: Running on `localhost`, where minimal auth friction is desired
- **Team/shared server**: Exposed on a network, requiring user identification
- **Remote access via ngrok**: Publicly accessible, demanding strong authentication
- **API integrations**: Programmatic access from scripts and CI/CD pipelines

A one-size-fits-all approach would either be too restrictive for local use or too permissive for shared/remote deployments.

## Decision

We implemented a **multi-layered authentication system** that can be progressively enabled based on deployment scenario.

**Layer 1 - API Key (Always Active):**

- Generated on first startup, stored in `~/.claude/wee/.secret`
- Bearer token in `Authorization` header
- Used for programmatic API access and MCP tool authentication
- Manageable via CLI commands

**Layer 2 - User Authentication (Optional):**

- Username/password with session cookies
- Session timeout configurable (default: 24 hours)
- User management in SQLite (`users` table)
- Enabled via `auth.user_auth_enabled` config

**Layer 3 - OAuth 2.0 (Optional):**

- Google and GitHub OAuth providers supported
- Auto-user creation on first OAuth login
- Email-based mapping to existing accounts
- Callback handling at `/api/auth/oauth/callback/{provider}`

**Layer 4 - Multi-Factor Authentication (Optional):**

- TOTP (Time-based One-Time Password)
- Backup codes for recovery
- QR code generation for authenticator apps
- Per-user enforcement

**Layer 5 - Access Control (Optional):**

- Domain-based allowlisting (`*@company.com`)
- Individual user allowlisting
- Admin role designation
- First-user auto-admin option
- Custom deny messages

**Middleware stack order:** CORS -> Session Auth -> API Key Auth -> Access Control -> Rate Limiting -> Handler

## Consequences

### Positive

- Zero-friction for local development (API key auto-generated, no login required)
- Progressive security - enable layers as exposure increases
- OAuth integration enables SSO with existing identity providers
- MFA support meets enterprise security requirements
- Access control supports team deployments with domain-based policies
- API key enables automation without user session management
- Audit logging tracks all authentication events

### Negative

- Multiple auth paths increase the attack surface to audit
- Configuration complexity - users must understand which layers to enable
- OAuth requires external provider setup (client IDs, secrets, redirect URIs)
- Session management adds state that must be handled during upgrades
- MFA recovery (backup codes) introduces a social engineering vector
- Testing all auth combinations is complex (reflected in security test skip files)
- API key stored in plaintext on disk (mitigated by 0600 permissions)
