package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertsCreateErrWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "alerts", "create")
	assert.Equal(t, "provide command input in JSON format by specifying --json option", err.Error())
}
