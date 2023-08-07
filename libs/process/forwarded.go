package process

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/log"
)

func Forwarded(ctx context.Context, args []string, opts ...execOption) error {
	commandStr := strings.Join(args, " ")
	log.Debugf(ctx, "starting: %s", commandStr)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	// make sure to sync on writing to stdout
	reader, writer := io.Pipe()
	go io.CopyBuffer(os.Stdout, reader, make([]byte, 128))
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

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go io.CopyBuffer(stdin, os.Stdin, make([]byte, 128))
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}
