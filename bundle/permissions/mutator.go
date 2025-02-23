package permissions

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

const (
	CAN_MANAGE = "CAN_MANAGE"
	CAN_VIEW   = "CAN_VIEW"
	CAN_RUN    = "CAN_RUN"
)

var unsupportedResources = []string{"clusters", "volumes", "schemas", "quality_monitors", "registered_models"}

var (
	allowedLevels = []string{CAN_MANAGE, CAN_VIEW, CAN_RUN}
	levelsMap     = map[string](map[string]string){
		"jobs": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_VIEW",
			CAN_RUN:    "CAN_MANAGE_RUN",
		},
		"pipelines": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_VIEW",
			CAN_RUN:    "CAN_RUN",
		},
		"experiments": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_READ",
		},
		"models": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_READ",
		},
		"model_serving_endpoints": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_VIEW",
			CAN_RUN:    "CAN_QUERY",
		},
		"dashboards": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_READ",
		},
		"apps": {
			CAN_MANAGE: "CAN_MANAGE",
			CAN_VIEW:   "CAN_USE",
		},
	}
)

type bundlePermissions struct{}

func ApplyBundlePermissions() bundle.Mutator {
	return &bundlePermissions{}
}

func (m *bundlePermissions) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := validate(b)
	if err != nil {
		return diag.FromErr(err)
	}

	patterns := make(map[string]dyn.Pattern, 0)
	for key := range levelsMap {
		patterns[key] = dyn.NewPattern(
			dyn.Key("resources"),
			dyn.Key(key),
			dyn.AnyKey(),
		)
	}

	err = b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		for key, pattern := range patterns {
			v, err = dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				var permissions []resources.Permission
				pv, err := dyn.Get(v, "permissions")
				// If the permissions field is not found, we set to an empty array
				if err != nil {
					pv = dyn.V([]dyn.Value{})
				}

				err = convert.ToTyped(&permissions, pv)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to convert permissions: %w", err)
				}

				permissions = append(permissions, convertPermissions(
					ctx,
					b.Config.Permissions,
					permissions,
					key,
					levelsMap[key],
				)...)

				pv, err = convert.FromTyped(permissions, dyn.NilValue)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to convert permissions: %w", err)
				}

				return dyn.Set(v, "permissions", pv)
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}

		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func validate(b *bundle.Bundle) error {
	for _, p := range b.Config.Permissions {
		if !slices.Contains(allowedLevels, p.Level) {
			return fmt.Errorf("invalid permission level: %s, allowed values: [%s]", p.Level, strings.Join(allowedLevels, ", "))
		}
	}

	return nil
}

func (m *bundlePermissions) Name() string {
	return "ApplyBundlePermissions"
}
