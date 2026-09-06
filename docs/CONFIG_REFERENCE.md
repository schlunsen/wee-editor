# Configuration Reference

Quick reference for all configuration options in Wee.

## File Locations

- **Config**: `~/.claude/wee/config.json`
- **API Key**: `~/.claude/wee/.secret`
- **Database**: `~/.claude/wee/wee.db`
- **Logs**: `~/.claude/wee/logs/`

## Configuration Schema

### Complete config.json Example

```json
{
  "tls": {
    "enabled": true,
    "cert_path": "~/.claude/wee/certs/cert.pem",
    "key_path": "~/.claude/wee/certs/key.pem"
  },
  "auth": {
    "enabled": true,
    "api_key_path": "~/.claude/wee/.secret",
    "user_auth_enabled": true,
    "require_login": false,
    "session_timeout_hours": 24,
    "oauth": {
      "enabled": true,
      "providers": {
        "google": {
          "enabled": true,
          "client_id": "YOUR-CLIENT-ID.apps.googleusercontent.com",
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
        "user@company.com"
      ],
      "allowed_domains": [
        "*@company.com",
        "*@example.org"
      ],
      "deny_message": "Your email is not authorized"
    },
    "admin_users": [
      "admin@example.com",
      "admin"
    ]
  },
  "server": {
    "port": 3333,
    "host": "127.0.0.1",
    "quiet": false,
    "verbose": false
  },
  "cors": {
    "allowed_origins": [
      "http://localhost:3333",
      "https://localhost:3333"
    ]
  },
  "agent": {
    "model": "claude-sonnet-4-5-20250929",
    "max_concurrent_sessions": 10,
    "session_retention_days": 30,
    "cleanup_enabled": true,
    "cleanup_interval_hours": 24
  },
  "terminal": {
    "enabled": true,
    "default_shell": "/bin/zsh",
    "max_concurrent_terminals": 5,
    "idle_timeout_minutes": 30,
    "max_output_buffer_kb": 1024
  }
}
```

## Environment Variable Overrides

All configuration can be overridden via environment variables:

### Pattern: `WEE_{SECTION}_{SUBSECTION}_{FIELD}`

```bash
# Server
export WEE_SERVER_PORT=3333
export WEE_SERVER_HOST=127.0.0.1
export WEE_SERVER_VERBOSE=true

# Auth
export WEE_AUTH_ENABLED=true
export WEE_AUTH_USER_AUTH_ENABLED=true
export WEE_AUTH_REQUIRE_LOGIN=false
export WEE_AUTH_SESSION_TIMEOUT_HOURS=24

# OAuth
export WEE_AUTH_OAUTH_ENABLED=true
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="secret"

# Access Control
export WEE_AUTH_ACCESS_CONTROL_ENABLED=true
export WEE_AUTH_ACCESS_CONTROL_ALLOW_FIRST_USER=true
export WEE_AUTH_ACCESS_CONTROL_ALLOWED_USERS="user@example.com,admin"
export WEE_AUTH_ACCESS_CONTROL_ALLOWED_DOMAINS="*@company.com"
export WEE_AUTH_ACCESS_CONTROL_DENY_MESSAGE="Access denied"

# Admin Users
export WEE_AUTH_ADMIN_USERS="admin@example.com,root"

# Agent
export WEE_AGENT_MODEL="claude-sonnet-4-5-20250929"
export WEE_AGENT_MAX_CONCURRENT_SESSIONS=10

# Terminal
export WEE_TERMINAL_ENABLED=true
export WEE_TERMINAL_DEFAULT_SHELL="/bin/bash"
```

## Configuration Defaults

### TLS
- `enabled`: `true`
- Auto-generates self-signed certificates on first run

### Auth
- `enabled`: `true`
- `user_auth_enabled`: `false` (password auth disabled by default)
- `require_login`: `false` (GET requests allowed without login)
- `session_timeout_hours`: `24`

### OAuth
- `enabled`: `false` (disabled by default)
- All providers disabled by default
- `auto_create_user`: `true`
- `auto_enable_mfa`: `false`

### Access Control
- `enabled`: `false` (disabled by default)
- `allow_first_user`: `true`
- `allowed_users`: `[]` (empty)
- `allowed_domains`: `[]` (empty)

### Server
- `port`: `3333`
- `host`: `127.0.0.1` (localhost only for security)
- `quiet`: `false`
- `verbose`: `false`

### CORS
- `allowed_origins`: localhost:3333 (http/https)

### Agent
- `model`: `claude-sonnet-4-5-20250929`
- `max_concurrent_sessions`: `10`
- `session_retention_days`: `30`
- `cleanup_enabled`: `true`
- `cleanup_interval_hours`: `24`

### Terminal
- `enabled`: `true`
- `default_shell`: `/bin/zsh` (falls back to bash/sh)
- `max_concurrent_terminals`: `5`
- `idle_timeout_minutes`: `30`
- `max_output_buffer_kb`: `1024`

## Boolean Value Parsing

Environment variables accept multiple boolean representations:

```bash
# TRUE values
export WEE_AUTH_ENABLED=true
export WEE_AUTH_ENABLED=1
export WEE_AUTH_ENABLED=yes
export WEE_AUTH_ENABLED=on

# FALSE values (anything else)
export WEE_AUTH_ENABLED=false
export WEE_AUTH_ENABLED=0
export WEE_AUTH_ENABLED=""
```

## Comma-Separated Lists

For list-type settings, use comma-separated values:

```bash
export WEE_AUTH_ACCESS_CONTROL_ALLOWED_USERS="user1@example.com,user2@example.com,admin"
export WEE_AUTH_ACCESS_CONTROL_ALLOWED_DOMAINS="*@company.com,*@example.org"
export WEE_AUTH_ADMIN_USERS="admin@company.com,superuser"
```

## Configuration Loading Order

1. Default values (hardcoded)
2. `config.json` (if exists)
3. Environment variable overrides
4. Secrets from environment vars

## OAuth Secrets

OAuth client secrets MUST be set via environment variable:

```bash
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-secret"
export WEE_OAUTH_GITHUB_CLIENT_SECRET="your-secret"
export WEE_OAUTH_CUSTOM_CLIENT_SECRET="your-secret"
```

Format: `WEE_OAUTH_{PROVIDER}_CLIENT_SECRET`

**Never** store secrets in `config.json`!

## Minimum Required Configuration

### For Basic Usage

```json
{
  "server": {
    "port": 3333
  }
}
```

### For Password Authentication

```json
{
  "auth": {
    "enabled": true,
    "user_auth_enabled": true,
    "require_login": true
  }
}
```

### For OAuth with Google

```json
{
  "auth": {
    "enabled": true,
    "user_auth_enabled": true,
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

Plus environment variable:
```bash
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-secret"
```

### For Access Control

```json
{
  "auth": {
    "access_control": {
      "enabled": true,
      "allowed_domains": ["*@company.com"]
    }
  }
}
```

## Startup Diagnostics

When starting the server with verbose mode, configuration is logged:

```bash
./wee --analytics --verbose
```

Check the output for:
- Configuration loaded from file
- Environment variable overrides applied
- OAuth configuration status
- Access control status
- TLS certificate status
- Database initialization

## Troubleshooting Configuration

### Check Effective Configuration

View what configuration is actually loaded:

```bash
# Start in verbose mode
./wee --analytics --verbose 2>&1 | grep -i config

# Check specific setting
cat ~/.claude/wee/config.json | jq .auth.oauth.enabled
```

### Validate Configuration

The server validates configuration on startup and will error if:

- OAuth provider missing required fields
- OAuth provider enabled but client_id not set
- OAuth provider enabled but client_secret not available
- Invalid domain patterns in access control

### Reset to Defaults

```bash
# Remove config to reset to defaults
rm ~/.claude/wee/config.json

# Server will recreate defaults on next start
./wee --analytics
```

### View Saved Configuration

```bash
cat ~/.claude/wee/config.json | jq .
```

## Performance Tuning

### For High Concurrency

```json
{
  "agent": {
    "max_concurrent_sessions": 50
  },
  "terminal": {
    "max_concurrent_terminals": 20
  }
}
```

### For Low Memory Environments

```json
{
  "agent": {
    "max_concurrent_sessions": 3
  },
  "terminal": {
    "max_concurrent_terminals": 2,
    "max_output_buffer_kb": 512
  }
}
```

### For Long Sessions

```json
{
  "agent": {
    "session_retention_days": 90
  },
  "auth": {
    "session_timeout_hours": 72
  }
}
```

## Production Checklist

- [ ] Use proper TLS certificates (not self-signed)
- [ ] Set strong OAuth client secrets (env var)
- [ ] Enable access control whitelist
- [ ] Configure appropriate session timeout
- [ ] Set reasonable max concurrent limits
- [ ] Enable cleanup for old sessions
- [ ] Configure CORS allowed origins
- [ ] Set require_login to true
- [ ] Monitor auth audit logs
