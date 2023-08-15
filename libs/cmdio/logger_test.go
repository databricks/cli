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

	_, err := AskChoice(ctx, "what is a question?", "a", []string{"b", "c", "a"})
	assert.EqualError(t, err, "question prompts are not supported in json mode")
}

func TestAskChoiceDefaultValueAbsent(t *testing.T) {
	l := NewLogger(flags.ModeJson)
	ctx := NewContext(context.Background(), l)

	// Expect error that default value is missing from choices.
	_, err := AskChoice(ctx, "what is a question?", "a", []string{"b", "c", "d"})
	assert.EqualError(t, err, "failed to find default value \"a\" among choices: []string{\"b\", \"c\", \"d\"}")
}
