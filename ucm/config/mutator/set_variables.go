package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
// the DAB-parity resolution order: existing .Value > env var > lookup (deferred) > default.
func SetVariables() ucm.Mutator {
	return &setVariables{}
}

func (m *setVariables) Name() string { return "SetVariables" }

func (m *setVariables) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	err := u.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "variables", dyn.Foreach(func(p dyn.Path, raw dyn.Value) (dyn.Value, error) {
			name := p[1].Key()
			defn, ok := u.Config.Variables[name]
			if !ok {
				return dyn.InvalidValue, fmt.Errorf(`variable "%s" is not defined`, name)
			}
			return setVariable(ctx, raw, defn, name)
		}))
	})
	return diag.FromErr(err)
}

// setVariable applies the priority ladder to a single variable. Must be kept
// consistent with variable.Variable.Value docstring.
func setVariable(ctx context.Context, v dyn.Value, defn *variable.Variable, name string) (dyn.Value, error) {
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
	// is available. Leave the dyn value alone here.
	if defn.Lookup != nil {
		return v, nil
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

	return dyn.InvalidValue, fmt.Errorf(`no value assigned to required variable %s. Assign a default in ucm.yml or override via "--var %s=..." or the %s environment variable`, name, name, envVarName)
}
