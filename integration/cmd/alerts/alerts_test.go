package alerts_test

import (
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/assert"
)

func TestAlertsCreateErrWhenNoArguments(t *testing.T) {
	_, _, err := testcli.RequireErrorRun(t, "alerts-legacy", "create")
	assert.Equal(t, "please provide command input in JSON format by specifying the --json flag", err.Error())
}
