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
	"crypto/tls"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/databrickscfg/profile"
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
}

// New returns sdk-go clients that authenticate with cfg. cfg must already be
// resolved (config.Config.EnsureResolved) and is shared, not copied, because
// config.Config embeds a sync.Mutex.
func New(ctx context.Context, cfg *config.Config) (*Clients, error) {
	s, err := newServices(ctx, Options(cfg)...)
	if err != nil {
		return nil, err
	}
	return &Clients{services: s, Config: cfg}, nil
}

// Options returns the sdk-go client options that make a client authenticate
// and route requests the way the rest of the CLI does.
func Options(cfg *config.Config) []client.Option {
	opts := []client.Option{
		client.WithHost(cfg.CanonicalHostName()),
		client.WithCredentials(NewCredentials(cfg)),
		client.WithHTTPClient(HTTPClient(cfg)),
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

func baseTransport(cfg *config.Config) http.RoundTripper {
	if cfg.HTTPTransport != nil {
		return cfg.HTTPTransport
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return t
}
