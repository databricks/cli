//go:build unix

// The second-signal escape hatch is a Unix signal-delivery behavior: the test
// re-execs itself and sends SIGINT, which os/signal cannot deliver to self on
// Windows (notifyInterrupt only watches os.Interrupt there anyway).
package aircmd

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// notifyInterrupt flips signal dispositions process-wide, so the second-signal
// behavior can only be observed in a real child process where the test binary's
// own handlers don't mask it. Go resets dispositions to SIG_DFL across exec.
const signalChildEnv = "TEST_AIR_SIGNAL_CHILD"

// TestMain runs notifyInterrupt under test when re-executed as the child.
func TestMain(m *testing.M) {
	if os.Getenv(signalChildEnv) == "" {
		os.Exit(m.Run())
	}

	// TestMain has no *testing.T, so t.Context() is unavailable here.
	//nolint:gocritic
	ctx, stop := notifyInterrupt(context.Background())
	defer stop()

	// Stand in for a log stream still draining after the first Ctrl-C; the user
	// needs the escape hatch during this window.
	os.Stdout.WriteString("READY\n")
	<-ctx.Done()
	os.Stdout.WriteString("CANCELLED\n")
	time.Sleep(30 * time.Second)
	os.Stdout.WriteString("SURVIVED\n")
	os.Exit(0)
}

// TestNotifyInterruptSecondSignalStillKills is the regression test for the
// escape hatch: signal.NotifyContext disables the default SIGINT disposition
// process-wide, so relaying only the first signal would leave the user unable to
// abort. The first Ctrl-C must cancel the context (so handleWatchResult prints
// resume guidance), and the second must terminate the process outright.
func TestNotifyInterruptSecondSignalStillKills(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestNotifyInterruptSecondSignalStillKills")
	cmd.Env = append(os.Environ(), signalChildEnv+"=1")
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	lines := make(chan string, 8)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	await := func(want string) bool {
		deadline := time.After(30 * time.Second)
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					return false
				}
				if strings.Contains(l, want) {
					return true
				}
			case <-deadline:
				return false
			}
		}
	}

	require.True(t, await("READY"), "child never started")
	require.NoError(t, cmd.Process.Signal(os.Interrupt))
	require.True(t, await("CANCELLED"), "first interrupt did not cancel the context")

	// The second interrupt must reach the default disposition and kill the child
	// rather than being buffered and dropped by the handler.
	require.NoError(t, cmd.Process.Signal(os.Interrupt))
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case err := <-waitErr:
		// Killed by the signal, so a non-nil (non-exit-zero) error.
		assert.Error(t, err, "child should die by signal, not exit cleanly")
	case <-time.After(15 * time.Second):
		t.Fatal("second interrupt was swallowed: the user has no way to abort a hung stream")
	}
}
