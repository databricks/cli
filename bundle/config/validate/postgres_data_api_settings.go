package validate

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// postgresDataApiNonSettingKeys are the keys of a postgres_data_apis entry that
// are not sent as Data API settings.
var postgresDataApiNonSettingKeys = []string{"parent", "lifecycle"}

// PostgresDataApiSettings rejects a Data API that sets nothing besides parent.
// The API requires at least one setting with a non-default value, so such a
// config would otherwise pass validation and fail at deploy time.
func PostgresDataApiSettings() bundle.ReadOnlyMutator {
	return &postgresDataApiSettings{}
}

type postgresDataApiSettings struct{ bundle.RO }

func (*postgresDataApiSettings) Name() string {
	return "validate:postgres_data_api_settings"
}

func (*postgresDataApiSettings) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Check the dynamic value rather than the typed struct: a setting given as
	// an unresolved ${...} reference is still zero in the typed struct.
	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("postgres_data_apis"), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Nil entries are reported by AllResourcesHaveValues.
			m, ok := v.AsMap()
			if !ok {
				return v, nil
			}
			for _, pair := range m.Pairs() {
				if !slices.Contains(postgresDataApiNonSettingKeys, pair.Key.MustString()) {
					return v, nil
				}
			}
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "postgres_data_apis." + p[2].Key() + ": at least one Data API setting besides parent is required, for example db_schemas",
				Locations: v.Locations(),
				Paths:     dyn.ToStructPaths(p),
			})
			return v, nil
		},
	)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
