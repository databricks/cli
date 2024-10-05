package mutator

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type initializeURLs struct {
	name string
}

// InitializeURLs makes sure the URL field of each resource is configured.
// NOTE: since this depends on an extra API call, this mutator adds some extra
// latency. As such, it should only be used when needed.
// This URL field is used for the output of the 'bundle summary' CLI command.
func InitializeURLs() bundle.Mutator {
	return &initializeURLs{}
}

func (m *initializeURLs) Name() string {
	return fmt.Sprintf("InitializeURLs(%s)", m.name)
}

func (m *initializeURLs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	workspaceId, err := b.WorkspaceClient().CurrentWorkspaceID(ctx)
	orgId := strconv.FormatInt(workspaceId, 10)
	if err != nil {
		return diag.FromErr(err)
	}
	urlPrefix := b.WorkspaceClient().Config.CanonicalHostName() + "/"
	initializeForWorkspace(b, orgId, urlPrefix)
	return nil
}

func initializeForWorkspace(b *bundle.Bundle, orgId string, urlPrefix string) {
	// Add ?o=<workspace id> only if <workspace id> wasn't in the subdomain already.
	// The ?o= is needed when vanity URLs / legacy workspace URLs are used.
	// If it's not needed we prefer to leave it out since these URLs are rather
	// long for most terminals.
	//
	// See https://docs.databricks.com/en/workspace/workspace-details.html for
	// further reading about the '?o=' suffix.
	urlSuffix := ""
	if !strings.Contains(urlPrefix, orgId) {
		urlSuffix = "?o=" + orgId
	}

	for _, rs := range b.Config.Resources.AllResources() {
		for _, r := range rs {
			r.InitializeURL(urlPrefix, urlSuffix)
		}
	}
}
