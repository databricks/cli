package statemgmt

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform"
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
	modes  []LoadMode
	engine engine.EngineType
}

func (l *load) Name() string {
	return "statemgmt.Load"
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var err error
	var state ExportedResourcesMap

	if l.engine.IsDirect() {
		_, fullPathDirect := b.StateFilenameDirect(ctx)
		state, err = b.DeploymentBundle.ExportState(ctx, fullPathDirect)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		var err error
		state, err = terraform.ParseResourcesState(ctx, b)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = l.validateState(state)
	if err != nil {
		return diag.FromErr(err)
	}

	// Merge state into configuration.
	err = StateToBundle(ctx, state, &b.Config)
	if err != nil {
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

func (l *load) validateState(state ExportedResourcesMap) error {
	if len(state) == 0 && slices.Contains(l.modes, ErrorOnEmptyState) {
		return errors.New("resource not found or not yet deployed. Did you forget to run 'databricks bundle deploy'?")
	}

	return nil
}

func Load(engine engine.EngineType, modes ...LoadMode) bundle.Mutator {
	return &load{modes: modes, engine: engine}
}
