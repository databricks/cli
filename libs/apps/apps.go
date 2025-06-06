package apps

import "context"

type App interface {
	PrepareEnvironment() error
	GetCommand(bool) ([]string, error)
}

func NewApp(ctx context.Context, config *Config, spec *AppSpec) App {
	// We only support python apps for now, but later we can add more types
	// based on AppSpec
	return NewPythonApp(ctx, config, spec)
}
