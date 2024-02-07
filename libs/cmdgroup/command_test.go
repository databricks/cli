package cmdgroup

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestCommandFlagGrouping(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test [flags]",
		Short: "test command",
		Run: func(cmd *cobra.Command, args []string) {
			// Do nothing
		},
	}

	wrappedCmd := NewCommandWithGroupFlag(cmd)
	jobGroup := wrappedCmd.AddFlagGroup("Job")
	fs := jobGroup.FlagSet()
	fs.String("job-name", "", "Name of the job")
	fs.String("job-type", "", "Type of the job")

	pipelineGroup := wrappedCmd.AddFlagGroup("Pipeline")
	fs = pipelineGroup.FlagSet()
	fs.String("pipeline-name", "", "Name of the pipeline")
	fs.String("pipeline-type", "", "Type of the pipeline")

	cmd.Flags().BoolP("bool", "b", false, "Bool flag")
	wrappedCmd.RefreshFlags()

	buf := bytes.NewBuffer(nil)
	cmd.SetOutput(buf)
	cmd.Usage()

	expected := `Usage:
  test [flags]

Job Flags:
      --job-name string   Name of the job
      --job-type string   Type of the job

Pipeline Flags:
      --pipeline-name string   Name of the pipeline
      --pipeline-type string   Type of the pipeline

Flags:
  -b, --bool   Bool flag`
	require.Equal(t, expected, buf.String())

	require.NotNil(t, cmd.Flags().Lookup("job-name"))
	require.NotNil(t, cmd.Flags().Lookup("job-type"))
	require.NotNil(t, cmd.Flags().Lookup("pipeline-name"))
	require.NotNil(t, cmd.Flags().Lookup("pipeline-type"))
	require.NotNil(t, cmd.Flags().Lookup("bool"))
}
