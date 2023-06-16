package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertsCreateErrWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "alerts", "create")
	assert.Equal(t, "accepts 3 arg(s), received 0", err.Error())
}
