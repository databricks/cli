package statemgmt

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type (
	ExportedResourcesMap = resourcestate.ExportedResourcesMap
	ResourceState        = resourcestate.ResourceState
	LoadMode             int
)

const ErrorOnEmptyState LoadMode = 0

type load struct {
	state ExportedResourcesMap
	modes []LoadMode
}

func (l *load) Name() string {
	return "statemgmt.Load"
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return applyState(ctx, b, l.state, l.modes)
}

// applyState merges the exported resource state into the bundle configuration.
func applyState(ctx context.Context, b *bundle.Bundle, state ExportedResourcesMap, modes []LoadMode) diag.Diagnostics {
	if err := validateLoadedState(state, modes); err != nil {
		return diag.FromErr(err)
	}

	if err := StateToBundle(ctx, state, &b.Config); err != nil {
		return diag.FromErr(err)
	}

	// Load each run's resolved job_id into ResolvedJobID (for the URL only). We
	// leave config's job_id ${resources.jobs.*.id} reference intact so it keeps
	// its plan dependency. Typed-only write, like the dashboard etag below.
	for resourceKey, rstate := range state {
		if !strings.HasPrefix(resourceKey, "resources.job_runs.") || rstate.JobID == 0 {
			continue
		}
		parts := strings.Split(resourceKey, ".")
		if len(parts) != 3 {
			continue
		}

		// A run in state but not config was deleted; skip it.
		jrconfig, ok := b.Config.Resources.JobRuns[parts[2]]
		if !ok {
			continue
		}
		jrconfig.ResolvedJobID = rstate.JobID
	}

	// Merge dashboard etags into configuration.
	for resourceKey, dstate := range state {
		// Check if this is a dashboard resource key
		if !strings.HasPrefix(resourceKey, "resources.dashboards.") {
			continue
		}
		// Extract dashboard name from "resources.dashboards.name"
		parts := strings.Split(resourceKey, ".")
		if len(parts) != 3 {
			continue
		}
		dashboardName := parts[2]

		dconfig, ok := b.Config.Resources.Dashboards[dashboardName]

		// Case: A dashboard is defined in state but not in configuration.
		// In this case the dashboard has been deleted and we do not need to load the etag.
		if !ok {
			continue
		}

		dconfig.Etag = dstate.ETag
	}

	return nil
}

func StateToBundle(ctx context.Context, state ExportedResourcesMap, cfg *config.Root) error {
	resourcesPath := structpath.NewStringKey(nil, "resources")
	if !cfg.View().Lookup(resourcesPath).IsValid() {
		if err := cfg.Set(resourcesPath, config.Resources{}); err != nil {
			return err
		}
	}

	for resourceKey, attrs := range state {
		// Parse resource key like "resources.jobs.foo" or "resources.jobs.foo.permissions"
		parts := strings.Split(resourceKey, ".")
		if len(parts) < 3 || parts[0] != "resources" {
			continue // Skip invalid resource keys
		}

		groupName := parts[1]
		resourceName := parts[2]

		// Skip permissions for now as they are sub-resources
		if len(parts) > 3 {
			continue
		}

		if !hasID(groupName) {
			continue
		}

		path := structpath.NewStringKeys(resourcesPath, groupName, resourceName)
		if !cfg.View().Lookup(path).IsValid() {
			if err := cfg.Set(structpath.NewStringKey(path, "modified_status"), resources.ModifiedStatusDeleted); err != nil {
				return err
			}
		}
		if err := cfg.Set(structpath.NewStringKey(path, "id"), attrs.ID); err != nil {
			return err
		}
	}

	return structvar.ForEach(cfg.View(), structpath.MustParsePattern("resources.*.*"), func(p *structpath.PathNode, inner structvar.View) error {
		group, _ := p.Parent().StringKey()
		if !hasID(group) || inner.Get("id").IsValid() || inner.Get("modified_status").IsValid() {
			return nil
		}
		return cfg.Set(structpath.NewStringKey(p, "modified_status"), resources.ModifiedStatusCreated)
	})
}

// hasID reports whether resources of group can hold the deployed id and status. Groups
// that are not part of the configuration, and internal snapshots, cannot.
func hasID(group string) bool {
	typ, ok := config.ResourcesTypes[group]
	if !ok {
		return false
	}
	_, ok = typ.FieldByName("ID")
	return ok
}

func validateLoadedState(state ExportedResourcesMap, modes []LoadMode) error {
	if len(state) == 0 && slices.Contains(modes, ErrorOnEmptyState) {
		return errors.New("resource not found or not yet deployed. Did you forget to run 'databricks bundle deploy'?")
	}
	return nil
}

// Load returns a mutator that merges the provided resource state into the bundle configuration.
func Load(state ExportedResourcesMap, modes ...LoadMode) bundle.Mutator {
	return &load{state: state, modes: modes}
}
