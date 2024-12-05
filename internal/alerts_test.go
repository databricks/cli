package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertsCreateErrWhenNoArguments(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "alerts-legacy", "create")
	assert.Equal(t, "please provide command input in JSON format by specifying the --json flag", err.Error())
}
