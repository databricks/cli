package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/log"
)

func Background(ctx context.Context, args []string, opts ...execOption) (string, error) {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "running: %s", commandStr)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdout := &bytes.Buffer{}
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	for _, o := range opts {
		err := o(cmd)
		if err != nil {
			return "", err
		}
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s %w", commandStr, stdout.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}
