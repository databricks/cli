package mutator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/env"
)

const bundleVarPrefix = "BUNDLE_VAR_"

type setVariables struct{}

func SetVariables() bundle.Mutator {
	return &setVariables{}
}

func (m *setVariables) Name() string {
	return "SetVariables"
}

func getDefaultVariableFilePath(target string) string {
	return ".databricks/bundle/" + target + "/variable-overrides.json"
}

func setVariable(ctx context.Context, v dyn.Value, variable *variable.Variable, name string, fileDefault dyn.Value) (dyn.Value, error) {
	// case: variable already has value initialized, so skip
	if variable.HasValue() {
		return v, nil
	}

	// case: read and set variable value from process environment
	envVarName := bundleVarPrefix + name
	if val, ok := env.Lookup(ctx, envVarName); ok {
		if variable.IsComplex() {
			return dyn.InvalidValue, fmt.Errorf(`setting via environment variables (%s) is not supported for complex variable %s`, envVarName, name)
		}

		v, err := dyn.Set(v, "value", dyn.V(val))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to assign value "%s" to variable %s from environment variable %s with error: %v`, val, name, envVarName, err)
		}
		return v, nil
	}

	// case: Defined a variable for named lookup for a resource
	// It will be resolved later in ResolveResourceReferences mutator
	if variable.Lookup != nil {
		return v, nil
	}

	// case: Set the variable to the default value from the variable file
	if fileDefault.Kind() != dyn.KindInvalid && fileDefault.Kind() != dyn.KindNil {
		hasComplexType := variable.IsComplex()
		hasComplexValue := fileDefault.Kind() == dyn.KindMap || fileDefault.Kind() == dyn.KindSequence

		if hasComplexType && !hasComplexValue {
			return dyn.InvalidValue, fmt.Errorf(`variable %s is of type complex, but the value in the variable file is not a complex type`, name)
		}
		if !hasComplexType && hasComplexValue {
			return dyn.InvalidValue, fmt.Errorf(`variable %s is not of type complex, but the value in the variable file is a complex type`, name)
		}

		v, err := dyn.Set(v, "value", fileDefault)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to assign default value from variable file to variable %s with error: %v`, name, err)
		}

		return v, nil
	}

	// case: Set the variable to its default value
	if variable.HasDefault() {
		vDefault, err := dyn.Get(v, "default")
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to get default value from config "%s" for variable %s with error: %v`, variable.Default, name, err)
		}

		v, err := dyn.Set(v, "value", vDefault)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to assign default value from config "%s" to variable %s with error: %v`, variable.Default, name, err)
		}
		return v, nil
	}

	// We should have had a value to set for the variable at this point.
	return dyn.InvalidValue, fmt.Errorf(`no value assigned to required variable %s. Assignment can be done using "--var", by setting the %s environment variable, or in %s file`, name, bundleVarPrefix+name, getDefaultVariableFilePath("<target>"))
}

func readVariablesFromFile(b *bundle.Bundle) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	filePath := filepath.Join(b.BundleRootPath, getDefaultVariableFilePath(b.Config.Bundle.Target))
	if _, err := os.Stat(filePath); err != nil {
		return dyn.InvalidValue, nil
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to read variables file: %w", err))
	}

	val, err := jsonloader.LoadJSON(f, filePath)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to parse variables file %s: %w", filePath, err))
	}

	if val.Kind() != dyn.KindMap {
		return dyn.InvalidValue, diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to parse variables file %s: invalid format", filePath),
			Detail:   "Variables file must be a JSON object with the following format:\n{\"var1\": \"value1\", \"var2\": \"value2\"}",
		})
	}

	return val, nil
}

func (m *setVariables) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	defaults, diags := readVariablesFromFile(b)
	if diags.HasError() {
		return diags
	}
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "variables", dyn.Foreach(func(p dyn.Path, variable dyn.Value) (dyn.Value, error) {
			name := p[1].Key()
			v, ok := b.Config.Variables[name]
			if !ok {
				return dyn.InvalidValue, fmt.Errorf(`variable "%s" is not defined`, name)
			}

			fileDefault, _ := dyn.Get(defaults, name)
			return setVariable(ctx, variable, v, name, fileDefault)
		}))
	})

	return diags.Extend(diag.FromErr(err))
}
