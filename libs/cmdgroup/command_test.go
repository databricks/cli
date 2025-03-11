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

	parent := &cobra.Command{
		Use: "parent",
	}

	parent.PersistentFlags().String("global", "", "Global flag")
	parent.AddCommand(cmd)

	wrappedCmd := NewCommandWithGroupFlag(cmd)
	jobGroup := NewFlagGroup("Job")
	jobGroup.SetDescription("Description.")
	fs := jobGroup.FlagSet()
	fs.String("job-name", "", "Name of the job")
	fs.String("job-type", "", "Type of the job")
	wrappedCmd.AddFlagGroup(jobGroup)

	pipelineGroup := NewFlagGroup("Pipeline")
	fs = pipelineGroup.FlagSet()
	fs.String("pipeline-name", "", "Name of the pipeline")
	fs.String("pipeline-type", "", "Type of the pipeline")
	wrappedCmd.AddFlagGroup(pipelineGroup)

	cmd.Flags().BoolP("bool", "b", false, "Bool flag")

	buf := bytes.NewBuffer(nil)
	cmd.SetOut(buf)
	err := cmd.Usage()
	require.NoError(t, err)

	expected := `Usage:
  parent test [flags]

Job Flags:
  Description.
      --job-name string   Name of the job
      --job-type string   Type of the job

Pipeline Flags:
      --pipeline-name string   Name of the pipeline
      --pipeline-type string   Type of the pipeline

Flags:
  -b, --bool   Bool flag

Global Flags:
      --global string   Global flag
`
	require.Equal(t, expected, buf.String())

	require.NotNil(t, cmd.Flags().Lookup("job-name"))
	require.NotNil(t, cmd.Flags().Lookup("job-type"))
	require.NotNil(t, cmd.Flags().Lookup("pipeline-name"))
	require.NotNil(t, cmd.Flags().Lookup("pipeline-type"))
	require.NotNil(t, cmd.Flags().Lookup("bool"))
}
