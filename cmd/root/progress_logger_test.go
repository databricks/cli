package root

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type progressLoggerTest struct {
	*cobra.Command
	*LogFlags
	*ProgressLoggerFlag
}

func initializeProgressLoggerTest(t *testing.T) (
	*progressLoggerTest,
	*flags.LogLevelFlag,
	*flags.LogFileFlag,
	*flags.ProgressLogFormat,
) {
	plt := &progressLoggerTest{
		Command: &cobra.Command{},
	}
	plt.LogFlags = InitLogFlags(plt.Command)
	plt.ProgressLoggerFlag = InitProgressLoggerFlag(plt.Command, plt.LogFlags)
	return plt, &plt.LogFlags.level, &plt.LogFlags.file, &plt.ProgressLoggerFlag.ProgressLogFormat
}

func TestInitializeErrorOnIncompatibleConfig(t *testing.T) {
	plt, logLevel, logFile, progressFormat := initializeProgressLoggerTest(t)
	require.NoError(t, logLevel.Set("info"))
	require.NoError(t, logFile.Set("stderr"))
	require.NoError(t, progressFormat.Set("inplace"))
	_, err := plt.ProgressLoggerFlag.InitializeContext(context.Background())
	assert.ErrorContains(t, err, "inplace progress logging cannot be used when log-file is stderr")
}

func TestNoErrorOnDisabledLogLevel(t *testing.T) {
	plt, logLevel, logFile, progressFormat := initializeProgressLoggerTest(t)
	require.NoError(t, logLevel.Set("disabled"))
	require.NoError(t, logFile.Set("stderr"))
	require.NoError(t, progressFormat.Set("inplace"))
	_, err := plt.ProgressLoggerFlag.InitializeContext(context.Background())
	assert.NoError(t, err)
}

func TestNoErrorOnNonStderrLogFile(t *testing.T) {
	plt, logLevel, logFile, progressFormat := initializeProgressLoggerTest(t)
	require.NoError(t, logLevel.Set("info"))
	require.NoError(t, logFile.Set("stdout"))
	require.NoError(t, progressFormat.Set("inplace"))
	_, err := plt.ProgressLoggerFlag.InitializeContext(context.Background())
	assert.NoError(t, err)
}

func TestDefaultLoggerModeResolution(t *testing.T) {
	plt, _, _, progressFormat := initializeProgressLoggerTest(t)
	require.Equal(t, *progressFormat, flags.ModeDefault)
	ctx, err := plt.ProgressLoggerFlag.InitializeContext(context.Background())
	require.NoError(t, err)
	logger, ok := cmdio.FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, logger.Mode, flags.ModeAppend)
}
