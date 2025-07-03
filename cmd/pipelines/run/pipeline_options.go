// Copied from bundle/run/pipeline_options.go and adapted for pipelines use.
package run

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	flag "github.com/spf13/pflag"
)

// PipelineOptions defines options for running a pipeline.
type PipelineOptions struct {
	// Perform a full graph run.
	RefreshAll bool

	// List of tables to run.
	Refresh []string

	// Perform a full graph reset and recompute.
	FullRefreshAll bool

	// List of tables to reset and recompute.
	FullRefresh []string
}

func (o *PipelineOptions) Define(fs *flag.FlagSet) {
	fs.BoolVar(&o.RefreshAll, "refresh-all", false, "Perform a full graph run.")
	fs.StringSliceVar(&o.Refresh, "refresh", nil, "List of tables to run.")
	fs.BoolVar(&o.FullRefreshAll, "full-refresh-all", false, "Perform a full graph reset and recompute.")
	fs.StringSliceVar(&o.FullRefresh, "full-refresh", nil, "List of tables to reset and recompute.")
}

// Validate returns if the combination of options is valid.
func (o *PipelineOptions) Validate(pipeline *resources.Pipeline) error {
	var set []string
	if o.RefreshAll {
		set = append(set, "--refresh-all")
	}
	if len(o.Refresh) > 0 {
		set = append(set, "--refresh")
	}
	if o.FullRefreshAll {
		set = append(set, "--full-refresh-all")
	}
	if len(o.FullRefresh) > 0 {
		set = append(set, "--full-refresh")
	}
	if len(set) > 1 {
		return fmt.Errorf("pipeline run arguments are mutually exclusive (got %s)", strings.Join(set, ", "))
	}
	return nil
}

func (o *PipelineOptions) toPayload(pipeline *resources.Pipeline, pipelineID string) (*pipelines.StartUpdate, error) {
	if err := o.Validate(pipeline); err != nil {
		return nil, err
	}
	payload := &pipelines.StartUpdate{
		PipelineId:           pipelineID,
		RefreshSelection:     o.Refresh,
		FullRefresh:          o.FullRefreshAll,
		FullRefreshSelection: o.FullRefresh,
	}
	return payload, nil
}
