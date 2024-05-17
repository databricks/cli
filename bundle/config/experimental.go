package config

type Experimental struct {
	Scripts map[ScriptHook]Command `json:"scripts,omitempty"`

	// By default Python wheel tasks deployed as is to Databricks platform.
	// If notebook wrapper required (for example, used in DBR < 13.1 or other configuration differences), users can provide a following experimental setting
	// experimental:
	//    python_wheel_wrapper: true
	// In this case the configured wheel task will be deployed as a notebook task which install defined wheel in runtime and executes it.
	// For more details see https://github.com/databricks/cli/pull/797 and https://github.com/databricks/cli/pull/635
	PythonWheelWrapper bool `json:"python_wheel_wrapper,omitempty"`

	// Enable legacy run_as behavior. That is:
	// - Set the run_as identity as the owner of any pipelines in the bundle.
	// - Do not error in the presence of resources that do not support run_as.
	//   As of April 2024 this includes pipelines and model serving endpoints.
	//
	// This mode of run_as requires the deploying user to be a workspace and metastore
	// admin. Use of this flag is not recommend for new bundles, and it is only provided
	// to unblock customers that are stuck due to breaking changes in the run_as behavior
	// made in https://github.com/databricks/cli/pull/1233. This flag might
	// be removed in the future once we have a proper workaround like allowing IS_OWNER
	// as a top-level permission in the DAB.
	UseLegacyRunAs bool `json:"use_legacy_run_as,omitempty"`

	// PyDABs if enabled loads the 'databricks-pydabs' package.
	//
	// This allows to define bundle configuration using Python. For now, this option has to be used
	// together with 'venv' configuration.
	PyDABs PyDABs `json:"pydabs,omitempty"`

	// VEnv is the virtual environment configuration for the bundle.
	//
	// When enabled, the bundle will use Python interpreter from the virtual environment.
	VEnv VEnv `json:"venv,omitempty"`
}

type VEnv struct {
	// Path to the virtual environment.
	Path string `json:"path,omitempty"`
}

type PyDABs struct {
	// Enable is a flag to enable the feature.
	Enable bool `json:"enable,omitempty"`
}

type Command string
type ScriptHook string

// These hook names are subject to change and currently experimental
const (
	ScriptPreInit    ScriptHook = "preinit"
	ScriptPostInit   ScriptHook = "postinit"
	ScriptPreBuild   ScriptHook = "prebuild"
	ScriptPostBuild  ScriptHook = "postbuild"
	ScriptPreDeploy  ScriptHook = "predeploy"
	ScriptPostDeploy ScriptHook = "postdeploy"
)
