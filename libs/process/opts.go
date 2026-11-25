package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type execOption func(context.Context, *exec.Cmd) error

// ExecOption is an option of Background and Forwarded. Code that passes options on to them names the type with it.
type ExecOption = execOption

func WithEnv(key, value string) execOption {
	return func(ctx context.Context, c *exec.Cmd) error {
		v := fmt.Sprintf("%s=%s", key, value)
		c.Env = append(c.Env, v)
		return nil
	}
}

func WithEnvs(envs map[string]string) execOption {
	return func(ctx context.Context, c *exec.Cmd) error {
		for k, v := range envs {
			err := WithEnv(k, v)(ctx, c)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

// WithoutEnv removes key from the environment that the process inherits, so the process does not see the variable at all.
// It is not the same as setting the variable to an empty value, which a process may read as set.
func WithoutEnv(key string) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Env = slices.DeleteFunc(c.Env, func(kv string) bool {
			return envEntryHasKey(runtime.GOOS, kv, key)
		})
		return nil
	}
}

// envEntryHasKey reports whether the environment entry kv, of the form key=value, sets key. Windows variable names are
// case-insensitive, so there the case of key does not matter. goos is a parameter so that tests can check both.
func envEntryHasKey(goos, kv, key string) bool {
	name, _, _ := strings.Cut(kv, "=")
	if goos == "windows" {
		return strings.EqualFold(name, key)
	}
	return name == key
}

// WithWaitDelay bounds how long Background waits for the output of the process once the process has exited or been
// killed. Without it, a child that a launcher started and that holds the output open would block Wait past a timeout.
// Unlike WithProcessGroup, it does not change the process group, so the process stays in the terminal's foreground group.
func WithWaitDelay(delay time.Duration) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.WaitDelay = delay
		return nil
	}
}

func WithDir(dir string) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Dir = dir
		return nil
	}
}

func WithStderrWriter(dst io.Writer) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Stderr = dst
		return nil
	}
}

func WithStdoutWriter(dst io.Writer) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Stdout = dst
		return nil
	}
}

// safeWriter is a writer that is safe to use concurrently.
// It serializes writes to the underlying writer.
type safeWriter struct {
	w io.Writer
	m sync.Mutex
}

func (s *safeWriter) Write(p []byte) (n int, err error) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.w.Write(p)
}

func WithCombinedOutput(buf *bytes.Buffer) execOption {
	sw := &safeWriter{w: buf}
	return func(_ context.Context, c *exec.Cmd) error {
		c.Stdout = io.MultiWriter(sw, c.Stdout)
		c.Stderr = io.MultiWriter(sw, c.Stderr)
		return nil
	}
}
