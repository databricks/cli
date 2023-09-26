package process

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

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
	return fmt.Sprintf("%s: %s %s", perr.Command, perr.Stderr, perr.Err)
}

func Background(ctx context.Context, args []string, opts ...execOption) (string, error) {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "running: %s", commandStr)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	// For background processes, there's no standard input
	cmd.Stdin = nil
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	for _, o := range opts {
		err := o(cmd)
		if err != nil {
			return "", err
		}
	}
	if err := cmd.Run(); err != nil {
		return "", &ProcessError{
			Err:     err,
			Command: commandStr,
			Stdout:  stdout.String(),
			Stderr:  stderr.String(),
		}
	}
	// trim leading/trailing whitespace from the output
	return strings.TrimSpace(stdout.String()), nil
}
