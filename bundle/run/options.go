package run

import flag "github.com/spf13/pflag"

type Options struct {
	Job      JobOptions
	Pipeline PipelineOptions
}

func (o *Options) Define(fs *flag.FlagSet) {
	o.Job.Define(fs)
	o.Pipeline.Define(fs)
}
