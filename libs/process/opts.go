package process

import (
	"fmt"
	"os"
	"os/exec"
)

type execOption func(*exec.Cmd) error

func WithEnv(key, value string) execOption {
	return func(c *exec.Cmd) error {
		if c.Env == nil {
			c.Env = os.Environ()
		}
		v := fmt.Sprintf("%s=%s", key, value)
		c.Env = append(c.Env, v)
		return nil
	}
}

func WithEnvs(envs map[string]string) execOption {
	return func(c *exec.Cmd) error {
		for k, v := range envs {
			err := WithEnv(k, v)(c)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func WithDir(dir string) execOption {
	return func(c *exec.Cmd) error {
		c.Dir = dir
		return nil
	}
}
