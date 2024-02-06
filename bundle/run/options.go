package run

import (
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/spf13/cobra"
)

type Options struct {
	Job      JobOptions
	Pipeline PipelineOptions
	NoWait   bool
}

func (o *Options) Define(cmd *cobra.Command) {
	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	jobGroup := wrappedCmd.AddFlagGroup("Job")
	o.Job.Define(jobGroup.FlagSet())

	pipelineGroup := wrappedCmd.AddFlagGroup("Pipeline")
	o.Pipeline.Define(pipelineGroup.FlagSet())
}
