package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/files"
	"go.yaml.in/yaml/v3"
)

// Where a version's MLmodel is in the Files API, and the most of it air serve reads.
const (
	modelVersionFilesPrefix = "/Models/"
	mlmodelFile             = "MLmodel"
	maxMLmodelBytes         = 1 << 20
)

// MLmodel keys air serve reads: metadata.entrypoint is the command that serves a
// version, and python_function's model_dir artifact is a dedicated checkpoint.
const (
	mlmodelEntrypointKey     = "entrypoint"
	mlmodelTaskKey           = "task"
	mlmodelDedicatedArtifact = "model_dir"
	llmTaskPrefix            = "llm/v1/"
)

// versionPackage is what a version's MLmodel says about how it is served.
type versionPackage struct {
	// command is set for a version served by its own command.
	command bool
	// dedicated is set for a checkpoint registered for dedicated inference.
	dedicated bool
	// task is the task the version is served for, "" if the MLmodel names none.
	task string
}

// readVersionPackage reads version of model's MLmodel as the caller.
func readVersionPackage(ctx context.Context, w *databricks.WorkspaceClient, model, version string) (versionPackage, error) {
	described := fmt.Sprintf("version %s of %s", version, model)
	path := modelVersionFilesPrefix + strings.ReplaceAll(model, ".", "/") + "/" + version + "/" + mlmodelFile
	resp, err := w.Files.Download(ctx, files.DownloadRequest{FilePath: path})
	if errors.Is(err, apierr.ErrNotFound) || errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return versionPackage{}, withCode(codeInvalidArgs, false, fmt.Errorf("%s doesn't exist or has no MLmodel, so air serve can't tell what serves it", described))
	}
	if err != nil {
		return versionPackage{}, fmt.Errorf("reading the MLmodel of %s: %w", described, err)
	}
	defer resp.Contents.Close()
	content, err := io.ReadAll(io.LimitReader(resp.Contents, maxMLmodelBytes+1))
	if err != nil {
		return versionPackage{}, fmt.Errorf("reading the MLmodel of %s: %w", described, err)
	}
	if len(content) > maxMLmodelBytes {
		return versionPackage{}, withCode(codeInvalidArgs, false, fmt.Errorf("the MLmodel of %s is larger than %d bytes", described, maxMLmodelBytes))
	}
	var doc map[string]any
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return versionPackage{}, withCode(codeInvalidArgs, false, fmt.Errorf("the MLmodel of %s isn't valid YAML: %w", described, err))
	}
	return mlmodelServedBy(doc), nil
}

// mlmodelServedBy describes how an MLmodel's version is served. It follows the
// server's rule for a dedicated checkpoint: an LLM task, no entrypoint key whatever
// its value, and a model_dir artifact with a path. Fields of an unexpected shape read
// as absent, as the server's reader takes them.
func mlmodelServedBy(doc map[string]any) versionPackage {
	metadata, _ := doc["metadata"].(map[string]any)
	_, hasEntrypoint := metadata[mlmodelEntrypointKey]
	task := nonEmptyString(metadata[mlmodelTaskKey])
	flavors, _ := doc["flavors"].(map[string]any)
	pyfunc, _ := flavors["python_function"].(map[string]any)
	artifacts, _ := pyfunc["artifacts"].(map[string]any)
	checkpoint, _ := artifacts[mlmodelDedicatedArtifact].(map[string]any)
	return versionPackage{
		command:   nonEmptyString(metadata[mlmodelEntrypointKey]) != "",
		dedicated: !hasEntrypoint && isLLMTask(task) && nonEmptyString(checkpoint["path"]) != "",
		task:      task,
	}
}

// isLLMTask reports whether task is one dedicated inference serves: an llm/v1 task
// other than audio, which is served only by a command.
func isLLMTask(task string) bool {
	return strings.HasPrefix(task, llmTaskPrefix) && task != taskAudioTranscriptions && task != taskAudioTranslations
}

// nonEmptyString is v when it is a string with something besides spaces, else "".
func nonEmptyString(v any) string {
	s, _ := v.(string)
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}
