package testutil

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomName gives random name with optional prefix. e.g. qa.RandomName("tf-")
func RandomName(prefix ...string) string {
	randLen := 12
	b := make([]byte, randLen)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	if len(prefix) > 0 {
		return fmt.Sprintf("%s%s", strings.Join(prefix, ""), b)
	}
	return string(b)
}

func SkipUntil(t TestingT, date string) {
	deadline, err := time.Parse(time.DateOnly, date)
	require.NoError(t, err)

	if time.Now().Before(deadline) {
		t.Skipf("Skipping test until %s. Time right now: %s", deadline.Format(time.DateOnly), time.Now())
	}
}
