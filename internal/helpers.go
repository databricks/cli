package internal

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/bricks/cmd/root"
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

func run(t *testing.T, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := root.RootCmd
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	_, err := root.ExecuteC()
	if stdout.Len() > 0 {
		t.Logf("[stdout]: %s", stdout.String())
	}
	if stderr.Len() > 0 {
		t.Logf("[stderr]: %s", stderr.String())
	}
	return stdout, stderr, err
}

func writeFile(t *testing.T, name string, body string) string {
	f, err := os.Create(filepath.Join(t.TempDir(), name))
	require.NoError(t, err)
	_, err = f.WriteString(body)
	require.NoError(t, err)
	f.Close()
	return f.Name()
}
