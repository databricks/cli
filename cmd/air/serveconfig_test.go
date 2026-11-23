package aircmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commandServeConfig is a checkpoint served by a command.
const commandServeConfig = `registered_model_name: main.models.qwen
checkpoint: /Volumes/main/models/weights/qwen3_8b
command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080
task: llm/v1/chat
environment:
  version: 6
  dependencies: [einops==0.8.0]
endpoint_name: qwen-chat
compute:
  accelerator_type: GPU_1xA10
  num_accelerators: 1
  workload_size: Small
secrets:
  HF_TOKEN: llm/hf_token
env_variables:
  LOG_LEVEL: info
`

// dedicatedServeConfig is a checkpoint served with dedicated inference.
const dedicatedServeConfig = `registered_model_name: main.models.qwen
checkpoint: /Volumes/main/models/weights/qwen3_8b
endpoint_name: qwen-chat
compute:
  compute_profile: low_latency
  provisioned_model_units: 600
`

// registeredServeConfig serves a version that is already registered.
const registeredServeConfig = `registered_model_name: main.models.qwen
model_version: 7
endpoint_name: qwen-chat
`

func loadServeConfigText(t *testing.T, text string, overrides ...string) (*serveConfig, error) {
	t.Helper()
	return loadServeConfig(t.Context(), writeConfigFile(t, "serve.yaml", text), overrides)
}

func TestLoadServeConfigVariantIsInferredFromTheFields(t *testing.T) {
	for _, tt := range []struct {
		name    string
		text    string
		variant string
	}{
		{"checkpoint and command", commandServeConfig, variantCommand},
		{"checkpoint without command", dedicatedServeConfig, variantDedicated},
		{"no checkpoint", registeredServeConfig, variantRegistered},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadServeConfigText(t, tt.text)
			require.NoError(t, err)
			assert.Equal(t, tt.variant, cfg.variant())
			assert.Equal(t, tt.variant != variantRegistered, cfg.registers())
		})
	}
}

func TestLoadServeConfigCommand(t *testing.T) {
	cfg, err := loadServeConfigText(t, commandServeConfig)
	require.NoError(t, err)

	req := cfg.registrationRequest("key-1")
	assert.Equal(t, registerModelPackageRequest{
		RegisteredModel: "main.models.qwen",
		IdempotencyKey:  "key-1",
		Checkpoint: &modelPackageCheckpoint{
			VolumePath:     "/Volumes/main/models/weights/qwen3_8b",
			ServingCommand: "python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080",
			Task:           "llm/v1/chat",
			Environment:    modelPackageEnvironment{Runtime: "client.6-gpu", Dependencies: []string{"einops==0.8.0"}},
		},
	}, req)

	entity, err := cfg.servedEntity("3")
	require.NoError(t, err)
	assert.Equal(t, servedEntity{
		EntityName:      "main.models.qwen",
		EntityVersion:   "3",
		WorkloadType:    "GPU_MEDIUM",
		WorkloadSize:    "Small",
		EnvironmentVars: map[string]string{"LOG_LEVEL": "info", "HF_TOKEN": "{{secrets/llm/hf_token}}"},
	}, entity)
}

func TestLoadServeConfigCommandOnDatabricksAI(t *testing.T) {
	cfg, err := loadServeConfigText(t, strings.Replace(commandServeConfig, "  version: 6\n", "  version: databricks_ai_v6\n", 1))
	require.NoError(t, err)
	req := cfg.registrationRequest("key-1")
	assert.Equal(t, modelPackageEnvironment{Runtime: "client.6-gpu", Dependencies: []string{"einops==0.8.0"}, BaseEnvironment: "databricks-ai"}, req.Checkpoint.Environment)
}

func TestLoadServeConfigDedicated(t *testing.T) {
	cfg, err := loadServeConfigText(t, dedicatedServeConfig+"mlflow_experiment_path: /Shared/qwen\n")
	require.NoError(t, err)

	req := cfg.registrationRequest("key-1")
	assert.Equal(t, registerModelPackageRequest{
		RegisteredModel:     "main.models.qwen",
		IdempotencyKey:      "key-1",
		DedicatedCheckpoint: &dedicatedCheckpoint{VolumePath: "/Volumes/main/models/weights/qwen3_8b"},
		ExperimentPath:      "/Shared/qwen",
	}, req)

	dedicated, err := cfg.resolveDedicatedCompute()
	require.NoError(t, err)
	assert.Equal(t, &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_LOW_LATENCY", ProvisionedModelUnits: 600}, dedicated)
}

func TestLoadServeConfigRegisteredDefaultsToCPU(t *testing.T) {
	cfg, err := loadServeConfigText(t, registeredServeConfig)
	require.NoError(t, err)
	assert.False(t, cfg.registers())

	entity, err := cfg.servedEntity("7")
	require.NoError(t, err)
	assert.Equal(t, "CPU", entity.WorkloadType)
	assert.Equal(t, "Small", entity.WorkloadSize)
	assert.Nil(t, entity.EnvironmentVars)
}

func TestServeConfigCrossVariantFields(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			"task without a checkpoint",
			registeredServeConfig + "task: llm/v1/chat\n",
			"task can't be set: the file is read as a registered model, because it sets no checkpoint, and task applies only to a checkpoint served by command",
		},
		{
			"command without a checkpoint",
			registeredServeConfig + "command: python serve.py\n",
			"command can't be set: the file is read as a registered model, because it sets no checkpoint, and command applies only to a checkpoint served by command",
		},
		{
			"environment on a dedicated checkpoint",
			dedicatedServeConfig + "environment: {version: 6}\n",
			"environment can't be set: the file is read as a dedicated checkpoint, because it sets checkpoint but not command, and environment applies only to a checkpoint served by command",
		},
		{
			"model_version with a checkpoint",
			commandServeConfig + "model_version: 3\n",
			"model_version can't be set: the file is read as a checkpoint served by command, because it sets checkpoint and command, and model_version applies only to a registered model",
		},
		{
			"model_version on a dedicated checkpoint",
			dedicatedServeConfig + "model_version: 3\n",
			"model_version can't be set: the file is read as a dedicated checkpoint",
		},
		{
			"secrets on a dedicated checkpoint",
			dedicatedServeConfig + "secrets: {HF_TOKEN: llm/hf_token}\n",
			"secrets can't be set: the file is read as a dedicated checkpoint, because it sets checkpoint but not command, and secrets applies only to a command or registered model",
		},
		{
			"env_variables on a dedicated checkpoint",
			dedicatedServeConfig + "env_variables: {LOG_LEVEL: info}\n",
			"env_variables can't be set: the file is read as a dedicated checkpoint",
		},
		{
			"idempotency_token without a checkpoint",
			registeredServeConfig + "idempotency_token: key-1\n",
			"idempotency_token can't be set: the file is read as a registered model, because it sets no checkpoint, and idempotency_token applies only to a checkpoint, which air serve registers",
		},
		{
			"mlflow_experiment_path without a checkpoint",
			registeredServeConfig + "mlflow_experiment_path: /Shared/qwen\n",
			"mlflow_experiment_path can't be set: the file is read as a registered model",
		},
		{
			"serving compute on a dedicated checkpoint",
			strings.Replace(dedicatedServeConfig, "  compute_profile: low_latency\n", "  compute_profile: low_latency\n  accelerator_type: GPU_1xA10\n", 1),
			"compute.accelerator_type can't be set: the file is read as a dedicated checkpoint, because it sets checkpoint but not command, and compute.accelerator_type applies only to a command or registered model",
		},
		{
			"dedicated compute on a command",
			strings.Replace(commandServeConfig, "  workload_size: Small\n", "  workload_size: Small\n  compute_profile: low_latency\n", 1),
			"compute.compute_profile can't be set: the file is read as a checkpoint served by command, because it sets checkpoint and command, and compute.compute_profile applies only to a dedicated checkpoint",
		},
		{
			"reservation term on a command",
			strings.Replace(commandServeConfig, "  workload_size: Small\n", "  workload_size: Small\n  reservation_term: ONE_MONTH\n", 1),
			"compute.reservation_term can't be set: the file is read as a checkpoint served by command",
		},
		{
			"dedicated compute on a registered model",
			registeredServeConfig + "compute: {workload_size: Small, provisioned_model_units: 10}\n",
			"compute.provisioned_model_units can't be set: the file is read as a registered model",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadServeConfigText(t, tt.config)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestServeConfigRequiredFieldsPerVariant(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{"command without a task", dropLine("task:")(commandServeConfig), "task is required for a checkpoint served by command"},
		{"command without an environment", dropBlock("environment:")(commandServeConfig), "environment is required for a checkpoint served by command"},
		{"dedicated without compute", dropBlock("compute:")(dedicatedServeConfig), "compute is required for a dedicated checkpoint"},
		{"dedicated without a profile", dropLine("  compute_profile:")(dedicatedServeConfig), "compute.compute_profile is required for a dedicated checkpoint"},
		{"dedicated without units", dropLine("  provisioned_model_units:")(dedicatedServeConfig), "compute.provisioned_model_units is required for a dedicated checkpoint"},
		{"command compute without num_accelerators", dropLine("  num_accelerators:")(commandServeConfig), "compute.num_accelerators is required when compute is set"},
		{"registered compute without accelerator", registeredServeConfig + "compute: {workload_size: Small}\n", "compute.accelerator_type is required when compute is set"},
		{"no endpoint", dropLine("endpoint_name:")(commandServeConfig), `endpoint_name must be a serving endpoint name of letters, digits, '-' and '_', got ""`},
		{"no registered model", dropLine("registered_model_name:")(commandServeConfig), `registered_model_name must be a Unity Catalog model name as <catalog>.<schema>.<model>, got ""`},
		{"registered with a bad version", replaceFirst("model_version: 7", "model_version: v7")(registeredServeConfig), "model_version must be a model version number such as 3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadServeConfigText(t, tt.config)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// servingCompute is a config's serving compute for one accelerator, count and size.
func servingCompute(accelerator string, count int, size string) *serveComputeConfig {
	return &serveComputeConfig{AcceleratorType: &accelerator, NumAccelerators: &count, WorkloadSize: &size}
}

func TestServeServingComputeTable(t *testing.T) {
	cfg := serveConfig{Checkpoint: new("/Volumes/a/b/c/w"), Command: new("serve")}
	for _, tt := range []struct {
		accelerator string
		count       int
		workload    string
	}{
		{"GPU_1xA10", 1, "GPU_MEDIUM"},
		{"GPU_1xA10", 4, "MULTIGPU_MEDIUM"},
		{"GPU_1xA10", 8, "GPU_MEDIUM_8"},
		{"GPU_1xH100", 1, "GPU_XLARGE"},
		{"GPU_8xH100", 8, "GPU_XLARGE_8"},
		{"GPU_1xRTX6000", 1, "GPU_LARGE_RTX"},
	} {
		cfg.Compute = servingCompute(tt.accelerator, tt.count, "Medium")
		workload, size, err := cfg.resolveServingCompute()
		require.NoError(t, err, tt.accelerator)
		assert.Equal(t, tt.workload, workload, "%s x%d", tt.accelerator, tt.count)
		assert.Equal(t, "Medium", size)
	}
}

func TestServeServingComputeRejectsUnsupportedPairs(t *testing.T) {
	const pairs = "GPU_1xA10 x1, GPU_1xA10 x4, GPU_1xA10 x8, GPU_1xH100 x1, GPU_8xH100 x8, GPU_1xRTX6000 x1"
	cfg := serveConfig{Checkpoint: new("/Volumes/a/b/c/w"), Command: new("serve")}
	for _, tt := range []struct {
		accelerator string
		count       int
	}{
		{"GPU_1xA10", 2},
		{"GPU_1xH100", 8},
		{"GPU_8xH100", 1},
		{"GPU_8xB300", 8},
		{"GPU_1xRTX6000", 2},
	} {
		cfg.Compute = servingCompute(tt.accelerator, tt.count, "Small")
		_, _, err := cfg.resolveServingCompute()
		assert.ErrorContains(t, err, "must be a supported pair, got "+tt.accelerator+" with ", tt.accelerator)
		assert.ErrorContains(t, err, "the supported pairs are "+pairs, tt.accelerator)
	}

	cfg.Compute = servingCompute("GPU_9xA10", 1, "Small")
	_, _, err := cfg.resolveServingCompute()
	assert.ErrorContains(t, err, `compute.accelerator_type: "GPU_9xA10" is not a recognized accelerator type; the supported pairs are `+pairs)
}

func TestServeServingComputeDefaults(t *testing.T) {
	command := serveConfig{Checkpoint: new("/Volumes/a/b/c/w"), Command: new("serve")}
	workload, size, err := command.resolveServingCompute()
	require.NoError(t, err)
	assert.Equal(t, "GPU_MEDIUM", workload)
	assert.Equal(t, "Small", size)

	registered := serveConfig{}
	workload, size, err = registered.resolveServingCompute()
	require.NoError(t, err)
	assert.Equal(t, "CPU", workload)
	assert.Equal(t, "Small", size)
}

func TestServeServingComputeRejectsBadSize(t *testing.T) {
	cfg := serveConfig{Checkpoint: new("/Volumes/a/b/c/w"), Command: new("serve"), Compute: servingCompute("GPU_1xA10", 1, "Huge")}
	_, _, err := cfg.resolveServingCompute()
	assert.ErrorContains(t, err, `compute.workload_size must be one of Small, Medium, Large, got "Huge"`)
}

func TestServeDedicatedComputeMapping(t *testing.T) {
	cfg := serveConfig{Checkpoint: new("/Volumes/a/b/c/w")}
	for profile, want := range map[string]string{
		"low_latency":  "COMPUTE_PROFILE_LOW_LATENCY",
		"FAST":         "COMPUTE_PROFILE_FAST",
		"Throughput":   "COMPUTE_PROFILE_THROUGHPUT",
		"minimal_cost": "COMPUTE_PROFILE_MINIMAL_COST",
	} {
		cfg.Compute = &serveComputeConfig{ComputeProfile: new(profile), ProvisionedModelUnits: new(600)}
		dedicated, err := cfg.resolveDedicatedCompute()
		require.NoError(t, err, profile)
		assert.Equal(t, want, dedicated.ComputeProfile, profile)
	}
}

func TestServeDedicatedComputeRejects(t *testing.T) {
	cfg := serveConfig{Checkpoint: new("/Volumes/a/b/c/w")}
	for _, tt := range []struct {
		name    string
		compute serveComputeConfig
		wantErr string
	}{
		{"unknown profile", serveComputeConfig{ComputeProfile: new("fastest"), ProvisionedModelUnits: new(600)}, `compute.compute_profile must be one of low_latency, fast, throughput, minimal_cost, got "fastest"`},
		{"zero units", serveComputeConfig{ComputeProfile: new("fast"), ProvisionedModelUnits: new(0)}, "compute.provisioned_model_units must be above 0, got 0"},
		{"negative units", serveComputeConfig{ComputeProfile: new("fast"), ProvisionedModelUnits: new(-1)}, "compute.provisioned_model_units must be above 0, got -1"},
		{"a reservation term", serveComputeConfig{ComputeProfile: new("fast"), ProvisionedModelUnits: new(600), ReservationTerm: new("ONE_MONTH")}, "compute.reservation_term isn't supported yet"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Compute = &tt.compute
			_, err := cfg.resolveDedicatedCompute()
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestServeConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(string) string
		wantErr string
	}{
		{"bad model name", replaceFirst("main.models.qwen", "qwen"), "registered_model_name must be a Unity Catalog model name"},
		{"backticked model name", replaceFirst("main.models.qwen", "main.models.`qwen`"), "registered_model_name must be a Unity Catalog model name"},
		{"command without task", replaceFirst("task: llm/v1/chat\n", ""), "task is required for a checkpoint served by command"},
		{"empty task", replaceFirst("task: llm/v1/chat", `task: " "`), "task cannot be empty"},
		{"path outside volumes", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/dbfs/weights"), "checkpoint must be a Unity Catalog volume directory"},
		{"path too long", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("a", 1100)), "checkpoint must be a Unity Catalog volume directory"},
		{"path too long in characters", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("é", 1020)), "checkpoint must be a Unity Catalog volume directory"},
		{"empty command", replaceFirst("command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080", `command: " "`), "command cannot be empty"},
		{"command too long", replaceFirst("command: python", "command: python "+strings.Repeat("x", maxServingCommandChars)), "command must be at most 32768 characters"},
		{"AI environment before version 5", replaceFirst("  version: 6\n", "  version: databricks_ai_v4\n"), "databricks_ai_v in environment.version requires AI Runtime version 5 or higher"},
		{"bad environment version", replaceFirst("  version: 6\n", "  version: six\n"), "unsupported client image version"},
		{"zero environment version", replaceFirst("  version: 6\n", "  version: 0\n"), "environment.version must be a serverless environment version such as 6 or databricks_ai_v6"},
		{"no environment version", replaceFirst("  version: 6\n", ""), "environment.version is required"},
		{"empty dependency", replaceFirst("[einops==0.8.0]", `[""]`), "environment.dependencies[0] cannot be empty"},
		{"no environment", dropBlock("environment:"), "environment is required for a checkpoint served by command"},
		{"no endpoint", dropLine("endpoint_name:"), `endpoint_name must be a serving endpoint name of letters, digits, '-' and '_', got ""`},
		{"endpoint with a path", replaceFirst("endpoint_name: qwen-chat", "endpoint_name: x/../../jobs"), "endpoint_name must be a serving endpoint name"},
		{"no workload size", dropLine("  workload_size:"), "compute.workload_size is required when compute is set"},
		{"lowercase workload size", replaceFirst("workload_size: Small", "workload_size: medium"), "compute.workload_size must be one of Small, Medium, Large"},
		{"unsupported accelerator pair", replaceFirst("num_accelerators: 1", "num_accelerators: 2"), "must be a supported pair, got GPU_1xA10 with 2"},
		{"secret without key", replaceFirst("llm/hf_token", "llm"), "expected format 'scope/key'"},
		{"env var and secret", replaceFirst("LOG_LEVEL: info", "HF_TOKEN: x"), `"HF_TOKEN" is set in both env_variables and secrets`},
		{"bad idempotency token", appendText("idempotency_token: a b\n"), "idempotency_token must be 1 to 128 letters"},
		{"long idempotency token", appendText("idempotency_token: " + strings.Repeat("k", 129) + "\n"), "idempotency_token must be 1 to 128 letters"},
		{"relative experiment", appendText("mlflow_experiment_path: Users/me/x\n"), "mlflow_experiment_path must be an absolute workspace path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadServeConfigText(t, tt.edit(commandServeConfig))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestServeConfigDedicatedValidation(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(string) string
		wantErr string
	}{
		{"bad path", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/dbfs/weights"), "checkpoint must be a Unity Catalog volume directory"},
		{"bad endpoint", replaceFirst("endpoint_name: qwen-chat", "endpoint_name: a/b"), "endpoint_name must be a serving endpoint name"},
		{"bad token", appendText("idempotency_token: a b\n"), "idempotency_token must be 1 to 128 letters"},
		{"unknown profile", replaceFirst("low_latency", "fastest"), "compute.compute_profile must be one of"},
		{"zero units", replaceFirst("provisioned_model_units: 600", "provisioned_model_units: 0"), "compute.provisioned_model_units must be above 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadServeConfigText(t, tt.edit(dedicatedServeConfig))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestServeConfigCountsCharactersNotBytes(t *testing.T) {
	// 1,000 two-byte characters fit the 1,024-character cap on a volume path.
	_, err := loadServeConfigText(t, replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("é", 1000))(commandServeConfig))
	assert.NoError(t, err)
}

func TestLoadServeConfigOverride(t *testing.T) {
	cfg, err := loadServeConfigText(t, commandServeConfig, "compute.num_accelerators=4", "environment.version=5")
	require.NoError(t, err)
	entity, err := cfg.servedEntity("3")
	require.NoError(t, err)
	assert.Equal(t, "MULTIGPU_MEDIUM", entity.WorkloadType)
	assert.Equal(t, "client.5-gpu", cfg.registrationRequest("k").Checkpoint.Environment.Runtime)
}

func TestLoadServeConfigOverrideIsValidatedAgainstTheVariant(t *testing.T) {
	_, err := loadServeConfigText(t, commandServeConfig, "compute.compute_profile=low_latency")
	assert.ErrorContains(t, err, "compute.compute_profile can't be set: the file is read as a checkpoint served by command")
}

func TestLoadServeConfigRejectsUnknownOverride(t *testing.T) {
	_, err := loadServeConfigText(t, commandServeConfig, "compute.workers=2")
	assert.ErrorContains(t, err, `"workers" is not a known field`)
}

func TestLoadServeConfigRejectsUnknownField(t *testing.T) {
	_, err := loadServeConfigText(t, commandServeConfig+"experiment_name: x\n")
	assert.ErrorContains(t, err, "field experiment_name not found")
}

func TestLoadServeConfigRejectsOldShape(t *testing.T) {
	_, err := loadServeConfigText(t, "registered_model: main.models.qwen\nendpoint: qwen-chat\n")
	assert.ErrorContains(t, err, "field registered_model not found")
}

func TestLoadServeConfigRejectsEmptyFile(t *testing.T) {
	_, err := loadServeConfigText(t, "")
	assert.ErrorContains(t, err, "is empty")
}

func TestServeSchemaHelp(t *testing.T) {
	root, err := resolveSchemaField(serveSchema(), "config")
	require.NoError(t, err)
	var out strings.Builder
	renderConfigField(&out, serveCommandPath, root, nil)
	help := out.String()
	assert.Contains(t, help, "    databricks air serve -h config.checkpoint\n")
	assert.Contains(t, help, "  Model:\n")
	assert.Contains(t, help, "  Serving:\n")
	assert.Contains(t, help, "  Registration:\n")
	assert.Contains(t, help, `For details on a field, run "databricks air serve -h config.<field>".`)

	field, err := resolveSchemaField(serveSchema(), "compute.compute_profile")
	require.NoError(t, err)
	assert.Equal(t, "when checkpoint is set without command", field.required)

	_, err = resolveSchemaField(serveSchema(), "checkpont")
	assert.ErrorContains(t, err, `did you mean "checkpoint"?`)

	_, err = resolveSchemaField(serveSchema(), "secrets.HF_TOKEN")
	assert.ErrorContains(t, err, "holds user-defined keys")
}

// Every field a variant names must be a schema path, so the variant checks can't
// drift from the YAML.
func TestServeVariantFieldsAreSchemaPaths(t *testing.T) {
	var cfg serveConfig
	for _, f := range cfg.variantFields() {
		_, err := resolveSchemaField(serveSchema(), f.name)
		assert.NoError(t, err, f.name)
	}
}

func replaceFirst(old, with string) func(string) string {
	return func(s string) string {
		if !strings.Contains(s, old) {
			panic("fixture does not contain " + old)
		}
		return strings.Replace(s, old, with, 1)
	}
}

func appendText(extra string) func(string) string {
	return func(s string) string { return s + extra }
}

// dropLine removes the first line starting with prefix.
func dropLine(prefix string) func(string) string {
	return func(s string) string {
		lines := strings.SplitAfter(s, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, prefix) {
				return strings.Join(append(lines[:i:i], lines[i+1:]...), "")
			}
		}
		panic("fixture has no line starting with " + prefix)
	}
}

// dropBlock removes the top-level key starting with prefix and its indented lines.
func dropBlock(prefix string) func(string) string {
	return func(s string) string {
		var kept []string
		dropping := false
		for _, line := range strings.SplitAfter(s, "\n") {
			switch {
			case strings.HasPrefix(line, prefix):
				dropping = true
			case dropping && strings.HasPrefix(line, "  "):
			default:
				dropping = false
				kept = append(kept, line)
			}
		}
		return strings.Join(kept, "")
	}
}
