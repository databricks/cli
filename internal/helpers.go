package internal

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GetEnvOrSkipTest proceeds with test only with that env variable
func GetEnvOrSkipTest(t *testing.T, name string) string {
	value := os.Getenv(name)
	if value == "" {
		t.Skipf("Environment variable %s is missing", name)
	}
	return value
}

// RandomName gives random name with optional prefix. e.g. qa.RandomName("tf-")
func RandomName(prefix ...string) string {
	rand.Seed(time.Now().UnixNano())
	randLen := 12
	b := make([]byte, randLen)
	for i := range b {
		b[i] = charset[rand.Intn(randLen)]
	}
	if len(prefix) > 0 {
		return fmt.Sprintf("%s%s", strings.Join(prefix, ""), b)
	}
	return string(b)
}

func BuildBinary(t *testing.T) string {
	// Absolute path to project root.
	abs, err := filepath.Abs("..")
	require.NoError(t, err)

	// Build binary and store it in temporary directory.
	dir := t.TempDir()
	bricksPath := path.Join(dir, "bricks")
	cmd := exec.Command("go", "build", "-o", bricksPath, "main.go")
	cmd.Dir = abs
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	require.NoError(t, err)

	return bricksPath
}

func run(t *testing.T, args ...string) ([]byte, error) {
	cmd := exec.Command(BuildBinary(t), args...)
	return cmd.Output()
}
