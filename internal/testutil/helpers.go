package testutil

import (
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
)

// GetEnvOrSkipTest proceeds with test only with that env variable.
func GetEnvOrSkipTest(t TestingT, name string) string {
	value := env.Get(t.Context(), name)
	if value == "" {
		t.Skipf("Environment variable %s is missing", name)
	}
	return value
}

// RandomName gives random name with optional prefix. e.g. qa.RandomName("tf-")
func RandomName(prefix ...string) string {
	var sb strings.Builder
	for _, p := range prefix {
		sb.WriteString(p)
	}
	sb.WriteString(strings.ReplaceAll(uuid.New().String(), "-", ""))
	return sb.String()
}
