package mutator

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/bricks/bundle"
)

const bundleVarPrefix = "BUNDLE_VAR_"

type loadVariablesFromEnvironment struct{}

func LoadVariablesFromEnvionment() bundle.Mutator {
	return &loadVariablesFromEnvironment{}
}

func (m *loadVariablesFromEnvironment) Name() string {
	return "LoadVariablesFromEnvionment"
}

func parseBundleVar(s string) (key string, val string, err error) {
	// env var name show have the bundle prefix to parse it
	if !strings.HasPrefix(s, bundleVarPrefix) {
		return "", "", fmt.Errorf("environment variable %s does not have expected prefix %s", s, bundleVarPrefix)
	}

	// split the env var entry at '=' to get the variable name and variable value
	components := strings.Split(s, "=")
	if len(components) != 2 {
		return "", "", fmt.Errorf("unexpected format for environment variable: %s", s)
	}
	key = strings.ToLower(
		strings.TrimPrefix(components[0], bundleVarPrefix),
	)
	val = components[1]
	return
}

func (m *loadVariablesFromEnvironment) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	envVars := os.Environ()
	for _, envVar := range envVars {
		if !strings.HasPrefix(envVar, bundleVarPrefix) {
			continue
		}
		varName, varVal, err := parseBundleVar(envVar)
		if err != nil {
			return nil, err
		}
		b.Config.Variables[varName] = varVal
	}
	return nil, nil
}
