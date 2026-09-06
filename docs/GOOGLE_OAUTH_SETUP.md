# Google OAuth Setup Guide

Step-by-step guide to set up Google OAuth authentication for Wee.

## Prerequisites

- Google Cloud Account (or create one at https://cloud.google.com)
- Wee running
- A domain or localhost for OAuth callback

## Step 1: Create Google Cloud Project

### 1.1 Create New Project

1. Open [Google Cloud Console](https://console.cloud.google.com)
2. Click on the project dropdown at the top
3. Click "NEW PROJECT"
4. Enter project name: `Wee` (or your preferred name)
5. Click "CREATE"
6. Wait for project to be created (2-3 minutes)

### 1.2 Enable Google+ API

1. In the top search bar, search for "Google+ API"
2. Click the result and click "ENABLE"
3. You may also need to enable:
   - "Google Drive API" (for profile data)
   - "Gmail API" (for email access)

## Step 2: Create OAuth 2.0 Credentials

### 2.1 Create OAuth Consent Screen

1. Go to "APIs & Services" in the left sidebar
2. Click "OAuth consent screen"
3. Choose "External" (unless you have Google Workspace)
4. Click "CREATE"
5. Fill in the form:
   - **App name**: Wee
   - **User support email**: your-email@example.com
   - **Developer contact**: your-email@example.com
6. Click "SAVE AND CONTINUE"
7. For scopes, click "ADD OR REMOVE SCOPES"
8. Search and add:
   - `openid`
   - `email`
   - `profile`
9. Click "UPDATE"
10. Click "SAVE AND CONTINUE" through the remaining screens

### 2.2 Create OAuth Client ID

1. Go to "APIs & Services" → "Credentials"
2. Click "CREATE CREDENTIALS" → "OAuth client ID"
3. Choose "Web application"
4. Enter name: `Wee`
5. Add Authorized redirect URIs:

```
https://localhost:3333/api/auth/oauth/callback/google
```

Or for production:

```
https://your-domain.com/api/auth/oauth/callback/google
```

6. Click "CREATE"
7. A dialog appears with your credentials
8. Copy:
   - **Client ID** (looks like: `123456789.apps.googleusercontent.com`)
   - **Client Secret** (a long string)

**⚠️ IMPORTANT**: Save these credentials securely. You'll need them next.

## Step 3: Configure Wee

### 3.1 Update config.json

Edit `~/.claude/wee/config.json`:

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
    }
  }
}
```

Replace `YOUR-CLIENT-ID` with the Client ID from Google Cloud Console.

### 3.2 Set Client Secret

Store the client secret in an environment variable:

```bash
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-client-secret-here"
```

**IMPORTANT**: Never store the client secret in `config.json`!

### 3.3 (Optional) Configure Access Control

To restrict access to specific domains:

```json
{
  "auth": {
    "access_control": {
      "enabled": true,
      "allowed_domains": [
        "*@company.com",
        "*@example.org"
      ],
      "allow_first_user": true
    }
  }
}
```

### 3.4 (Optional) Configure Admin Users

```json
{
  "auth": {
    "admin_users": [
      "admin@company.com",
      "your-email@company.com"
    ]
  }
}
```

## Step 4: Start the Server

```bash
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-client-secret"
./wee --analytics
```

You should see:

```
🔐 User authentication enabled (0 users)
🔐 MFA (Multi-Factor Authentication) system initialized
🌐 OAuth enabled with Google provider
🔒 Access control enabled
```

## Step 5: Test OAuth Login

### 5.1 Open the Dashboard

Navigate to:

```
https://localhost:3333/login
```

(Accept the self-signed certificate warning)

### 5.2 Click "Sign in with Google"

1. You'll be redirected to Google's login page
2. Sign in with your Google account
3. Click "Continue" to authorize
4. You'll be redirected back to the dashboard

## Troubleshooting

### "OAuth provider 'google' is not enabled"

**Cause**: Provider not enabled or misconfigured in config.json

**Solution**:
```json
{
  "oauth": {
    "enabled": true,
    "providers": {
      "google": {
        "enabled": true
      }
    }
  }
}
```

### "Failed to exchange code for token"

**Cause**: Client secret not set or invalid

**Solution**:
```bash
# Verify secret is set
echo $WEE_OAUTH_GOOGLE_CLIENT_SECRET

# If empty, set it
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="your-secret"
```

### "redirect_uri mismatch"

**Cause**: Callback URL doesn't match Google Cloud Console settings

**Solution**:
1. In Google Cloud Console, check "Authorized redirect URIs"
2. Make sure it matches exactly in config.json
3. Both must use HTTPS (except localhost)

For development with localhost:
```
https://localhost:3333/api/auth/oauth/callback/google
```

### "Your email domain or address is not authorized"

**Cause**: Access control enabled and your email not in whitelist

**Solution**:
```json
{
  "access_control": {
    "enabled": true,
    "allowed_users": ["your-email@example.com"],
    "allowed_domains": ["*@example.com"]
  }
}
```

Or temporarily disable:
```json
{
  "access_control": {
    "enabled": false
  }
}
```

### Google redirects to redirect_uri is not whitelisted

**Cause**: Callback URL not exactly matching

**Solution**:
1. Go to Google Cloud Console
2. APIs & Services → Credentials
3. Find your OAuth 2.0 client
4. Click edit
5. Check "Authorized redirect URIs"
6. Make sure it exactly matches your config.json

Common mistakes:
- `http` vs `https` (must be https except localhost)
- Trailing `/` mismatch
- Port number mismatch
- Exact path mismatch

## Security Considerations

### Client Secret Protection

- Store client secret in `WEE_OAUTH_GOOGLE_CLIENT_SECRET` environment variable
- Never commit it to version control
- Rotate periodically
- Don't share with untrusted parties

### Redirect URI Security

- Must use HTTPS in production
- Use exact URL matching (no wildcards)
- Keep redirect_uri consistent across Google Cloud and config.json

### Session Management

- Sessions expire after 24 hours (configurable)
- User can log out to clear session
- Tokens are HTTP-only (prevent XSS theft)
- Secure flag set (HTTPS only)

### Access Control

- Use domain patterns for company-wide access
- Whitelist specific domains instead of allowing all
- Monitor auth audit logs for suspicious activity

```bash
# View recent login attempts
sqlite3 ~/.claude/wee/wee.db \
  "SELECT username, email, action, reason, created_at FROM auth_audit_logs ORDER BY created_at DESC LIMIT 20;"
```

## Production Deployment

### 1. Update Redirect URI

In Google Cloud Console:
1. Change redirect_uri to production domain:
   ```
   https://your-domain.com/api/auth/oauth/callback/google
   ```

2. Update config.json:
   ```json
   {
     "oauth": {
       "providers": {
         "google": {
           "redirect_uri": "https://your-domain.com/api/auth/oauth/callback/google"
         }
       }
     }
   }
   ```

### 2. Use Proper TLS Certificate

Instead of self-signed, use CA-signed certificate:

```bash
./wee cert --use-ca-cert /path/to/cert.pem /path/to/key.pem
```

### 3. Configure Access Control

```json
{
  "access_control": {
    "enabled": true,
    "allow_first_user": false,
    "allowed_domains": ["*@company.com"],
    "admin_users": ["admin@company.com"]
  }
}
```

### 4. Set Environment Variables

```bash
export WEE_OAUTH_GOOGLE_CLIENT_SECRET="production-secret"
export WEE_SERVER_HOST="0.0.0.0"  # Listen on all interfaces
export WEE_AUTH_REQUIRE_LOGIN=true
./wee --analytics
```

### 5. Enable Audit Logging

Monitor access attempts:

```bash
# Check audit log regularly
watch -n 5 'sqlite3 ~/.claude/wee/wee.db \
  "SELECT COUNT(*), action FROM auth_audit_logs GROUP BY action;"'
```

## Multiple Domains

To allow multiple domains:

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

## Testing with Different Google Accounts

1. Use OAuth test users in Google Cloud Console
2. Or use separate Google accounts for testing
3. Each account must match access control rules

## Rotating Client Secret

1. In Google Cloud Console, create new OAuth client ID
2. Update config.json with new Client ID
3. Set new `WEE_OAUTH_GOOGLE_CLIENT_SECRET`
4. Restart the server
5. Delete old OAuth client from Google Cloud Console

## Additional Resources

- [Google OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [Google Cloud Console](https://console.cloud.google.com)
- [OpenID Connect Documentation](https://openid.net/connect/)

## Support

If you encounter issues:

1. Check browser console for errors (F12)
2. Check server logs: `~/.claude/wee/logs/`
3. Enable verbose mode: `./wee --analytics --verbose`
4. Check audit log:
   ```bash
   sqlite3 ~/.claude/wee/wee.db "SELECT * FROM auth_audit_logs ORDER BY created_at DESC LIMIT 10;"
   ```
