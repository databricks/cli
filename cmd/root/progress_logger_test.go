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
	*logFlags
	*progressLoggerFlag
}

func initializeProgressLoggerTest(t *testing.T) (
	*progressLoggerTest,
	*flags.ProgressLogFormat,
) {
	plt := &progressLoggerTest{
		Command: &cobra.Command{},
	}
	plt.logFlags = initLogFlags(plt.Command)
	plt.progressLoggerFlag = initProgressLoggerFlag(plt.Command, plt.logFlags)
	return plt, &plt.ProgressLogFormat
}

func TestDefaultLoggerModeResolution(t *testing.T) {
	plt, progressFormat := initializeProgressLoggerTest(t)
	require.Equal(t, *progressFormat, flags.ModeDefault)
	ctx, err := plt.progressLoggerFlag.initializeContext(context.Background())
	require.NoError(t, err)
	logger, ok := cmdio.FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, logger.Mode, flags.ModeAppend)
}
