package testutil

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/stretchr/testify/require"
)

func RunCommand(t TestingT, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}

func CaptureCommandOutput(t TestingT, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err)
	return stdout.String()
}
