package mutator

import (
	"fmt"

	"github.com/databricks/bricks/bundle/config"
)

type defineDefaultEnvironment struct {
	name string
}

func DefineDefaultEnvironment() Mutator {
	return &defineDefaultEnvironment{
		name: "default",
	}
}

func (m *defineDefaultEnvironment) Name() string {
	return fmt.Sprintf("DefineDefaultEnvironment(%s)", m.name)
}

func (m *defineDefaultEnvironment) Apply(root *config.Root) ([]Mutator, error) {
	// Nothing to do if the configuration has at least 1 environment.
	if root.Environments != nil || len(root.Environments) > 0 {
		return nil, nil
	}

	// Define default environment.
	root.Environments = make(map[string]*config.Environment)
	root.Environments[m.name] = &config.Environment{}
	return nil, nil
}
