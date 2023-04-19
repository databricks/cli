package cmdio

import (
	"testing"

	"github.com/databricks/bricks/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestAskFailedInJsonMode(t *testing.T) {
	l := NewLogger(flags.ModeJson)
	_, err := l.Ask("What is your spirit animal?")
	assert.ErrorContains(t, err, "question prompts are not supported in json mode")
}
