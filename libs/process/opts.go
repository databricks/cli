package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type execOption func(context.Context, *exec.Cmd) error

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

func WithDir(dir string) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Dir = dir
		return nil
	}
}

func WithStdoutPipe(dst *io.ReadCloser) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		outPipe, err := c.StdoutPipe()
		if err != nil {
			return err
		}
		*dst = outPipe
		return nil
	}
}

func WithStdinReader(src io.Reader) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Stdin = src
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

// safeMultiWriter is a thread-safe io.Writer that writes to multiple writers.
// It is functionality equivalent to io.MultiWriter, but is safe for concurrent use.
type safeMultiWriter struct {
	writers []io.Writer
	mu      sync.Mutex
}

// newSafeMultiWriter creates a new safeMultiWriter that writes to the provided writers.
func newSafeMultiWriter(writers ...io.Writer) *safeMultiWriter {
	return &safeMultiWriter{writers: writers}
}

// Write implements the io.Writer interface for safeMultiWriter.
func (t *safeMultiWriter) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

func WithCombinedOutput(buf *bytes.Buffer) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Stdout = newSafeMultiWriter(buf, c.Stdout)
		c.Stderr = newSafeMultiWriter(buf, c.Stderr)
		return nil
	}
}
