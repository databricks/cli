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

	"github.com/databricks/cli/internal/build"
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
	if err := checkUnknownConfigFields(path, document); err != nil {
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

// unknownConfigKey is a YAML key absent from the run config schema.
type unknownConfigKey struct {
	line   int
	key    string
	parent configField
}

// checkUnknownConfigFields reports every YAML key missing from the run config
// schema by its YAML path, so users see the field they wrote rather than the
// strict decoder's Go type names.
func checkUnknownConfigFields(path string, document *yaml.Node) error {
	var unknown []unknownConfigKey
	collectUnknownConfigKeys(document.Content[0], configSchema(), &unknown)
	if len(unknown) == 0 {
		return nil
	}

	blocks := make([]string, 0, len(unknown))
	suggestUpdate := false
	for _, u := range unknown {
		names := make([]string, 0, len(u.parent.children))
		for _, c := range u.parent.children {
			if !c.hidden {
				names = append(names, configLeafName(c.path))
			}
		}
		slices.Sort(names)

		fieldPath := strings.TrimPrefix(u.parent.path+"."+u.key, configHelpRoot+".")
		block := fmt.Sprintf("line %d: unknown field %q", u.line, fieldPath)
		if suggestion, ok := closestConfigField(u.key, names); ok {
			block += fmt.Sprintf("; did you mean %q?", suggestion)
		} else {
			suggestUpdate = true
		}
		block += fmt.Sprintf("\nfields under %q are: %s", u.parent.path, strings.Join(names, ", "))
		// `-h` rejects hidden legacy paths, so only point at documented ones.
		if !u.parent.hidden {
			block += fmt.Sprintf("\nrun \"databricks air run -h %s\" for details", u.parent.path)
		}
		blocks = append(blocks, block)
	}

	msg := strings.Join(blocks, "\n\n")
	if len(blocks) > 1 {
		msg = fmt.Sprintf("%d unknown fields\n\n%s", len(blocks), msg)
	}
	if suggestUpdate {
		msg += fmt.Sprintf("\n\nfields from newer documentation may not be supported by this CLI (version %s); update the Databricks CLI (run 'databricks version --check')", build.GetInfo().Version)
	}
	return fmt.Errorf("invalid config %s: %s", path, msg)
}

// collectUnknownConfigKeys walks node against schema, appending unknown keys in
// document order. Non-object and free-form fields are left to the decoder.
func collectUnknownConfigKeys(node *yaml.Node, schema configField, unknown *[]unknownConfigKey) {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if len(schema.children) == 0 || schema.freeForm {
		return
	}
	switch node.Kind {
	case yaml.SequenceNode:
		for _, item := range node.Content {
			collectUnknownConfigKeys(item, schema, unknown)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			// The decoder folds "<<: *anchor" into this mapping, so its keys belong to schema.
			if key.ShortTag() == "!!merge" {
				collectUnknownConfigKeys(value, schema, unknown)
				continue
			}
			child, ok := findConfigChild(schema.children, key.Value)
			if !ok {
				*unknown = append(*unknown, unknownConfigKey{line: key.Line, key: key.Value, parent: schema})
				continue
			}
			collectUnknownConfigKeys(value, child, unknown)
		}
	default:
		// A scalar where an object is expected is a type error the decoder reports.
	}
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
