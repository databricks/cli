package root

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitializeErrorOnIncompatibleConfig(t *testing.T) {
	logLevel.Set("info")
	logFile.Set("stderr")
	progressFormat.Set("inplace")
	_, err := initializeProgressLogger(context.Background())
	assert.ErrorContains(t, err, "inplace progress logging cannot be used when log-file is stderr")
}

func TestNoErrorOnDisabledLogLevel(t *testing.T) {
	logLevel.Set("disabled")
	logFile.Set("stderr")
	progressFormat.Set("inplace")
	_, err := initializeProgressLogger(context.Background())
	assert.NoError(t, err)
}

func TestNoErrorOnNonStderrLogFile(t *testing.T) {
	logLevel.Set("info")
	logFile.Set("stdout")
	progressFormat.Set("inplace")
	_, err := initializeProgressLogger(context.Background())
	assert.NoError(t, err)
}
