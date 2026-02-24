package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
)

// Auth type names returned by credential providers.
const (
	AuthTypeDatabricksCli   = "databricks-cli"
	AuthTypePat             = "pat"
	AuthTypeBasic           = "basic"
	AuthTypeAzureCli        = "azure-cli"
	AuthTypeOAuthM2M        = "oauth-m2m"
	AuthTypeAzureMSI        = "azure-msi"
	AuthTypeAzureSecret     = "azure-client-secret"
	AuthTypeGoogleCreds     = "google-credentials"
	AuthTypeGoogleID        = "google-id"
	AuthTypeGitHubOIDC      = "github-oidc-azure"
	AuthTypeMetadataService = "metadata-service"
)

// authTypeDisplayNames maps auth type identifiers to human-readable names.
var authTypeDisplayNames = map[string]string{
	AuthTypeDatabricksCli:   "OAuth (databricks-cli)",
	AuthTypePat:             "Personal Access Token (pat)",
	AuthTypeBasic:           "Basic",
	AuthTypeAzureCli:        "Azure CLI (azure-cli)",
	AuthTypeOAuthM2M:        "OAuth Machine-to-Machine (oauth-m2m)",
	AuthTypeAzureMSI:        "Azure Managed Identity (azure-msi)",
	AuthTypeAzureSecret:     "Azure Client Secret (azure-client-secret)",
	AuthTypeGoogleCreds:     "Google Credentials (google-credentials)",
	AuthTypeGoogleID:        "Google Default Credentials (google-id)",
	AuthTypeGitHubOIDC:      "GitHub OIDC for Azure (github-oidc-azure)",
	AuthTypeMetadataService: "Metadata Service (metadata-service)",
}

// AuthTypeDisplayName returns a human-readable name for the given auth type.
// Falls back to the raw identifier if no display name is registered.
func AuthTypeDisplayName(authType string) string {
	if name, ok := authTypeDisplayNames[strings.ToLower(authType)]; ok {
		return name
	}
	return authType
}

// RewriteAuthError rewrites the error message for invalid refresh token error.
// It returns whether the error was rewritten and the rewritten error.
func RewriteAuthError(ctx context.Context, host, accountId, profile string, err error) (bool, error) {
	target := &u2m.InvalidRefreshTokenError{}
	if errors.As(err, &target) {
		oauthArgument, err := AuthArguments{
			Host:      host,
			AccountID: accountId,
		}.ToOAuthArgument()
		if err != nil {
			return false, err
		}
		msg := `A new access token could not be retrieved because the refresh token is invalid. To reauthenticate, run the following command:
  $ ` + BuildLoginCommand(ctx, profile, oauthArgument)
		return true, errors.New(msg)
	}
	return false, err
}

// EnrichAuthError appends identity context and remediation steps to 401/403 API errors.
// For non-API errors or other status codes, the original error is returned unchanged.
func EnrichAuthError(ctx context.Context, cfg *config.Config, err error) error {
	var apiErr *apierr.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	if apiErr.StatusCode != http.StatusUnauthorized && apiErr.StatusCode != http.StatusForbidden {
		return err
	}

	var b strings.Builder

	// Identity context.
	if cfg.Profile != "" {
		fmt.Fprintf(&b, "\nProfile:   %s", cfg.Profile)
	}
	if cfg.Host != "" {
		fmt.Fprintf(&b, "\nHost:      %s", cfg.Host)
	}
	if cfg.AuthType != "" {
		fmt.Fprintf(&b, "\nAuth type: %s", AuthTypeDisplayName(cfg.AuthType))
	}

	fmt.Fprint(&b, "\n\nNext steps:")

	if apiErr.StatusCode == http.StatusUnauthorized {
		writeReauthSteps(ctx, cfg, &b)
	} else {
		fmt.Fprint(&b, "\n  - Verify you have the required permissions for this operation")
	}

	// Always suggest checking identity.
	fmt.Fprintf(&b, "\n  - Check your identity: %s", BuildDescribeCommand(cfg))

	// Nudge toward profiles when using env-var-based auth.
	if cfg.Profile == "" {
		fmt.Fprint(&b, "\n  - Consider configuring a profile: databricks configure --profile <name>")
	}

	return fmt.Errorf("%w\n%s", err, b.String())
}

// writeReauthSteps writes auth-type-aware re-authentication suggestions for 401 errors.
func writeReauthSteps(ctx context.Context, cfg *config.Config, b *strings.Builder) {
	switch strings.ToLower(cfg.AuthType) {
	case AuthTypeDatabricksCli:
		// When profile is set, BuildLoginCommand uses --profile and ignores
		// the OAuthArgument, so skip the conversion entirely.
		if cfg.Profile != "" {
			fmt.Fprintf(b, "\n  - Re-authenticate: databricks auth login --profile %s", cfg.Profile)
			return
		}
		oauthArg, argErr := AuthArguments{
			Host:          cfg.Host,
			AccountID:     cfg.AccountID,
			WorkspaceID:   cfg.WorkspaceID,
			IsUnifiedHost: cfg.Experimental_IsUnifiedHost,
		}.ToOAuthArgument()
		if argErr != nil {
			fmt.Fprint(b, "\n  - Re-authenticate: databricks auth login")
			return
		}
		loginCmd := BuildLoginCommand(ctx, "", oauthArg)
		// For unified hosts, BuildLoginCommand (via OAuthArgument) doesn't carry
		// workspace-id. Append it so the command is actionable.
		if cfg.Experimental_IsUnifiedHost && cfg.WorkspaceID != "" {
			loginCmd += " --workspace-id " + cfg.WorkspaceID
		}
		fmt.Fprintf(b, "\n  - Re-authenticate: %s", loginCmd)

	case AuthTypePat:
		if cfg.Profile != "" {
			fmt.Fprintf(b, "\n  - Regenerate your access token or run: databricks configure --profile %s", cfg.Profile)
		} else {
			fmt.Fprint(b, "\n  - Regenerate your access token")
		}

	case AuthTypeBasic:
		if cfg.Profile != "" {
			fmt.Fprintf(b, "\n  - Check your username/password or run: databricks configure --profile %s", cfg.Profile)
		} else {
			fmt.Fprint(b, "\n  - Check your username and password")
		}

	case AuthTypeAzureCli:
		fmt.Fprint(b, "\n  - Re-authenticate with Azure: az login")

	case AuthTypeOAuthM2M:
		fmt.Fprint(b, "\n  - Check your service principal client ID and secret")

	default:
		fmt.Fprint(b, "\n  - Check your authentication credentials")
	}
}

// BuildLoginCommand builds the login command for the given OAuth argument or profile.
func BuildLoginCommand(ctx context.Context, profile string, arg u2m.OAuthArgument) string {
	cmd := []string{
		"databricks",
		"auth",
		"login",
	}
	if profile != "" {
		cmd = append(cmd, "--profile", profile)
	} else {
		switch arg := arg.(type) {
		case u2m.UnifiedOAuthArgument:
			cmd = append(cmd, "--host", arg.GetHost(), "--account-id", arg.GetAccountId(), "--experimental-is-unified-host")
		case u2m.AccountOAuthArgument:
			cmd = append(cmd, "--host", arg.GetAccountHost(), "--account-id", arg.GetAccountId())
		case u2m.WorkspaceOAuthArgument:
			cmd = append(cmd, "--host", arg.GetWorkspaceHost())
		}
	}
	return strings.Join(cmd, " ")
}

// BuildDescribeCommand builds the describe command for the given config.
// When a profile is set, it uses --profile. Otherwise it emits a bare command
// since `databricks auth describe` resolves env vars (DATABRICKS_HOST, etc.)
// automatically.
func BuildDescribeCommand(cfg *config.Config) string {
	if cfg.Profile != "" {
		return "databricks auth describe --profile " + cfg.Profile
	}
	return "databricks auth describe"
}
