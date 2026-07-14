package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

// decodeRunConfig reads and decodes the run YAML into the schema. Unknown keys
// are rejected (KnownFields).
//
// The `_bases_` composition feature is not yet ported; a config using `_bases_`
// is currently rejected as an unknown field. CLI `--override` handling lives in
// runconfig_override.go and is applied to the parsed map before this decode.
func decodeRunConfig(path string) (*runConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return decodeRunConfigReader(f, path)
}

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
func loadRunConfig(path string) (*runConfig, error) {
	cfg, err := decodeRunConfig(path)
	if err != nil {
		return nil, err
	}
	if err := validateRunConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadRunConfigWithOverrides decodes a run YAML config, applies any
// --override KEY=VALUE entries to the parsed map, then re-decodes (with unknown
// keys rejected) and structurally validates the result. Applying overrides to
// the map — rather than the typed config — lets the single decode+validate
// pipeline enforce path existence, type coercion, and the semantic rules at
// once. ctx is used only to log applied overrides.
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

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	if m == nil {
		// An empty file decodes to a nil map; start from an empty one so overrides
		// can populate it (the re-decode still enforces required fields).
		m = map[string]any{}
	}
	if err := applyOverrides(ctx, m, entries); err != nil {
		return nil, err
	}

	merged, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}
	cfg, err := decodeRunConfigReader(bytes.NewReader(merged), path)
	if err != nil {
		return nil, err
	}
	if err := validateRunConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
