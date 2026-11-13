package sandbox

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCommandValidatesArgumentsWithoutAuthentication(t *testing.T) {
	for _, args := range [][]string{
		{"--sshd-config", "config", "--port", "22"},
		{"--sshd-config", "config", "--idle-timeout", "0s"},
		{"--sshd-config", "config", "unexpected"},
	} {
		cmd := newServerCommand()
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(t.Context())
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "credential")
	}
}

func TestServerCommandReportsReadiness(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sandbox servers run on Linux")
	}
	binary := filepath.Join(t.TempDir(), "sshd")
	require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700))
	cmd := newServerCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--sshd", binary, "--sshd-config", "prepared-config", "--port", "0", "--idle-timeout", "20ms"})
	require.NoError(t, cmd.ExecuteContext(t.Context()))
	var ready struct {
		Port int `json:"port"`
	}
	require.NoError(t, json.Unmarshal(output.Bytes(), &ready))
	assert.GreaterOrEqual(t, ready.Port, 1024)
}

func TestServerCommandReportsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	require.NoError(t, err)
	defer listener.Close()
	cmd := newServerCommand()
	cmd.SetArgs([]string{"--sshd-config", "prepared-config", "--port", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "listen for SSH WebSockets")
}
