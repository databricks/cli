package mutator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/variable"
)

// ucmVarPrefix is the env-var prefix that can assign ucm variables without
// touching the ucm.yml. `DATABRICKS_UCM_VAR_foo=bar` resolves to the same
// effective state as `--var foo=bar` and as the `variables.foo.default` key.
const ucmVarPrefix = "DATABRICKS_UCM_VAR_"

type setVariables struct{}

// SetVariables walks Config.Variables and assigns each one a value based on
// the DAB-parity resolution order: existing .Value > env var > lookup
// (deferred) > override file > default.
func SetVariables() ucm.Mutator {
	return &setVariables{}
}

func (m *setVariables) Name() string { return "SetVariables" }

func getDefaultVariableFilePath(target string) string {
	return ".databricks/ucm/" + target + "/variable-overrides.json"
}

func (m *setVariables) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	defaults, diags := readVariablesFromFile(u)
	if diags.HasError() {
		return diags
	}
	err := u.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "variables", dyn.Foreach(func(p dyn.Path, raw dyn.Value) (dyn.Value, error) {
			name := p[1].Key()
			defn, ok := u.Config.Variables[name]
			if !ok {
				return dyn.InvalidValue, fmt.Errorf(`variable "%s" is not defined`, name)
			}
			fileDefault, _ := dyn.Get(defaults, name)
			return setVariable(ctx, raw, defn, name, fileDefault)
		}))
	})
	return diags.Extend(diag.FromErr(err))
}

// setVariable applies the priority ladder to a single variable. Must be kept
// consistent with variable.Variable.Value docstring.
func setVariable(ctx context.Context, v dyn.Value, defn *variable.Variable, name string, fileDefault dyn.Value) (dyn.Value, error) {
	// Already assigned — e.g. by --var before the mutator chain ran.
	if defn.HasValue() {
		return v, nil
	}

	envVarName := ucmVarPrefix + name
	if val, ok := env.Lookup(ctx, envVarName); ok {
		if defn.IsComplex() {
			return dyn.InvalidValue, fmt.Errorf(`setting via environment variables (%s) is not supported for complex variable %s`, envVarName, name)
		}
		nv, err := dyn.Set(v, "value", dyn.V(val))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to assign value "%s" to variable %s from environment variable %s: %w`, val, name, envVarName, err)
		}
		return nv, nil
	}

	// Lookup is resolved later by a separate mutator once a WorkspaceClient
	// is available. Leave the dyn value alone here. This short-circuit runs
	// before the file-override branch so a lookup variable is not clobbered
	// by a matching entry in variable-overrides.json.
	if defn.Lookup != nil {
		return v, nil
	}

	if fileDefault.Kind() != dyn.KindInvalid && fileDefault.Kind() != dyn.KindNil {
		hasComplexType := defn.IsComplex()
		hasComplexValue := fileDefault.Kind() == dyn.KindMap || fileDefault.Kind() == dyn.KindSequence

		if hasComplexType && !hasComplexValue {
			return dyn.InvalidValue, fmt.Errorf(`variable %s is of type complex, but the value in the variable file is not a complex type`, name)
		}
		if !hasComplexType && hasComplexValue {
			return dyn.InvalidValue, fmt.Errorf(`variable %s is not of type complex, but the value in the variable file is a complex type`, name)
		}

		nv, err := dyn.Set(v, "value", fileDefault)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to assign default value from variable file to variable %s with error: %v`, name, err)
		}
		return nv, nil
	}

	if defn.HasDefault() {
		vDefault, err := dyn.Get(v, "default")
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to get default value for variable %s: %w`, name, err)
		}
		nv, err := dyn.Set(v, "value", vDefault)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf(`failed to assign default value to variable %s: %w`, name, err)
		}
		return nv, nil
	}

	return dyn.InvalidValue, fmt.Errorf(`no value assigned to required variable %s. Variables are usually assigned in ucm.yml, and they can be overridden using "--var", the %s environment variable, or %s`, name, envVarName, getDefaultVariableFilePath("<target>"))
}

func readVariablesFromFile(u *ucm.Ucm) (dyn.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	filePath := filepath.Join(u.RootPath, getDefaultVariableFilePath(u.Config.Ucm.Target))
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
