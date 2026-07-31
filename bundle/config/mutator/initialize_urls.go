package mutator

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/diag"
)

type initializeURLs struct{}

// InitializeURLs makes sure the URL field of each resource is configured.
// NOTE: since this depends on an extra API call, this mutator adds some extra
// latency. As such, it should only be used when needed.
// This URL field is used for the output of the 'bundle summary' CLI command.
func InitializeURLs() bundle.Mutator {
	return &initializeURLs{}
}

func (m *initializeURLs) Name() string {
	return "InitializeURLs"
}

func (m *initializeURLs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	baseURL, err := WorkspaceBaseURL(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			r.InitializeURL(baseURL)
		}
	}

	return nil
}

// WorkspaceBaseURL returns the workspace URL used as the base for resource links,
// with ?w=<workspace id> appended when the workspace ID isn't already in the
// hostname. Resolving the workspace ID may cost an extra API call; see
// auth.ResolveWorkspaceID.
func WorkspaceBaseURL(ctx context.Context, b *bundle.Bundle) (url.URL, error) {
	workspaceID, err := auth.ResolveWorkspaceID(ctx, b.WorkspaceClient(ctx))
	if err != nil {
		return url.URL{}, err
	}
	host := b.WorkspaceClient(ctx).Config.CanonicalHostName()
	return workspaceBaseURL(host, workspaceID)
}

func workspaceBaseURL(host, workspaceID string) (url.URL, error) {
	baseURL, err := url.Parse(host)
	if err != nil {
		return url.URL{}, err
	}

	// Add ?w=<workspace id> only if <workspace id> wasn't in the subdomain
	// already. The parameter is needed when vanity URLs / legacy workspace
	// URLs are used. If it's not needed we prefer to leave it out since these
	// URLs are rather long for most terminals.
	//
	// The legacy ?o= spelling is also accepted by the platform; we emit ?w=
	// here to match the new workspace addressing convention.
	if !strings.Contains(baseURL.Hostname(), workspaceID) {
		values := baseURL.Query()
		values.Add("w", workspaceID)
		baseURL.RawQuery = values.Encode()
	}

	return *baseURL, nil
}
