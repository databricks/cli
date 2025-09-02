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

	// PyDABs is deprecated use Python instead.
	PyDABs PyDABs `json:"pydabs,omitempty"`

	// Python configures loading of Python code defined with 'databricks-bundles' package.
	Python Python `json:"python,omitempty"`

	// SkipArtifactCleanup determines whether to skip cleaning up the .internal folder
	// containing build artifacts such as wheels. When set to true, the .internal folder
	// and its contents will be preserved after bundle operations complete.
	SkipArtifactCleanup bool `json:"skip_artifact_cleanup,omitempty"`

	// SkipNamePrefixForSchema skips adding the `presets.name_prefix` prefix
	// to UC schemas defined in the bundle. Currently this is a opt-in behavior
	// because turning it on by default will cause schema recreates and lose
	// customer data.
	// Eventually this can be made the default once we have native CRUD in DABs
	// at which point we can deprecate or remove this field all together.
	SkipNamePrefixForSchema bool `json:"skip_name_prefix_for_schema,omitempty"`
}

type Python struct {
	// Resources contains a list of fully qualified function paths to load resources
	// defined in Python code.
	//
	// Example: ["my_project.resources:load_resources"]
	Resources []string `json:"resources,omitempty"`

	// Mutators contains a list of fully qualified function paths to mutator functions.
	//
	// Example: ["my_project.mutators:add_default_cluster"]
	Mutators []string `json:"mutators,omitempty"`

	// VEnvPath is path to the virtual environment.
	//
	// If enabled, Python code will execute within this environment. If disabled,
	// it defaults to using the Python interpreter available in the current shell.
	VEnvPath string `json:"venv_path,omitempty"`
}

// PyDABs is deprecated use Python instead
type PyDABs struct {
	// Enabled is a flag to enable the feature.
	Enabled bool `json:"enabled,omitempty"`
}

type (
	Command    string
	ScriptHook string
)

// These hook names are subject to change and currently experimental
const (
	ScriptPreInit    ScriptHook = "preinit"
	ScriptPostInit   ScriptHook = "postinit"
	ScriptPreBuild   ScriptHook = "prebuild"
	ScriptPostBuild  ScriptHook = "postbuild"
	ScriptPreDeploy  ScriptHook = "predeploy"
	ScriptPostDeploy ScriptHook = "postdeploy"
)
