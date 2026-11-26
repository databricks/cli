package deploy

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
)

type genieAgentState struct {
	Name string
	ID   string
	ETag string
}

func collectGenieAgentsFromState(ctx context.Context, b *bundle.Bundle) []genieAgentState {
	state := b.DeploymentBundle.ExportState(ctx)

	var genieAgents []genieAgentState
	for _, resourceKey := range slices.Sorted(maps.Keys(state)) {
		instance := state[resourceKey]
		if !strings.HasPrefix(resourceKey, "resources.genie_agents.") {
			continue
		}
		parts := strings.Split(resourceKey, ".")
		if len(parts) != 3 {
			continue
		}
		resourceName := parts[2]

		genieAgents = append(genieAgents, genieAgentState{
			Name: resourceName,
			ID:   instance.ID,
			ETag: instance.ETag,
		})
	}

	return genieAgents
}

type checkGenieAgentsModifiedRemotely struct {
	isPlan bool
}

func (l *checkGenieAgentsModifiedRemotely) Name() string {
	return "CheckGenieAgentsModifiedRemotely"
}

func (l *checkGenieAgentsModifiedRemotely) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Config.Resources.GenieAgents) == 0 {
		return nil
	}

	if b.Config.Bundle.Force {
		return nil
	}

	genieAgents := collectGenieAgentsFromState(ctx, b)

	var diags diag.Diagnostics
	for _, genieAgent := range genieAgents {
		// Removed resources will be destroyed; older state may not have a deployment etag.
		if _, ok := b.Config.Resources.GenieAgents[genieAgent.Name]; !ok || genieAgent.ETag == "" {
			continue
		}

		path := structpath.NewPath(nil, "resources", "genie_agents", genieAgent.Name)
		loc := b.Config.GetLocationOf(path)
		actual, err := b.WorkspaceClient(ctx).GenieAgents.GetGenieAgent(ctx, genieagents.GetGenieAgentRequest{Name: "agents/" + genieAgent.ID})
		if apierr.IsMissing(err) {
			// Missing agents follow the normal recreate path.
			continue
		}
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to get genie agent %q", genieAgent.Name),
				Detail:    err.Error(),
				Paths:     []*structpath.PathNode{path},
				Locations: []diag.Location{loc},
			})
			continue
		}

		if actual.Etag == genieAgent.ETag {
			continue
		}

		// Downgrade this to a warning in plan mode.
		severity := diag.Error
		if l.isPlan {
			severity = diag.Warning
		}

		diags = diags.Append(diag.Diagnostic{
			Severity: severity,
			Summary:  fmt.Sprintf("genie agent %q has been modified remotely", genieAgent.Name),
			Detail: "" +
				"This genie agent has been modified remotely since the last bundle deployment.\n" +
				"These modifications are untracked and will be overwritten on deploy.\n" +
				"\n" +
				"Make sure that the local genie agent definition matches what you intend to deploy\n" +
				"before proceeding with the deployment.\n" +
				"\n" +
				"To overwrite the remote changes with your local version, use --force.\n" +
				"The remote modifications will be lost." + agent.AgentNotice(),
			Paths:     []*structpath.PathNode{path},
			Locations: []diag.Location{loc},
		})
	}

	return diags
}

func CheckGenieAgentsModifiedRemotely(isPlan bool) *checkGenieAgentsModifiedRemotely {
	return &checkGenieAgentsModifiedRemotely{isPlan: isPlan}
}
