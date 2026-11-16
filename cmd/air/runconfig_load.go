package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

// decodeRunConfigReader decodes and unknown-key-checks a run YAML from r. path is
// used only for error messages.
func decodeRunConfigReader(r io.Reader, path string) (*runConfig, error) {
	return decodeConfigReader[runConfig](r, path)
}

// decodeConfigReader decodes and unknown-key-checks a YAML config from r into a
// T. path is used only for error messages.
func decodeConfigReader[T any](r io.Reader, path string) (*T, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	var cfg T
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty", path)
		}
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &cfg, nil
}

// validateRunConfig runs structural validation over a decoded config.
func validateRunConfig(cfg *runConfig) error {
	return cfg.validate()
}

// loadRunConfig decodes and structurally validates a run YAML config file.
// The `_bases_` composition feature is not yet supported and is rejected as an
// unknown field.
func loadRunConfig(path string) (*runConfig, error) {
	document, raw, err := readRunConfigDocument(path)
	if err != nil {
		return nil, err
	}
	if len(document.Content) == 0 {
		return nil, fmt.Errorf("config %s is empty", path)
	}
	return finishRunConfigLoad(path, document, raw)
}

// loadRunConfigWithOverrides decodes a run YAML config, applies any --override
// KEY=VALUE entries to its ordered YAML tree, then re-decodes (with unknown keys
// rejected) and structurally validates the result. The serialized tree is kept
// for training_config.yaml, while validation may normalize the typed config used
// to submit the workload. ctx is used only to log applied overrides.
func loadRunConfigWithOverrides(ctx context.Context, path string, overrides []string) (*runConfig, error) {
	if len(overrides) == 0 {
		return loadRunConfig(path)
	}
	document, validationYAML, err := readConfigWithOverrides(ctx, path, overrides, configSchema())
	if err != nil {
		return nil, err
	}
	return finishRunConfigLoad(path, document, validationYAML)
}

// readConfigWithOverrides reads a YAML config and applies --override KEY=VALUE
// entries, checked against schema, to its ordered YAML tree. It returns the tree
// and its serialization, which the caller decodes with unknown keys rejected.
// ctx is used only to log applied overrides.
func readConfigWithOverrides(ctx context.Context, path string, overrides []string, schema configField) (*yaml.Node, []byte, error) {
	entries, err := parseOverrides(overrides)
	if err != nil {
		return nil, nil, err
	}
	if err := validateOverridePaths(entries, schema); err != nil {
		return nil, nil, err
	}
	document, _, err := readRunConfigDocument(path)
	if err != nil {
		return nil, nil, err
	}
	if len(document.Content) == 0 {
		document.Kind = yaml.DocumentNode
		document.Content = []*yaml.Node{newMappingNode()}
	}
	if err := applyOverrides(ctx, document, entries); err != nil {
		return nil, nil, err
	}
	validationYAML, err := yaml.Marshal(document)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize config %s: %w", path, err)
	}
	return document, validationYAML, nil
}

func readRunConfigDocument(path string) (*yaml.Node, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &document, raw, nil
}

func finishRunConfigLoad(path string, document *yaml.Node, validationYAML []byte) (*runConfig, error) {
	// Validate before stripping comments so YAML errors retain source line numbers.
	cfg, err := decodeRunConfigReader(bytes.NewReader(validationYAML), path)
	if err != nil {
		return nil, err
	}
	// YAML decodes omitted and null pointers alike. Preserve explicit nulls so validation rejects them.
	if cfg.CodeSource != nil {
		if cfg.CodeSource.Type == nil && codeSourceFieldPresent(document, "type") {
			cfg.CodeSource.Type = new(string)
		}
		if cfg.CodeSource.LegacySnapshot == nil && codeSourceFieldPresent(document, "snapshot") {
			cfg.CodeSource.LegacySnapshot = &snapshotSourceConfig{}
		}
	}
	if err := validateRunConfig(cfg); err != nil {
		return nil, err
	}

	stripYAMLComments(document)
	artifactYAML, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config %s: %w", path, err)
	}
	cfg.artifactYAML = artifactYAML
	return cfg, nil
}

func codeSourceFieldPresent(document *yaml.Node, field string) bool {
	codeSource, _ := mappingValue(document.Content[0], "code_source")
	if codeSource == nil || codeSource.Kind != yaml.MappingNode {
		return false
	}
	fieldNode, _ := mappingValue(codeSource, field)
	return fieldNode != nil
}

func stripYAMLComments(node *yaml.Node) {
	node.HeadComment = ""
	node.LineComment = ""
	node.FootComment = ""
	for _, child := range node.Content {
		stripYAMLComments(child)
	}
}

var runtimeVersionRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[0-9]+$`) })

const databricksAIPrefix = "databricks_ai_v"

func validateRuntimeVersion(version, source string) (string, error) {
	normalized := strings.ToLower(version)
	numeric := normalized
	usesDatabricksAI := strings.HasPrefix(normalized, databricksAIPrefix)
	if usesDatabricksAI {
		numeric = strings.TrimPrefix(normalized, databricksAIPrefix)
	}
	if !runtimeVersionRe().MatchString(numeric) {
		return "", fmt.Errorf("unsupported client image version %q in %s: version must be an integer, optionally prefixed with databricks_ai_v", version, source)
	}
	if !usesDatabricksAI {
		return numeric, nil
	}
	major, err := strconv.Atoi(numeric)
	if err != nil {
		return "", fmt.Errorf("failed to parse client image version %q in %s: %w", version, source, err)
	}
	if major < 5 {
		return "", fmt.Errorf("databricks_ai_v in %s requires AI Runtime version 5 or higher, got %q", source, version)
	}
	return databricksAIPrefix + numeric, nil
}
