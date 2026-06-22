package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/filer"
	"go.yaml.in/yaml/v3"
)

// Launch artifact basenames, uploaded into the run's cli_launch directory. The
// server-side launcher derives requirements.yaml / hyperparameters.yaml from the
// same directory, so these names are part of the contract.
const (
	trainingConfigName  = "training_config.yaml"
	commandScriptName   = "command.sh"
	requirementsName    = "requirements.yaml"
	hyperparametersName = "hyperparameters.yaml"
	envVarsName         = "env_vars.json"
	secretEnvVarsName   = "secret_env_vars.json"
)

// maxConfigYAMLBytes caps training_config.yaml. It is referenced by the Jobs
// payload and rendered on the run page, so an oversized parameters/command block
// is rejected here; full parameters still ship in hyperparameters.yaml.
const maxConfigYAMLBytes = 1024 * 1024

// uploadItem is a single artifact to write into the launch directory.
type uploadItem struct {
	name string
	data []byte
}

// fileWriter is the subset of filer.Filer the upload path needs; a narrow
// interface keeps buildArtifacts/upload testable without a live workspace.
type fileWriter interface {
	Write(ctx context.Context, name string, reader io.Reader, mode ...filer.WriteMode) error
}

// requirementsDoc mirrors the on-disk requirements.yaml format so the worker
// parses synthesized inline dependencies identically to a user-provided file.
type requirementsDoc struct {
	Version      string   `yaml:"version,omitempty"`
	Dependencies []string `yaml:"dependencies"`
}

// buildArtifacts assembles the files to upload for a run: the merged config, the
// inline command as a script, requirements (from a file or synthesized from
// inline dependencies), and hyperparameters. configPath is the local YAML path.
func buildArtifacts(cfg *runConfig, configPath string) ([]uploadItem, error) {
	// TODO(DABs): with no _bases_/overrides ported yet, the merged config is the
	// file as-is; once those land, upload the re-serialized merged YAML instead.
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %s: %w", configPath, err)
	}
	if len(configData) > maxConfigYAMLBytes {
		return nil, fmt.Errorf("config YAML is %.2f MB, over the %d MB limit; reduce 'parameters' or 'command'",
			float64(len(configData))/(1024*1024), maxConfigYAMLBytes/(1024*1024))
	}

	items := []uploadItem{
		{trainingConfigName, configData},
		{commandScriptName, []byte(*cfg.Command)},
	}

	switch reqPath, ok := cfg.requirementsFile(); {
	case ok:
		// Resolve a relative requirements path against the config's directory.
		if !filepath.IsAbs(reqPath) {
			reqPath = filepath.Join(filepath.Dir(configPath), reqPath)
		}
		data, err := os.ReadFile(reqPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read requirements file %s: %w", reqPath, err)
		}
		items = append(items, uploadItem{requirementsName, data})
	default:
		if deps, ok := cfg.inlineDependencies(); ok {
			version, _ := cfg.runtimeVersion()
			data, err := yaml.Marshal(requirementsDoc{Version: version, Dependencies: deps})
			if err != nil {
				return nil, fmt.Errorf("failed to synthesize requirements.yaml: %w", err)
			}
			items = append(items, uploadItem{requirementsName, data})
		}
	}

	if len(cfg.Parameters) > 0 {
		data, err := yaml.Marshal(cfg.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize parameters: %w", err)
		}
		items = append(items, uploadItem{hyperparametersName, data})
	}

	// The ai_runtime_task proto carries no inline env vars or secrets; stage them
	// as JSON files co-located with command.sh for the server-side launcher.
	if len(cfg.EnvVariables) > 0 {
		data, err := json.Marshal(envVarEntries(cfg.EnvVariables))
		if err != nil {
			return nil, fmt.Errorf("failed to serialize env_variables: %w", err)
		}
		items = append(items, uploadItem{envVarsName, data})
	}
	if len(cfg.Secrets) > 0 {
		data, err := json.Marshal(secretEnvVarEntries(cfg.Secrets))
		if err != nil {
			return nil, fmt.Errorf("failed to serialize secrets: %w", err)
		}
		items = append(items, uploadItem{secretEnvVarsName, data})
	}

	return items, nil
}

// envVarEntry is one entry in env_vars.json.
type envVarEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// secretEnvVarEntry is one entry in secret_env_vars.json. The YAML side is
// {ENV_VAR: "scope/key"}; the launcher wants the split form.
type secretEnvVarEntry struct {
	Name        string `json:"name"`
	SecretScope string `json:"secret_scope"`
	SecretKey   string `json:"secret_key"`
}

// envVarEntries renders env_variables sorted by name for deterministic output.
func envVarEntries(vars map[string]string) []envVarEntry {
	out := make([]envVarEntry, 0, len(vars))
	for _, name := range slices.Sorted(maps.Keys(vars)) {
		out = append(out, envVarEntry{Name: name, Value: vars[name]})
	}
	return out
}

// secretEnvVarEntries renders secrets sorted by name for deterministic output.
func secretEnvVarEntries(secrets map[string]string) []secretEnvVarEntry {
	out := make([]secretEnvVarEntry, 0, len(secrets))
	for _, name := range slices.Sorted(maps.Keys(secrets)) {
		scope, key, _ := strings.Cut(secrets[name], "/")
		out = append(out, secretEnvVarEntry{Name: name, SecretScope: scope, SecretKey: key})
	}
	return out
}

// uploadArtifacts writes each artifact into the launch directory, overwriting and
// creating parents as needed.
//
// TODO(DABs): this client-side upload could move onto libs/sync / a bundle deploy
// so the CLI reuses DABs' file-staging machinery instead of writing files itself.
func uploadArtifacts(ctx context.Context, w fileWriter, items []uploadItem) error {
	for _, it := range items {
		if err := w.Write(ctx, it.name, bytes.NewReader(it.data), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
			return fmt.Errorf("failed to upload %s: %w", it.name, err)
		}
	}
	return nil
}
