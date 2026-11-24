package aircmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commandRegisterConfig is a checkpoint served by a command.
const commandRegisterConfig = `model_source: /Volumes/main/models/weights/qwen3_8b
uc_model: main.models.qwen
engine:
  environment:
    version: 6
    dependencies: [einops==0.8.0]
  command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080
  task: llm/v1/chat
`

// dedicatedRegisterConfig is a checkpoint served with dedicated inference.
const dedicatedRegisterConfig = `model_source: /Volumes/main/models/weights/qwen3_8b
uc_model: main.models.qwen
`

func loadRegisterConfigText(t *testing.T, text string, overrides ...string) (*registerConfig, error) {
	t.Helper()
	return loadRegisterConfig(t.Context(), registerOptions{file: writeConfigFile(t, "register.yaml", text), overrides: overrides})
}

func TestLoadRegisterConfigVariant(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, commandRegisterConfig)
	require.NoError(t, err)
	assert.Equal(t, variantCommand, cfg.variant())

	cfg, err = loadRegisterConfigText(t, dedicatedRegisterConfig)
	require.NoError(t, err)
	assert.Equal(t, variantDedicated, cfg.variant())
}

func TestLoadRegisterConfigCommand(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, commandRegisterConfig)
	require.NoError(t, err)
	assert.Equal(t, registerModelPackageRequest{
		ModelSource:    "/Volumes/main/models/weights/qwen3_8b",
		UCModel:        "main.models.qwen",
		IdempotencyKey: "key-1",
		Engine: &modelPackageEngine{
			Environment: modelPackageEnvironment{Version: "6", Dependencies: []string{"einops==0.8.0"}},
			Command:     "python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080",
			Task:        "llm/v1/chat",
		},
	}, cfg.registrationRequest("key-1"))
}

func TestLoadRegisterConfigCommandTask(t *testing.T) {
	// The server serves llm/v1/chat for an omitted task.
	cfg, err := loadRegisterConfigText(t, dropLine("  task:")(commandRegisterConfig))
	require.NoError(t, err)
	assert.Empty(t, cfg.registrationRequest("k").Engine.Task)

	cfg, err = loadRegisterConfigText(t, replaceFirst("task: llm/v1/chat", "task: llm/v1/audio/transcriptions")(commandRegisterConfig))
	require.NoError(t, err)
	assert.Equal(t, "llm/v1/audio/transcriptions", cfg.registrationRequest("k").Engine.Task)
}

func TestLoadRegisterConfigCommandOnDatabricksAI(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, replaceFirst("    version: 6\n", "    version: Databricks_AI_v6\n")(commandRegisterConfig))
	require.NoError(t, err)
	assert.Equal(t, modelPackageEnvironment{Version: "databricks_ai_v6", Dependencies: []string{"einops==0.8.0"}}, cfg.registrationRequest("k").Engine.Environment)
}

func TestRegistrationRequestSendsTheCanonicalVersion(t *testing.T) {
	for given, want := range map[string]string{
		"6":                 "6",
		`"06"`:              "6",
		"databricks_ai_v6":  "databricks_ai_v6",
		"Databricks_AI_v06": "databricks_ai_v6",
	} {
		cfg, err := loadRegisterConfigText(t, replaceFirst("    version: 6\n", "    version: "+given+"\n")(commandRegisterConfig))
		require.NoError(t, err, given)
		assert.Equal(t, want, cfg.registrationRequest("k").Engine.Environment.Version, given)
	}
}

func TestLoadRegisterConfigDedicated(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, dedicatedRegisterConfig)
	require.NoError(t, err)
	assert.Equal(t, registerModelPackageRequest{
		ModelSource:    "/Volumes/main/models/weights/qwen3_8b",
		UCModel:        "main.models.qwen",
		IdempotencyKey: "key-1",
	}, cfg.registrationRequest("key-1"))
}

func TestLoadRegisterConfigFromFlags(t *testing.T) {
	cfg, err := loadRegisterConfig(t.Context(), registerOptions{modelSource: "/Volumes/main/models/weights/gemma", ucModel: "main.models.gemma"})
	require.NoError(t, err)
	assert.Equal(t, &registerConfig{ModelSource: "/Volumes/main/models/weights/gemma", UCModel: "main.models.gemma"}, cfg)
}

func TestLoadRegisterConfigFlagsOverrideTheFile(t *testing.T) {
	cfg, err := loadRegisterConfig(t.Context(), registerOptions{
		file:        writeConfigFile(t, "register.yaml", commandRegisterConfig),
		modelSource: "/Volumes/main/models/weights/other",
		ucModel:     "main.models.other",
	})
	require.NoError(t, err)
	assert.Equal(t, "/Volumes/main/models/weights/other", cfg.ModelSource)
	assert.Equal(t, "main.models.other", cfg.UCModel)
	assert.Equal(t, variantCommand, cfg.variant())
}

func TestLoadRegisterConfigRefusesAnEmptyEngine(t *testing.T) {
	_, err := loadRegisterConfigText(t, dedicatedRegisterConfig+"engine:\n  # command: python serve.py\n")
	assert.ErrorContains(t, err, "engine is empty: set engine.command and engine.environment, or remove engine to register the checkpoint for dedicated inference")
}

func TestLoadRegisterConfigChecksFlagsByName(t *testing.T) {
	tests := []struct {
		name    string
		opts    registerOptions
		wantErr string
	}{
		{"an empty model source", registerOptions{ucModel: "a.b.c", flagsSet: map[string]bool{modelSourceFlag: true}}, "--model-source cannot be empty"},
		{"an empty model over a file", registerOptions{file: "unused.yaml", flagsSet: map[string]bool{ucModelFlag: true}}, "--uc-model cannot be empty"},
		{"a bad model source", registerOptions{modelSource: "/dbfs/w", ucModel: "a.b.c"}, `--model-source must be a Unity Catalog volume directory as /Volumes/<catalog>/<schema>/<volume>/<path>, of at most 1024 characters, got "/dbfs/w"`},
		{"a bad model", registerOptions{modelSource: "/Volumes/a/b/c/w", ucModel: "qwen"}, `--uc-model must be a Unity Catalog model name as <catalog>.<schema>.<model>, got "qwen"`},
		{"a logged model", registerOptions{modelSource: "models:/m-1", ucModel: "a.b.c"}, "model_source can't be a logged model yet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadRegisterConfig(t.Context(), tt.opts)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestLoadRegisterConfigNeedsAFileOrBothFlags(t *testing.T) {
	for _, opts := range []registerOptions{
		{},
		{modelSource: "/Volumes/a/b/c/w"},
		{ucModel: "a.b.c"},
	} {
		_, err := loadRegisterConfig(t.Context(), opts)
		assert.ErrorContains(t, err, "set --file, or both --model-source and --uc-model")
	}
	_, err := loadRegisterConfig(t.Context(), registerOptions{modelSource: "/Volumes/a/b/c/w", ucModel: "a.b.c", overrides: []string{"engine.task=x"}})
	assert.ErrorContains(t, err, "--override applies to a file")
}

func TestRegisterConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(string) string
		wantErr string
	}{
		{"no model source", dropLine("model_source:"), `model_source must be a Unity Catalog volume directory as /Volumes/<catalog>/<schema>/<volume>/<path>, of at most 1024 characters, got ""`},
		{"a logged model", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "models:/m-0586da44211e4ed2bace93129a99bf1e"), `model_source can't be a logged model yet, got "models:/m-0586da44211e4ed2bace93129a99bf1e": register it with MLflow's register_model, then serve the version with databricks air serve`},
		{"path outside volumes", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/dbfs/weights"), "model_source must be a Unity Catalog volume directory"},
		{"path too long", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("a", 1100)), "of at most 1024 characters"},
		{"path too long in characters", replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("é", 1020)), "model_source must be a Unity Catalog volume directory"},
		{"no model name", dropLine("uc_model:"), `uc_model must be a Unity Catalog model name as <catalog>.<schema>.<model>, got ""`},
		{"bad model name", replaceFirst("uc_model: main.models.qwen", "uc_model: qwen"), "uc_model must be a Unity Catalog model name"},
		{"backticked model name", replaceFirst("uc_model: main.models.qwen", "uc_model: main.models.`qwen`"), "uc_model must be a Unity Catalog model name"},
		{"engine without a command", dropLine("  command:"), "engine.command is required"},
		{"empty command", replaceFirst("command: python -m vllm.entrypoints.openai.api_server --model qwen3_8b --port 8080", `command: " "`), "engine.command cannot be empty"},
		{"command too long", replaceFirst("command: python", "command: python "+strings.Repeat("x", maxServingCommandChars)), "engine.command must be at most 32768 characters"},
		{"empty task", replaceFirst("task: llm/v1/chat", `task: " "`), "engine.task cannot be empty; leave it out to serve llm/v1/chat"},
		{"engine without an environment", dropBlock("  environment:"), "engine.environment is required"},
		{"environment without a version", dropLine("    version:"), "engine.environment.version is required"},
		{"AI environment before version 5", replaceFirst("    version: 6\n", "    version: databricks_ai_v4\n"), "databricks_ai_v in engine.environment.version requires AI Runtime version 5 or higher"},
		{"bad environment version", replaceFirst("    version: 6\n", "    version: six\n"), "unsupported client image version"},
		{"zero environment version", replaceFirst("    version: 6\n", "    version: 0\n"), "engine.environment.version must be a serverless environment version such as 6 or databricks_ai_v6"},
		{"empty dependency", replaceFirst("[einops==0.8.0]", `[""]`), "engine.environment.dependencies[0] cannot be empty"},
		{"an unknown field", appendText("mlflow_experiment_path: /Shared/qwen\n"), "field mlflow_experiment_path not found"},
		{"a serve field", appendText("endpoint_name: qwen-chat\n"), "field endpoint_name not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadRegisterConfigText(t, tt.edit(commandRegisterConfig))
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRegisterConfigCountsCharactersNotBytes(t *testing.T) {
	// 1,000 two-byte characters fit the 1,024-character cap on a volume path.
	_, err := loadRegisterConfigText(t, replaceFirst("/Volumes/main/models/weights/qwen3_8b", "/Volumes/"+strings.Repeat("é", 1000))(commandRegisterConfig))
	assert.NoError(t, err)
}

func TestLoadRegisterConfigOverride(t *testing.T) {
	cfg, err := loadRegisterConfigText(t, commandRegisterConfig, "engine.environment.version=5")
	require.NoError(t, err)
	assert.Equal(t, "5", cfg.registrationRequest("k").Engine.Environment.Version)

	_, err = loadRegisterConfigText(t, commandRegisterConfig, "engine.workers=2")
	assert.ErrorContains(t, err, `"workers" is not a known field`)
}

func TestLoadRegisterConfigRejectsEmptyFile(t *testing.T) {
	_, err := loadRegisterConfigText(t, "")
	assert.ErrorContains(t, err, "is empty")
}

func TestRegisterSchemaHelp(t *testing.T) {
	root, err := resolveSchemaField(registerSchema(), "config")
	require.NoError(t, err)
	var out strings.Builder
	renderConfigField(&out, registerCommandPath, root, nil)
	assert.Contains(t, out.String(), "    databricks air register -h config.model_source\n")

	field, err := resolveSchemaField(registerSchema(), "engine.environment.version")
	require.NoError(t, err)
	assert.Equal(t, "yes", field.required)

	_, err = resolveSchemaField(registerSchema(), "model_sorce")
	assert.ErrorContains(t, err, `did you mean "model_source"?`)
}

// The examples in the schema help must load, so the help can't drift from the YAML.
func TestRegisterSchemaHelpExamplesLoad(t *testing.T) {
	for _, example := range helpExamples(t, registerSchema(), 2) {
		_, err := loadRegisterConfigText(t, example)
		assert.NoError(t, err, example)
	}
}

// helpExamples returns the YAML examples in a schema's help: the blocks after its
// opening sentence and -h example, without their titles and indentation.
func helpExamples(t *testing.T, schema configField, want int) []string {
	t.Helper()
	blocks := strings.Split(schema.help, "\n\n  ")[2:]
	require.Len(t, blocks, want)
	var examples []string
	for _, block := range blocks {
		_, body, _ := strings.Cut(block, ":\n")
		var lines []string
		for line := range strings.SplitSeq(body, "\n") {
			lines = append(lines, strings.TrimPrefix(line, "    "))
		}
		examples = append(examples, strings.Join(lines, "\n")+"\n")
	}
	return examples
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

// dropBlock removes the first line starting with prefix, a key at the indentation
// prefix has, and the lines indented under it.
func dropBlock(prefix string) func(string) string {
	nested := strings.Repeat(" ", len(prefix)-len(strings.TrimLeft(prefix, " "))) + "  "
	return func(s string) string {
		var kept []string
		dropping, dropped := false, false
		for _, line := range strings.SplitAfter(s, "\n") {
			switch {
			case !dropped && strings.HasPrefix(line, prefix):
				dropping, dropped = true, true
			case dropping && strings.HasPrefix(line, nested):
			default:
				dropping = false
				kept = append(kept, line)
			}
		}
		if !dropped {
			panic("fixture has no line starting with " + prefix)
		}
		return strings.Join(kept, "")
	}
}
