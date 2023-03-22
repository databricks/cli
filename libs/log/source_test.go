package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestReplaceSourceAttrSourceKey(t *testing.T) {
	attr := slog.String(slog.SourceKey, "bricks/bundle/phases/phase.go:30")
	out := ReplaceSourceAttr([]string{}, attr)
	assert.Equal(t, "phase.go:30", out.Value.String())
}

func TestReplaceSourceAttrOtherKey(t *testing.T) {
	attr := slog.String("foo", "bar")
	out := ReplaceSourceAttr([]string{}, attr)
	assert.Equal(t, attr, out)
}
