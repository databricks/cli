package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
		t.Skip("sandbox command output uses Linux pipes")
	}
	const marker = "server output write interrupted\n"
	if mode := os.Getenv("DATABRICKS_TEST_SERVER_OUTPUT"); mode != "" {
		// An inherited pipe remains writable even when its permissions prevent reopening.
		require.NoError(t, os.Stdout.Chmod(0))
		output, err := openServerOutput(os.Stdout)
		require.NoError(t, err)
		defer output.Close()
		file := output.(*os.File)
		if mode == "deadline" {
			require.NoError(t, file.SetWriteDeadline(time.Now().Add(100*time.Millisecond)))
		} else {
			timer := time.AfterFunc(100*time.Millisecond, func() { file.Close() })
			defer timer.Stop()
		}
		_, err = file.Write(make([]byte, 16<<20))
		if mode == "deadline" {
			require.ErrorIs(t, err, os.ErrDeadlineExceeded)
		} else {
			require.ErrorIs(t, err, fs.ErrClosed)
		}
		fmt.Fprint(os.Stderr, marker)
		return
	}
	for _, mode := range []string{"deadline", "close"} {
		t.Run(mode, func(t *testing.T) {
			executable, err := os.Executable()
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestServerOutputInterruptsInheritedStdout$")
			cmd.Env = append(os.Environ(), "DATABRICKS_TEST_SERVER_OUTPUT="+mode)
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
		})
	}
}

func TestServerCommandValidatesArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--sshd-config", "config", "--port", "22"},
		{"--sshd-config", "config", "--connect-timeout", "0s"},
		{"--sshd-config", "config", "unexpected"},
	} {
		cmd := NewServerCommand()
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
	cmd := NewServerCommand()
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
	cmd := NewServerCommand()
	cmd.SetArgs([]string{"--sshd-config", "prepared-config", "--port", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "listen for SSH WebSockets")
}

func TestServerCommandRejectsNonForwardablePorts(t *testing.T) {
	for _, port := range []string{"9999", "10101", "18767", "19090", "65536"} {
		t.Run(port, func(t *testing.T) {
			cmd := NewServerCommand()
			cmd.SetArgs([]string{"--sshd-config", "config", "--port", port})
			require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "port must be zero")
		})
	}
}
