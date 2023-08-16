package log

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceSourceAttrSourceKey(t *testing.T) {
	attr := slog.String(slog.SourceKey, "cli/bundle/phases/phase.go:30")
	out := ReplaceSourceAttr([]string{}, attr)
	assert.Equal(t, "phase.go:30", out.Value.String())
}

func TestReplaceSourceAttrOtherKey(t *testing.T) {
	attr := slog.String("foo", "bar")
	out := ReplaceSourceAttr([]string{}, attr)
	assert.Equal(t, attr, out)
}
