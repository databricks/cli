package mutator

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/bricks/bundle"
)

const bundleVarPrefix = "BUNDLE_VAR_"

type setVariables struct{}

func SetVariables() bundle.Mutator {
	return &setVariables{}
}

func (m *setVariables) Name() string {
	return "SetVariables"
}

func (m *setVariables) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	for name, variable := range b.Config.Variables {
		// variable already has value initialized, so skip
		if variable.HasValue() {
			continue
		}

		// read and set variable value from environment
		if val, ok := os.LookupEnv(bundleVarPrefix + name); ok {
			err := variable.Set(val)
			if err != nil {
				return nil, fmt.Errorf("failed to assign %s to %s: %s", val, name, err)
			}
		}

		// Set the varliable to it's default value
		if variable.HasDefault() {
			err := variable.Set(variable.Default)
			return nil, fmt.Errorf("failed to assign %s to %s: %s", variable.Default, name, err)
		}

		return nil, fmt.Errorf(`no value assigned to required variable %s. Assignment can be done through the "--var" flag or by setting the %s environment variable`, name, bundleVarPrefix+name)
	}
	return nil, nil
}
