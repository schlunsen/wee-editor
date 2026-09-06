# OAuth2 & User/Domain Whitelist Access Control

Complete guide for setting up OAuth2 authentication with Google and configuring user/domain access control in Wee.

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Google OAuth Setup](#google-oauth-setup)
4. [Configuration](#configuration)
5. [Environment Variables](#environment-variables)
6. [Access Control Rules](#access-control-rules)
7. [Usage Examples](#usage-examples)
8. [API Endpoints](#api-endpoints)
9. [Troubleshooting](#troubleshooting)
10. [Security Considerations](#security-considerations)

## Overview

Wee now supports:

- **OAuth2 Authentication**: Secure login via Google (extensible to other providers)
- **User/Domain Whitelisting**: Restrict access by specific emails or domain patterns
- **Auto-user Creation**: Automatically create user accounts for OAuth users
- **Access Audit Logging**: Track all authentication attempts and access denials
- **Admin User Detection**: Automatically assign admin status to configured users

## Quick Start

### 1. Enable OAuth in Configuration

```bash
# Edit ~/.claude/wee/config.json
{
  "auth": {
    "user_auth_enabled": true,
    "oauth": {
      "enabled": true,
      "providers": {
        "google": {
          "enabled": true,
          "client_id": "YOUR-GOOGLE-CLIENT-ID.apps.googleusercontent.com",
          "redirect_uri": "https://localhost:3333/api/auth/oauth/callback/google",
          "scopes": ["openid", "email", "profile"],
          "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
          "token_url": "https://oauth2.googleapis.com/token",
          "user_info_url": "https://openidconnect.googleapis.com/v1/userinfo",
          "auto_create_user": true,
          "auto_enable_mfa": false
        }
      }
    },
    "access_control": {
      "enabled": true,
      "allow_first_user": true,
      "allowed_users": [
        "admin@example.com",
        "john.doe@company.com"
      ],
      "allowed_domains": [
        "*@company.com",
        "*@example.org"
      ],
      "deny_message": "Your email is not authorized to access this service"
    },
    "admin_users": [
      "admin@example.com",
      "admin"
    ]
  }
}
```

### 2. Set OAuth Client Secret

```bash
# Store the client secret in environment variable (never in config.json)
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-client-secret-here"

# Start the server
./wee --analytics
```

### 3. Access the Dashboard

Navigate to `https://localhost:3333/login` and click "Sign in with Google"

## Google OAuth Setup

### Step 1: Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project
3. Enable the "Google+ API"

### Step 2: Create OAuth 2.0 Credentials

1. Go to "APIs & Services" → "Credentials"
2. Click "Create Credentials" → "OAuth client ID"
3. Choose "Web application"
4. Add authorized redirect URIs:
   - `https://localhost:3333/api/auth/oauth/callback/google` (development)
   - `https://your-domain.com/api/auth/oauth/callback/google` (production)
5. Copy the Client ID and Client Secret

### Step 3: Configure Wee

Update your `~/.claude/wee/config.json`:

```json
{
  "auth": {
    "oauth": {
      "enabled": true,
      "providers": {
        "google": {
          "enabled": true,
          "client_id": "YOUR-CLIENT-ID.apps.googleusercontent.com"
        }
      }
    }
  }
}
```

Set the client secret as an environment variable:

```bash
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-client-secret"
./wee --analytics
```

## Configuration

### config.json Structure

```json
{
  "auth": {
    "enabled": true,
    "user_auth_enabled": true,
    "require_login": true,
    "session_timeout_hours": 24,
    "oauth": {
      "enabled": true,
      "providers": {
        "google": {
          "enabled": true,
          "client_id": "string",
          "redirect_uri": "https://localhost:3333/api/auth/oauth/callback/google",
          "scopes": ["openid", "email", "profile"],
          "auth_url": "string",
          "token_url": "string",
          "user_info_url": "string",
          "auto_create_user": true,
          "auto_enable_mfa": false
        }
      }
    },
    "access_control": {
      "enabled": true,
      "allow_first_user": true,
      "allowed_users": ["email@example.com", "username"],
      "allowed_domains": ["*@company.com", "*@example.org"],
      "deny_message": "Custom message shown to denied users"
    },
    "admin_users": ["admin@example.com", "admin"]
  }
}
```

### OAuth Provider Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `enabled` | Yes | Enable/disable this provider |
| `client_id` | Yes | OAuth application client ID |
| `client_secret` | Yes | OAuth application secret (from env var) |
| `redirect_uri` | Yes | Callback URL after OAuth authorization |
| `scopes` | Yes | OAuth scopes to request (e.g., openid, email, profile) |
| `auth_url` | Yes | Authorization endpoint URL |
| `token_url` | Yes | Token endpoint URL |
| `user_info_url` | Yes | User info endpoint URL |
| `auto_create_user` | No | Auto-create users on first OAuth login |
| `auto_enable_mfa` | No | Automatically enable MFA for OAuth users |

### Access Control Configuration

| Field | Required | Description |
|-------|----------|-------------|
| `enabled` | Yes | Enable/disable access control |
| `allow_first_user` | No | Allow first user to bypass whitelist (default: true) |
| `allowed_users` | No | List of allowed usernames or emails (exact match) |
| `allowed_domains` | No | List of allowed email domains (wildcard patterns) |
| `deny_message` | No | Custom message shown when access is denied |

### Admin Users Configuration

```json
{
  "auth": {
    "admin_users": [
      "admin@company.com",        // Exact email match
      "admin"                     // Exact username match
    ]
  }
}
```

Users in the `admin_users` list are automatically created with admin privileges.

## Environment Variables

All configuration can be overridden via environment variables using the pattern: `WEE_{SECTION}_{SUBSECTION}_{FIELD}`

### OAuth Settings

```bash
# Enable/disable OAuth
export WEE_AUTH_OAUTH_ENABLED=true

# Google OAuth secrets (REQUIRED)
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-secret-here"

# Other OAuth provider secrets follow same pattern
export WEE_OAUTH_GITHUB_CLIENT_SECRET="..."
```

### Access Control Settings

```bash
# Enable/disable access control
export WEE_AUTH_ACCESS_CONTROL_ENABLED=true

# Allow first user to bypass restrictions
export WEE_AUTH_ACCESS_CONTROL_ALLOW_FIRST_USER=true

# Allowed users (comma-separated)
export WEE_AUTH_ACCESS_CONTROL_ALLOWED_USERS="admin@example.com,john.doe@company.com"

# Allowed domains (comma-separated)
export WEE_AUTH_ACCESS_CONTROL_ALLOWED_DOMAINS="*@company.com,*@example.org"

# Custom deny message
export WEE_AUTH_ACCESS_CONTROL_DENY_MESSAGE="Your organization is not authorized"
```

### Admin Users

```bash
# Comma-separated list of admin users
export WEE_AUTH_ADMIN_USERS="admin@example.com,root"
```

### Server Settings

```bash
export WEE_SERVER_PORT=3333
export WEE_SERVER_HOST=127.0.0.1
```

### Auth Settings

```bash
export WEE_AUTH_ENABLED=true
export WEE_AUTH_USER_AUTH_ENABLED=true
export WEE_AUTH_REQUIRE_LOGIN=true
export WEE_AUTH_SESSION_TIMEOUT_HOURS=24
```

## Access Control Rules

### Domain Pattern Matching

Domains support wildcard patterns for flexible authorization:

```javascript
// Exact domain match
"*@example.com"        // user@example.com ✓

// Subdomain matching
"*@example.com"        // user@dept.example.com ✓
"*@example.com"        // user@team.dept.example.com ✓

// Multi-level domains
"*@company.co.uk"      // user@company.co.uk ✓
"*@company.co.uk"      // user@dept.company.co.uk ✓

// Exact email match
"admin@special.org"    // admin@special.org ✓
```

### Validation Flow

1. **Password Login**:
   - User enters username/password
   - Password verified
   - Email/username checked against whitelist
   - MFA verified (if enabled)
   - Session created

2. **OAuth Login**:
   - OAuth provider authorizes
   - Email checked against whitelist
   - User auto-created (if enabled)
   - Admin status assigned (if matched)
   - Session created

3. **Access Denied**:
   - User logs access denial event
   - Custom deny message shown
   - No session created

## Usage Examples

### Allow All Users in a Company Domain

```json
{
  "access_control": {
    "enabled": true,
    "allowed_domains": ["*@company.com"]
  }
}
```

### Allow Specific Users and Domains

```json
{
  "access_control": {
    "enabled": true,
    "allowed_users": [
      "external.partner@partner.com"
    ],
    "allowed_domains": [
      "*@company.com",
      "*@subsidiary.org"
    ]
  }
}
```

### Allow First User, Then Lock Down

```json
{
  "access_control": {
    "enabled": true,
    "allow_first_user": true,
    "allowed_domains": ["*@company.com"]
  }
}
```

First user can access without restriction, then whitelist is enforced for all subsequent users.

### Multi-Admin Setup

```json
{
  "auth": {
    "admin_users": [
      "admin@company.com",
      "manager@company.com",
      "ops@company.com"
    ]
  },
  "access_control": {
    "enabled": true,
    "allowed_domains": ["*@company.com"]
  }
}
```

### Environment-Based Configuration

```bash
#!/bin/bash

if [ "$ENV" = "production" ]; then
  export WEE_AUTH_ACCESS_CONTROL_ENABLED=true
  export WEE_AUTH_ACCESS_CONTROL_ALLOWED_DOMAINS="*@company.com"
  export WEE_AUTH_ADMIN_USERS="admin@company.com"
else
  export WEE_AUTH_ACCESS_CONTROL_ENABLED=false
fi

./wee --analytics
```

## API Endpoints

### OAuth Endpoints

#### Initiate OAuth Login

```bash
GET /api/auth/oauth/authorize?provider=google
```

**Response:**
```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "provider": "google",
  "state": "secure-random-token"
}
```

#### OAuth Callback

```bash
POST /api/auth/oauth/callback
GET /api/auth/oauth/callback/google?code=...&state=...
```

**Response (Success):**
```json
{
  "token": "session-token",
  "username": "auto-generated-username",
  "expires_at": "2024-11-09T12:00:00Z"
}
```

**Response (Access Denied):**
```json
{
  "error": "Your email domain or address is not authorized for this service"
}
```

### Authentication Endpoints

#### Check Auth Status

```bash
GET /api/auth/status
```

**Response:**
```json
{
  "enabled": true,
  "authenticated": false,
  "require_login": true
}
```

#### Login (Password)

```bash
POST /api/auth/login
```

**Body:**
```json
{
  "username": "user@example.com",
  "password": "password"
}
```

## Troubleshooting

### OAuth Login Not Working

**Issue**: OAuth button appears but clicking it does nothing

**Solution**:
1. Verify `WEE_OAUTH_GOOGLE_CLIENT_SECRET` is set:
   ```bash
   echo $WEE_OAUTH_GOOGLE_CLIENT_SECRET
   ```
2. Check config.json has correct `client_id`
3. Verify `redirect_uri` matches Google Cloud Console settings
4. Check browser console for errors

### "Access Denied" on OAuth Login

**Issue**: OAuth succeeds but access is denied

**Solution**:
1. Check access control config:
   ```bash
   cat ~/.claude/wee/config.json | jq .auth.access_control
   ```
2. Verify your email matches allowed domains/users
3. Check audit logs:
   ```bash
   sqlite3 ~/.claude/wee/wee.db "SELECT * FROM auth_audit_logs ORDER BY created_at DESC LIMIT 10;"
   ```

### First User Bootstrap

**Issue**: Can't create first user with access control enabled

**Solution**:
1. Set `allow_first_user: true` in config:
   ```json
   {
     "access_control": {
       "enabled": true,
       "allow_first_user": true
     }
   }
   ```
2. First user bypasses whitelist restrictions
3. Subsequent users must match whitelist

### Admin User Not Assigned

**Issue**: User isn't getting admin privileges

**Solution**:
1. Verify user is in `admin_users` list:
   ```bash
   cat ~/.claude/wee/config.json | jq .auth.admin_users
   ```
2. Check both username and email are listed
3. Matching is case-insensitive but must be exact
4. User must be created/logged in after admin config

## Security Considerations

### Secrets Management

**CRITICAL**: Never store OAuth client secrets in `config.json`:

```bash
# ✓ CORRECT - Use environment variables
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="secret"

# ✗ WRONG - Never put secrets in config.json
# {
#   "oauth": {
#     "client_secret": "secret"  // Don't do this!
#   }
# }
```

### CSRF Protection

OAuth implementation includes CSRF protection:

- State tokens are generated for each OAuth flow
- State tokens expire after 15 minutes
- State tokens are single-use (consumed on validation)
- Nonce tokens added for OpenID Connect compliance

### Access Audit Logging

All authentication events are logged to database:

```sql
SELECT * FROM auth_audit_logs
WHERE action IN ('password_login_success', 'password_login_denied', 'oauth_login_success', 'oauth_login_denied')
ORDER BY created_at DESC
LIMIT 20;
```

Log columns:
- `username`: Username (if available)
- `email`: Email address
- `action`: Type of action (login, deny, etc.)
- `reason`: Reason for denial (if applicable)
- `ip_address`: User's IP address
- `user_agent`: Browser user agent
- `created_at`: Timestamp

### TLS/HTTPS

OAuth requires HTTPS in production:

- Development: Self-signed certs (auto-generated)
- Production: Use proper CA-signed certificates
- Set `WEE_DISABLE_TLS=false` (default)

### Session Security

Sessions are secured with:

- HTTP-only cookies (prevent XSS theft)
- Secure flag set (HTTPS only)
- SameSite=Lax (CSRF protection)
- 24-hour expiration (configurable)

## Advanced Configuration

### Custom Deny Message

```json
{
  "access_control": {
    "enabled": true,
    "allowed_domains": ["*@company.com"],
    "deny_message": "Sorry, only @company.com emails are allowed. Contact your admin for access."
  }
}
```

### Gradual Rollout

Start permissive, then restrict:

```json
{
  "access_control": {
    "enabled": false  // Phase 1: Allow all
  }
}
```

Then later:

```json
{
  "access_control": {
    "enabled": true,
    "allow_first_user": true,
    "allowed_domains": ["*@company.com"]  // Phase 2: Enforce whitelist
  }
}
```

### Multiple Domains

```json
{
  "access_control": {
    "allowed_domains": [
      "*@company.com",
      "*@subsidiary.org",
      "*@partner.io"
    ]
  }
}
```

### Mixed Authentication

Support both password and OAuth:

```json
{
  "auth": {
    "user_auth_enabled": true,
    "oauth": {
      "enabled": true,
      "providers": {
        "google": { "enabled": true }
      }
    }
  }
}
```

Users can choose either method at login page.

## Support & Troubleshooting

For issues or questions:

1. Check logs: `~/.claude/wee/logs/`
2. View audit trail: `sqlite3 ~/.claude/wee/wee.db "SELECT * FROM auth_audit_logs"`
3. Enable verbose mode: `./wee --analytics --verbose`
4. Check configuration: `cat ~/.claude/wee/config.json | jq .auth`
