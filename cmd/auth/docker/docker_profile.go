package docker

import (
	"context"
	"fmt"
	"os"

	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
)

type dockerProfileDeps struct {
	profiler               profile.Profiler
	newWorkspaceClient     func(*databricks.Config) (*databricks.WorkspaceClient, error)
	resolveWorkspaceID     func(context.Context, *databricks.WorkspaceClient) (string, error)
	resolveWorkspaceRegion func(context.Context, *databricks.WorkspaceClient) (string, error)
	validateWorkspaceHost  func(string) error
	executable             func() (string, error)
	registryHost           func(string, string, string) (string, error)
}

func defaultDockerProfileDeps() dockerProfileDeps {
	return dockerProfileDeps{
		profiler: profile.DefaultProfiler,
		newWorkspaceClient: func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
			return databricks.NewWorkspaceClient(cfg)
		},
		resolveWorkspaceID: authlib.ResolveWorkspaceID,
		resolveWorkspaceRegion: func(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
			summary, err := w.Metastores.Summary(ctx)
			if err != nil {
				return "", err
			}
			return summary.Region, nil
		},
		validateWorkspaceHost: dockercredentials.ValidateWorkspaceHost,
		executable:            os.Executable,
		registryHost:          dockercredentials.RegistryHost,
	}
}

func validateDockerCredentialProfile(p profile.Profile) error {
	if p.HasClientCredentials {
		return fmt.Errorf("profile %q uses client credentials. Docker credential helper requires a profile created by databricks auth login", p.Name)
	}
	if p.AuthType != authlib.AuthTypeDatabricksCli {
		return fmt.Errorf("profile %q uses auth_type %q. Docker credential helper requires a profile created by databricks auth login", p.Name, p.AuthType)
	}
	if isDockerCredentialAccountOnlyProfile(p) {
		return fmt.Errorf("profile %q does not target a workspace. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name)
	}
	return nil
}

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
