package cmdio

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/databricks/bricks/libs/flags"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

type cmdIO struct {
	interactive  bool
	outputFormat flags.Output
	template     string
	out          io.Writer
}

func NewIO(outputFormat flags.Output, out io.Writer, template string) *cmdIO {
	return &cmdIO{
		interactive:  !color.NoColor,
		outputFormat: outputFormat,
		template:     template,
		out:          out,
	}
}

func (c *cmdIO) IsTTY() bool {
	f, ok := c.out.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func (c *cmdIO) Render(v any) error {
	// TODO: add terminal width & white/dark theme detection
	switch c.outputFormat {
	case flags.OutputJSON:
		return renderJson(c.out, v)
	case flags.OutputText:
		if c.template != "" {
			return newFunction(c.out, c.template, v)
		}
		return renderJson(c.out, v)
	default:
		return fmt.Errorf("invalid output format: %s", c.outputFormat)
	}
}

func Render(ctx context.Context, v any) error {
	c := fromContext(ctx)
	return c.Render(v)
}

type cmdIOType int

var cmdIOKey cmdIOType

func InContext(ctx context.Context, io *cmdIO) context.Context {
	return context.WithValue(ctx, cmdIOKey, io)
}

func fromContext(ctx context.Context) *cmdIO {
	io, ok := ctx.Value(cmdIOKey).(*cmdIO)
	if !ok {
		panic("no cmdIO found in the context. Please report it as an issue")
	}
	return io
}
