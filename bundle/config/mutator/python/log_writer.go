package python

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/databricks/cli/libs/log"
)

type logWriter struct {
	ctx    context.Context
	prefix string
	buf    bytes.Buffer
}

// newLogWriter creates a new io.Writer that writes to log with specified prefix.
func newLogWriter(ctx context.Context, prefix string) io.Writer {
	return &logWriter{
		ctx:    ctx,
		prefix: prefix,
	}
}

func (p *logWriter) Write(bytes []byte) (n int, err error) {
	p.buf.Write(bytes)

	scanner := bufio.NewScanner(&p.buf)

	for scanner.Scan() {
		line := scanner.Text()

		log.Debugf(p.ctx, "%s%s", p.prefix, line)
	}

	remaining := p.buf.Bytes()
	p.buf.Reset()
	p.buf.Write(remaining)

	return len(bytes), nil
}
