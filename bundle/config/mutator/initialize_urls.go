package mutator

import (
	"context"
	"net/url"
	"strconv"
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
	client := b.WorkspaceClient(ctx)
	host := client.Config.CanonicalHostName()

	// A configured, non-numeric workspace ID (a UUID connection-style
	// identifier) can't be validated against CurrentWorkspaceID, which
	// strictly parses the org ID header as an integer and would error on a
	// UUID. Skip the API call entirely and pass it through unchanged.
	cfgID := client.Config.WorkspaceID
	if cfgID != "" && cfgID != auth.WorkspaceIDNone {
		if _, err := strconv.ParseInt(cfgID, 10, 64); err != nil {
			if err := initializeForWorkspace(b, cfgID, host); err != nil {
				return diag.FromErr(err)
			}
			return nil
		}
	}

	// Otherwise, always resolve the workspace ID from the API. This is the
	// only way to detect a numeric workspace ID that is stale or mis-scoped
	// (e.g. a leftover value from a different profile or an old pasted SPOG
	// URL): if the configured value disagrees with the org ID the workspace
	// actually reports, embedding it in ?w= would silently navigate to the
	// wrong workspace, so we error out instead.
	apiID, err := client.CurrentWorkspaceID(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	orgID := strconv.FormatInt(apiID, 10)

	if cfgID != "" && cfgID != auth.WorkspaceIDNone && cfgID != orgID {
		return diag.Errorf(
			"workspace_id %s in your configuration does not match the connected workspace (ID: %s); "+
				"remove or correct workspace_id in your profile or bundle config to disambiguate",
			cfgID, orgID,
		)
	}

	if err := initializeForWorkspace(b, orgID, host); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func initializeForWorkspace(b *bundle.Bundle, workspaceID, host string) error {
	baseURL, err := url.Parse(host)
	if err != nil {
		return err
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

	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			r.InitializeURL(*baseURL)
		}
	}

	return nil
}
