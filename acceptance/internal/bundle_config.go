package internal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"dario.cat/mergo"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var filenames = []string{"databricks.yml", "databricks.yml.tmpl"}

func MergeBundleConfig(source string, bundleConfig map[string]any) (string, error) {
	config := make(map[string]any)

	err := yaml.Unmarshal([]byte(source), &config)
	if err != nil {
		return "", err
	}

	err = mergo.Merge(
		&config,
		bundleConfig,
		mergo.WithoutDereference,
	)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	err = enc.Encode(config)
	if err != nil {
		return "", err
	}

	updated := buf.String()
	return updated, nil
}

func ProcessBundleConfig(t *testing.T, dir string, bundleConfig map[string]any) {
	var configPath string
	for _, name := range filenames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return
	}

	var config map[string]any

	if configPath != "" {
		data := testutil.ReadFile(t, configPath)
		err := yaml.Unmarshal([]byte(data), &config)
		require.NoError(t, err, "failed to parse bundle config file: %s", configPath)
	}

	if len(config) == 0 {
		config = bundleConfig
	} else {
		err := mergo.Merge(
			&config,
			bundleConfig,
			mergo.WithoutDereference,
		)
		require.NoError(t, err, "Error during config merge\nconfigPath=%s\nconfig=%#v\nbundleConfig=%#v", configPath, config, bundleConfig)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	err := enc.Encode(config)
	require.NoError(t, err)

	updated := buf.String()
	testutil.WriteFile(t, configPath, updated)
}
