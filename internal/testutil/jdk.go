package testutil

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/stretchr/testify/require"
)

func RequireJDK(t TestingT, ctx context.Context, version string) {
	var stderr bytes.Buffer

	cmd := exec.Command("javac", "-version")
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "Unable to run javac -version")

	// Get the first line of the output
	line := strings.Split(stderr.String(), "\n")[0]
	require.Contains(t, line, version, "Expected JDK version %s, got %s", version, line)
}
