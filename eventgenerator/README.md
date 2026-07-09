# Event Generator

The Event Generator polls metrics from CF Log Cache, aggregates them, evaluates scaling rules, and triggers scaling events when thresholds are breached.

## Log Cache Authentication

The Event Generator needs to authenticate with CF's Log Cache to read application metrics. Two authentication modes are supported:

### Client Credentials (default)

Uses a UAA client with `client_credentials` grant. The client needs the `logs.admin` authority. This mode uses the standard go-log-cache OAuth2 client internally.

```yaml
uaa:
  url: https://uaa.sys.example.com
  client_id: autoscaler_client
  client_secret: my-secret
  skip_ssl_validation: false
```

### Password Grant

Uses the `password` grant type with CF user credentials. This is useful when a dedicated UAA client with `logs.admin` is not available — instead, an org manager user with Log Cache access can be used.

The default client ID is `cf` (CF's built-in public UAA client with an empty secret), matching `cf login` behavior.

```yaml
uaa:
  url: https://uaa.sys.example.com
  client_id: cf
  client_secret: ""
  grant_type: password
  username: org-manager@example.com
  password: my-password
  skip_ssl_validation: false
```

**Required fields for password grant:**
- `url` — UAA base URL (e.g., `https://uaa.sys.example.com`); `/oauth/token` is appended automatically
- `grant_type` — must be `password`
- `username` — CF user with access to app metrics via Log Cache
- `password` — user password

**Optional fields:**
- `client_id` — defaults to `cf` if empty
- `client_secret` — empty for the `cf` client (public client)
- `skip_ssl_validation` — defaults to `false`

### How it works

The password grant OAuth2 client:
1. Authenticates using HTTP Basic auth header (`client_id:client_secret`) with username/password in the request body
2. Caches the access token until shortly before expiry (30-second buffer, or half the token lifetime for short-lived tokens)
3. Automatically refreshes on 401 responses with stale-token detection under lock to prevent thundering herd
4. Retries once after a forced token refresh

## Configuration

See [`default_config.json`](./default_config.json) for all available configuration options.
