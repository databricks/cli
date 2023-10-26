package cmdio

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestAskFailedInJsonMode(t *testing.T) {
	l := NewLogger(flags.ModeJson)
	_, err := l.Ask("What is your spirit animal?", "")
	assert.ErrorContains(t, err, "question prompts are not supported in json mode")
}

func TestAskChoiceFailsInJsonMode(t *testing.T) {
	l := NewLogger(flags.ModeJson)
	ctx := NewContext(context.Background(), l)

	_, err := AskSelect(ctx, "what is a question?", []string{"b", "c", "a"})
	assert.EqualError(t, err, "question prompts are not supported in json mode")
}

func TestSplitAtLastNewLine(t *testing.T) {
	first, last := splitAtLastNewLine("hello\nworld")
	assert.Equal(t, "hello\n", first)
	assert.Equal(t, "world", last)

	first, last = splitAtLastNewLine("hello\r\nworld")
	assert.Equal(t, "hello\r\n", first)
	assert.Equal(t, "world", last)

	first, last = splitAtLastNewLine("hello world")
	assert.Equal(t, "", first)
	assert.Equal(t, "hello world", last)

	first, last = splitAtLastNewLine("hello\nworld\n")
	assert.Equal(t, "hello\nworld\n", first)
	assert.Equal(t, "", last)

	first, last = splitAtLastNewLine("\nhello world")
	assert.Equal(t, "\n", first)
	assert.Equal(t, "hello world", last)
}
