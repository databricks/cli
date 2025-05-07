package apps

type App interface {
	PrepareEnvironment() error
	GetCommand(bool) ([]string, error)
}

func NewApp(config *Config, spec *AppSpec) App {
	// We only support python apps for now, but later we can add more types
	// based on AppSpec
	return NewPythonApp(config, spec)
}
