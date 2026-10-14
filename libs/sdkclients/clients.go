// Package sdkclients builds sdk-go API clients from the CLI's resolved
// authentication config.
//
// The CLI resolves host, profile and credentials once (libs/auth, cmd/root)
// using config.Config. Every sdk-go client built here reuses that resolution:
// requests are signed with config.Authenticate and no client re-reads
// ~/.databrickscfg or the environment.
package sdkclients

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/bundledeployments"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/qualitymonitors"
	"github.com/databricks/cli/libs/sqllegacy"
	"github.com/databricks/cli/libs/workspaceapi"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/sdk-go/options/client"
)

// Clients holds the resolved config and one sdk-go client per API module.
// The same type serves workspace and account level commands; which APIs make
// sense depends on whether Config points at a workspace or an account host.
type Clients struct {
	*services

	// Config is the resolved authentication config the clients were built from.
	Config *config.Config

	// Hand-written clients for APIs sdk-go does not provide.
	Workspace         *workspaceapi.Client
	BundleDeployments *bundledeployments.Client
	QualityMonitors   *qualitymonitors.Client
	SqlLegacy         *sqllegacy.Client
}

// New returns sdk-go clients that authenticate with cfg. cfg must already be
// resolved (config.Config.EnsureResolved) and is shared, not copied, because
// config.Config embeds a sync.Mutex.
func New(ctx context.Context, cfg *config.Config) (*Clients, error) {
	if err := cfg.EnsureResolved(); err != nil {
		return nil, err
	}
	// Resolve the credentials provider up front, as the previous SDK's client
	// constructor did as a side effect: callers inspect cfg.AuthType right
	// after constructing clients. Errors surface on the first request.
	_ = cfg.GetTokenSource()
	s, err := newServices(ctx, Options(cfg)...)
	if err != nil {
		return nil, err
	}
	api, err := apiclient.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Clients{
		services:          s,
		Config:            cfg,
		Workspace:         workspaceapi.New(api),
		BundleDeployments: bundledeployments.New(api),
		QualityMonitors:   qualitymonitors.New(api),
		SqlLegacy:         sqllegacy.New(api),
	}, nil
}

// Options returns the sdk-go client options that make a client authenticate
// and route requests the way the rest of the CLI does.
func Options(cfg *config.Config) []client.Option {
	opts := []client.Option{
		client.WithHost(cfg.CanonicalHostName()),
		client.WithCredentials(apiclient.NewCredentials(cfg)),
		client.WithHTTPClient(apiclient.HTTPClient(cfg)),
		client.WithoutConfigFile(),
		client.WithoutEnv(),
	}
	// The workspace routing header is needed on unified ("SPOG") hosts; the
	// CLI's "none" sentinel means "no workspace ID", so it is not forwarded.
	if id := cfg.WorkspaceID; id != "" && id != profile.WorkspaceIDNone {
		opts = append(opts, client.WithWorkspaceID(id))
	}
	if cfg.AccountID != "" {
		opts = append(opts, client.WithAccountID(cfg.AccountID))
	}
	if cfg.HTTPTimeoutSeconds > 0 {
		opts = append(opts, client.WithTimeout(time.Duration(cfg.HTTPTimeoutSeconds)*time.Second))
	}
	return opts
}
