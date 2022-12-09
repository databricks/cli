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

	"github.com/databricks/bricks/cmd/root"
	"github.com/stretchr/testify/assert"
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

// Helper for running the bricks root command in the background.
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
	cmd := root.RootCmd
	cmd.SetOut(&t.stdout)
	cmd.SetErr(&t.stderr)
	cmd.SetArgs(t.args)

	errch := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	// Run command in background.
	go func() {
		err := cmd.ExecuteContext(ctx)
		if err != nil {
			t.Logf("Error running command: %s", err)
		}

		if t.stdout.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(t.stdout.Bytes()))
			for scanner.Scan() {
				t.Logf("[bricks stdout]: %s", scanner.Text())
			}
		}

		if t.stderr.Len() > 0 {
			// Make a copy of the buffer such that it remains "unread".
			scanner := bufio.NewScanner(bytes.NewBuffer(t.stderr.Bytes()))
			for scanner.Scan() {
				t.Logf("[bricks stderr]: %s", scanner.Text())
			}
		}

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

// Like [assert.Eventually] but errors if the underlying command has failed.
func (c *cobraTestRunner) Eventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) bool {
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case err := <-c.errch:
			assert.Fail(c, "Command failed", err)
		case <-timer.C:
			assert.Fail(c, "Condition never satisfied", msgAndArgs...)
		case <-tick:
			tick = nil
			go func() { ch <- condition() }()
		case v := <-ch:
			if v {
				return true
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

func writeFile(t *testing.T, name string, body string) string {
	f, err := os.Create(filepath.Join(t.TempDir(), name))
	require.NoError(t, err)
	_, err = f.WriteString(body)
	require.NoError(t, err)
	f.Close()
	return f.Name()
}
