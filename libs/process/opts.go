package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/databricks/cli/libs/env"
)

type execOption func(context.Context, *exec.Cmd) error

func WithEnv(key, value string) execOption {
	return func(ctx context.Context, c *exec.Cmd) error {
		// we pull the env through lib/env such that we can run
		// parallel tests with anything using libs/process.
		if c.Env == nil {
			for k, v := range env.All(ctx) {
				c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}
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

func WithCombinedOutput(buf *bytes.Buffer) execOption {
	return func(_ context.Context, c *exec.Cmd) error {
		c.Stdout = io.MultiWriter(buf, c.Stdout)
		c.Stderr = io.MultiWriter(buf, c.Stderr)
		return nil
	}
}
