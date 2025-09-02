package testutil

import (
	"os"
	"strings"

	"github.com/google/uuid"
)

// GetEnvOrSkipTest proceeds with test only with that env variable.
func GetEnvOrSkipTest(t TestingT, name string) string {
	value := os.Getenv(name)
	if value == "" {
		t.Skipf("Environment variable %s is missing", name)
	}
	return value
}

// RandomName gives random name with optional prefix. e.g. qa.RandomName("tf-")
func RandomName(prefix ...string) string {
	out := ""
	for _, p := range prefix {
		out += p
	}
	out += strings.ReplaceAll(uuid.New().String(), "-", "")
	return out
}

func ReplaceWindowsLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
