package internal

import (
	"bytes"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

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
