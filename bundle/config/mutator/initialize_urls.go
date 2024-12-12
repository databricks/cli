package mutator

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
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
	workspaceId, err := b.WorkspaceClient().CurrentWorkspaceID(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	orgId := strconv.FormatInt(workspaceId, 10)
	host := b.WorkspaceClient().Config.CanonicalHostName()
	err = initializeForWorkspace(b, orgId, host)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func initializeForWorkspace(b *bundle.Bundle, orgId, host string) error {
	baseURL, err := url.Parse(host)
	if err != nil {
		return err
	}

	// Add ?o=<workspace id> only if <workspace id> wasn't in the subdomain already.
	// The ?o= is needed when vanity URLs / legacy workspace URLs are used.
	// If it's not needed we prefer to leave it out since these URLs are rather
	// long for most terminals.
	//
	// See https://docs.databricks.com/en/workspace/workspace-details.html for
	// further reading about the '?o=' suffix.
	if !strings.Contains(baseURL.Hostname(), orgId) {
		values := baseURL.Query()
		values.Add("o", orgId)
		baseURL.RawQuery = values.Encode()
	}

	for _, group := range b.Config.Resources.AllResources() {
		for _, r := range group.Resources {
			r.InitializeURL(*baseURL)
		}
	}

	return nil
}
