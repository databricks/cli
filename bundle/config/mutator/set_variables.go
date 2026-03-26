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

// variableOverrideSource tracks which override method resolved a variable's value.
type variableOverrideSource int

const (
	variableOverrideSourceCLI     variableOverrideSource = iota // --var flag (value already set)
	variableOverrideSourceEnvVar                                // BUNDLE_VAR_* environment variable
	variableOverrideSourceFile                                  // variable-overrides.json
	variableOverrideSourceDefault                               // default value from config
	variableOverrideSourceNone                                  // no value resolved (lookup or error)
)

func setVariable(ctx context.Context, v dyn.Value, variable *variable.Variable, name string, fileDefault dyn.Value) (dyn.Value, variableOverrideSource, error) {
	// case: variable already has value initialized, so skip.
	// This happens when the value is set via the --var CLI flag.
	if variable.HasValue() {
		return v, variableOverrideSourceCLI, nil
	}

	// case: read and set variable value from process environment
	envVarName := bundleVarPrefix + name
	if val, ok := env.Lookup(ctx, envVarName); ok {
		if variable.IsComplex() {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`setting via environment variables (%s) is not supported for complex variable %s`, envVarName, name)
		}

		v, err := dyn.Set(v, "value", dyn.V(val))
		if err != nil {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`failed to assign value "%s" to variable %s from environment variable %s with error: %v`, val, name, envVarName, err)
		}
		return v, variableOverrideSourceEnvVar, nil
	}

	// case: Defined a variable for named lookup for a resource
	// It will be resolved later in ResolveResourceReferences mutator
	if variable.Lookup != nil {
		return v, variableOverrideSourceNone, nil
	}

	// case: Set the variable to the default value from the variable file
	if fileDefault.Kind() != dyn.KindInvalid && fileDefault.Kind() != dyn.KindNil {
		hasComplexType := variable.IsComplex()
		hasComplexValue := fileDefault.Kind() == dyn.KindMap || fileDefault.Kind() == dyn.KindSequence

		if hasComplexType && !hasComplexValue {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`variable %s is of type complex, but the value in the variable file is not a complex type`, name)
		}
		if !hasComplexType && hasComplexValue {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`variable %s is not of type complex, but the value in the variable file is a complex type`, name)
		}

		v, err := dyn.Set(v, "value", fileDefault)
		if err != nil {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`failed to assign default value from variable file to variable %s with error: %v`, name, err)
		}

		return v, variableOverrideSourceFile, nil
	}

	// case: Set the variable to its default value
	if variable.HasDefault() {
		vDefault, err := dyn.Get(v, "default")
		if err != nil {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`failed to get default value from config "%s" for variable %s with error: %v`, variable.Default, name, err)
		}

		v, err := dyn.Set(v, "value", vDefault)
		if err != nil {
			return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`failed to assign default value from config "%s" to variable %s with error: %v`, variable.Default, name, err)
		}
		return v, variableOverrideSourceDefault, nil
	}

	// We should have had a value to set for the variable at this point.
	return dyn.InvalidValue, variableOverrideSourceNone, fmt.Errorf(`no value assigned to required variable %s. Variables are usually assigned in databricks.yml, and they can be overridden using "--var", the %s environment variable, or %s`, name, bundleVarPrefix+name, getDefaultVariableFilePath("<target>"))
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

	envVarUsed := false
	fileUsed := false
	cliUsed := false

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "variables", dyn.Foreach(func(p dyn.Path, variable dyn.Value) (dyn.Value, error) {
			name := p[1].Key()
			v, ok := b.Config.Variables[name]
			if !ok {
				return dyn.InvalidValue, fmt.Errorf(`variable "%s" is not defined`, name)
			}

			fileDefault, _ := dyn.Get(defaults, name)
			result, source, err := setVariable(ctx, variable, v, name, fileDefault)
			switch source {
			case variableOverrideSourceCLI:
				cliUsed = true
			case variableOverrideSourceEnvVar:
				envVarUsed = true
			case variableOverrideSourceFile:
				fileUsed = true
			}
			return result, err
		}))
	})

	if envVarUsed {
		b.Metrics.SetBoolValue("variable_override_env_var_used", true)
	}
	if fileUsed {
		b.Metrics.SetBoolValue("variable_override_file_used", true)
	}
	if cliUsed {
		b.Metrics.SetBoolValue("variable_override_cli_flag_used", true)
	}

	return diags.Extend(diag.FromErr(err))
}
