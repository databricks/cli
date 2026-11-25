package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

// This file defines the `air register` YAML schema and its validation. engine selects
// what serves the checkpoint: the user's command when it's set, dedicated inference when
// it isn't. The server validates what needs the workspace, such as task names, runtimes
// and dependencies.

// registerCommandPath is the command whose -h documents registerConfig.
const registerCommandPath = "databricks air register"

// What serves a registered checkpoint.
const (
	// variantCommand: the command in engine.
	variantCommand = "command"
	// variantDedicated: Databricks, with dedicated inference.
	variantDedicated = "dedicated"
)

// Prefixes of model_source: a volume directory holding a checkpoint, or a logged
// model, named as MLflow names it.
const (
	volumeSourcePrefix      = "/Volumes/"
	loggedModelSourcePrefix = "models:/"
)

// Limits the server enforces on a registration, checked here to fail early.
const (
	maxServingCommandChars = 32 * 1024
	maxVolumePathChars     = 1024
	maxDependencies        = 1000
)

// registrationKeyRe matches the registration keys the server records.
var registrationKeyRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`) })

// registerConfig is the `air register` YAML schema: a checkpoint to register as a
// new version of a Unity Catalog model.
type registerConfig struct {
	ModelSource string                `yaml:"model_source" help:"Unity Catalog volume directory holding a checkpoint's weights, as /Volumes/<catalog>/<schema>/<volume>/<path>. Model Serving copies it, with its subdirectories, into the new version." required:"yes"`
	UCModel     string                `yaml:"uc_model" help:"Unity Catalog model, as <catalog>.<schema>.<model>. The checkpoint is registered as a new version of it, creating the model if it doesn't exist." required:"yes"`
	Engine      *registerEngineConfig `yaml:"engine" help:"Your own model server for the checkpoint: the command that starts it and the serverless environment it runs on. Without engine, Databricks serves the checkpoint with dedicated inference."`
}

// registerEngineConfig is the `engine` block: the model server of a checkpoint.
type registerEngineConfig struct {
	Environment *registerEnvironmentConfig `yaml:"environment" help:"Serverless environment to serve the command on, and the packages to install. The model uses its Python version." required:"yes"`
	Command     *string                    `yaml:"command" help:"Shell command that starts a model server over the checkpoint's weights on port 8080, at most 32,768 characters. It runs from the directory where the checkpoint appears under its own name." required:"yes"`
	Task        *string                    `yaml:"task" help:"The task Model Serving routes to the command: llm/v1/chat (the default), or another such as llm/v1/embeddings or llm/v1/audio/transcriptions."`
}

// registerEnvironmentConfig is the `engine.environment` block.
type registerEnvironmentConfig struct {
	Version      *stringOrInt `yaml:"version" required:"yes" help:"Serverless environment to serve on: 6 for the Standard environment or databricks_ai_v6 for the Databricks AI environment, as for air run. The model uses its Python version. Standard includes CUDA but not PyTorch; the Databricks AI environment adds PyTorch, Transformers and vLLM, and needs GPU compute. A command that doesn't install SGLang runs on the Databricks AI environment either way."`
	Dependencies []string     `yaml:"dependencies" help:"pip requirement lines to install, e.g. einops==0.8.0, at most 1,000."`
}

// variant is what serves the checkpoint: its command when engine is set, else
// dedicated inference.
func (c *registerConfig) variant() string {
	if c.Engine != nil {
		return variantCommand
	}
	return variantDedicated
}

// validate checks the whole config, returning the first failure. Fields are checked
// in declaration order to keep errors stable.
func (c *registerConfig) validate() error {
	// TODO: register a logged model (models:/m-<id>), which needs the server to copy
	// the logged model's files into the version.
	if strings.HasPrefix(c.ModelSource, loggedModelSourcePrefix) {
		return fmt.Errorf("model_source can't be a logged model yet, got %q: register it with MLflow's register_model, then serve the version with %s", c.ModelSource, serveCommandPath)
	}
	if err := validateVolumeDirectory("model_source", c.ModelSource); err != nil {
		return err
	}
	if err := validateUCModelName(c.UCModel, "uc_model"); err != nil {
		return err
	}
	if c.Engine == nil {
		return nil
	}
	return c.Engine.validate()
}

// validate checks the engine of a checkpoint served by command.
func (e *registerEngineConfig) validate() error {
	if e.Command == nil {
		return errors.New("engine.command is required: the shell command that starts your model server on port 8080")
	}
	if strings.TrimSpace(*e.Command) == "" {
		return errors.New("engine.command cannot be empty")
	}
	if n := serverLength(*e.Command); n > maxServingCommandChars {
		return fmt.Errorf("engine.command must be at most %d characters, got %d", maxServingCommandChars, n)
	}
	if e.Task != nil && strings.TrimSpace(*e.Task) == "" {
		return errors.New("engine.task cannot be empty; leave it out to serve llm/v1/chat")
	}
	if e.Environment == nil {
		return errors.New("engine.environment is required: the serverless environment and packages to serve the command on")
	}
	return e.Environment.validate()
}

// validateVolumeDirectory checks a Unity Catalog volume directory the server
// copies into the version.
func validateVolumeDirectory(field, v string) error {
	if !strings.HasPrefix(v, volumeSourcePrefix) || serverLength(v) > maxVolumePathChars {
		return fmt.Errorf("%s must be a Unity Catalog volume directory as /Volumes/<catalog>/<schema>/<volume>/<path>, of at most %d characters, got %q", field, maxVolumePathChars, v)
	}
	return nil
}

// validate checks the environment version and dependency lines.
func (e *registerEnvironmentConfig) validate() error {
	if e.Version == nil {
		return errors.New("engine.environment.version is required: the serverless environment to serve the command on, such as 6 or databricks_ai_v6")
	}
	if _, _, err := e.runtimeVersion(); err != nil {
		return err
	}
	if len(e.Dependencies) > maxDependencies {
		return fmt.Errorf("engine.environment.dependencies can list at most %d packages, got %d", maxDependencies, len(e.Dependencies))
	}
	for i, dependency := range e.Dependencies {
		if strings.TrimSpace(dependency) == "" {
			return fmt.Errorf("engine.environment.dependencies[%d] cannot be empty", i)
		}
	}
	return nil
}

// runtimeVersion is engine.environment.version as a version number, and whether it names
// the Databricks AI environment, as air run reads the same field.
func (e *registerEnvironmentConfig) runtimeVersion() (int, bool, error) {
	if e.Version == nil {
		return 0, false, errors.New("engine.environment.version is required")
	}
	normalized, err := validateRuntimeVersion(e.Version.raw, "engine.environment.version")
	if err != nil {
		return 0, false, err
	}
	ai := strings.HasPrefix(normalized, databricksAIPrefix)
	major, err := strconv.Atoi(strings.TrimPrefix(normalized, databricksAIPrefix))
	if err != nil || major < 1 {
		return 0, false, fmt.Errorf("engine.environment.version must be a serverless environment version such as 6 or databricks_ai_v6, got %q", e.Version.raw)
	}
	return major, ai, nil
}

// validateRegistrationKey checks a registration key from the --idempotency-key flag
// against the server's rule.
func validateRegistrationKey(key string) error {
	if !registrationKeyRe().MatchString(key) {
		return fmt.Errorf("--idempotency-key must be 1 to 128 letters, digits, '.', '_' or '-', got %q", key)
	}
	return nil
}

// registrationRequest builds the request that registers the checkpoint under key: the
// file's fields as they are, with the environment version in its canonical form, such
// as 6 for "06" or databricks_ai_v6 for "Databricks_AI_v6". The version was validated
// when the config loaded; the server defaults an omitted task to llm/v1/chat.
func (c *registerConfig) registrationRequest(key string) registerModelPackageRequest {
	req := registerModelPackageRequest{ModelSource: c.ModelSource, UCModel: c.UCModel, IdempotencyKey: key}
	if c.Engine == nil {
		return req
	}
	major, ai, _ := c.Engine.Environment.runtimeVersion()
	version := strconv.Itoa(major)
	if ai {
		version = databricksAIPrefix + version
	}
	req.Engine = &modelPackageEngine{
		Environment: modelPackageEnvironment{Version: version, Dependencies: c.Engine.Environment.Dependencies},
		Command:     *c.Engine.Command,
	}
	if c.Engine.Task != nil {
		req.Engine.Task = *c.Engine.Task
	}
	return req
}

// registerSchema is the reflection walk over registerConfig, shared by
// `air register -h config.<field>` and --override path validation.
func registerSchema() configField {
	return configField{
		path: configHelpRoot,
		help: `Registration YAML schema. For details on a field, pass its path to -h, for example:

    databricks air register -h config.model_source

  A checkpoint served by a command of your own:
    model_source: /Volumes/main/weights/qwen3_30b_a3b
    uc_model: main.serving.qwen30b_sglang
    engine:
      environment:
        version: 6
        dependencies: [sglang==0.5.23]
      command: python -m sglang.launch_server --model-path qwen3_30b_a3b --attention-backend triton --sampling-backend pytorch --host 0.0.0.0 --port 8080

  A checkpoint served with dedicated inference:
    model_source: /Volumes/main/weights/gemma4_e4b
    uc_model: main.serving.gemma4`,
		children: describeStruct(reflect.TypeFor[registerConfig](), configHelpRoot),
	}
}

// loadModelConfig reads a model command's YAML, applies --override entries against
// schema, and decodes it with unknown keys rejected, naming command's -h for the
// list of fields. It also returns the document, in
// which a key set to null is still present. ctx is used only to log applied overrides.
// The caller validates the result.
func loadModelConfig[T any](ctx context.Context, path string, overrides []string, schema configField, command string) (*T, *yaml.Node, error) {
	var (
		document       *yaml.Node
		validationYAML []byte
		err            error
	)
	if len(overrides) == 0 {
		document, validationYAML, err = readRunConfigDocument(path)
	} else {
		document, validationYAML, err = readConfigWithOverrides(ctx, path, overrides, schema)
	}
	if err != nil {
		return nil, nil, err
	}
	if len(document.Content) == 0 {
		return nil, nil, fmt.Errorf("config %s is empty", path)
	}
	if err := checkUnknownSchemaFields(path, document.Content[0], schema, command); err != nil {
		return nil, nil, err
	}
	cfg, err := decodeConfigReader[T](bytes.NewReader(validationYAML), path)
	return cfg, document, err
}
