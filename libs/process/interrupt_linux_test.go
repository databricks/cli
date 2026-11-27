package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// terminalHelperEnv makes TestBackgroundInGroupInterruptChildReadsTerminal
// run BackgroundInGroup in a helper process that has a terminal, which the
// test interrupts.
const terminalHelperEnv = "PROCESS_TEST_TERMINAL_HELPER_PIDFILE"

// runTerminalHelper is the helper process side of
// TestBackgroundInGroupInterruptChildReadsTerminal.
func runTerminalHelper(t *testing.T, pidFile string) {
	// On SIGTERM the child reads the terminal before it exits. The grandchild
	// records its own PID: $! is known right after fork, while the forked shell
	// still has the TERM trap, and a SIGTERM landing then can be lost, leaving
	// the sleep to hold the output pipe until the grace period ends.
	script := `trap 'read -r line </dev/tty; exit 1' TERM; sh -c 'echo $$ > "$1"; exec sleep 300' sh "$1" & wait`
	_, _ = BackgroundInGroup(t.Context(), []string{"sh", "-c", script, "sh", pidFile})
	// The raised signal ends the process asynchronously.
	time.Sleep(5 * time.Second)
	t.Fatal("the interrupt did not end the process")
}

// TestBackgroundInGroupInterruptChildReadsTerminal covers a child that reads
// the terminal once it got SIGTERM, as Claude Code does. It is Linux only
// because opening a pseudo-terminal takes platform-specific ioctls.
func TestBackgroundInGroupInterruptChildReadsTerminal(t *testing.T) {
	if pidFile := os.Getenv(terminalHelperEnv); pidFile != "" {
		runTerminalHelper(t, pidFile)
		return
	}
	ptmx, tty := openPty(t)
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	helper := exec.Command(os.Args[0], "-test.run=^TestBackgroundInGroupInterruptChildReadsTerminal$")
	helper.Env = append(os.Environ(), terminalHelperEnv+"="+pidFile)
	helper.Stdin, helper.Stdout, helper.Stderr = tty, tty, tty
	// Like a CLI started from a shell, the helper is in the foreground group of
	// its controlling terminal.
	helper.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}

	// Read the terminal so the helper never blocks writing to it, and keep what
	// it wrote to report it if the test fails. The copy ends once the helper
	// exits and the terminal has no other process left.
	var out bytes.Buffer
	outDone := make(chan struct{})
	go func() {
		defer close(outDone)
		_, _ = io.Copy(&out, ptmx)
	}()
	output := func() string {
		select {
		case <-outDone:
			return out.String()
		case <-time.After(time.Second):
			return "(the helper's terminal is still open)"
		}
	}

	require.NoError(t, helper.Start())
	// The helper leads its own session; kill it if the test fails before it ends.
	t.Cleanup(func() {
		_ = syscall.Kill(-helper.Process.Pid, syscall.SIGKILL)
		if t.Failed() {
			t.Logf("helper output:\n%s", output())
		}
	})
	require.NoError(t, tty.Close())

	grandchildPid := waitForPid(t, pidFile)
	// The child of BackgroundInGroup leads a session of its own, so killing the
	// helper does not stop it; kill its group if the test fails before it ends.
	childPgid, err := syscall.Getpgid(grandchildPid)
	require.NoError(t, err)
	require.NotEqual(t, syscall.Getpgrp(), childPgid, "the child must not share the test's process group")
	t.Cleanup(func() { _ = syscall.Kill(-childPgid, syscall.SIGKILL) })

	require.NoError(t, helper.Process.Signal(os.Interrupt))

	waitErr := make(chan error, 1)
	go func() { waitErr <- helper.Wait() }()
	// A child stopped by SIGTTIN would hold the helper until the SIGKILL at the
	// end of the grace period.
	select {
	case err := <-waitErr:
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		require.True(t, ok, "helper exited with %v", err)
		status := exitErr.Sys().(syscall.WaitStatus)
		assert.True(t, status.Signaled(), "the helper must end as the interrupt would have, got %v", err)
		assert.Equal(t, syscall.SIGINT, status.Signal())
	case <-time.After(processGroupGracePeriod / 2):
		t.Fatal("the interrupt did not end the helper before the grace period")
	}

	assert.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(grandchildPid, 0), syscall.ESRCH)
	}, 5*time.Second, 20*time.Millisecond, "grandchild %d outlived the interrupt", grandchildPid)
}

// openPty opens a pseudo-terminal and returns its master and slave ends.
func openPty(t *testing.T) (ptmx, tty *os.File) {
	ptmx, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ptmx.Close() })
	require.NoError(t, unix.IoctlSetPointerInt(int(ptmx.Fd()), unix.TIOCSPTLCK, 0))
	n, err := unix.IoctlGetUint32(int(ptmx.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	tty, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	require.NoError(t, err)
	return ptmx, tty
}
