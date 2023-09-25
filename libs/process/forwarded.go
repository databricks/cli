package process

import (
	"context"
	"io"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/log"
)

func Forwarded(ctx context.Context, args []string, src io.Reader, dst io.Writer, opts ...execOption) error {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "starting: %s", commandStr)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// make sure to sync on writing to stdout
	reader, writer := io.Pipe()
	go io.CopyBuffer(dst, reader, make([]byte, 128))
	defer reader.Close()
	defer writer.Close()
	cmd.Stdout = writer
	cmd.Stderr = writer

	// apply common options
	for _, o := range opts {
		err := o(cmd)
		if err != nil {
			return err
		}
	}

	// pipe standard input to the child process, so that we can allow terminal UX
	// see the PoC at https://github.com/databricks/cli/pull/637
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go io.CopyBuffer(stdin, src, make([]byte, 128))
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}
