package statemgmt

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

	// Restore each job run's resolved job_id from state. In config a run's
	// job_id is a ${resources.jobs.*.id} reference that is only resolved at
	// deploy, so at read time it is 0; the deployed value lives in state and is
	// needed to build the run URL. This is written into the dynamic config (not
	// just the typed struct) because job_id is a serialized field and a later
	// config round-trip would otherwise reset it to 0 before the URL is built.
	if err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		for resourceKey, rstate := range state {
			if !strings.HasPrefix(resourceKey, "resources.job_runs.") || rstate.JobID == 0 {
				continue
			}
			parts := strings.Split(resourceKey, ".")
			if len(parts) != 3 {
				continue
			}

			// Only restore for runs still present in config; a run in state but
			// not config was deleted and has no URL to build.
			runPath := dyn.Path{dyn.Key("resources"), dyn.Key("job_runs"), dyn.Key(parts[2])}
			if run, _ := dyn.GetByPath(v, runPath); !run.IsValid() {
				continue
			}

			var err error
			v, err = dyn.SetByPath(v, dyn.Path{dyn.Key("resources"), dyn.Key("job_runs"), dyn.Key(parts[2]), dyn.Key("job_id")}, dyn.V(rstate.JobID))
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return v, nil
	}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ensureMap(v dyn.Value, path dyn.Path) (dyn.Value, error) {
	item, _ := dyn.GetByPath(v, path)
	if !item.IsValid() {
		var err error
		v, err = dyn.SetByPath(v, path, dyn.V(dyn.NewMapping()))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("internal error: failed to create %s: %s", path, err)
		}
	}
	return v, nil
}

func StateToBundle(ctx context.Context, state ExportedResourcesMap, config *config.Root) error {
	return config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		v, err = ensureMap(v, dyn.Path{dyn.Key("resources")})
		if err != nil {
			return v, err
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

			var err error
			v, err = ensureMap(v, dyn.Path{dyn.Key("resources"), dyn.Key(groupName)})
			if err != nil {
				return v, err
			}

			path := dyn.Path{dyn.Key("resources"), dyn.Key(groupName), dyn.Key(resourceName)}
			resource, err := dyn.GetByPath(v, path)
			if !resource.IsValid() {
				m := dyn.NewMapping()
				m.SetLoc("id", nil, dyn.V(attrs.ID))
				m.SetLoc("modified_status", nil, dyn.V(resources.ModifiedStatusDeleted))
				v, err = dyn.SetByPath(v, path, dyn.V(m))
				if err != nil {
					return dyn.InvalidValue, err
				}
			} else if err != nil {
				return dyn.InvalidValue, err
			} else {
				v, err = dyn.SetByPath(v, dyn.Path{dyn.Key("resources"), dyn.Key(groupName), dyn.Key(resourceName), dyn.Key("id")}, dyn.V(attrs.ID))
				if err != nil {
					return dyn.InvalidValue, err
				}
			}
		}

		return dyn.MapByPattern(v, dyn.Pattern{dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()}, func(p dyn.Path, inner dyn.Value) (dyn.Value, error) {
			idPath := dyn.Path{dyn.Key("id")}
			statusPath := dyn.Path{dyn.Key("modified_status")}
			id, _ := dyn.GetByPath(inner, idPath)
			status, _ := dyn.GetByPath(inner, statusPath)
			if !id.IsValid() && !status.IsValid() {
				return dyn.SetByPath(inner, statusPath, dyn.V(resources.ModifiedStatusCreated))
			}
			return inner, nil
		})
	})
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
