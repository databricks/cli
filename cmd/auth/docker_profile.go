package auth

import (
	"fmt"

	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
)

// validateDockerCredentialProfile requires metadata eligible for workspace-scoped U2M authentication.
func validateDockerCredentialProfile(p profile.Profile) error {
	if p.HasClientCredentials {
		return fmt.Errorf("profile %q uses client credentials. Docker credential helper requires a profile created by databricks auth login", p.Name)
	}
	if p.AuthType != authTypeDatabricksCLI {
		return fmt.Errorf("profile %q uses auth_type %q. Docker credential helper requires a profile created by databricks auth login", p.Name, p.AuthType)
	}
	if isDockerCredentialAccountOnlyProfile(p) {
		return fmt.Errorf("profile %q does not target a workspace. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name)
	}
	return nil
}

// isDockerCredentialAccountOnlyProfile treats classic account hosts and unrouted account profiles as unsafe for workspace requests.
func isDockerCredentialAccountOnlyProfile(p profile.Profile) bool {
	if p.Host == "" {
		return true
	}
	cfg := &config.Config{Host: p.Host, AccountID: p.AccountID, WorkspaceID: p.WorkspaceID}
	if authlib.IsClassicAccountHost(cfg.CanonicalHostName()) {
		return true
	}
	return p.AccountID != "" && (p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone)
}
