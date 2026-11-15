package aircmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const checkpointDeployConfig = `registered_model: main.models.qwen
model:
  type: checkpoint
  path: /Volumes/main/models/weights/qwen3_8b
task: llm/v1/chat
command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080
environment:
  dependencies: [einops==0.8.0]
endpoint: qwen-chat
compute:
  workload_type: GPU_MEDIUM
  workload_size: Small
secrets:
  HF_TOKEN: llm/hf_token
env_variables:
  LOG_LEVEL: info
`

const codeDeployConfig = `registered_model: main.agents.support
model:
  type: code
  entry: agent.py
task: agent/v1/responses
code_source:
  root_path: .
  include_paths: [agent.py, tools]
environment:
  version: 5
  dependencies: [mlflow==3.5.0]
endpoint: support-agent
compute:
  workload_type: CPU
  workload_size: Small
  scale_to_zero: true
`

func loadDeployConfigText(t *testing.T, text string, overrides ...string) (*deployConfig, error) {
	t.Helper()
	return loadDeployConfig(t.Context(), writeConfigFile(t, "deploy.yaml", text), overrides)
}

func TestLoadDeployConfigCheckpoint(t *testing.T) {
	cfg, err := loadDeployConfigText(t, checkpointDeployConfig)
	require.NoError(t, err)
	assert.True(t, cfg.registers())
	assert.Equal(t, "client.6-gpu", cfg.servingRuntime())
	assert.Equal(t, modelPackageEnvironment{Runtime: "client.6-gpu", Dependencies: []string{"einops==0.8.0"}}, cfg.servingEnvironment())
	assert.Equal(t, map[string]string{"LOG_LEVEL": "info", "HF_TOKEN": "{{secrets/llm/hf_token}}"}, cfg.servedEnvironmentVars())

	req := cfg.registrationRequest("key-1", nil)
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
}

func TestLoadDeployConfigCodeOnCPU(t *testing.T) {
	cfg, err := loadDeployConfigText(t, codeDeployConfig)
	require.NoError(t, err)
	assert.Equal(t, "client.5", cfg.servingRuntime())
	assert.Nil(t, cfg.servedEnvironmentVars())

	entity := cfg.servedEntity("4")
	assert.Equal(t, "main.agents.support", entity.EntityName)
	assert.Equal(t, "4", entity.EntityVersion)
	assert.True(t, entity.ScaleToZeroEnabled)
	assert.Contains(t, entity.ForceSendFields, "ScaleToZeroEnabled")
}

func TestLoadDeployConfigRegisteredVersion(t *testing.T) {
	cfg, err := loadDeployConfigText(t, `registered_model: main.ml.churn
model: {type: registered, version: 7}
endpoint: churn
compute: {workload_type: CPU, workload_size: Small}
`)
	require.NoError(t, err)
	assert.False(t, cfg.registers())
	assert.Equal(t, "7", *cfg.Model.Version)
}

func TestLoadDeployConfigSavedModelDir(t *testing.T) {
	cfg, err := loadDeployConfigText(t, `registered_model: main.ml.churn
model: {type: saved_model_dir, path: /Volumes/main/ml/runs/model}
mlflow_experiment_path: /Shared/churn
endpoint: churn
compute: {workload_type: CPU, workload_size: Small}
`)
	require.NoError(t, err)
	assert.Equal(t, registerModelPackageRequest{
		RegisteredModel: "main.ml.churn",
		IdempotencyKey:  "k",
		SavedModel:      &modelPackageSavedModel{VolumePath: "/Volumes/main/ml/runs/model"},
		ExperimentPath:  "/Shared/churn",
	}, cfg.registrationRequest("k", nil))
}

func TestLoadDeployConfigOverride(t *testing.T) {
	cfg, err := loadDeployConfigText(t, checkpointDeployConfig, "compute.workload_size=Medium", "environment.version=5")
	require.NoError(t, err)
	assert.Equal(t, "Medium", cfg.Compute.WorkloadSize)
	assert.Equal(t, "client.5-gpu", cfg.servingRuntime())
}

func TestLoadDeployConfigRejectsUnknownOverride(t *testing.T) {
	_, err := loadDeployConfigText(t, checkpointDeployConfig, "compute.accelerator_type=GPU_1xH100")
	assert.ErrorContains(t, err, `invalid --override "compute.accelerator_type": "accelerator_type" is not a known field; available fields are: scale_to_zero, workload_size, workload_type`)
}

func TestLoadDeployConfigRejectsUnknownField(t *testing.T) {
	_, err := loadDeployConfigText(t, checkpointDeployConfig+"experiment_name: x\n")
	assert.ErrorContains(t, err, "field experiment_name not found")
}

func TestLoadDeployConfigRejectsEmptyFile(t *testing.T) {
	_, err := loadDeployConfigText(t, "")
	assert.ErrorContains(t, err, "is empty")
}

func TestDeployConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(string) string
		wantErr string
	}{
		{"bad model name", replaceFirst("main.models.qwen", "qwen"), "registered_model must be a Unity Catalog model name"},
		{"backticked model name", replaceFirst("main.models.qwen", "main.models.`qwen`"), "registered_model must be a Unity Catalog model name"},
		{"unknown type", replaceFirst("type: checkpoint", "type: pickle"), `model.type must be one of checkpoint, code, saved_model_dir, registered, got "pickle"`},
		{"checkpoint without task", replaceFirst("task: llm/v1/chat\n", ""), "task is required when model.type is checkpoint"},
		{"checkpoint without command", dropLine("command:"), "command is required when model.type is checkpoint"},
		{"checkpoint with entry", replaceFirst("  type: checkpoint\n", "  type: checkpoint\n  entry: a.py\n"), "model.entry can't be set when model.type is checkpoint"},
		{"checkpoint with code_source", appendText("code_source: {root_path: .}\n"), "code_source can't be set when model.type is checkpoint"},
		{"path outside volumes", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/dbfs/weights"), "model.path must be a Unity Catalog volume directory"},
		{"path too long", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("a", 1100)), "model.path must be a Unity Catalog volume directory"},
		{"empty command", replaceFirst("command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080", `command: " "`), "command cannot be empty"},
		{"command too long", replaceFirst("command: python", "command: python "+strings.Repeat("x", maxServingCommandChars)), "command must be at most 32768 characters"},
		{"path too long in characters", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("é", 1020)), "model.path must be a Unity Catalog volume directory"},
		{"checkpoint on CPU", replaceFirst("GPU_MEDIUM", "CPU"), "a checkpoint is served on GPU"},
		{"unknown workload type", replaceFirst("GPU_MEDIUM", "GPU_MEDUIM"), `compute.workload_type must be one of CPU, CPU_LARGE, CPU_MEDIUM, GPU_LARGE, GPU_MEDIUM, GPU_SMALL, GPU_XLARGE, GPU_XLARGE_8, MULTIGPU_MEDIUM, got "GPU_MEDUIM"`},
		{"lowercase workload type", replaceFirst("GPU_MEDIUM", "gpu_medium"), "compute.workload_type must be one of"},
		{"agent task on a checkpoint", replaceFirst("task: llm/v1/chat", "task: agent/v1/responses"), `task of a checkpoint must be one of llm/v1/chat, llm/v1/completions, llm/v1/embeddings, got "agent/v1/responses"`},
		{"zero environment version", replaceFirst("  dependencies: [einops==0.8.0]", "  version: 0"), "environment.version must be a serverless environment version"},
		{"empty dependency", replaceFirst("[einops==0.8.0]", `[""]`), "environment.dependencies[0] cannot be empty"},
		{"no endpoint", dropLine("endpoint:"), `endpoint must be a serving endpoint name of letters, digits, '-' and '_', got ""`},
		{"endpoint with a path", replaceFirst("endpoint: qwen-chat", "endpoint: x/../../jobs"), "endpoint must be a serving endpoint name"},
		{"no compute", dropBlock("compute:"), "compute: section is required"},
		{"no workload size", dropLine("  workload_size:"), `compute.workload_size must be one of Small, Medium, Large, got ""`},
		{"lowercase workload size", replaceFirst("workload_size: Small", "workload_size: medium"), "compute.workload_size must be one of Small, Medium, Large"},
		{"secret without key", replaceFirst("llm/hf_token", "llm"), "expected format 'scope/key'"},
		{"env var and secret", replaceFirst("LOG_LEVEL: info", "HF_TOKEN: x"), `"HF_TOKEN" is set in both env_variables and secrets`},
		{"bad idempotency token", appendText("idempotency_token: a b\n"), "idempotency_token must be 1 to 128 letters"},
		{"long idempotency token", appendText("idempotency_token: " + strings.Repeat("k", 129) + "\n"), "idempotency_token must be 1 to 128 letters"},
		{"relative experiment", appendText("mlflow_experiment_path: Users/me/x\n"), "mlflow_experiment_path must be an absolute workspace path"},
		{"no model", dropBlock("model:"), "model: section is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadDeployConfigText(t, tt.edit(checkpointDeployConfig))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestDeployConfigSourceFields(t *testing.T) {
	base := "endpoint: e\ncompute: {workload_type: CPU, workload_size: Small}\nregistered_model: a.b.c\n"
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{"code without entry", "model: {type: code}\ncode_source: {root_path: .}\n", "model.entry is required when model.type is code"},
		{"code without code_source", "model: {type: code, entry: agent.py}\n", "code_source is required when model.type is code"},
		{"code with command", "model: {type: code, entry: agent.py}\ncode_source: {root_path: .}\ncommand: x\n", "command can't be set when model.type is code"},
		{"code entry in a directory", "model: {type: code, entry: src/agent.py}\ncode_source: {root_path: .}\n", "model.entry must name a .py file directly under code_source.root_path"},
		{"code entry not python", "model: {type: code, entry: agent.ipynb}\ncode_source: {root_path: .}\n", "model.entry must name a .py file"},
		{"code without include_paths", "model: {type: code, entry: agent.py}\ncode_source: {root_path: .}\n", "code_source.include_paths is required: list the files and directories to send"},
		{"code include pathspec magic", "model: {type: code, entry: agent.py}\ncode_source: {root_path: ., include_paths: [\":!creds.json\"]}\n", `code_source.include_paths must name files and directories, without patterns or surrounding spaces, got ":!creds.json"`},
		{"code include glob", "model: {type: code, entry: agent.py}\ncode_source: {root_path: ., include_paths: [\"*\"]}\n", `code_source.include_paths must name files and directories, without patterns or surrounding spaces, got "*"`},
		{"code include padded", "model: {type: code, entry: agent.py}\ncode_source: {root_path: ., include_paths: [\" agent.py\"]}\n", `got " agent.py"`},
		{"code include traversal", "model: {type: code, entry: agent.py}\ncode_source: {root_path: ., include_paths: [../x]}\n", "code_source.include_paths cannot contain '..' traversal"},
		{"saved model with task", "model: {type: saved_model_dir, path: /Volumes/a/b/c/m}\ntask: llm/v1/chat\n", "task can't be set when model.type is saved_model_dir"},
		{"saved model with environment", "model: {type: saved_model_dir, path: /Volumes/a/b/c/m}\nenvironment: {version: 6}\n", "environment can't be set when model.type is saved_model_dir"},
		{"registered without version", "model: {type: registered}\n", "model.version is required when model.type is registered"},
		{"registered with path", "model: {type: registered, version: 3, path: /Volumes/a/b/c/m}\n", "model.path can't be set when model.type is registered"},
		{"registered with token", "model: {type: registered, version: 3}\nidempotency_token: k\n", "idempotency_token can't be set when model.type is registered"},
		{"registered bad version", "model: {type: registered, version: v3}\n", "model.version must be a model version number"},
		{"registered version zero", "model: {type: registered, version: 0}\n", "model.version must be a model version number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadDeployConfigText(t, base+tt.source)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestDeployConfigCountsCharactersNotBytes(t *testing.T) {
	// 1,000 two-byte characters fit the 1,024-character cap on a volume path.
	_, err := loadDeployConfigText(t, replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("é", 1000))(checkpointDeployConfig))
	assert.NoError(t, err)
}

func TestDeployConfigServingRuntimeByWorkload(t *testing.T) {
	for workloadType, want := range map[string]string{
		"CPU":             "client.6",
		"CPU_LARGE":       "client.6",
		"GPU_SMALL":       "client.6-gpu",
		"MULTIGPU_MEDIUM": "client.6-gpu",
	} {
		cfg := deployConfig{Compute: &deployComputeConfig{WorkloadType: workloadType}}
		assert.Equal(t, want, cfg.servingRuntime(), workloadType)
	}
}

func TestDeploySchemaHelp(t *testing.T) {
	root, err := resolveSchemaField(deploySchema(), "config")
	require.NoError(t, err)
	var out strings.Builder
	renderConfigField(&out, deployCommandPath, root, nil)
	help := out.String()
	assert.Contains(t, help, "    databricks air deploy -h config.model.type\n")
	assert.Contains(t, help, "  Model:\n")
	assert.Contains(t, help, "  Serving:\n")
	assert.Contains(t, help, "  Registration:\n")
	assert.Contains(t, help, `For details on a field, run "databricks air deploy -h config.<field>".`)

	field, err := resolveSchemaField(deploySchema(), "model.type")
	require.NoError(t, err)
	assert.Equal(t, "yes", field.required)

	_, err = resolveSchemaField(deploySchema(), "model.typ")
	assert.ErrorContains(t, err, `did you mean "type"?`)

	_, err = resolveSchemaField(deploySchema(), "secrets.HF_TOKEN")
	assert.ErrorContains(t, err, "holds user-defined keys")
}

// Every source field named in sourceFields and requiredSourceFields must be a
// schema path, so the per-source checks can't drift from the YAML.
func TestDeploySourceFieldsAreSchemaPaths(t *testing.T) {
	for _, table := range []map[string][]string{sourceFields, requiredSourceFields} {
		for _, fields := range table {
			for _, f := range fields {
				_, err := resolveSchemaField(deploySchema(), f)
				assert.NoError(t, err, f)
			}
		}
	}
	for _, modelType := range modelTypes {
		assert.Contains(t, requiredSourceFields, modelType)
		assert.Contains(t, sourceFields, modelType)
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
