package cmdio

import (
	"bufio"
	"context"
	"io"
)

type Test struct {
	Done context.CancelFunc

	Stdin  *bufio.Writer
	Stdout *bufio.Reader
	Stderr *bufio.Reader
}

func SetupTest(ctx context.Context) (context.Context, *Test) {
	rin, win := io.Pipe()
	rout, wout := io.Pipe()
	rerr, werr := io.Pipe()

	cmdio := &cmdIO{
		interactive: true,
		in:          rin,
		out:         wout,
		err:         werr,
	}

	ctx, cancel := context.WithCancel(ctx)
	ctx = InContext(ctx, cmdio)

	// Wait for context to be done, so we can drain stdin and close the pipes.
	go func() {
		<-ctx.Done()
		rin.Close()
		wout.Close()
		werr.Close()
	}()

	return ctx, &Test{
		Done:   cancel,
		Stdin:  bufio.NewWriter(win),
		Stdout: bufio.NewReader(rout),
		Stderr: bufio.NewReader(rerr),
	}
}
