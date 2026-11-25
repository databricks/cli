package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalServeConfig serves the newest READY version on the compute it was
// registered for.
const minimalServeConfig = `uc_model: main.models.qwen
endpoint_name: qwen-chat
`

// gpuServeConfig serves version 3 on replicas.
const gpuServeConfig = minimalServeConfig + `model_version: 3
compute:
  gpu:
    accelerator_type: GPU_1xA10
    min_replicas: 1
    max_replicas: 2
`

// dedicatedServeConfig serves a dedicated checkpoint on dedicated compute.
const dedicatedServeConfig = minimalServeConfig + `compute:
  mode: throughput_optimized
  model_units:
    min_units: 900
    max_units: 900
`

// Versions as their MLmodel describes them.
var (
	commandPackage   = versionPackage{command: true, task: "llm/v1/chat"}
	dedicatedPackage = versionPackage{dedicated: true}
	otherPackage     = versionPackage{}
)

func loadServeConfigText(t *testing.T, text string, overrides ...string) (*serveConfig, error) {
	t.Helper()
	return loadServeConfig(t.Context(), serveOptions{file: writeConfigFile(t, "serve.yaml", text), overrides: overrides})
}

func TestServeConfigDefaultsFollowThePackage(t *testing.T) {
	cfg, err := loadServeConfigText(t, minimalServeConfig)
	require.NoError(t, err)
	tests := []struct {
		name string
		pkg  versionPackage
		want servedEntity
	}{
		{"a command", commandPackage, servedEntity{EntityName: "main.models.qwen", EntityVersion: "3", Serving: servingCompute{WorkloadType: "GPU_MEDIUM", MinConcurrency: 4, MaxConcurrency: 4}}},
		{"a dedicated checkpoint", dedicatedPackage, servedEntity{EntityName: "main.models.qwen", EntityVersion: "3", Dedicated: &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_LOW_LATENCY", ProvisionedModelUnits: 600}}},
		{"any other version", otherPackage, servedEntity{EntityName: "main.models.qwen", EntityVersion: "3", Serving: servingCompute{WorkloadType: "CPU", MinConcurrency: 4, MaxConcurrency: 4}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, err := cfg.servedEntity("3", tt.pkg)
			require.NoError(t, err)
			assert.Equal(t, tt.want, entity)
		})
	}
}

func TestServeConfigGPU(t *testing.T) {
	cfg, err := loadServeConfigText(t, gpuServeConfig)
	require.NoError(t, err)
	assert.Equal(t, "3", *cfg.ModelVersion)
	entity, err := cfg.servedEntity("3", commandPackage)
	require.NoError(t, err)
	assert.Equal(t, servingCompute{WorkloadType: "GPU_MEDIUM", MinConcurrency: 4, MaxConcurrency: 8}, entity.Serving)
	assert.Nil(t, entity.Dedicated)
}

func TestServeConfigDedicated(t *testing.T) {
	cfg, err := loadServeConfigText(t, dedicatedServeConfig)
	require.NoError(t, err)
	entity, err := cfg.servedEntity("3", dedicatedPackage)
	require.NoError(t, err)
	assert.Equal(t, &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_THROUGHPUT", ProvisionedModelUnits: 900}, entity.Dedicated)
}

func TestServeConfigPartialBlocksTakeDefaults(t *testing.T) {
	tests := []struct {
		name string
		text string
		pkg  versionPackage
		want servedEntity
	}{
		{"only a mode", minimalServeConfig + "compute: {mode: Throughput_Optimized}\n", dedicatedPackage, servedEntity{Dedicated: &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_THROUGHPUT", ProvisionedModelUnits: 600}}},
		{"only min_units", minimalServeConfig + "compute: {model_units: {min_units: 50}}\n", dedicatedPackage, servedEntity{Dedicated: &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_LOW_LATENCY", ProvisionedModelUnits: 50}}},
		{"only max_units", minimalServeConfig + "compute: {model_units: {max_units: 100}}\n", dedicatedPackage, servedEntity{Dedicated: &dedicatedCompute{ComputeProfile: "COMPUTE_PROFILE_LOW_LATENCY", ProvisionedModelUnits: 100}}},
		{"only an accelerator", minimalServeConfig + "compute: {gpu: {accelerator_type: GPU_1xH100}}\n", commandPackage, servedEntity{Serving: servingCompute{WorkloadType: "GPU_XLARGE", MinConcurrency: 4, MaxConcurrency: 4}}},
		{"only min_replicas", minimalServeConfig + "compute: {gpu: {min_replicas: 2}}\n", commandPackage, servedEntity{Serving: servingCompute{WorkloadType: "GPU_MEDIUM", MinConcurrency: 8, MaxConcurrency: 8}}},
		{"scale to zero", minimalServeConfig + "compute: {gpu: {min_replicas: 0}}\n", otherPackage, servedEntity{Serving: servingCompute{WorkloadType: "CPU", MinConcurrency: 0, MaxConcurrency: 4, ScaleToZero: true}}},
		{"only max_replicas", minimalServeConfig + "compute: {gpu: {max_replicas: 5}}\n", otherPackage, servedEntity{Serving: servingCompute{WorkloadType: "CPU", MinConcurrency: 4, MaxConcurrency: 20}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadServeConfigText(t, tt.text)
			require.NoError(t, err)
			entity, err := cfg.servedEntity("3", tt.pkg)
			require.NoError(t, err)
			tt.want.EntityName, tt.want.EntityVersion = "main.models.qwen", "3"
			assert.Equal(t, tt.want, entity)
		})
	}
}

func TestServeConfigAcceleratorTable(t *testing.T) {
	for accelerator, workload := range map[string]string{
		"GPU_1xA10":     "GPU_MEDIUM",
		"GPU_4xA10":     "MULTIGPU_MEDIUM",
		"GPU_8xA10":     "GPU_MEDIUM_8",
		"GPU_1xH100":    "GPU_XLARGE",
		"GPU_8xH100":    "GPU_XLARGE_8",
		"GPU_1xRTX6000": "GPU_LARGE_RTX",
		"CPU":           "CPU",
	} {
		cfg, err := loadServeConfigText(t, minimalServeConfig+"compute: {gpu: {accelerator_type: "+accelerator+"}}\n")
		require.NoError(t, err, accelerator)
		entity, err := cfg.servedEntity("3", otherPackage)
		require.NoError(t, err, accelerator)
		assert.Equal(t, workload, entity.Serving.WorkloadType, accelerator)
	}
}

func TestServeConfigRefusesComputeOfTheOtherKind(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		pkg     versionPackage
		wantErr string
	}{
		{"replicas for a dedicated checkpoint", gpuServeConfig, dedicatedPackage, "compute.gpu can't be set: version 3 of main.models.qwen is a checkpoint registered without engine, which is served with dedicated inference; set compute.model_units, or leave compute out"},
		{"model units for a command", dedicatedServeConfig, commandPackage, "compute.model_units can't be set: it's for a checkpoint registered without engine, and version 3 of main.models.qwen isn't one; set compute.gpu, or leave compute out"},
		{"model units for another version", dedicatedServeConfig, otherPackage, "compute.model_units can't be set"},
		{"a mode for a command", minimalServeConfig + "compute: {mode: throughput_optimized}\n", commandPackage, "compute.mode can't be set: version 3 of main.models.qwen is served on replicas, which have no mode yet; leave it out"},
		{"a mode for another version", minimalServeConfig + "compute: {mode: latency_sensitive}\n", otherPackage, "compute.mode can't be set"},
		{"CPU for a command", minimalServeConfig + "compute: {gpu: {accelerator_type: CPU}}\n", commandPackage, "compute.gpu.accelerator_type can't be CPU: version 3 of main.models.qwen is served by its command, whose environment is registered for a GPU"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadServeConfigText(t, tt.text)
			require.NoError(t, err)
			_, err = cfg.servedEntity("3", tt.pkg)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestServeConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr string
	}{
		{"no model name", "endpoint_name: qwen-chat\n", `uc_model must be a Unity Catalog model name as <catalog>.<schema>.<model>, got ""`},
		{"bad model name", "uc_model: qwen\nendpoint_name: qwen-chat\n", "uc_model must be a Unity Catalog model name"},
		{"no endpoint", "uc_model: main.models.qwen\n", `endpoint_name must be a serving endpoint name of letters, digits, '-' and '_', got ""`},
		{"endpoint with a path", "uc_model: main.models.qwen\nendpoint_name: x/../../jobs\n", "endpoint_name must be a serving endpoint name"},
		{"a version that isn't a number", minimalServeConfig + "model_version: v3\n", `model_version must be a model version number such as 3, got "v3"`},
		{"version zero", minimalServeConfig + "model_version: 0\n", "model_version must be a model version number"},
		{"an unknown mode", minimalServeConfig + "compute: {mode: realtime}\n", `compute.mode must be one of latency_sensitive, throughput_optimized, got "realtime"`},
		{"both kinds of compute", minimalServeConfig + "compute: {gpu: {min_replicas: 1}, model_units: {min_units: 1}}\n", "compute can set gpu or model_units, not both"},
		{"a mode with replicas", minimalServeConfig + "compute: {mode: latency_sensitive, gpu: {min_replicas: 1}}\n", "compute.mode can't be set with compute.gpu: it selects the profile of dedicated inference, and replicas have no mode yet"},
		{"a minimum whose concurrency overflows", minimalServeConfig + "compute: {gpu: {min_replicas: 4611686018427387904}}\n", "compute.gpu.min_replicas must be at most 536870911, got 4611686018427387904"},
		{"an unknown accelerator", minimalServeConfig + "compute: {gpu: {accelerator_type: GPU_2xA10}}\n", `compute.gpu.accelerator_type must be one of GPU_1xA10, GPU_4xA10, GPU_8xA10, GPU_1xH100, GPU_8xH100, GPU_1xRTX6000, CPU, got "GPU_2xA10"`},
		{"a lowercase accelerator", minimalServeConfig + "compute: {gpu: {accelerator_type: gpu_1xa10}}\n", "compute.gpu.accelerator_type must be one of"},
		{"a negative minimum", minimalServeConfig + "compute: {gpu: {min_replicas: -1}}\n", "compute.gpu.min_replicas must be 0 or more, got -1"},
		{"a zero maximum", minimalServeConfig + "compute: {gpu: {min_replicas: 0, max_replicas: 0}}\n", "compute.gpu.max_replicas must be at least 1, got 0"},
		{"a minimum above the maximum", minimalServeConfig + "compute: {gpu: {min_replicas: 3, max_replicas: 2}}\n", "compute.gpu.min_replicas must be at most max_replicas, got 3 and 2"},
		{"a maximum whose concurrency overflows", minimalServeConfig + "compute: {gpu: {max_replicas: 4611686018427387904}}\n", "compute.gpu.max_replicas must be at most 536870911, got 4611686018427387904"},
		{"a range on an H100", minimalServeConfig + "compute: {gpu: {accelerator_type: GPU_1xH100, min_replicas: 1, max_replicas: 2}}\n", "compute.gpu.min_replicas must equal max_replicas on GPU_1xH100, because endpoints there don't scale; got 1 and 2"},
		{"scale to zero on an H100", minimalServeConfig + "compute: {gpu: {accelerator_type: GPU_8xH100, min_replicas: 0}}\n", "compute.gpu.min_replicas must equal max_replicas on GPU_8xH100"},
		{"half a replica", minimalServeConfig + "compute: {gpu: {min_replicas: 0.5}}\n", `"0.5" is not a whole number`},
		{"a unit range", minimalServeConfig + "compute: {model_units: {min_units: 600, max_units: 900}}\n", "compute.model_units.min_units must equal max_units, because dedicated endpoints don't scale yet; got 600 and 900"},
		{"zero units", minimalServeConfig + "compute: {model_units: {min_units: 0}}\n", "compute.model_units must be above 0, got 0"},
		{"a reservation term", minimalServeConfig + "compute: {model_units: {reservation_term: ONE_MONTH}}\n", "compute.model_units.reservation_term isn't supported yet"},
		{"fractional units", minimalServeConfig + "compute: {model_units: {min_units: 600.7}}\n", `"600.7" is not a whole number`},
		{"a register field", minimalServeConfig + "model_source: /Volumes/a/b/c/w\n", `line 3: unknown field "model_source"; run "databricks air serve -h config" for the list of fields`},
		{"environment variables", minimalServeConfig + "env_variables: {LOG_LEVEL: info}\n", `line 3: unknown field "env_variables"; run "databricks air serve -h config" for the list of fields`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadServeConfigText(t, tt.text)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestLoadServeConfigFromFlags(t *testing.T) {
	cfg, err := loadServeConfig(t.Context(), serveOptions{ucModel: "main.models.qwen", endpoint: "qwen-chat", modelVersion: "4"})
	require.NoError(t, err)
	assert.Equal(t, "main.models.qwen", cfg.UCModel)
	assert.Equal(t, "qwen-chat", cfg.Endpoint)
	assert.Equal(t, "4", *cfg.ModelVersion)
	assert.Nil(t, cfg.Compute)

	cfg, err = loadServeConfig(t.Context(), serveOptions{file: writeConfigFile(t, "serve.yaml", gpuServeConfig), endpoint: "other", modelVersion: "5", flagsSet: map[string]bool{endpointNameFlag: true, modelVersionFlag: true}})
	require.NoError(t, err)
	assert.Equal(t, "other", cfg.Endpoint)
	assert.Equal(t, "5", *cfg.ModelVersion)
	assert.NotNil(t, cfg.Compute.GPU)
}

func TestLoadServeConfigNeedsAFileOrBothFlags(t *testing.T) {
	for _, opts := range []serveOptions{{}, {ucModel: "a.b.c"}, {endpoint: "ep"}} {
		_, err := loadServeConfig(t.Context(), opts)
		assert.ErrorContains(t, err, "set --file, or both --uc-model and --endpoint-name")
	}
	_, err := loadServeConfig(t.Context(), serveOptions{ucModel: "a.b.c", endpoint: "ep", overrides: []string{"compute.mode=x"}})
	assert.ErrorContains(t, err, "--override applies to a file")
}

func TestLoadServeConfigChecksFlagsByName(t *testing.T) {
	tests := []struct {
		name    string
		opts    serveOptions
		wantErr string
	}{
		{"an empty version", serveOptions{ucModel: "a.b.c", endpoint: "ep", flagsSet: map[string]bool{modelVersionFlag: true}}, "--model-version cannot be empty"},
		{"an empty endpoint over a file", serveOptions{file: "unused.yaml", flagsSet: map[string]bool{endpointNameFlag: true}}, "--endpoint-name cannot be empty"},
		{"a bad version", serveOptions{ucModel: "a.b.c", endpoint: "ep", modelVersion: "latest"}, `--model-version must be a model version number such as 3, got "latest"`},
		{"a bad model", serveOptions{ucModel: "qwen", endpoint: "ep"}, `--uc-model must be a Unity Catalog model name as <catalog>.<schema>.<model>, got "qwen"`},
		{"a bad endpoint", serveOptions{ucModel: "a.b.c", endpoint: "a/b"}, `--endpoint-name must be a serving endpoint name of letters, digits, '-' and '_', got "a/b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadServeConfig(t.Context(), tt.opts)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestLoadServeConfigOverride(t *testing.T) {
	cfg, err := loadServeConfigText(t, gpuServeConfig, "compute.gpu.max_replicas=4")
	require.NoError(t, err)
	entity, err := cfg.servedEntity("3", commandPackage)
	require.NoError(t, err)
	assert.Equal(t, 16, entity.Serving.MaxConcurrency)

	_, err = loadServeConfigText(t, gpuServeConfig, "compute.workers=2")
	assert.ErrorContains(t, err, `"workers" is not a known field`)
}

func TestServeSchemaHelp(t *testing.T) {
	field, err := resolveSchemaField(serveSchema(), "compute.gpu.min_replicas")
	require.NoError(t, err)
	assert.Empty(t, field.required)

	field, err = resolveSchemaField(serveSchema(), "endpoint_name")
	require.NoError(t, err)
	assert.Equal(t, "yes", field.required)
}

// The examples in the schema help must load, so the help can't drift from the YAML.
func TestServeSchemaHelpExamplesLoad(t *testing.T) {
	for _, example := range helpExamples(t, serveSchema(), 3) {
		_, err := loadServeConfigText(t, example)
		assert.NoError(t, err, example)
	}
}
