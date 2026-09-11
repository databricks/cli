//go:build unix

// The second-signal escape hatch is a Unix signal-delivery behavior: the test
// re-execs itself and sends SIGINT/SIGTERM, which os/signal does not support on
// Windows (the CLI's own signal handling there is likewise a no-op path).
package environments

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The second-signal behavior can only be observed in a real process: signal
// dispositions are process-wide, and the test binary's own handlers would mask
// them. So the test re-executes itself as a child (Go resets dispositions to
// SIG_DFL across exec, unlike a shell background job, which inherits SIGINT as
// SIG_IGN and would silently invalidate the result).
const signalChildEnv = "TEST_ENVIRONMENTS_SIGNAL_CHILD"

// TestMain runs the signal-handler-under-test when re-executed as the child.
func TestMain(m *testing.M) {
	if os.Getenv(signalChildEnv) == "" {
		os.Exit(m.Run())
	}

	// TestMain has no *testing.T, so t.Context() is unavailable here.
	//nolint:gocritic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := watchInterruptSignals(ctx, cancel)
	defer stop()

	// Stand in for the pipeline still draining the uv process group during the
	// SIGKILL grace window — the interval in which the user needs the hatch.
	os.Stdout.WriteString("READY\n")
	<-ctx.Done()
	os.Stdout.WriteString("CANCELLED\n")
	time.Sleep(30 * time.Second)
	os.Stdout.WriteString("SURVIVED\n")
	os.Exit(0)
}

// TestWatchInterruptSignalsSecondSignalStillKills is the regression test for the
// escape hatch: signal.Notify disables the default SIGINT/SIGTERM disposition
// process-wide, so a handler that only relays the first signal leaves the user
// unable to abort. The first signal must cancel the context, and the second must
// terminate the process outright.
func TestWatchInterruptSignalsSecondSignalStillKills(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestWatchInterruptSignalsSecondSignalStillKills")
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
			require.NoError(t, cmd.Process.Signal(sig))
			require.True(t, await("CANCELLED"), "first %s did not cancel the context", sig)

			// The second signal must reach the default disposition and kill the
			// child rather than being buffered and dropped by the handler.
			require.NoError(t, cmd.Process.Signal(sig))
			waitErr := make(chan error, 1)
			go func() { waitErr <- cmd.Wait() }()
			select {
			case err := <-waitErr:
				// Killed by the signal, so a non-nil (non-exit-zero) error.
				assert.Error(t, err, "child should die by signal, not exit cleanly")
			case <-time.After(15 * time.Second):
				t.Fatalf("second %s was swallowed: the user has no escape hatch "+
					"during the process group's SIGKILL grace window", sig)
			}
		})
	}
}

// TestWatchInterruptSignalsStopsWithoutSignal covers the no-signal path: stop()
// must return (joining its goroutine) rather than leaving it parked on the
// channel for the life of the process.
func TestWatchInterruptSignalsStopsWithoutSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stop := watchInterruptSignals(ctx, cancel)

	returned := make(chan struct{})
	go func() {
		stop()
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("stop() blocked: the signal goroutine was never joined")
	}
}
