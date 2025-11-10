package query

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
)

func testContextWithIO() context.Context {
	ctx := env.Set(context.Background(), "TERM", "dumb")
	in := strings.NewReader("")
	ioStreams := cmdio.NewIO(ctx, flags.OutputText, in, io.Discard, io.Discard, "", "")
	return cmdio.InContext(ctx, ioStreams)
}

func TestDetermineFormatDefaults(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(testContextWithIO())

	f, err := determineFormat(cmd, "", "")
	require.NoError(t, err)
	require.Equal(t, formatJSON, f)

	f, err = determineFormat(cmd, "", "out.json")
	require.NoError(t, err)
	require.Equal(t, formatJSON, f)
}

func TestNormalizeWaitTimeout(t *testing.T) {
	val, err := normalizeWaitTimeout(10 * time.Second)
	require.NoError(t, err)
	require.Equal(t, "10s", val)

	_, err = normalizeWaitTimeout(3 * time.Second)
	require.Error(t, err)

	val, err = normalizeWaitTimeout(0)
	require.NoError(t, err)
	require.Equal(t, "0s", val)
}
