package cmdio

import (
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestAskFailedInJsonMode(t *testing.T) {
	l := NewLogger(flags.ModeJson)
	_, err := l.AskYesOrNo("What is your spirit animal?")
	assert.ErrorContains(t, err, "question prompts are not supported in json mode")
}
