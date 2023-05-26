package internal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	_ "github.com/databricks/cli/cmd/version"
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

// Helper for running the root command in the background.
// It ensures that the background goroutine terminates upon
// test completion through cancelling the command context.
type cobraTestRunner struct {
	*testing.T

	args   []string
	stdout bytes.Buffer
	stderr bytes.Buffer

	errch <-chan error
}

func (t *cobraTestRunner) RunBackground() {
	root := root.RootCmd
	root.SetOut(&t.stdout)
	root.SetErr(&t.stderr)
	root.SetArgs(t.args)

	errch := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	// Run command in background.
	go func() {
		cmd, err := root.ExecuteContextC(ctx)
		if err != nil {
			t.Logf("Error running command: %s", err)
		}

		if t.stdout.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(t.stdout.Bytes()))
			for scanner.Scan() {
				t.Logf("[databricks stdout]: %s", scanner.Text())
			}
		}

		if t.stderr.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(t.stderr.Bytes()))
			for scanner.Scan() {
				t.Logf("[databricks stderr]: %s", scanner.Text())
			}
		}

		// Reset context on command for the next test.
		// These commands are globals so we have to clean up to the best of our ability after each run.
		// See https://github.com/spf13/cobra/blob/a6f198b635c4b18fff81930c40d464904e55b161/command.go#L1062-L1066
		//lint:ignore SA1012 cobra sets the context and doesn't clear it
		cmd.SetContext(nil)

		// Make caller aware of error.
		errch <- err
		close(errch)
	}()

	// Ensure command terminates upon test completion (success or failure).
	t.Cleanup(func() {
		// Signal termination of command.
		cancel()
		// Wait for goroutine to finish.
		<-errch
	})

	t.errch = errch
}

func (t *cobraTestRunner) Run() (bytes.Buffer, bytes.Buffer, error) {
	t.RunBackground()
	err := <-t.errch
	return t.stdout, t.stderr, err
}

// Like [require.Eventually] but errors if the underlying command has failed.
func (c *cobraTestRunner) Eventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) {
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case err := <-c.errch:
			require.Fail(c, "Command failed", err)
			return
		case <-timer.C:
			require.Fail(c, "Condition never satisfied", msgAndArgs...)
			return
		case <-tick:
			tick = nil
			go func() { ch <- condition() }()
		case v := <-ch:
			if v {
				return
			}
			tick = ticker.C
		}
	}
}

func NewCobraTestRunner(t *testing.T, args ...string) *cobraTestRunner {
	return &cobraTestRunner{
		T:    t,
		args: args,
	}
}

func RequireSuccessfulRun(t *testing.T, args ...string) (bytes.Buffer, bytes.Buffer) {
	c := NewCobraTestRunner(t, args...)
	stdout, stderr, err := c.Run()
	require.NoError(t, err)
	return stdout, stderr
}

func RequireErrorRun(t *testing.T, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	c := NewCobraTestRunner(t, args...)
	stdout, stderr, err := c.Run()
	require.Error(t, err)
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
