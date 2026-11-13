package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerOutputInterruptsInheritedStdout(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux sandbox command output uses /proc/self/fd")
	}
	const marker = "server output write interrupted\n"
	if os.Getenv("DATABRICKS_TEST_SERVER_OUTPUT") == "1" {
		output, err := openServerOutput(os.Stdout)
		require.NoError(t, err)
		defer output.Close()
		file := output.(*os.File)
		require.NoError(t, file.SetWriteDeadline(time.Now().Add(100*time.Millisecond)))
		_, err = file.Write(make([]byte, 16<<20))
		require.ErrorIs(t, err, os.ErrDeadlineExceeded)
		fmt.Fprint(os.Stderr, marker)
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestServerOutputInterruptsInheritedStdout$")
	cmd.Env = append(os.Environ(), "DATABRICKS_TEST_SERVER_OUTPUT=1")
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	// Keep stdout full until the child's write has been interrupted.
	line, err := bufio.NewReader(stderr).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, marker, line)
	_, err = io.Copy(io.Discard, stdout)
	require.NoError(t, err)
	require.NoError(t, cmd.Wait())
}

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
