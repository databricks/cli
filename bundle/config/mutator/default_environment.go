package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
)

type defineDefaultEnvironment struct {
	name string
}

// DefineDefaultEnvironment adds an environment named "default"
// to the configuration if none have been defined.
func DefineDefaultEnvironment() bundle.Mutator {
	return &defineDefaultEnvironment{
		name: "default",
	}
}

func (m *defineDefaultEnvironment) Name() string {
	return fmt.Sprintf("DefineDefaultEnvironment(%s)", m.name)
}

func (m *defineDefaultEnvironment) Apply(_ context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// Nothing to do if the configuration has at least 1 environment.
	if b.Config.Environments != nil || len(b.Config.Environments) > 0 {
		return nil, nil
	}

	// Define default environment.
	b.Config.Environments = make(map[string]*config.Environment)
	b.Config.Environments[m.name] = &config.Environment{}
	return nil, nil
}
