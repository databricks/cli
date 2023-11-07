package process

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

func Forwarded(ctx context.Context, args []string, src io.Reader, outWriter, errWriter io.Writer, opts ...execOption) error {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "starting: %s", commandStr)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// empirical tests showed buffered copies being more responsive
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	cmd.Stdin = src
	// we pull the env through lib/env such that we can run
	// parallel tests with anything using libs/process.
	for k, v := range env.All(ctx) {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// apply common options
	for _, o := range opts {
		err := o(ctx, cmd)
		if err != nil {
			return err
		}
	}

	return runCmd(ctx, cmd)
}
