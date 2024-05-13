package python

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/databricks/cli/libs/log"
)

type stderrLogger struct {
	ctx context.Context
	buf bytes.Buffer
}

type level string

const levelError level = "ERROR"
const levelInfo level = "INFO"
const levelDebug level = "DEBUG"
const levelWarning level = "WARNING"

type logEntry struct {
	Level   level  `json:"level,omitempty"`
	Message string `json:"message"`
}

// NewStderrLogger creates a new io.Writer that parses structured logs from stderr
// and logs them using CLI logger with appropriate log level.
func NewStderrLogger(ctx context.Context) io.Writer {
	return &stderrLogger{
		ctx: ctx,
		buf: bytes.Buffer{},
	}
}

func (p *stderrLogger) Write(bytes []byte) (n int, err error) {
	p.buf.Write(bytes)

	scanner := bufio.NewScanner(&p.buf)

	for scanner.Scan() {
		line := scanner.Text()

		if parsed, ok := parseLogEntry(line); ok {
			p.writeLogEntry(parsed)
		} else {
			log.Debugf(p.ctx, "stderr: %s", line)
		}
	}

	remaining := p.buf.Bytes()
	p.buf.Reset()
	p.buf.Write(remaining)

	return len(bytes), nil
}

func (p *stderrLogger) writeLogEntry(entry logEntry) {
	switch entry.Level {
	case levelInfo:
		log.Infof(p.ctx, "%s", entry.Message)
	case levelError:
		log.Errorf(p.ctx, "%s", entry.Message)
	case levelWarning:
		log.Warnf(p.ctx, "%s", entry.Message)
	case levelDebug:
		log.Debugf(p.ctx, "%s", entry.Message)
	default:
		log.Debugf(p.ctx, "%s", entry.Message)
	}
}

func parseLogEntry(line string) (logEntry, bool) {
	if !strings.HasPrefix(line, "{") {
		return logEntry{}, false
	}

	if !strings.HasSuffix(line, "}") {
		return logEntry{}, false
	}

	var out logEntry

	err := json.Unmarshal([]byte(line), &out)

	if err != nil {
		return logEntry{}, false
	}

	return out, true
}
