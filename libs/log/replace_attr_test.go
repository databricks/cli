package log

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testReplaceA(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "foo" {
		return slog.Int("foo", int(a.Value.Int64())+1)
	}
	return a
}

func TestReplaceAttrGroup(t *testing.T) {
	var foo, bar, out slog.Attr

	fn := ReplaceAttrFunctions{
		testReplaceA,
		testReplaceA,
	}

	foo = slog.Int("foo", 0)
	out = fn.ReplaceAttr([]string{}, foo)
	assert.EqualValues(t, 2, out.Value.Int64())

	bar = slog.Int("bar", 0)
	out = fn.ReplaceAttr([]string{}, bar)
	assert.EqualValues(t, 0, out.Value.Int64())
}
