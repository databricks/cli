package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

func setVariable(ctx context.Context, v dyn.Value, variable *variable.Variable, name string) (dyn.Value, error) {
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
	return dyn.InvalidValue, fmt.Errorf(`no value assigned to required variable %s. Assignment can be done through the "--var" flag or by setting the %s environment variable`, name, bundleVarPrefix+name)

}

func (m *setVariables) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Updating variable "value" locaiton to its variable "default" locaiton
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "variables", dyn.Foreach(func(p dyn.Path, variable dyn.Value) (dyn.Value, error) {
			name := p[1].Key()
			v, ok := b.Config.Variables[name]
			if !ok {
				return dyn.InvalidValue, fmt.Errorf(`variable "%s" is not defined`, name)
			}

			return setVariable(ctx, variable, v, name)
		}))
	})

	return diag.FromErr(err)
}
