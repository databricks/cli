package validate

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	// Check the configuration view rather than the typed struct alone: a setting given
	// as an unresolved ${...} reference is still zero in the typed struct.
	for name, v := range b.Config.View().Get("resources").Get("postgres_data_apis").MapItems() {
		// Nil entries are reported by AllResourcesHaveValues.
		if v.Kind() != structvar.KindMap {
			continue
		}
		hasSetting := false
		for key := range v.MapItems() {
			if !slices.Contains(postgresDataApiNonSettingKeys, key) {
				hasSetting = true
				break
			}
		}
		if hasSetting {
			continue
		}
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "postgres_data_apis." + name + ": at least one Data API setting besides parent is required, for example db_schemas",
			Locations: v.Locations(),
			Paths:     []*structpath.PathNode{structpath.NewPath(nil, "resources", "postgres_data_apis", name)},
		})
	}

	return diags
}
