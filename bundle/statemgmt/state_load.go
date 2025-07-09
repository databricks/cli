package statemgmt

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type (
	ExportedResourcesMap = resourcestate.ExportedResourcesMap
	ResourceState        = resourcestate.ResourceState
	loadMode             int
)

const ErrorOnEmptyState loadMode = 0

type load struct {
	modes []loadMode
}

func (l *load) Name() string {
	return "statemgmt.Load"
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var state ExportedResourcesMap
	var err error

	if b.DirectDeployment {
		err = b.OpenResourceDatabase(ctx)
		if err != nil {
			return diag.FromErr(err)
		}
		state = b.ResourceDatabase.ExportState(ctx)
	} else {
		tf := b.Terraform
		if tf == nil {
			return diag.Errorf("terraform not initialized")
		}

		err = tf.Init(ctx, tfexec.Upgrade(true))
		if err != nil {
			return diag.Errorf("terraform init: %v", err)
		}

		state, err = terraform.ParseResourcesState(ctx, b)
	}
	if err != nil {
		return diag.FromErr(err)
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

		for groupName, group := range state {
			var err error
			v, err = ensureMap(v, dyn.Path{dyn.Key("resources"), dyn.Key(groupName)})
			if err != nil {
				return v, err
			}

			for resourceName, attrs := range group {
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
		return errors.New("no deployment state. Did you forget to run 'databricks bundle deploy'?")
	}

	return nil
}

func Load(modes ...loadMode) bundle.Mutator {
	return &load{modes: modes}
}
