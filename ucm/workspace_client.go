package ucm

import (
	"fmt"
	"sync"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// workspaceClientConfig builds the SDK config used to construct a workspace
// client for this Ucm. Mirrors bundle/config.Workspace.Config + Client() with
// only the subset of auth attributes UCM currently supports.
func (u *Ucm) workspaceClientConfig() *sdkconfig.Config {
	return &sdkconfig.Config{
		Host:    u.Config.Workspace.Host,
		Profile: u.Config.Workspace.Profile,
	}
}

// buildWorkspaceClient resolves auth configuration and constructs a workspace
// client. Mirrors bundle/config.Workspace.Client(): when only the host is set,
// we install the ResolveProfileFromHost loader so the SDK picks up a unique
// matching profile from ~/.databrickscfg. On ambiguity the loader returns the
// errMultipleProfiles error detected by databrickscfg.AsMultipleProfiles.
func (u *Ucm) buildWorkspaceClient() (*databricks.WorkspaceClient, error) {
	cfg := u.workspaceClientConfig()

	if cfg.Host != "" && cfg.Profile == "" {
		cfg.Loaders = []sdkconfig.Loader{
			sdkconfig.ConfigAttributes,
			databrickscfg.ResolveProfileFromHost,
		}
	}

	if err := cfg.EnsureResolved(); err != nil {
		return nil, err
	}

	if cfg.Host != "" && cfg.Profile != "" {
		if err := databrickscfg.ValidateConfigAndProfileHost(cfg, cfg.Profile); err != nil {
			return nil, err
		}
	}

	return databricks.NewWorkspaceClient((*databricks.Config)(cfg))
}

func (u *Ucm) initClientOnce() {
	u.getClient = sync.OnceValues(func() (*databricks.WorkspaceClient, error) {
		w, err := u.buildWorkspaceClient()
		if err != nil {
			return nil, fmt.Errorf("cannot resolve ucm auth configuration: %w", err)
		}
		return w, nil
	})
}

// WorkspaceClientE returns the memoized workspace client, building it from
// Config.Workspace on first call.
func (u *Ucm) WorkspaceClientE() (*databricks.WorkspaceClient, error) {
	if u.getClient == nil {
		u.initClientOnce()
	}
	return u.getClient()
}

// WorkspaceClient is the panicking convenience wrapper around WorkspaceClientE.
// Prefer WorkspaceClientE in new code so callers can surface auth errors.
func (u *Ucm) WorkspaceClient() *databricks.WorkspaceClient {
	client, err := u.WorkspaceClientE()
	if err != nil {
		panic(err)
	}
	return client
}

// SetWorkspaceClient injects a pre-built client, primarily for tests.
func (u *Ucm) SetWorkspaceClient(w *databricks.WorkspaceClient) {
	u.getClient = func() (*databricks.WorkspaceClient, error) {
		return w, nil
	}
}

// ClearWorkspaceClient resets the memoized client so the next WorkspaceClientE
// call rebuilds it. Used after Config.Workspace is mutated (e.g. when a
// profile is selected via the ambiguity picker).
func (u *Ucm) ClearWorkspaceClient() {
	u.initClientOnce()
}
