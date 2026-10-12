package docker

import (
	"context"
	"fmt"
	"os"
	"strings"

	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
)

// dockerWorkspace holds the selected profile and its resolved workspace ID.
type dockerWorkspace struct {
	profile    profile.Profile
	id         string
	executable string
	client     *databricks.WorkspaceClient
}

type dockerTarget struct {
	profile      profile.Profile
	workspaceID  string
	registryHost string
	executable   string
}

type dockerProfileHostError struct {
	profileName string
	cause       error
}

func (e *dockerProfileHostError) Error() string {
	return fmt.Sprintf("profile %q does not target a supported Databricks workspace host. Select a workspace profile, or create one with databricks auth login --host <workspace-url> --profile <name>", e.profileName)
}

func (e *dockerProfileHostError) Unwrap() error {
	return e.cause
}

// loadDockerWorkspace validates inputs before loading credentials or making requests.
// A nil region means inference; an explicitly empty region remains an error.
func loadDockerWorkspace(ctx context.Context, name string, region *string) (*dockerWorkspace, error) {
	p, err := dockercredentials.LoadProfile(ctx, profile.DefaultProfiler, name)
	if err != nil {
		return nil, err
	}
	if err := dockercredentials.ValidateWorkspaceHost(p.Host); err != nil {
		return nil, &dockerProfileHostError{profileName: p.Name, cause: err}
	}
	if region != nil {
		if err := dockercredentials.ValidateRegion(*region); err != nil {
			return nil, err
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate databricks executable: %w", err)
	}
	var client *databricks.WorkspaceClient
	if region == nil || p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone {
		client, err = databricks.NewWorkspaceClient(&databricks.Config{
			Profile:           p.Name,
			Host:              p.Host,
			AccountID:         p.AccountID,
			AuthType:          p.AuthType,
			ConfigFile:        env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
			Loaders:           databrickscfg.ProfileAuthLoaders,
			DatabricksCliPath: executable,
		})
		if err != nil {
			err = fmt.Errorf("load workspace profile %q: %w. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name, err)
			return nil, dockercredentials.RewriteProfileError(ctx, p, err)
		}
	}
	id, err := dockercredentials.WorkspaceID(ctx, p, client)
	if err != nil {
		return nil, err
	}
	return &dockerWorkspace{profile: p, id: id, executable: executable, client: client}, nil
}

// target resolves the registry after any command-specific checks on the workspace ID.
func (w *dockerWorkspace) target(ctx context.Context, explicitRegion *string) (*dockerTarget, error) {
	var region string
	if explicitRegion != nil {
		region = strings.TrimSpace(*explicitRegion)
	} else {
		w.client.Config.WorkspaceID = w.id
		var err error
		region, err = dockercredentials.WorkspaceRegion(ctx, w.profile, w.client.Metastores)
		if err != nil {
			return nil, err
		}
	}
	host, err := dockercredentials.RegistryHost(w.id, region, w.profile.Host)
	if err != nil {
		return nil, err
	}
	return &dockerTarget{
		profile:      w.profile,
		workspaceID:  w.id,
		registryHost: host,
		executable:   w.executable,
	}, nil
}
