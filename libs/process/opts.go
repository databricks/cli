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
