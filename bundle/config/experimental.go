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
