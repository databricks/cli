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

	"go.yaml.in/yaml/v3"
)

// This file defines the `air deploy` YAML schema and its validation. The fields a
// file sets select its variant, and each variant reads its own fields. Fields shared
// with `air run` (environment, env_variables, secrets, idempotency_token) keep their
// meaning; the server validates what needs the workspace, such as task names,
// runtimes and dependencies.

// Variants of a deployment, named after what serves the checkpoint.
const (
	// variantCommand: a checkpoint served by the command the file sets.
	variantCommand = "command"
	// variantDedicated: a checkpoint Databricks serves with dedicated inference.
	variantDedicated = "dedicated"
	// variantRegistered: a model version already registered in Unity Catalog, which
	// air deploy serves without registering another.
	variantRegistered = "registered"
)

// variantDescriptions and variantReasons say how a file was read, in errors about
// its fields: "the file is read as <description>, because <reason>".
var (
	variantDescriptions = map[string]string{
		variantCommand:    "a checkpoint served by command",
		variantDedicated:  "a dedicated checkpoint",
		variantRegistered: "a registered model",
	}
	variantReasons = map[string]string{
		variantCommand:    "it sets checkpoint and command",
		variantDedicated:  "it sets checkpoint but not command",
		variantRegistered: "it sets no checkpoint",
	}
)

// Audio tasks: their requests are multipart uploads of a recording, sent to the
// endpoint's audio route rather than to invocations.
const (
	taskAudioTranscriptions = "llm/v1/audio/transcriptions"
	taskAudioTranslations   = "llm/v1/audio/translations"
)

// deployCommandPath is the command whose -h documents deployConfig.
const deployCommandPath = "databricks air deploy"

// Serving compute a command or registered model gets when compute is omitted: one
// A10 for a command, CPU for a registered model.
const (
	defaultCommandWorkloadType    = "GPU_MEDIUM"
	defaultRegisteredWorkloadType = "CPU"
	defaultWorkloadSize           = "Small"
)

// Limits the server enforces on a registration, checked here to fail early.
const (
	maxServingCommandChars = 32 * 1024
	maxVolumePathChars     = 1024
	maxDependencies        = 1000
)

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

// servingShape is serving compute the server offers: an accelerator type, the number
// of accelerators, and the workload type that serves them.
type servingShape struct {
	accelerator  gpuType
	count        int
	workloadType string
}

// servingShapes lists the serving compute a command or registered model can use, in
// the order errors name them.
var servingShapes = []servingShape{
	{gpuType1xA10, 1, "GPU_MEDIUM"},
	{gpuType1xA10, 4, "MULTIGPU_MEDIUM"},
	{gpuType1xA10, 8, "GPU_MEDIUM_8"},
	{gpuType1xH100, 1, "GPU_XLARGE"},
	{gpuType8xH100, 8, "GPU_XLARGE_8"},
	{gpuType1xRTX6000, 1, "GPU_LARGE_RTX"},
}

// computeProfiles are the dedicated compute profiles, lowercase as the file names
// them. The API takes each as COMPUTE_PROFILE_ followed by its uppercase form.
var computeProfiles = []string{"low_latency", "fast", "throughput", "minimal_cost"}

// deployConfig is the `air deploy` YAML schema: a model to serve on an endpoint,
// and the checkpoint to register as a version of it, if any.
type deployConfig struct {
	RegisteredModel string                   `yaml:"registered_model_name" help:"Unity Catalog model, as <catalog>.<schema>.<model>. A checkpoint is registered as a new version of it, creating the model if it doesn't exist. Without a checkpoint, the model must already have a version to serve." required:"yes" group:"Model"`
	Checkpoint      *string                  `yaml:"checkpoint" help:"Unity Catalog volume directory holding a checkpoint's weights, as /Volumes/<catalog>/<schema>/<volume>/<path>. Model Serving copies it, with its subdirectories, into the version air deploy registers. Leave it out to serve a version that is already registered." group:"Model"`
	Command         *string                  `yaml:"command" help:"Shell command that starts a model server over the checkpoint's weights on port 8080, at most 32,768 characters. It runs from the directory where the checkpoint appears under its own name. Set it to serve the checkpoint with your own server; without it, Databricks serves the checkpoint with dedicated inference." group:"Model"`
	Task            *string                  `yaml:"task" help:"The task Model Serving routes to the command, such as llm/v1/chat, llm/v1/embeddings or llm/v1/audio/transcriptions." required:"when command is set" group:"Model"`
	Environment     *deployEnvironmentConfig `yaml:"environment" help:"Serverless environment to serve the command on, and the packages to install. The model uses its Python version." required:"when command is set" group:"Model"`
	ModelVersion    *string                  `yaml:"model_version" help:"Version of registered_model to serve, e.g. 3. Without it, the newest READY version is served." group:"Model"`

	Endpoint     string               `yaml:"endpoint_name" help:"Serving endpoint to serve the model on. It is created if it doesn't exist. An endpoint that serves one version of registered_model is updated to serve this version instead, with the compute, environment variables and secrets in this file replacing the ones it has; air deploy doesn't change an endpoint that serves anything else." required:"yes" group:"Serving"`
	Compute      *deployComputeConfig `yaml:"compute" help:"Compute to serve on. A command or registered model takes serving compute, and a dedicated checkpoint takes dedicated compute. Without compute, a command is served on one GPU_1xA10 with workload size Small, and a registered model on CPU with workload size Small." required:"when checkpoint is set without command" group:"Serving"`
	EnvVariables map[string]string    `yaml:"env_variables" help:"Plain environment variables of the served model, as NAME: value. A name here cannot also appear in secrets. A command or registered model only." group:"Serving"`
	Secrets      map[string]string    `yaml:"secrets" help:"Environment variables of the served model sourced from secrets, as NAME: scope/key. A command or registered model only." group:"Serving"`

	IdempotencyToken     *string `yaml:"idempotency_token" help:"Key of the registration: 1 to 128 letters, digits, '.', '_' or '-'. Running air deploy again with the same key and unchanged inputs continues that registration instead of registering another version. Defaults to a new key, which air deploy prints. A checkpoint only." group:"Registration"`
	MLflowExperimentPath *string `yaml:"mlflow_experiment_path" help:"Workspace path of the MLflow experiment that records the registered model (default: /Users/<you>/air-deployments). A checkpoint only." group:"Registration"`
}

// deployEnvironmentConfig is the `environment` block of a command.
type deployEnvironmentConfig struct {
	Version      *stringOrInt `yaml:"version" required:"yes" help:"Serverless environment to serve on: 6 for the Standard environment or databricks_ai_v6 for the Databricks AI environment, as for air run. The model uses its Python version. Standard includes CUDA but not PyTorch; the Databricks AI environment adds PyTorch, Transformers and vLLM, and needs GPU compute. A command that doesn't install SGLang runs on the Databricks AI environment either way."`
	Dependencies []string     `yaml:"dependencies" help:"pip requirement lines to install, e.g. einops==0.8.0, at most 1,000."`
}

// deployComputeConfig is the `compute` block. A command or registered model reads
// the serving fields, and a dedicated checkpoint the dedicated fields.
type deployComputeConfig struct {
	AcceleratorType *string `yaml:"accelerator_type" help:"Accelerator to serve on, as in air run: GPU_1xA10, GPU_1xH100, GPU_8xH100 or GPU_1xRTX6000. Each takes the num_accelerators listed for it: GPU_1xA10 takes 1, 4 or 8; GPU_1xH100 takes 1; GPU_8xH100 takes 8; GPU_1xRTX6000 takes 1." required:"when compute is set" group:"Serving compute"`
	NumAccelerators *int    `yaml:"num_accelerators" help:"Number of accelerators, from the pairs listed under accelerator_type." required:"when compute is set" group:"Serving compute"`
	WorkloadSize    *string `yaml:"workload_size" help:"How far the endpoint scales: Small, Medium or Large." required:"when compute is set" group:"Serving compute"`

	ComputeProfile        *string `yaml:"compute_profile" help:"Dedicated compute profile: low_latency, fast, throughput or minimal_cost, case-insensitive." required:"when checkpoint is set without command" group:"Dedicated compute"`
	ProvisionedModelUnits *int    `yaml:"provisioned_model_units" help:"Model units to provision, above 0." required:"when checkpoint is set without command" group:"Dedicated compute"`
	ReservationTerm       *string `yaml:"reservation_term" help:"Commitment term of the provisioned units, such as ONE_MONTH. Not supported yet: dedicated endpoints don't accept a commitment term, so a file that sets it is refused before anything is registered." group:"Dedicated compute"`
}

// variantField is a field that only some variants use.
type variantField struct {
	name  string
	set   bool
	users []string
	// applies says what the field is for, as the error that rejects it names it.
	applies string
}

// variantFields lists the optional fields with the variants that use them, in the
// order their errors are reported.
func (c *deployConfig) variantFields() []variantField {
	var compute deployComputeConfig
	if c.Compute != nil {
		compute = *c.Compute
	}
	servingVariants := []string{variantCommand, variantRegistered}
	registeringVariants := []string{variantCommand, variantDedicated}
	return []variantField{
		{"command", c.Command != nil, []string{variantCommand}, "a checkpoint served by command"},
		{"task", c.Task != nil, []string{variantCommand}, "a checkpoint served by command"},
		{"environment", c.Environment != nil, []string{variantCommand}, "a checkpoint served by command"},
		{"model_version", c.ModelVersion != nil, []string{variantRegistered}, "a registered model"},
		{"env_variables", c.EnvVariables != nil, servingVariants, "a command or registered model"},
		{"secrets", c.Secrets != nil, servingVariants, "a command or registered model"},
		{"idempotency_token", c.IdempotencyToken != nil, registeringVariants, "a checkpoint, which air deploy registers"},
		{"mlflow_experiment_path", c.MLflowExperimentPath != nil, registeringVariants, "a checkpoint, which air deploy registers"},
		{"compute.accelerator_type", compute.AcceleratorType != nil, servingVariants, "a command or registered model"},
		{"compute.num_accelerators", compute.NumAccelerators != nil, servingVariants, "a command or registered model"},
		{"compute.workload_size", compute.WorkloadSize != nil, servingVariants, "a command or registered model"},
		{"compute.compute_profile", compute.ComputeProfile != nil, []string{variantDedicated}, "a dedicated checkpoint"},
		{"compute.provisioned_model_units", compute.ProvisionedModelUnits != nil, []string{variantDedicated}, "a dedicated checkpoint"},
		{"compute.reservation_term", compute.ReservationTerm != nil, []string{variantDedicated}, "a dedicated checkpoint"},
	}
}

// variant is the deployment the file describes, inferred from the fields it sets: a
// checkpoint with a command is served by that command, a checkpoint without one is
// served with dedicated inference, and no checkpoint means a registered version.
func (c *deployConfig) variant() string {
	switch {
	case c.Checkpoint == nil:
		return variantRegistered
	case c.Command != nil:
		return variantCommand
	default:
		return variantDedicated
	}
}

// readAs says how the file was read and why, for errors about its fields.
func (c *deployConfig) readAs() string {
	v := c.variant()
	return fmt.Sprintf("the file is read as %s, because %s", variantDescriptions[v], variantReasons[v])
}

// registers reports whether air deploy registers a new version from the checkpoint.
func (c *deployConfig) registers() bool {
	return c.Checkpoint != nil
}

// validate runs structural validation over the whole config, returning the
// first failure. Fields are checked in declaration order to keep errors stable.
func (c *deployConfig) validate() error {
	if !registeredModelRe().MatchString(c.RegisteredModel) {
		return fmt.Errorf("registered_model_name must be a Unity Catalog model name as <catalog>.<schema>.<model>, got %q", c.RegisteredModel)
	}
	if err := c.validateSource(); err != nil {
		return err
	}
	if !endpointNameRe().MatchString(c.Endpoint) {
		return fmt.Errorf("endpoint_name must be a serving endpoint name of letters, digits, '-' and '_', got %q", c.Endpoint)
	}
	if err := c.validateCompute(); err != nil {
		return err
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

// validateSource checks that the fields the file sets fit its variant, and that the
// variant's own fields are valid.
func (c *deployConfig) validateSource() error {
	if err := c.checkVariantFields(); err != nil {
		return err
	}
	if c.Checkpoint != nil {
		if err := validateVolumeDirectory("checkpoint", *c.Checkpoint); err != nil {
			return err
		}
	}
	switch c.variant() {
	case variantCommand:
		return c.validateCommandSource()
	case variantRegistered:
		if c.ModelVersion != nil && !modelVersionRe().MatchString(*c.ModelVersion) {
			return fmt.Errorf("model_version must be a model version number such as 3, got %q", *c.ModelVersion)
		}
	}
	return nil
}

// checkVariantFields rejects a field the file's variant doesn't use, naming the
// field, the variant and why the file is read as that variant.
func (c *deployConfig) checkVariantFields() error {
	variant := c.variant()
	for _, f := range c.variantFields() {
		if f.set && !slices.Contains(f.users, variant) {
			return fmt.Errorf("%s can't be set: %s, and %s applies only to %s", f.name, c.readAs(), f.name, f.applies)
		}
	}
	return nil
}

// validateCommandSource checks the fields only a checkpoint served by command has.
func (c *deployConfig) validateCommandSource() error {
	if c.Task == nil {
		return errors.New("task is required for a checkpoint served by command: the task Model Serving routes to the command, such as llm/v1/chat")
	}
	if strings.TrimSpace(*c.Task) == "" {
		return errors.New("task cannot be empty")
	}
	if strings.TrimSpace(*c.Command) == "" {
		return errors.New("command cannot be empty")
	}
	if n := serverLength(*c.Command); n > maxServingCommandChars {
		return fmt.Errorf("command must be at most %d characters, got %d", maxServingCommandChars, n)
	}
	if c.Environment == nil {
		return errors.New("environment is required for a checkpoint served by command: the serverless environment and packages to serve it on")
	}
	return c.Environment.validate()
}

// validateCompute checks the compute the variant reads.
func (c *deployConfig) validateCompute() error {
	if c.variant() == variantDedicated {
		_, err := c.resolveDedicatedCompute()
		return err
	}
	_, _, err := c.resolveServingCompute()
	return err
}

// resolveServingCompute returns the workload type and size a command or registered
// model is served on. Without compute, the defaults apply; with it, all three fields
// are required and the accelerator and count must be a pair the server offers.
func (c *deployConfig) resolveServingCompute() (string, string, error) {
	if c.Compute == nil {
		if c.variant() == variantRegistered {
			return defaultRegisteredWorkloadType, defaultWorkloadSize, nil
		}
		return defaultCommandWorkloadType, defaultWorkloadSize, nil
	}
	compute := c.Compute
	switch {
	case compute.AcceleratorType == nil:
		return "", "", errors.New("compute.accelerator_type is required when compute is set, e.g. GPU_1xA10")
	case compute.NumAccelerators == nil:
		return "", "", errors.New("compute.num_accelerators is required when compute is set, e.g. 1")
	case compute.WorkloadSize == nil:
		return "", "", errors.New("compute.workload_size is required when compute is set: Small, Medium or Large")
	}
	if !slices.Contains(workloadSizes, *compute.WorkloadSize) {
		return "", "", fmt.Errorf("compute.workload_size must be one of %s, got %q", strings.Join(workloadSizes, ", "), *compute.WorkloadSize)
	}
	accelerator, err := parseGPUType(*compute.AcceleratorType)
	if err != nil {
		return "", "", fmt.Errorf("compute.accelerator_type: %w; the supported pairs are %s", err, servingPairs())
	}
	for _, shape := range servingShapes {
		if shape.accelerator == accelerator && shape.count == *compute.NumAccelerators {
			return shape.workloadType, *compute.WorkloadSize, nil
		}
	}
	return "", "", fmt.Errorf("compute.accelerator_type and compute.num_accelerators must be a supported pair, got %s with %d; the supported pairs are %s", *compute.AcceleratorType, *compute.NumAccelerators, servingPairs())
}

// servingPairs lists the accelerator and count pairs serving compute supports.
func servingPairs() string {
	pairs := make([]string, 0, len(servingShapes))
	for _, shape := range servingShapes {
		pairs = append(pairs, fmt.Sprintf("%s x%d", shape.accelerator, shape.count))
	}
	return strings.Join(pairs, ", ")
}

// dedicatedCompute is the dedicated compute a dedicated served entity gets.
type dedicatedCompute struct {
	ComputeProfile        string
	ProvisionedModelUnits int64
}

// resolveDedicatedCompute returns the dedicated compute of a dedicated checkpoint. The
// server has no default for it, so compute, its profile and its units are required.
func (c *deployConfig) resolveDedicatedCompute() (*dedicatedCompute, error) {
	if c.Compute == nil {
		return nil, errors.New("compute is required for a dedicated checkpoint: set compute.compute_profile and compute.provisioned_model_units")
	}
	compute := c.Compute
	if compute.ComputeProfile == nil {
		return nil, errors.New("compute.compute_profile is required for a dedicated checkpoint: one of low_latency, fast, throughput or minimal_cost")
	}
	profile := strings.ToLower(*compute.ComputeProfile)
	if !slices.Contains(computeProfiles, profile) {
		return nil, fmt.Errorf("compute.compute_profile must be one of %s, got %q", strings.Join(computeProfiles, ", "), *compute.ComputeProfile)
	}
	if compute.ProvisionedModelUnits == nil {
		return nil, errors.New("compute.provisioned_model_units is required for a dedicated checkpoint: the model units to provision, above 0")
	}
	if *compute.ProvisionedModelUnits <= 0 {
		return nil, fmt.Errorf("compute.provisioned_model_units must be above 0, got %d", *compute.ProvisionedModelUnits)
	}
	// The server refuses a commitment term for a dedicated endpoint, which it would
	// only do after the weights are copied.
	if compute.ReservationTerm != nil {
		return nil, errors.New("compute.reservation_term isn't supported yet: dedicated endpoints don't accept a commitment term; leave it out")
	}
	return &dedicatedCompute{
		ComputeProfile:        "COMPUTE_PROFILE_" + strings.ToUpper(profile),
		ProvisionedModelUnits: int64(*compute.ProvisionedModelUnits),
	}, nil
}

// validateVolumeDirectory checks a Unity Catalog volume directory the server
// copies into the version.
func validateVolumeDirectory(field, v string) error {
	if !strings.HasPrefix(v, "/Volumes/") || serverLength(v) > maxVolumePathChars {
		return fmt.Errorf("%s must be a Unity Catalog volume directory as /Volumes/<catalog>/<schema>/<volume>/<path>, of at most %d characters, got %q", field, maxVolumePathChars, v)
	}
	return nil
}

// validate checks the environment version and dependency lines.
func (e *deployEnvironmentConfig) validate() error {
	if e.Version == nil {
		return errors.New("environment.version is required: the serverless environment to serve the command on, such as 6 or databricks_ai_v6")
	}
	if _, _, err := e.runtimeVersion(); err != nil {
		return err
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

// runtimeVersion is environment.version as a version number, and whether it names
// the Databricks AI environment, as air run reads the same field.
func (e *deployEnvironmentConfig) runtimeVersion() (int, bool, error) {
	if e.Version == nil {
		return 0, false, errors.New("environment.version is required")
	}
	normalized, err := validateRuntimeVersion(e.Version.raw, "environment.version")
	if err != nil {
		return 0, false, err
	}
	ai := strings.HasPrefix(normalized, databricksAIPrefix)
	major, err := strconv.Atoi(strings.TrimPrefix(normalized, databricksAIPrefix))
	if err != nil || major < 1 {
		return 0, false, fmt.Errorf("environment.version must be a serverless environment version such as 6 or databricks_ai_v6, got %q", e.Version.raw)
	}
	return major, ai, nil
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

    databricks air deploy -h config.checkpoint

  A checkpoint served by a command of your own:
    registered_model_name: main.serving.qwen30b_sglang
    endpoint_name: qwen3-30b-sglang
    checkpoint: /Volumes/main/weights/qwen3_30b_a3b
    command: python -m sglang.launch_server --model-path qwen3_30b_a3b --attention-backend triton --sampling-backend pytorch --host 0.0.0.0 --port 8080
    task: llm/v1/chat
    environment:
      version: 6
      dependencies: [sglang==0.5.23]
    compute:
      accelerator_type: GPU_1xH100
      num_accelerators: 1
      workload_size: Small

  A checkpoint served with dedicated inference:
    registered_model_name: main.serving.gemma4
    endpoint_name: gemma4
    checkpoint: /Volumes/main/weights/gemma4_e4b
    compute:
      compute_profile: low_latency
      provisioned_model_units: 600

  A version already registered in Unity Catalog:
    registered_model_name: main.ml.churn
    endpoint_name: churn
    model_version: 3`,
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
