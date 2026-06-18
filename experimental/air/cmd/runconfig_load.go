package aircmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

// loadRunConfig reads a run YAML config file, decodes it into the schema, and
// runs structural validation. Unknown keys are rejected (KnownFields), mirroring
// the Python schema's extra="forbid".
//
// The `_bases_` composition feature and CLI `--override` handling are not yet
// ported; a config using `_bases_` is currently rejected as an unknown field.
func loadRunConfig(path string) (*runConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var cfg runConfig
	if err := dec.Decode(&cfg); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty", path)
		}
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}
