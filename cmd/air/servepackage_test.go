package aircmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// MLmodels of the versions air register writes, and of one MLflow logged.
const (
	commandMLmodel = `flavors:
  python_function:
    env: {virtualenv: python_env.yaml}
    python_version: "3.12"
metadata:
  task: llm/v1/chat
  entrypoint: python -m vllm.entrypoints.openai.api_server --model base --port 8080
  volume_artifacts_path: /Volumes/main/models/weights/base
`
	dedicatedMLmodel = `flavors:
  python_function:
    artifacts:
      model_dir: {path: artifacts/gemma4_e4b, uri: /Volumes/main/models/weights/gemma4_e4b}
    python_version: "3.12"
metadata:
  task: llm/v1/chat
  volume_artifacts_path: /Volumes/main/models/weights/gemma4_e4b
`
	loggedMLmodel = `flavors:
  python_function:
    loader_module: mlflow.sklearn
    python_version: "3.12"
  sklearn: {pickled_model: model.pkl}
`
)

func TestMLmodelServedBy(t *testing.T) {
	tests := []struct {
		name    string
		mlmodel string
		want    versionPackage
	}{
		{"a command", commandMLmodel, versionPackage{command: true, task: "llm/v1/chat"}},
		{"a dedicated checkpoint", dedicatedMLmodel, versionPackage{dedicated: true, task: "llm/v1/chat"}},
		{"a logged model", loggedMLmodel, versionPackage{}},
		// The server writes MLmodel as JSON, which is valid YAML.
		{"a dedicated checkpoint written as JSON", `{"flavors": {"python_function": {"artifacts": {"model_dir": {"path": "artifacts/gemma", "uri": "/Volumes/a/b/c/gemma"}}}}, "metadata": {"task": "llm/v1/chat"}}`, versionPackage{dedicated: true, task: "llm/v1/chat"}},
		{"a command written as JSON", `{"metadata": {"task": "llm/v1/embeddings", "entrypoint": "python serve.py"}}`, versionPackage{command: true, task: "llm/v1/embeddings"}},
		{"an audio command", replaceFirst("task: llm/v1/chat", "task: llm/v1/audio/transcriptions")(commandMLmodel), versionPackage{command: true, task: "llm/v1/audio/transcriptions"}},
		// The server refuses a custom entrypoint for dedicated inference.
		{"a model_dir with an entrypoint", dedicatedMLmodel + "  entrypoint: python serve.py\n", versionPackage{command: true, task: "llm/v1/chat"}},
		{"a model_dir without a path", replaceFirst("path: artifacts/gemma4_e4b, ", "")(dedicatedMLmodel), versionPackage{task: "llm/v1/chat"}},
		{"a blank entrypoint", replaceFirst("entrypoint: python -m vllm.entrypoints.openai.api_server --model base --port 8080", `entrypoint: " "`)(commandMLmodel), versionPackage{task: "llm/v1/chat"}},
		{"a model_dir that isn't a map", "flavors: {python_function: {artifacts: {model_dir: weights}}}\n", versionPackage{}},
		// The server serves only an LLM task with dedicated inference, so a custom
		// Python model's model_dir artifact doesn't make it a dedicated checkpoint.
		{"a model_dir without a task", replaceFirst("  task: llm/v1/chat\n", "")(dedicatedMLmodel), versionPackage{}},
		{"a model_dir with an audio task", replaceFirst("task: llm/v1/chat", "task: llm/v1/audio/transcriptions")(dedicatedMLmodel), versionPackage{task: "llm/v1/audio/transcriptions"}},
		{"a model_dir with another task", replaceFirst("task: llm/v1/chat", "task: classification")(dedicatedMLmodel), versionPackage{task: "classification"}},
		// The server refuses any entrypoint key, whatever its value.
		{"a model_dir with a null entrypoint", dedicatedMLmodel + "  entrypoint:\n", versionPackage{task: "llm/v1/chat"}},
		{"metadata that isn't a map", "metadata: []\nflavors: {python_function: x}\n", versionPackage{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(tt.mlmodel), &doc))
			assert.Equal(t, tt.want, mlmodelServedBy(doc))
		})
	}
}

// filesServer serves the Files API's download of path with body, and 404 for any
// other file.
func filesServer(t *testing.T, path, body string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/2.0/fs/files"+path:
			_, _ = w.Write([]byte(body))
		case strings.HasPrefix(r.URL.Path, "/api/2.0/fs/files/"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code": "NOT_FOUND", "message": "The file being accessed is not found."}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestReadVersionPackage(t *testing.T) {
	srv := filesServer(t, "/Models/main/models/qwen/3/MLmodel", dedicatedMLmodel)
	pkg, err := readVersionPackage(t.Context(), newTestWorkspaceClient(t, srv.URL), "main.models.qwen", "3")
	require.NoError(t, err)
	assert.Equal(t, versionPackage{dedicated: true, task: "llm/v1/chat"}, pkg)
}

func TestReadVersionPackageFailures(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		version string
		wantErr string
	}{
		{"no MLmodel", dedicatedMLmodel, "4", "version 4 of main.models.qwen doesn't exist or has no MLmodel, so air serve can't tell what serves it"},
		{"not YAML", "flavors: [", "3", "the MLmodel of version 3 of main.models.qwen isn't valid YAML"},
		{"not a mapping", "- flavors\n", "3", "the MLmodel of version 3 of main.models.qwen isn't valid YAML"},
		{"too large", "metadata: {task: " + string(make([]byte, maxMLmodelBytes)) + "}", "3", "the MLmodel of version 3 of main.models.qwen is larger than 1048576 bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := filesServer(t, "/Models/main/models/qwen/3/MLmodel", tt.body)
			_, err := readVersionPackage(t.Context(), newTestWorkspaceClient(t, srv.URL), "main.models.qwen", tt.version)
			assert.ErrorContains(t, err, tt.wantErr)
			code, _ := modelErrorCode(err)
			assert.Equal(t, codeInvalidArgs, code)
		})
	}
}
