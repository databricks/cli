package testutil

import (
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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

func SkipUntil(t TestingT, date string) {
	deadline, err := time.Parse(time.DateOnly, date)
	require.NoError(t, err)

	if time.Now().Before(deadline) {
		t.Skipf("Skipping test until %s. Time right now: %s", deadline.Format(time.DateOnly), time.Now())
	}
}

func ReplaceWindowsLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
