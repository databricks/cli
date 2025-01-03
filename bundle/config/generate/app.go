package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

func ConvertAppToValue(app *apps.App, sourceCodePath string, appConfig map[string]any) (dyn.Value, error) {
	ac, err := convert.FromTyped(appConfig, dyn.NilValue)
	if err != nil {
		return dyn.NilValue, err
	}

	ar, err := convert.FromTyped(app.Resources, dyn.NilValue)
	if err != nil {
		return dyn.NilValue, err
	}

	// The majority of fields of the app struct are read-only.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"name":             dyn.NewValue(app.Name, []dyn.Location{{Line: 1}}),
		"description":      dyn.NewValue(app.Description, []dyn.Location{{Line: 2}}),
		"source_code_path": dyn.NewValue(sourceCodePath, []dyn.Location{{Line: 3}}),
	}

	if ac.Kind() != dyn.KindNil {
		dv["config"] = ac.WithLocations([]dyn.Location{{Line: 4}})
	}

	if ar.Kind() != dyn.KindNil {
		dv["resources"] = ar.WithLocations([]dyn.Location{{Line: 5}})
	}

	return dyn.V(dv), nil
}
