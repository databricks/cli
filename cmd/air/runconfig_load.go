package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

// decodeRunConfigReader decodes and unknown-key-checks a run YAML from r. path is
// used only for error messages.
func decodeRunConfigReader(r io.Reader, path string) (*runConfig, error) {
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)

	var cfg runConfig
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

	entries, err := parseOverrides(overrides)
	if err != nil {
		return nil, err
	}
	if err := validateOverridePaths(entries); err != nil {
		return nil, err
	}
	document, _, err := readRunConfigDocument(path)
	if err != nil {
		return nil, err
	}
	if len(document.Content) == 0 {
		document.Kind = yaml.DocumentNode
		document.Content = []*yaml.Node{newMappingNode()}
	}
	if err := applyOverrides(ctx, document, entries); err != nil {
		return nil, err
	}
	validationYAML, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config %s: %w", path, err)
	}
	return finishRunConfigLoad(path, document, validationYAML)
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
	if err := checkUnknownConfigFields(path, document.Content[0]); err != nil {
		return nil, err
	}
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

// checkUnknownConfigFields rejects keys absent from the run config schema. The
// strict decode that follows rejects them too, but its error names the Go struct
// rather than the field's path and offers no suggestion.
func checkUnknownConfigFields(path string, root *yaml.Node) error {
	schema := configSchema()
	unknown := unknownConfigFields(root, schema, schema.path, nil)
	switch len(unknown) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("invalid config %s: %s", path, unknown[0])
	default:
		return fmt.Errorf("invalid config %s:\n  %s", path, strings.Join(unknown, "\n  "))
	}
}

// unknownConfigFields describes each key under node that field's schema does not
// define. helpPath is the nearest documented ancestor, since hidden fields have
// no `-h config.<field>` page. Sequences add no path segment, matching how the
// schema addresses list elements.
func unknownConfigFields(node *yaml.Node, field configField, helpPath string, keys []string) []string {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	var unknown []string
	switch node.Kind {
	case yaml.SequenceNode:
		for _, element := range node.Content {
			unknown = append(unknown, unknownConfigFields(element, field, helpPath, keys)...)
		}
	case yaml.MappingNode:
		// Only objects have schema-defined keys; free-form maps and leaves do not.
		if len(field.children) == 0 {
			return nil
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			// A merge key pulls in another mapping's keys; the strict decode checks those.
			if key.Tag == "!!merge" {
				continue
			}
			keyPath := append(slices.Clone(keys), key.Value)
			child, ok := findConfigChild(field.children, key.Value)
			if !ok {
				msg := fmt.Sprintf("line %d: unknown field %q", key.Line, strings.Join(keyPath, "."))
				if suggestion, ok := closestConfigField(key.Value, visibleConfigChildNames(field)); ok {
					msg += fmt.Sprintf("; did you mean %q? Run", suggestion)
				} else {
					msg += "; run"
				}
				unknown = append(unknown, fmt.Sprintf("%s \"databricks air run -h %s\" for the list of fields", msg, helpPath))
				continue
			}
			childHelpPath := helpPath
			if !child.hidden {
				childHelpPath = child.path
			}
			unknown = append(unknown, unknownConfigFields(value, child, childHelpPath, keyPath)...)
		}
	default:
		// Scalars hold no keys.
	}
	return unknown
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
