package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
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

func setVariable(ctx context.Context, v *variable.Variable, name string) error {
	// case: variable already has value initialized, so skip
	if v.HasValue() {
		return nil
	}

	// case: read and set variable value from process environment
	envVarName := bundleVarPrefix + name
	if val, ok := env.Lookup(ctx, envVarName); ok {
		err := v.Set(val)
		if err != nil {
			return fmt.Errorf(`failed to assign value "%s" to variable %s from environment variable %s with error: %w`, val, name, envVarName, err)
		}
		return nil
	}

	// case: Set the variable to its default value
	if v.HasDefault() {
		err := v.Set(*v.Default)
		if err != nil {
			return fmt.Errorf(`failed to assign default value from config "%s" to variable %s with error: %w`, *v.Default, name, err)
		}
		return nil
	}

	// We should have had a value to set for the variable at this point.
	// TODO: use cmdio to request values for unassigned variables if current
	// terminal is a tty. Tracked in https://github.com/databricks/cli/issues/379
	return fmt.Errorf(`no value assigned to required variable %s. Assignment can be done through the "--var" flag or by setting the %s environment variable`, name, bundleVarPrefix+name)
}

func (m *setVariables) Apply(ctx context.Context, b *bundle.Bundle) error {
	for name, variable := range b.Config.Variables {
		err := setVariable(ctx, variable, name)
		if err != nil {
			return err
		}
	}
	return nil
}
