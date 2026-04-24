package mutator

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
)

type initializeURLs struct{}

// InitializeURLs populates the URL field on every UC resource in the config
// tree. Mirrors bundle/config/mutator.InitializeURLs but for the UCM resource
// set and without the orgId query-string handling (UCM only supports hosts
// where the workspace id already appears in the subdomain).
//
// The URL field is used by `ucm summary` to surface clickable workspace
// console links. An unset Workspace.Host is reported as a warning, not an
// error, so summary still runs for locally-authored configs.
func InitializeURLs() ucm.Mutator { return &initializeURLs{} }

func (m *initializeURLs) Name() string { return "InitializeURLs" }

func (m *initializeURLs) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	host := strings.TrimRight(u.Config.Workspace.Host, "/")
	if host == "" {
		return diag.Warningf("cannot initialize resource URLs: workspace.host is not set")
	}

	baseURL, err := url.Parse(host)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, r := range u.Config.Resources.Catalogs {
		r.InitializeURL(*baseURL)
	}
	for _, r := range u.Config.Resources.Schemas {
		r.InitializeURL(*baseURL)
	}
	for _, r := range u.Config.Resources.Volumes {
		r.InitializeURL(*baseURL)
	}
	for _, r := range u.Config.Resources.StorageCredentials {
		r.InitializeURL(*baseURL)
	}
	for _, r := range u.Config.Resources.ExternalLocations {
		r.InitializeURL(*baseURL)
	}
	for _, r := range u.Config.Resources.Connections {
		r.InitializeURL(*baseURL)
	}

	return nil
}
