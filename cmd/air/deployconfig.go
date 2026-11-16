package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/databricks/databricks-sdk-go/service/serving"
	"go.yaml.in/yaml/v3"
)

// This file defines the `air deploy` YAML schema and its structural validation.
// Fields shared with `air run` (environment, code_source, env_variables, secrets,
// idempotency_token) keep their meaning; the server validates what needs the
// workspace, such as task names, runtimes and dependencies.

// Model sources, as model.type.
const (
	modelTypeCheckpoint    = "checkpoint"
	modelTypeCode          = "code"
	modelTypeSavedModelDir = "saved_model_dir"
	modelTypeRegistered    = "registered"
)

// modelTypes lists model.type values in the order the help and errors name them.
var modelTypes = []string{modelTypeCheckpoint, modelTypeCode, modelTypeSavedModelDir, modelTypeRegistered}

// checkpointTasks are the tasks a checkpoint can serve, as the registration API
// documents them; models from code serve agent tasks too.
var checkpointTasks = []string{"llm/v1/chat", "llm/v1/completions", "llm/v1/embeddings"}

// deployCommandPath is the command whose -h documents deployConfig.
const deployCommandPath = "databricks air deploy"

// defaultServingEnvironmentVersion is the serverless environment version a model
// is served on when environment.version is unset, as for `air run`.
const defaultServingEnvironmentVersion = 6

// Limits the server enforces on a registration, checked here to fail early.
const (
	maxServingCommandChars = 32 * 1024
	maxVolumePathChars     = 1024
	maxDependencies        = 1000
)

// cpuWorkloadTypePrefix starts the name of every CPU serving workload type (CPU,
// CPU_MEDIUM, CPU_LARGE); the others are GPU workload types.
const cpuWorkloadTypePrefix = "CPU"

// registeredModelRe matches <catalog>.<schema>.<model> with the characters the
// server accepts in each part.
var registeredModelRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile("^[^\\s./`]+\\.[^\\s./`]+\\.[^\\s./`]+$") })

// deployIdempotencyKeyRe matches the registration keys the server records.
var deployIdempotencyKeyRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`) })

// modelVersionRe matches a Unity Catalog model version number.
var modelVersionRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[1-9][0-9]*$`) })

// endpointNameRe matches a serving endpoint name: letters, digits, '-' and '_'.
// The name is part of the API's URL paths, so nothing else may reach a request.
var endpointNameRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Za-z0-9_-]+$`) })

// workloadSizes are the concurrency tiers a served entity scales to.
var workloadSizes = []string{"Small", "Medium", "Large"}

// entryFileRe matches the name of a model from code's entry file, which the server
// places at the model's root.
var entryFileRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*\.py$`) })

// deployConfig is the `air deploy` YAML schema: a model to register in Unity
// Catalog and the serving endpoint that serves it.
type deployConfig struct {
	RegisteredModel string                   `yaml:"registered_model" help:"Unity Catalog model, as <catalog>.<schema>.<model>. air deploy registers a new version of it, creating the model if it doesn't exist, unless model.type is registered." required:"yes" group:"Model"`
	Model           *deployModelConfig       `yaml:"model" help:"What to serve: a checkpoint, a model from code, a model directory saved by MLflow, or an existing version." required:"yes" group:"Model"`
	Task            *string                  `yaml:"task" help:"The model's task as MLflow names it: llm/v1/chat, llm/v1/completions or llm/v1/embeddings for a checkpoint, or e.g. agent/v1/responses for a model from code. A model from code without a task has no signature." required:"when model.type is checkpoint" group:"Model"`
	Command         *string                  `yaml:"command" help:"Shell command that starts the model server of a checkpoint, at most 32,768 characters. It runs from the directory where model.path appears under its own name." required:"when model.type is checkpoint" group:"Model"`
	Environment     *deployEnvironmentConfig `yaml:"environment" help:"Serverless environment version to serve on and the packages to install, for a checkpoint or a model from code." group:"Model"`
	CodeSource      *deployCodeSourceConfig  `yaml:"code_source" help:"Local code of a model from code, sent with the registration and stored in the model version, where anyone who can read the model can read it. Only the files under include_paths are sent, leaving out files git ignores, hidden files and directories (names starting with '.') and __pycache__ directories; run with --dry-run to list them." required:"when model.type is code" group:"Model"`

	Endpoint     string               `yaml:"endpoint" help:"Serving endpoint to serve the model on. It is created if it doesn't exist. An endpoint that serves one version of registered_model is updated to serve this version instead, with the compute, environment variables and secrets in this file replacing the ones it has; air deploy doesn't change an endpoint that serves anything else." required:"yes" group:"Serving"`
	Compute      *deployComputeConfig `yaml:"compute" help:"Serving compute and how the endpoint scales." required:"yes" group:"Serving"`
	EnvVariables map[string]string    `yaml:"env_variables" help:"Plain environment variables of the served model, as NAME: value. A name here cannot also appear in secrets." group:"Serving"`
	Secrets      map[string]string    `yaml:"secrets" help:"Environment variables of the served model sourced from secrets, as NAME: scope/key." group:"Serving"`

	IdempotencyToken     *string `yaml:"idempotency_token" help:"Key of the registration: 1 to 128 letters, digits, '.', '_' or '-'. Running air deploy again with the same key and unchanged inputs continues that registration instead of registering another version. Defaults to a new key, which air deploy prints." group:"Registration"`
	MLflowExperimentPath *string `yaml:"mlflow_experiment_path" help:"Workspace path of the MLflow experiment that records the registered model (default: /Users/<you>/air-deployments)." group:"Registration"`
}

// deployModelConfig is the `model` block: where the served model comes from.
type deployModelConfig struct {
	Type    string  `yaml:"type" help:"One of checkpoint (weights in a volume, started by command), code (a Python file from code_source), saved_model_dir (a model directory saved by MLflow in a volume) or registered (an existing version of registered_model)." required:"yes"`
	Path    *string `yaml:"path" help:"Unity Catalog volume directory, as /Volumes/<catalog>/<schema>/<volume>/<path>, holding a checkpoint's weights or a model saved by MLflow. Model Serving copies it into the version." required:"when type is checkpoint or saved_model_dir"`
	Entry   *string `yaml:"entry" help:"Python file directly under code_source.root_path that builds the model and calls mlflow.models.set_model, e.g. agent.py." required:"when type is code"`
	Version *string `yaml:"version" help:"Version of registered_model to serve, e.g. 3." required:"when type is registered"`
}

// deployEnvironmentConfig is the `environment` block of a deployment.
type deployEnvironmentConfig struct {
	Version      *int     `yaml:"version" help:"Serverless environment version to serve on, e.g. 6 (default: 6). The model uses its Python version."`
	Dependencies []string `yaml:"dependencies" help:"pip requirement lines to install, e.g. einops==0.8.0, at most 1,000. A model from code must list mlflow or mlflow-skinny."`
}

// deployCodeSourceConfig is the `code_source` block of a model from code.
type deployCodeSourceConfig struct {
	RootPath     string   `yaml:"root_path" help:"Directory holding the model's code. A relative path resolves against the YAML file's directory." required:"yes"`
	IncludePaths []string `yaml:"include_paths" help:"Files and directories to send, relative to root_path and without \"..\", e.g. [agent.py, tools]." required:"yes"`
}

// deployComputeConfig is the `compute` block: the served entity's workload.
type deployComputeConfig struct {
	WorkloadType string `yaml:"workload_type" help:"Serving compute, e.g. CPU, GPU_SMALL, GPU_MEDIUM or GPU_LARGE. A checkpoint is served on a GPU workload type." required:"yes"`
	WorkloadSize string `yaml:"workload_size" help:"How far the endpoint scales: Small, Medium or Large." required:"yes"`
	ScaleToZero  bool   `yaml:"scale_to_zero" help:"Whether the endpoint scales to zero when idle (default: false)."`
}

// sourceFields lists, per model.type, the optional fields that apply to it, in
// addition to the required ones in requiredSourceFields. Every other source field
// is rejected, so a field the source can't use is never silently dropped.
var sourceFields = map[string][]string{
	modelTypeCheckpoint:    {"environment", "idempotency_token", "mlflow_experiment_path"},
	modelTypeCode:          {"task", "environment", "idempotency_token", "mlflow_experiment_path"},
	modelTypeSavedModelDir: {"idempotency_token", "mlflow_experiment_path"},
	modelTypeRegistered:    {},
}

// requiredSourceFields lists the fields each model.type requires.
var requiredSourceFields = map[string][]string{
	modelTypeCheckpoint:    {"model.path", "task", "command"},
	modelTypeCode:          {"model.entry", "code_source"},
	modelTypeSavedModelDir: {"model.path"},
	modelTypeRegistered:    {"model.version"},
}

// validate runs structural validation over the whole config, returning the
// first failure. Fields are checked in declaration order to keep errors stable.
func (c *deployConfig) validate() error {
	if !registeredModelRe().MatchString(c.RegisteredModel) {
		return fmt.Errorf("registered_model must be a Unity Catalog model name as <catalog>.<schema>.<model>, got %q", c.RegisteredModel)
	}
	if c.Model == nil {
		return errors.New("model: section is required")
	}
	if err := c.validateSource(); err != nil {
		return err
	}
	if !endpointNameRe().MatchString(c.Endpoint) {
		return fmt.Errorf("endpoint must be a serving endpoint name of letters, digits, '-' and '_', got %q", c.Endpoint)
	}
	if c.Compute == nil {
		return errors.New("compute: section is required")
	}
	if workloadTypes := servingWorkloadTypes(); !slices.Contains(workloadTypes, c.Compute.WorkloadType) {
		return fmt.Errorf("compute.workload_type must be one of %s, got %q", strings.Join(workloadTypes, ", "), c.Compute.WorkloadType)
	}
	if !slices.Contains(workloadSizes, c.Compute.WorkloadSize) {
		return fmt.Errorf("compute.workload_size must be one of %s, got %q", strings.Join(workloadSizes, ", "), c.Compute.WorkloadSize)
	}
	if c.Model.Type == modelTypeCheckpoint && !c.servesOnGPU() {
		return fmt.Errorf("a checkpoint is served on GPU; set compute.workload_type to a GPU workload type such as GPU_MEDIUM, got %q", c.Compute.WorkloadType)
	}
	if err := validateEnvironmentVariables(c.EnvVariables, c.Secrets); err != nil {
		return err
	}
	if c.IdempotencyToken != nil {
		if err := validateDeployIdempotencyKey(*c.IdempotencyToken, "idempotency_token"); err != nil {
			return err
		}
	}
	if c.MLflowExperimentPath != nil && !strings.HasPrefix(*c.MLflowExperimentPath, "/") {
		return fmt.Errorf("mlflow_experiment_path must be an absolute workspace path such as /Users/<you>/air-deployments, got %q", *c.MLflowExperimentPath)
	}
	return nil
}

// validateSource checks model.type and the fields that describe the source.
func (c *deployConfig) validateSource() error {
	m := c.Model
	if !slices.Contains(modelTypes, m.Type) {
		return fmt.Errorf("model.type must be one of %s, got %q", strings.Join(modelTypes, ", "), m.Type)
	}

	set := map[string]bool{
		"model.path":             m.Path != nil,
		"model.entry":            m.Entry != nil,
		"model.version":          m.Version != nil,
		"task":                   c.Task != nil,
		"command":                c.Command != nil,
		"environment":            c.Environment != nil,
		"code_source":            c.CodeSource != nil,
		"idempotency_token":      c.IdempotencyToken != nil,
		"mlflow_experiment_path": c.MLflowExperimentPath != nil,
	}
	required := requiredSourceFields[m.Type]
	for _, name := range required {
		if !set[name] {
			return fmt.Errorf("%s is required when model.type is %s", name, m.Type)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(set)) {
		if set[name] && !slices.Contains(required, name) && !slices.Contains(sourceFields[m.Type], name) {
			return fmt.Errorf("%s can't be set when model.type is %s", name, m.Type)
		}
	}

	if m.Path != nil {
		v := *m.Path
		if !strings.HasPrefix(v, "/Volumes/") || serverLength(v) > maxVolumePathChars {
			return fmt.Errorf("model.path must be a Unity Catalog volume directory as /Volumes/<catalog>/<schema>/<volume>/<path>, of at most %d characters, got %q", maxVolumePathChars, v)
		}
	}
	if m.Entry != nil && !entryFileRe().MatchString(*m.Entry) {
		return fmt.Errorf("model.entry must name a .py file directly under code_source.root_path, e.g. agent.py, got %q", *m.Entry)
	}
	if m.Version != nil && !modelVersionRe().MatchString(*m.Version) {
		return fmt.Errorf("model.version must be a model version number such as 3, got %q", *m.Version)
	}
	if c.Task != nil && strings.TrimSpace(*c.Task) == "" {
		return errors.New("task cannot be empty")
	}
	if m.Type == modelTypeCheckpoint && !slices.Contains(checkpointTasks, *c.Task) {
		return fmt.Errorf("task of a checkpoint must be one of %s, got %q", strings.Join(checkpointTasks, ", "), *c.Task)
	}
	if c.Command != nil {
		if err := validateCommand(*c.Command); err != nil {
			return err
		}
		if n := serverLength(*c.Command); n > maxServingCommandChars {
			return fmt.Errorf("command must be at most %d characters, got %d", maxServingCommandChars, n)
		}
	}
	if c.Environment != nil {
		if err := c.Environment.validate(); err != nil {
			return err
		}
	}
	if c.CodeSource != nil {
		// The files are stored in the model version for anyone who can read the
		// model, so the user names them rather than sending a whole directory.
		if c.CodeSource.IncludePaths == nil {
			return errors.New("code_source.include_paths is required: list the files and directories to send, e.g. include_paths: [agent.py, tools]")
		}
		// git reads a leading ':' as pathspec magic, such as ":!x" for everything
		// but x, and expands globs, either of which would send files the list
		// doesn't name; padding would match nothing.
		for _, p := range c.CodeSource.IncludePaths {
			if strings.HasPrefix(p, ":") || strings.ContainsAny(p, "*?[") || p != strings.TrimSpace(p) {
				return fmt.Errorf("code_source.include_paths must name files and directories, without patterns or surrounding spaces, got %q", p)
			}
		}
		snapshot := snapshotSourceConfig{RootPath: c.CodeSource.RootPath, IncludePaths: c.CodeSource.IncludePaths}
		if err := snapshot.validate("code_source"); err != nil {
			return err
		}
	}
	return nil
}

// validate checks the environment version and dependency lines.
func (e *deployEnvironmentConfig) validate() error {
	if e.Version != nil && *e.Version < 1 {
		return fmt.Errorf("environment.version must be a serverless environment version such as 6, got %d", *e.Version)
	}
	if len(e.Dependencies) > maxDependencies {
		return fmt.Errorf("environment.dependencies can list at most %d packages, got %d", maxDependencies, len(e.Dependencies))
	}
	for i, dependency := range e.Dependencies {
		if strings.TrimSpace(dependency) == "" {
			return fmt.Errorf("environment.dependencies[%d] cannot be empty", i)
		}
	}
	return nil
}

// validateDeployIdempotencyKey checks a registration key from source (the YAML
// field or the flag) against the server's rule.
func validateDeployIdempotencyKey(key, source string) error {
	if !deployIdempotencyKeyRe().MatchString(key) {
		return fmt.Errorf("%s must be 1 to 128 letters, digits, '.', '_' or '-', got %q", source, key)
	}
	return nil
}

// serverLength is the length of s as the server's limits count it, in UTF-16
// code units.
func serverLength(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// servingWorkloadTypes lists the served entity workload types, sorted.
func servingWorkloadTypes() []string {
	var workloadType serving.ServingModelWorkloadType
	var types []string
	for _, t := range workloadType.Values() {
		types = append(types, string(t))
	}
	slices.Sort(types)
	return types
}

// registers reports whether air deploy registers a new version, rather than
// serving an existing one.
func (c *deployConfig) registers() bool {
	return c.Model.Type != modelTypeRegistered
}

// servesOnGPU reports whether compute.workload_type is a GPU workload type.
func (c *deployConfig) servesOnGPU() bool {
	return !strings.HasPrefix(c.Compute.WorkloadType, cpuWorkloadTypePrefix)
}

// servingRuntime is the serverless runtime the model is served on, as the server
// names it: client.<version>, with a -gpu suffix on GPU compute.
func (c *deployConfig) servingRuntime() string {
	version := defaultServingEnvironmentVersion
	if c.Environment != nil && c.Environment.Version != nil {
		version = *c.Environment.Version
	}
	runtime := "client." + strconv.Itoa(version)
	if c.servesOnGPU() {
		runtime += "-gpu"
	}
	return runtime
}

// servingEnvironment is the registration's runtime and dependencies.
func (c *deployConfig) servingEnvironment() modelPackageEnvironment {
	environment := modelPackageEnvironment{Runtime: c.servingRuntime()}
	if c.Environment != nil {
		environment.Dependencies = c.Environment.Dependencies
	}
	return environment
}

// servedEnvironmentVars merges env_variables and secrets into the served
// entity's environment variables, with each secret as {{secrets/scope/key}}.
func (c *deployConfig) servedEnvironmentVars() map[string]string {
	if len(c.EnvVariables) == 0 && len(c.Secrets) == 0 {
		return nil
	}
	vars := maps.Clone(c.EnvVariables)
	if vars == nil {
		vars = map[string]string{}
	}
	for name, ref := range c.Secrets {
		vars[name] = "{{secrets/" + ref + "}}"
	}
	return vars
}

// deploySchema is the reflection walk over deployConfig, shared by
// `air deploy -h config.<field>` and --override path validation.
func deploySchema() configField {
	return configField{
		path: configHelpRoot,
		help: `Deployment YAML schema. For details on a field, pass its path to -h, for example:

    databricks air deploy -h config.model.type

  Minimal example, a checkpoint served by vLLM:
    registered_model: main.models.qwen
    model:
      type: checkpoint
      path: /Volumes/main/models/weights/qwen3_8b
    task: llm/v1/chat
    command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --served-model-name base --host 0.0.0.0 --port 8080
    endpoint: qwen-chat
    compute:
      workload_type: GPU_MEDIUM
      workload_size: Small`,
		children: describeStruct(reflect.TypeFor[deployConfig](), configHelpRoot),
	}
}

// loadDeployConfig reads a deployment YAML, applies --override entries, decodes
// it with unknown keys rejected and validates it. ctx is used only to log applied
// overrides.
func loadDeployConfig(ctx context.Context, path string, overrides []string) (*deployConfig, error) {
	var (
		document       *yaml.Node
		validationYAML []byte
		err            error
	)
	if len(overrides) == 0 {
		document, validationYAML, err = readRunConfigDocument(path)
	} else {
		document, validationYAML, err = readConfigWithOverrides(ctx, path, overrides, deploySchema())
	}
	if err != nil {
		return nil, err
	}
	if len(document.Content) == 0 {
		return nil, fmt.Errorf("config %s is empty", path)
	}

	cfg, err := decodeConfigReader[deployConfig](bytes.NewReader(validationYAML), path)
	if err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
