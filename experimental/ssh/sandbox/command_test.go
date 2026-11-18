package sandbox_test

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCommandValidatesArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--sshd-config", "config", "--port", "22"},
		{"--sshd-config", "config", "--connect-timeout", "0s"},
		{"--sshd-config", "config", "unexpected"},
	} {
		cmd := sandbox.NewServerCommand()
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(t.Context())
		require.Error(t, err)
	}
}

func TestServerCommandReportsReadiness(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sandbox servers run on Linux")
	}
	binary := filepath.Join(t.TempDir(), "sshd")
	require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700))
	cmd := sandbox.NewServerCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--sshd", binary, "--sshd-config", "prepared-config", "--port", "0", "--connect-timeout", "20ms"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	var ready struct {
		Port int `json:"port"`
	}
	require.NoError(t, json.Unmarshal(output.Bytes(), &ready))
	assert.GreaterOrEqual(t, ready.Port, 10000)
}

func TestServerCommandReportsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	require.NoError(t, err)
	defer listener.Close()
	cmd := sandbox.NewServerCommand()
	cmd.SetArgs([]string{"--sshd-config", "prepared-config", "--port", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "listen for SSH WebSockets")
}

func TestServerCommandRejectsNonForwardablePorts(t *testing.T) {
	for _, port := range []string{"9999", "10101", "18767", "19090", "65536"} {
		t.Run(port, func(t *testing.T) {
			cmd := sandbox.NewServerCommand()
			cmd.SetArgs([]string{"--sshd-config", "config", "--port", port})
			require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "port must be zero")
		})
	}
}
