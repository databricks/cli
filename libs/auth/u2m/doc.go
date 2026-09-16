/*
Package u2m supports the user-to-machine (U2M) OAuth flow for authenticating with Databricks.

Databricks uses the authorization code flow from OAuth 2.0 to authenticate users. This flow
consists of four steps:
 1. Retrieve an authorization code for a user by opening a browser and directing them to the
    Databricks authorization URL.
 2. Exchange the authorization code for an access token.
 3. Use the access token to authenticate with Databricks.
 4. When the access token expires, use the refresh token to get a new access token.

The token and authorization endpoints for Databricks vary depending on whether the host is
an account- or workspace-level host. Account-level endpoints are fixed based on the account
ID and host, while workspace-level endpoints are discovered using the OIDC discovery endpoint
at /oidc/.well-known/oauth-authorization-server.

For host-agnostic login through login.databricks.com, use WithDiscoveryLogin together with a
DiscoveryOAuthArgument during Challenge(). This is a bootstrap flow only. Once the callback
reveals the workspace host, construct the usual host-based OAuthArgument for future
PersistentAuth instances.

To trigger the authorization flow, construct a PersistentAuth object with an
OAuthArgument and call Challenge:

	arg, err := NewProfileWorkspaceOAuthArgument(host, profile)
	if err != nil {
		return err
	}
	auth, err := NewPersistentAuth(ctx,
		WithOAuthArgument(arg),
		WithTokenStore(tokenStore),
	)
	if err != nil {
		return err
	}
	defer auth.Close()
	token, err := auth.Challenge()
	if err != nil {
		return err
	}

Challenge returns the new token without storing it. The caller is responsible
for persisting it after any configuration associated with the login is ready.
WithTokenStore configures the store used by Token and ForceRefreshToken.
*/
package u2m
