package run

import (
	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/spf13/cobra"
)

type Options struct {
	Job      JobOptions
	Pipeline PipelineOptions
	NoWait   bool
}

func (o *Options) Define(cmd *cobra.Command, cliType clis.CLIType) {
	if cliType == clis.DLT {
		// Only show the DLT flags, and don't group them
		o.Pipeline.Define(cmd.Flags())
		return
	}

	jobGroup := cmdgroup.NewFlagGroup("Job")
	o.Job.DefineJobOptions(jobGroup.FlagSet())

	jobTaskGroup := cmdgroup.NewFlagGroup("Job Task")
	jobTaskGroup.SetDescription(`Note: please prefer use of job-level parameters (--param) over task-level parameters.
For more information, see https://docs.databricks.com/en/workflows/jobs/create-run-jobs.html#pass-parameters-to-a-databricks-job-task`)
	o.Job.DefineTaskOptions(jobTaskGroup.FlagSet())

	pipelineGroup := cmdgroup.NewFlagGroup("DLT")
	o.Pipeline.Define(pipelineGroup.FlagSet())

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(jobGroup)
	wrappedCmd.AddFlagGroup(jobTaskGroup)
	wrappedCmd.AddFlagGroup(pipelineGroup)
}
