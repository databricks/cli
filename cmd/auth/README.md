# Auth challenge (happy path)

Simplified description of [PKCE](https://oauth.net/2/pkce/) implementation:

```mermaid
sequenceDiagram
    autonumber
    actor User

    User ->> CLI: type `databricks auth login HOST`
    CLI ->>+ HOST: request OIDC endpoints
    HOST ->>- CLI: auth & token endpoints
    CLI ->> CLI: start embedded server to consume redirects (lock)
    CLI -->>+ Auth Endpoint: open browser with RND1 + SHA256(RND2)

    User ->>+ Auth Endpoint: Go through SSO
    Auth Endpoint ->>- CLI: AUTH CODE + 'RND1 (redirect)

    CLI ->>+ Token Endpoint: Exchange: AUTH CODE + RND2
    Token Endpoint ->>- CLI: Access Token (JWT) + refresh + expiry
    CLI ->> Token cache: Save Access Token (JWT) + refresh + expiry
    CLI ->> User: success
```

# Token refresh (happy path)

```mermaid
sequenceDiagram
    autonumber
    actor User

    User ->> CLI: type `databricks token HOST`

    CLI ->> CLI: acquire lock (same local addr as redirect server)
    CLI ->>+ Token cache: read token

    critical token not expired
    Token cache ->>- User: JWT (without refresh)

    option token is expired
    CLI ->>+ HOST: request OIDC endpoints
    HOST ->>- CLI: auth & token endpoints
    CLI ->>+ Token Endpoint: refresh token
    Token Endpoint ->>- CLI: JWT (refreshed)
    CLI ->> Token cache: save JWT (refreshed)
    CLI ->> User: JWT (refreshed)

    option no auth for host
    CLI -X User: no auth configured
    end
```

# Device code login (experimental)

`databricks auth login --use-device-code` logs in from a machine without a browser.
It is a proof of concept: the flag is hidden and requires
`DATABRICKS_CLI_DEVICE_CODE=true`. Only workspace hosts are supported. See
[docs/device-code-login.md](../../docs/device-code-login.md) for the user guide.

Databricks implements a variant of [RFC 8628](https://datatracker.ietf.org/doc/html/rfc8628):
instead of polling the token endpoint until the user approves the login, the
browser shows the user a confirmation code, which the CLI sends together with
the device code in a single token request.

```mermaid
sequenceDiagram
    autonumber
    actor User

    User ->> CLI: type `databricks auth login HOST --use-device-code`
    CLI ->>+ HOST: request OIDC endpoints
    HOST ->>- CLI: token & device_authorization endpoints
    CLI ->>+ Device Auth Endpoint: client_id + scopes
    Device Auth Endpoint ->>- CLI: DEVICE CODE + USER CODE + verification URI
    CLI ->> User: show verification URI + USER CODE

    User ->>+ Verification Page: open on another device, go through SSO
    User ->> Verification Page: enter USER CODE and approve
    Verification Page ->>- User: CONFIRMATION CODE

    User ->> CLI: type CONFIRMATION CODE
    CLI ->>+ Token Endpoint: Exchange: DEVICE CODE + CONFIRMATION CODE
    Token Endpoint ->>- CLI: Access Token (JWT) + refresh + expiry
    CLI ->> Token cache: Save Access Token (JWT) + refresh + expiry
    CLI ->> User: success
```

Notes on the server behavior the CLI relies on:

- Workspaces advertise `device_authorization_endpoint` in
  `/oidc/.well-known/oauth-authorization-server` only when device code login
  is enabled for them. The CLI fails early if it is missing.
- The token request carries the confirmation code in the `confirmation_code`
  form parameter. A wrong confirmation code does not consume the device code;
  a successful exchange does. The server answers a wrong confirmation code
  with `invalid_request`, and the CLI lets the user enter it up to 3 times in
  total.
- Approving the login in the browser extends the device code's lifetime, so
  the CLI does not enforce the `expires_in` from the device authorization
  response.
- The refresh token is single-use (rotating) and has an absolute lifetime of
  24 hours by default. After that, `databricks auth token` fails with a
  refresh error and the user needs to log in again.
