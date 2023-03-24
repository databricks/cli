package run

import (
	"github.com/databricks/bricks/libs/flags"
	flag "github.com/spf13/pflag"
)

type Options struct {
	Job            JobOptions
	Pipeline       PipelineOptions
	ProgressFormat flags.ProgressLogFormat
}

func (o *Options) Define(fs *flag.FlagSet) {
	o.Job.Define(fs)
	o.Pipeline.Define(fs)

	o.ProgressFormat = flags.NewProgressLogFormat()
	fs.Var(&o.ProgressFormat, "progress-format", "format for progress logs (append, inplace, json)")
}
