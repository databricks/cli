package process

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

type ProcessError struct {
	Command string
	Err     error
	Stdout  string
	Stderr  string
}

func (perr *ProcessError) Unwrap() error {
	return perr.Err
}

func (perr *ProcessError) Error() string {
	return fmt.Sprintf("%s: %s", perr.Command, perr.Err)
}

func Background(ctx context.Context, args []string, opts ...execOption) (string, error) {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "running: %s", commandStr)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	// For background processes, there's no standard input
	cmd.Stdin = nil
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// we pull the env through lib/env such that we can run
	// parallel tests with anything using libs/process.
	for k, v := range env.All(ctx) {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for _, o := range opts {
		err := o(ctx, cmd)
		if err != nil {
			return "", err
		}
	}
	if err := runCmd(ctx, cmd); err != nil {
		return stdout.String(), &ProcessError{
			Err:     err,
			Command: commandStr,
			Stdout:  stdout.String(),
			Stderr:  stderr.String(),
		}
	}
	return stdout.String(), nil
}
